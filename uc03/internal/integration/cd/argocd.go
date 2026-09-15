package cd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pr3s3nt/final_idp/uc03/internal/integration/kubernetes"
)

var applicationGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}

// ArgoCD publishes desired state as commits to a GitOps repository and lets
// the Argo CD instance inside the target cluster sync it.
type ArgoCD struct {
	RepoURL        string // git@github.com:owner/repo.git
	Branch         string
	WriteKeyFile   string
	ReadKeyFile    string
	KnownHostsFile string
	WorkDir        string
	Kube           *kubernetes.Adapter

	mu sync.Mutex
}

func (a *ArgoCD) appName(ds DesiredState) string {
	return fmt.Sprintf("%s-%s", ds.Application, strings.ToLower(ds.Environment))
}

func (a *ArgoCD) appPath(ds DesiredState) string {
	return fmt.Sprintf("%s/%s", ds.Target, a.appName(ds))
}

func (a *ArgoCD) git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes -o UserKnownHostsFile=%s -o StrictHostKeyChecking=yes", a.WriteKeyFile, a.KnownHostsFile),
		"GIT_AUTHOR_NAME=idp-uc03", "GIT_AUTHOR_EMAIL=idp-uc03@users.noreply.github.com",
		"GIT_COMMITTER_NAME=idp-uc03", "GIT_COMMITTER_EMAIL=idp-uc03@users.noreply.github.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *ArgoCD) checkout(ctx context.Context) (string, error) {
	dir := filepath.Join(a.WorkDir, "repo")
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		if err := os.MkdirAll(a.WorkDir, 0o700); err != nil {
			return "", err
		}
		if _, err := a.git(ctx, a.WorkDir, "clone", "--branch", a.Branch, a.RepoURL, "repo"); err != nil {
			return "", err
		}
	}
	if _, err := a.git(ctx, dir, "fetch", "origin", a.Branch); err != nil {
		return "", err
	}
	if _, err := a.git(ctx, dir, "reset", "--hard", "origin/"+a.Branch); err != nil {
		return "", err
	}
	return dir, nil
}

