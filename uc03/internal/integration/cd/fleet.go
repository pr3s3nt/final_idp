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
)

var gitRepoGVR = schema.GroupVersionResource{Group: "fleet.cattle.io", Version: "v1alpha1", Resource: "gitrepos"}

// fleetNamespace is where GitRepo objects live in a single-cluster install;
// Fleet wires that namespace to the cluster it runs on.
const fleetNamespace = "fleet-local"

// fleetPollingInterval is how often Fleet re-reads the repository. Fleet pulls
// on a timer rather than being poked, so this is what decides how quickly a
// published commit is picked up.
const fleetPollingInterval = "10s"

// Fleet lets the Fleet instance inside the target cluster sync the desired
// state from the application's delivery repository. Everything Fleet-specific
// stays in this file: outside it, only the neutral Status values of the CD
// abstraction are visible.
type Fleet struct{ gitDelivery }

// NewFleet builds the Fleet adapter over a delivery configuration.
func NewFleet(d Delivery) *Fleet { return &Fleet{gitDelivery{cfg: d}} }

// PublishDesiredDeploymentState applies secret objects to the cluster, then
// commits the manifests of this wave to the application's delivery repository.
// The commit SHA is the delivery reference.
func (f *Fleet) PublishDesiredDeploymentState(ctx context.Context, ds DesiredState) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ra, err := f.delivery(ctx, ds)
	if err != nil {
		return "", err
	}
	if err := f.ensureGitRepo(ctx, ds, ra); err != nil {
		return "", fmt.Errorf("prepare Fleet GitRepo: %w", err)
	}
	if err := f.applyClusterObjects(ctx, ds); err != nil {
		return "", err
	}
	sha, err := f.commitDesiredState(ctx, ra, ds)
	if err != nil {
		return "", err
	}
	if err := f.forceSync(ctx, ds); err != nil {
		return "", err
	}
	return "git:" + sha, nil
}

