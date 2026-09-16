package cd

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var applicationGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}

// ArgoCD lets the Argo CD instance inside the target cluster sync the desired
// state from the application's delivery repository. Everything about Argo CD
// stays in this file; the repository half is shared with the other adapters.
type ArgoCD struct{ gitDelivery }

// NewArgoCD builds the Argo CD adapter over a delivery configuration.
func NewArgoCD(d Delivery) *ArgoCD { return &ArgoCD{gitDelivery{cfg: d}} }

// PublishDesiredDeploymentState applies secret objects to the cluster, then
// commits the manifests of this wave to the application's delivery repository
// and asks Argo CD to look at the result. The commit SHA is the delivery
// reference.
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
	if err := a.applyClusterObjects(ctx, ds); err != nil {
		return "", err
	}
	sha, err := a.commitDesiredState(ctx, ra, ds)
	if err != nil {
		return "", err
	}
	if err := a.refresh(ctx, ds); err != nil {
		return "", err
	}
	return "git:" + sha, nil
}

func (a *ArgoCD) ensureApplication(ctx context.Context, ds DesiredState, ra *deliveryAccess) error {
	c, err := a.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	readKey, err := a.readKey(ctx, ra, ds.Application)
	if err != nil {
		return err
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
		"metadata":   map[string]any{"name": deliveryName(ds), "namespace": "argocd"},
		"spec": map[string]any{
			"project": "default",
			"source": map[string]any{
				"repoURL": ra.url, "targetRevision": ra.branch, "path": environmentPath(ds),
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
	c, err := a.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	patch := []byte(`{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}`)
	_, err = c.Dynamic.Resource(applicationGVR).Namespace("argocd").Patch(ctx, deliveryName(ds), types.MergePatchType, patch, metav1.PatchOptions{})
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
		if st.Sync == "SYNCED" && st.Revision != "" && (st.Revision == sha || a.containsCommit(ctx, ds, st.Revision, sha)) {
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

func (a *ArgoCD) GetCDStatus(ctx context.Context, ds DesiredState) (*Status, error) {
	c, err := a.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return nil, err
	}
	app, err := c.Dynamic.Resource(applicationGVR).Namespace("argocd").Get(ctx, deliveryName(ds), metav1.GetOptions{})
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

// RemoveApplication deletes the Argo CD application after its workloads were
// removed from Git and pruned, waits until Argo CD no longer has it (so no
// in-flight sync recreates the namespace), then removes the desired state of
// this environment from the application's repository.
func (a *ArgoCD) RemoveApplication(ctx context.Context, ds DesiredState) error {
	c, err := a.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	apps := c.Dynamic.Resource(applicationGVR).Namespace("argocd")
	err = apps.Delete(ctx, deliveryName(ds), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	deadline := time.Now().Add(3 * time.Minute)
	for {
		if _, err := apps.Get(ctx, deliveryName(ds), metav1.GetOptions{}); apierrors.IsNotFound(err) {
			break
		} else if err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Argo CD application %s was not deleted within 3m", deliveryName(ds))
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
	return a.removeEnvironmentPath(ctx, ra, ds)
}
