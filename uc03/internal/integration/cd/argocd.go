package cd

import (
	"context"
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

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/deliveryrepo"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/kubernetes"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
)

var applicationGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}

// ArgoCD publishes desired state as commits to the delivery repository of the
// application and lets the Argo CD instance inside the target cluster sync it.
// Each application has its own repository and its own key pair: the IDP pushes
// with the write key, Argo CD reads with the read key, and no application can
// reach another application's desired state.
type ArgoCD struct {
	Provider       deliveryrepo.Provider   // prepares the repository the first time
	Registry       DeliveryRepositoryStore // remembers it afterwards
	Secrets        secretstore.Store
	Branch         string
	KnownHostsFile string
	WorkDir        string
	Kube           *kubernetes.Adapter

	mu sync.Mutex // serializes publishes, which share a working copy

	repoMu sync.Mutex
	repos  map[string]*deliveryAccess
}

// deliveryAccess is everything needed to work with one application's delivery
// repository: where it is, the working copy, and the write key materialized as
// a file because that is what ssh needs.
type deliveryAccess struct {
	url          string
	branch       string
	dir          string
	writeKeyFile string
	readKeyRef   string
}

func (a *ArgoCD) appName(ds DesiredState) string {
	return fmt.Sprintf("%s-%s", ds.Application, strings.ToLower(ds.Environment))
}

// appPath is where the desired state of one environment lives inside the
// application's own repository.
func (a *ArgoCD) appPath(ds DesiredState) string {
	return fmt.Sprintf("%s/%s", ds.Target, strings.ToLower(ds.Environment))
}

func (a *ArgoCD) branch() string {
	if a.Branch != "" {
		return a.Branch
	}
	return "main"
}

// delivery returns the delivery repository of the application, preparing it on
// the application's first deployment: the provider creates the repository and
// the key pair, and the registry records where they are.
func (a *ArgoCD) delivery(ctx context.Context, ds DesiredState) (*deliveryAccess, error) {
	a.repoMu.Lock()
	defer a.repoMu.Unlock()
	if ra, ok := a.repos[ds.ApplicationID]; ok {
		return ra, nil
	}
	if ds.ApplicationID == "" {
		return nil, fmt.Errorf("desired state without an application: no delivery repository can be resolved")
	}
	stored, err := a.Registry.Find(ctx, ds.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("look up the delivery repository of %s: %w", ds.Application, err)
	}
	if stored == nil {
		prepared, err := a.Provider.EnsureDeliveryRepository(ctx, deliveryrepo.Application{ID: ds.ApplicationID, Name: ds.Application})
		if err != nil {
			return nil, fmt.Errorf("prepare the delivery repository of %s: %w", ds.Application, err)
		}
		stored, err = a.Registry.Save(ctx, domain.DeliveryRepository{
			ApplicationID: ds.ApplicationID, RepositoryURL: prepared.URL, Branch: prepared.Branch,
			WriteKeyReference: prepared.WriteKeyReference, ReadKeyReference: prepared.ReadKeyReference,
		})
		if err != nil {
			return nil, fmt.Errorf("record the delivery repository of %s: %w", ds.Application, err)
		}
	}

	base := filepath.Join(a.WorkDir, safeDir(ds.Application))
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	key, err := a.Secrets.Get(ctx, stored.WriteKeyReference)
	if err != nil {
		return nil, fmt.Errorf("write key of %s: %w", ds.Application, err)
	}
	keyFile := filepath.Join(base, "write-key")
	if err := os.WriteFile(keyFile, key, 0o600); err != nil {
		return nil, err
	}
	if err := a.ensureKnownHosts(ctx, stored.RepositoryURL); err != nil {
		return nil, err
	}
	ra := &deliveryAccess{url: stored.RepositoryURL, branch: stored.Branch, dir: filepath.Join(base, "repo"),
		writeKeyFile: keyFile, readKeyRef: stored.ReadKeyReference}
	if ra.branch == "" {
		ra.branch = a.branch()
	}
	if a.repos == nil {
		a.repos = map[string]*deliveryAccess{}
	}
	a.repos[ds.ApplicationID] = ra
	return ra, nil
}

// ensureKnownHosts records the host key of the Git hosting the first time it is
// needed, so every later connection is checked strictly against it.
func (a *ArgoCD) ensureKnownHosts(ctx context.Context, repoURL string) error {
	if info, err := os.Stat(a.KnownHostsFile); err == nil && info.Size() > 0 {
		return nil
	}
	host := repoURL
	if _, rest, ok := strings.Cut(host, "@"); ok {
		host = rest
	}
	host, _, _ = strings.Cut(host, ":")
	if host == "" {
		return fmt.Errorf("cannot tell the Git host from %q", repoURL)
	}
	out, err := exec.CommandContext(ctx, "ssh-keyscan", "-t", "rsa,ecdsa,ed25519", host).Output()
	if err != nil || len(out) == 0 {
		return fmt.Errorf("read the host key of %s: %v", host, err)
	}
	if err := os.MkdirAll(filepath.Dir(a.KnownHostsFile), 0o700); err != nil {
		return err
	}
	return os.WriteFile(a.KnownHostsFile, out, 0o600)
}