// PublishDesiredDeploymentState applies secret objects to the cluster, then
// commits the manifests of upserted workloads (and deletes removed ones) under
// <target>/<app>-<env>/workloads/<workload-id>/ and pushes. The commit SHA is
// the delivery reference.
func (a *ArgoCD) PublishDesiredDeploymentState(ctx context.Context, ds DesiredState) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.ensureApplication(ctx, ds); err != nil {
		return "", fmt.Errorf("prepare Argo CD application: %w", err)
	}
	labels := map[string]string{"idp.dev/application": ds.Application, "idp.dev/environment": strings.ToLower(ds.Environment)}
	if err := a.Kube.EnsureNamespace(ctx, ds.Cluster, ds.Namespace, labels); err != nil {
		return "", err
	}
	for _, w := range ds.Upsert {
		var keep []string
		for _, s := range w.Secrets {
			if err := a.Kube.ApplySecret(ctx, ds.Cluster, ds.Namespace, s.Name, s.Labels, s.Data); err != nil {
				return "", fmt.Errorf("apply secret %s: %w", s.Name, err)
			}
			keep = append(keep, s.Name)
		}
		if err := a.Kube.PruneWorkloadSecrets(ctx, ds.Cluster, ds.Namespace, w.WorkloadID, keep); err != nil {
			return "", fmt.Errorf("prune secrets of workload %s: %w", w.WorkloadID, err)
		}
	}

	var sha string
	for attempt := 1; ; attempt++ {
		dir, err := a.checkout(ctx)
		if err != nil {
			return "", err
		}
		base := filepath.Join(dir, filepath.FromSlash(a.appPath(ds)), "workloads")
		for _, w := range ds.Upsert {
			wd := filepath.Join(base, w.WorkloadID)
			if err := os.MkdirAll(wd, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(wd, "manifests.yaml"), w.Manifests, 0o644); err != nil {
				return "", err
			}
		}
		for _, id := range ds.Remove {
			if err := os.RemoveAll(filepath.Join(base, id)); err != nil {
				return "", err
			}
		}
		// Git does not keep empty directories; a fixed marker keeps the
		// application path valid when every workload has been removed.
		marker := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: idp-desired-state
  namespace: %s
  labels:
    app.kubernetes.io/managed-by: idp
data:
  application: %s
  environment: %s
`, ds.Namespace, ds.Application, strings.ToLower(ds.Environment))
		if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(base), "idp-app.yaml"), []byte(marker), 0o644); err != nil {
			return "", err
		}
		if _, err := a.git(ctx, dir, "add", "-A"); err != nil {
			return "", err
		}
		if status, _ := a.git(ctx, dir, "status", "--porcelain"); status != "" {
			msg := fmt.Sprintf("%s %s: deployment %s wave %d", ds.Application, ds.Environment, ds.DeploymentID, ds.Wave)
			if _, err := a.git(ctx, dir, "commit", "-m", msg); err != nil {
				return "", err
			}
			if _, err := a.git(ctx, dir, "push", "origin", "HEAD:"+a.Branch); err != nil {
				if attempt < 3 {
					continue // another publish moved the branch; rebuild on top of it
				}
				return "", err
			}
		}
		if sha, err = a.git(ctx, dir, "rev-parse", "HEAD"); err != nil {
			return "", err
		}
		break
	}
	if err := a.refresh(ctx, ds); err != nil {
		return "", err
	}
	return "git:" + sha, nil
}

func (a *ArgoCD) ensureApplication(ctx context.Context, ds DesiredState) error {
	c, err := a.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	readKey, err := os.ReadFile(a.ReadKeyFile)
	if err != nil {
		return fmt.Errorf("read-only deploy key: %w", err)
	}
	repoSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "idp-gitops-repo", Namespace: "argocd",
			Labels: map[string]string{"argocd.argoproj.io/secret-type": "repository"}},
		StringData: map[string]string{"type": "git", "url": a.RepoURL, "sshPrivateKey": string(readKey)},
	}
	secrets := c.Core.CoreV1().Secrets("argocd")
	if existing, err := secrets.Get(ctx, repoSecret.Name, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		if _, err := secrets.Create(ctx, repoSecret, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		repoSecret.ResourceVersion = existing.ResourceVersion
		if _, err := secrets.Update(ctx, repoSecret, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	app := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata":   map[string]any{"name": a.appName(ds), "namespace": "argocd"},
		"spec": map[string]any{
			"project": "default",
			"source": map[string]any{
				"repoURL": a.RepoURL, "targetRevision": a.Branch, "path": a.appPath(ds),
				"directory": map[string]any{"recurse": true},
			},
			"destination": map[string]any{"server": "https://kubernetes.default.svc", "namespace": ds.Namespace},
			"syncPolicy": map[string]any{
				"automated":   map[string]any{"prune": true, "selfHeal": true, "allowEmpty": true},
				"syncOptions": []any{"CreateNamespace=true"},
			},
		},
	}}
	apps := c.Dynamic.Resource(applicationGVR).Namespace("argocd")
	existing, err := apps.Get(ctx, app.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = apps.Create(ctx, app, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	app.SetResourceVersion(existing.GetResourceVersion())
	_, err = apps.Update(ctx, app, metav1.UpdateOptions{})
	return err
}

func (a *ArgoCD) refresh(ctx context.Context, ds DesiredState) error {
	c, err := a.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	patch := []byte(`{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}`)
	_, err = c.Dynamic.Resource(applicationGVR).Namespace("argocd").Patch(ctx, a.appName(ds), types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

// WaitForDelivery waits until Argo CD reports the application Synced at the
// delivered commit, or at a later commit that contains it.
func (a *ArgoCD) WaitForDelivery(ctx context.Context, ds DesiredState, ref string, timeout time.Duration) error {
	sha := strings.TrimPrefix(ref, "git:")
	deadline := time.Now().Add(timeout)
	lastRefresh := time.Now()
	var last *Status
	for {
		st, err := a.GetCDStatus(ctx, ds)
		if err != nil {
			return err
		}
		last = st
		if st.Sync == "SYNCED" && st.Revision != "" && (st.Revision == sha || a.contains(ctx, st.Revision, sha)) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Argo CD did not sync revision %s within %s (revision %q, sync %s, health %s: %s)",
				short(sha), timeout, short(last.Revision), last.Sync, last.Health, last.Message)
		}
		if time.Since(lastRefresh) > 30*time.Second {
			a.refresh(ctx, ds)
			lastRefresh = time.Now()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func (a *ArgoCD) contains(ctx context.Context, revision, sha string) bool {
	dir := filepath.Join(a.WorkDir, "repo")
	a.git(ctx, dir, "fetch", "origin", a.Branch)
	_, err := a.git(ctx, dir, "merge-base", "--is-ancestor", sha, revision)
	return err == nil
}

func (a *ArgoCD) GetCDStatus(ctx context.Context, ds DesiredState) (*Status, error) {
	c, err := a.Kube.Clients(ds.Cluster)
	if err != nil {
		return nil, err
	}
	app, err := c.Dynamic.Resource(applicationGVR).Namespace("argocd").Get(ctx, a.appName(ds), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return &Status{Sync: "UNKNOWN", Health: "MISSING", Message: "application not created"}, nil
	}
	if err != nil {
		return nil, err
	}
	st := &Status{Sync: "UNKNOWN", Health: "UNKNOWN"}
	st.Revision, _, _ = unstructured.NestedString(app.Object, "status", "sync", "revision")
	sync, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
	health, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
	phase, _, _ := unstructured.NestedString(app.Object, "status", "operationState", "phase")
	st.Message, _, _ = unstructured.NestedString(app.Object, "status", "operationState", "message")
	switch sync {
	case "Synced":
		st.Sync = "SYNCED"
	case "OutOfSync":
		st.Sync = "OUT_OF_SYNC"
	}
	if phase == "Running" {
		st.Sync = "SYNCING"
	}
	switch health {
	case "Healthy":
		st.Health = "HEALTHY"
	case "Progressing":
		st.Health = "PROGRESSING"
	case "Degraded":
		st.Health = "DEGRADED"
	case "Missing":
		st.Health = "MISSING"
	}
	if conds, ok, _ := unstructured.NestedSlice(app.Object, "status", "conditions"); ok && len(conds) > 0 {
		if m, ok := conds[0].(map[string]any); ok {
			st.Message = fmt.Sprint(m["message"])
		}
	}
	return st, nil
}

// RemoveApplication deletes the application after its workloads were removed
// from Git and pruned.
func (a *ArgoCD) RemoveApplication(ctx context.Context, ds DesiredState) error {
	c, err := a.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	err = c.Dynamic.Resource(applicationGVR).Namespace("argocd").Delete(ctx, a.appName(ds), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

var _ = errors.New