// forceSync asks Fleet to re-read the repository now instead of waiting for the
// next poll: raising forceSyncGeneration is how Fleet is poked.
func (f *Fleet) forceSync(ctx context.Context, ds DesiredState) error {
	c, err := f.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	repos := c.Dynamic.Resource(gitRepoGVR).Namespace(fleetNamespace)
	repo, err := repos.Get(ctx, deliveryName(ds), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	generation, _, _ := unstructured.NestedInt64(repo.Object, "spec", "forceSyncGeneration")
	if err := unstructured.SetNestedField(repo.Object, generation+1, "spec", "forceSyncGeneration"); err != nil {
		return err
	}
	_, err = repos.Update(ctx, repo, metav1.UpdateOptions{})
	return err
}

// ensureGitRepo gives Fleet the read key of this application and points it at
// the one environment path inside this application's repository.
func (f *Fleet) ensureGitRepo(ctx context.Context, ds DesiredState, ra *deliveryAccess) error {
	c, err := f.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	readKey, err := f.readKey(ctx, ra, ds.Application)
	if err != nil {
		return err
	}
	// Fleet verifies the Git host strictly and uses a single source of host
	// keys, so the recorded ones travel with the key.
	knownHosts, err := f.knownHosts()
	if err != nil {
		return fmt.Errorf("known hosts for Fleet: %w", err)
	}
	// The credential is named after the application and lives beside its
	// GitRepo, so the cluster holds one read key per application instead of one
	// key that opens every repository.
	name := "idp-delivery-" + safeDir(ds.Application)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: fleetNamespace},
		Type:       corev1.SecretTypeSSHAuth,
		Data:       map[string][]byte{corev1.SSHAuthPrivateKey: readKey, "known_hosts": knownHosts},
	}
	secrets := c.Core.CoreV1().Secrets(fleetNamespace)
	if existing, err := secrets.Get(ctx, name, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		if _, err := secrets.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		secret.ResourceVersion = existing.ResourceVersion
		if _, err := secrets.Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	repo := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "fleet.cattle.io/v1alpha1",
		"kind":       "GitRepo",
		"metadata":   map[string]any{"name": deliveryName(ds), "namespace": fleetNamespace},
		"spec": map[string]any{
			"repo":             ra.url,
			"branch":           ra.branch,
			"paths":            []any{environmentPath(ds)},
			"clientSecretName": name,
			// Everything this application delivers lands in its own namespace,
			// whatever the manifests say.
			"targetNamespace": ds.Namespace,
			"pollingInterval": fleetPollingInterval,
		},
	}}
	repos := c.Dynamic.Resource(gitRepoGVR).Namespace(fleetNamespace)
	existing, err := repos.Get(ctx, repo.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = repos.Create(ctx, repo, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	repo.SetResourceVersion(existing.GetResourceVersion())
	// Carry over the counter that forces an immediate re-read, so rewriting the
	// spec does not quietly reset it.
	if generation, ok, _ := unstructured.NestedInt64(existing.Object, "spec", "forceSyncGeneration"); ok {
		if err := unstructured.SetNestedField(repo.Object, generation, "spec", "forceSyncGeneration"); err != nil {
			return err
		}
	}
	_, err = repos.Update(ctx, repo, metav1.UpdateOptions{})
	return err
}

// WaitForDelivery waits until Fleet reports it has applied the delivered
// commit, or a later commit that contains it.
func (f *Fleet) WaitForDelivery(ctx context.Context, ds DesiredState, ref string, timeout time.Duration) error {
	sha := strings.TrimPrefix(ref, "git:")
	deadline := time.Now().Add(timeout)
	lastForce := time.Now()
	var last *Status
	for {
		st, err := f.GetCDStatus(ctx, ds)
		if err != nil {
			return err
		}
		last = st
		if st.Sync == "SYNCED" && st.Revision != "" && (st.Revision == sha || f.containsCommit(ctx, ds, st.Revision, sha)) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Fleet did not apply revision %s within %s (revision %q, sync %s, health %s: %s)",
				short(sha), timeout, short(last.Revision), last.Sync, last.Health, last.Message)
		}
		if time.Since(lastForce) > 30*time.Second {
			f.forceSync(ctx, ds)
			lastForce = time.Now()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

// GetCDStatus reads the GitRepo of this application + environment and reports
// it in the neutral terms of the CD abstraction.
func (f *Fleet) GetCDStatus(ctx context.Context, ds DesiredState) (*Status, error) {
	c, err := f.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return nil, err
	}
	repo, err := c.Dynamic.Resource(gitRepoGVR).Namespace(fleetNamespace).Get(ctx, deliveryName(ds), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return &Status{Sync: "UNKNOWN", Health: "MISSING", Message: "git repository not registered"}, nil
	}
	if err != nil {
		return nil, err
	}
	return fleetStatus(repo.Object), nil
}

// fleetStatus translates what Fleet reports into the neutral status of the CD
// abstraction. Fleet's own vocabulary does not leave this function, which is
// why it takes the raw object and is tested on its own.
func fleetStatus(obj map[string]any) *Status {
	st := &Status{Sync: "UNKNOWN", Health: "UNKNOWN"}
	st.Revision, _, _ = unstructured.NestedString(obj, "status", "commit")

	counter := func(name string) int64 {
		v, _, _ := unstructured.NestedInt64(obj, "status", "summary", name)
		return v
	}
	ready, desired := counter("ready"), counter("desiredReady")
	notReady, waitApplied, errApplied := counter("notReady"), counter("waitApplied"), counter("errApplied")
	outOfSync, modified := counter("outOfSync"), counter("modified")

	readyCondition, message := false, ""
	if conds, ok, _ := unstructured.NestedSlice(obj, "status", "conditions"); ok {
		for _, raw := range conds {
			cond, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			switch fmt.Sprint(cond["type"]) {
			case "Ready":
				readyCondition = fmt.Sprint(cond["status"]) == "True"
				if msg := fmt.Sprint(cond["message"]); msg != "" && msg != "<nil>" {
					message = msg
				}
			case "Stalled":
				if fmt.Sprint(cond["status"]) == "True" {
					if msg := fmt.Sprint(cond["message"]); msg != "" && msg != "<nil>" {
						message = msg
					}
				}
			}
		}
	}
	st.Message = message

	switch {
	case errApplied > 0:
		st.Sync, st.Health = "OUT_OF_SYNC", "DEGRADED"
	case outOfSync > 0 || modified > 0:
		st.Sync, st.Health = "OUT_OF_SYNC", "PROGRESSING"
	case readyCondition && desired > 0 && ready == desired:
		st.Sync, st.Health = "SYNCED", "HEALTHY"
	case notReady > 0 || waitApplied > 0 || desired == 0:
		st.Sync, st.Health = "SYNCING", "PROGRESSING"
	case readyCondition:
		// Fleet is done and reports nothing left to deploy.
		st.Sync, st.Health = "SYNCED", "HEALTHY"
	default:
		st.Sync, st.Health = "SYNCING", "PROGRESSING"
	}
	return st
}

// RemoveApplication deletes the GitRepo, which makes Fleet remove what it
// deployed, waits until it is gone, then removes the desired state of this
// environment from the application's repository. The repository and its key
// pair are kept: the application may be deployed again later.
func (f *Fleet) RemoveApplication(ctx context.Context, ds DesiredState) error {
	c, err := f.cfg.Kube.Clients(ds.Cluster)
	if err != nil {
		return err
	}
	repos := c.Dynamic.Resource(gitRepoGVR).Namespace(fleetNamespace)
	err = repos.Delete(ctx, deliveryName(ds), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	deadline := time.Now().Add(3 * time.Minute)
	for {
		if _, err := repos.Get(ctx, deliveryName(ds), metav1.GetOptions{}); apierrors.IsNotFound(err) {
			break
		} else if err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Fleet git repository %s was not deleted within 3m", deliveryName(ds))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	ra, err := f.delivery(ctx, ds)
	if err != nil {
		return err
	}
	return f.removeEnvironmentPath(ctx, ra, ds)
}