func (a *ArgoCD) git(ctx context.Context, ra *deliveryAccess, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes -o UserKnownHostsFile=%s -o StrictHostKeyChecking=yes", ra.writeKeyFile, a.KnownHostsFile),
		"GIT_AUTHOR_NAME=idp-uc03", "GIT_AUTHOR_EMAIL=idp-uc03@users.noreply.github.com",
		"GIT_COMMITTER_NAME=idp-uc03", "GIT_COMMITTER_EMAIL=idp-uc03@users.noreply.github.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *ArgoCD) checkout(ctx context.Context, ra *deliveryAccess) (string, error) {
	if _, err := os.Stat(filepath.Join(ra.dir, ".git")); err != nil {
		if err := os.MkdirAll(filepath.Dir(ra.dir), 0o700); err != nil {
			return "", err
		}
		if _, err := a.git(ctx, ra, filepath.Dir(ra.dir), "clone", "--branch", ra.branch, ra.url, filepath.Base(ra.dir)); err != nil {
			return "", err
		}
	}
	if _, err := a.git(ctx, ra, ra.dir, "fetch", "origin", ra.branch); err != nil {
		return "", err
	}
	if _, err := a.git(ctx, ra, ra.dir, "reset", "--hard", "origin/"+ra.branch); err != nil {
		return "", err
	}
	return ra.dir, nil
}

// PublishDesiredDeploymentState applies secret objects to the cluster, then
// commits the manifests of upserted workloads (and deletes removed ones) under
// <target>/<environment>/workloads/<workload-id>/ in the application's own
// delivery repository and pushes. The commit SHA is the delivery reference.
func (a *ArgoCD) PublishDesiredDeploymentState(ctx context.Context, ds DesiredState) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ra, err := a.delivery(ctx, ds)
	if err != nil {
		return "", err
	}
	if err := a.ensureApplication(ctx, ds, ra); err != nil {
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
		dir, err := a.checkout(ctx, ra)
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
		if _, err := a.git(ctx, ra, dir, "add", "-A"); err != nil {
			return "", err
		}
		if status, _ := a.git(ctx, ra, dir, "status", "--porcelain"); status != "" {
			msg := fmt.Sprintf("%s %s: deployment %s wave %d", ds.Application, ds.Environment, ds.DeploymentID, ds.Wave)
			if _, err := a.git(ctx, ra, dir, "commit", "-m", msg); err != nil {
				return "", err
			}
			if _, err := a.git(ctx, ra, dir, "push", "origin", "HEAD:"+ra.branch); err != nil {
				if attempt < 3 {
					continue // another publish moved the branch; rebuild on top of it
				}
				return "", err
			}
		}
		if sha, err = a.git(ctx, ra, dir, "rev-parse", "HEAD"); err != nil {
			return "", err
		}
		break
	}
	if err := a.refresh(ctx, ds); err != nil {
		return "", err
	}
	return "git:" + sha, nil
}

func (a *ArgoCD) ensureApplication(ctx context.Context, ds DesiredState, ra *deliveryAccess) error {
	c, err := a.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	readKey, err := a.Secrets.Get(ctx, ra.readKeyRef)
	if err != nil {
		return fmt.Errorf("read key of %s: %w", ds.Application, err)
	}
	// The credential is named after the application, so the cluster holds one
	// read key per application instead of one key that opens every repository.
	repoSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "idp-delivery-" + safeDir(ds.Application), Namespace: "argocd",
			Labels: map[string]string{"argocd.argoproj.io/secret-type": "repository"}},
		StringData: map[string]string{"type": "git", "url": ra.url, "sshPrivateKey": string(readKey)},
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
				"repoURL": ra.url, "targetRevision": ra.branch, "path": a.appPath(ds),
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
		if st.Sync == "SYNCED" && st.Revision != "" && (st.Revision == sha || a.contains(ctx, ds, st.Revision, sha)) {
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

func (a *ArgoCD) contains(ctx context.Context, ds DesiredState, revision, sha string) bool {
	ra, err := a.delivery(ctx, ds)
	if err != nil {
		return false
	}
	a.git(ctx, ra, ra.dir, "fetch", "origin", ra.branch)
	_, err = a.git(ctx, ra, ra.dir, "merge-base", "--is-ancestor", sha, revision)
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
// from Git and pruned, waits until Argo CD no longer has it (so no in-flight
// sync recreates the namespace), then removes the desired state of this
// environment from the application's repository. The repository and its key
// pair are kept: the application may be deployed again later.
func (a *ArgoCD) RemoveApplication(ctx context.Context, ds DesiredState) error {
	c, err := a.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	apps := c.Dynamic.Resource(applicationGVR).Namespace("argocd")
	err = apps.Delete(ctx, a.appName(ds), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	deadline := time.Now().Add(3 * time.Minute)
	for {
		if _, err := apps.Get(ctx, a.appName(ds), metav1.GetOptions{}); apierrors.IsNotFound(err) {
			break
		} else if err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Argo CD application %s was not deleted within 3m", a.appName(ds))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	ra, err := a.delivery(ctx, ds)
	if err != nil {
		return err
	}
	for attempt := 1; ; attempt++ {
		dir, err := a.checkout(ctx, ra)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(dir, filepath.FromSlash(a.appPath(ds)))); err != nil {
			return err
		}
		if _, err := a.git(ctx, ra, dir, "add", "-A"); err != nil {
			return err
		}
		if status, _ := a.git(ctx, ra, dir, "status", "--porcelain"); status == "" {
			return nil
		}
		if _, err := a.git(ctx, ra, dir, "commit", "-m", fmt.Sprintf("%s %s: remove application", ds.Application, ds.Environment)); err != nil {
			return err
		}
		if _, err := a.git(ctx, ra, dir, "push", "origin", "HEAD:"+ra.branch); err != nil {
			if attempt < 3 {
				continue
			}
			return err
		}
		return nil
	}
}

// safeDir keeps an application recognisable in a path or a Kubernetes object
// name without letting its name escape the directory it belongs in.
func safeDir(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
