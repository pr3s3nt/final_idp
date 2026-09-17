// Package kubernetes is the Workload Status Provider / Kubernetes Adapter. It
// connects to the cluster built by the k8s-cluster resource using that
// resource's outputs.
package kubernetes

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"sdp/internal/domain"
)

// ClusterAccess describes how to reach a cluster, derived from k8s-cluster outputs.
type ClusterAccess struct {
	Kind        string // kind | eks
	ClusterName string
	Kubeconfig  string // kind
	Endpoint    string // eks
	CAData      string // eks, base64
	Region      string // eks
}

// AccessFromOutputs builds cluster access from the outputs of a k8s-cluster resource.
func AccessFromOutputs(o domain.Outputs) (*ClusterAccess, error) {
	a := &ClusterAccess{Kind: o["cluster_kind"].Value, ClusterName: o["cluster_name"].Value}
	switch a.Kind {
	case "kind":
		a.Kubeconfig = o["kubeconfig"].Value
		if a.Kubeconfig == "" {
			return nil, errors.New("kind cluster outputs have no kubeconfig")
		}
	case "eks":
		a.Endpoint, a.CAData, a.Region = o["endpoint"].Value, o["ca_data"].Value, o["region"].Value
		if a.Endpoint == "" || a.CAData == "" {
			return nil, errors.New("eks cluster outputs need endpoint and ca_data")
		}
	default:
		return nil, fmt.Errorf("unsupported cluster kind %q", a.Kind)
	}
	return a, nil
}

func (a *ClusterAccess) key() string {
	sum := sha256.Sum256([]byte(a.Kind + a.ClusterName + a.Kubeconfig + a.Endpoint + a.CAData))
	return hex.EncodeToString(sum[:])
}

func (a *ClusterAccess) restConfig() (*rest.Config, error) {
	switch a.Kind {
	case "kind":
		return clientcmd.RESTConfigFromKubeConfig([]byte(a.Kubeconfig))
	case "eks":
		ca, err := base64.StdEncoding.DecodeString(a.CAData)
		if err != nil {
			return nil, err
		}
		return &rest.Config{
			Host:            a.Endpoint,
			TLSClientConfig: rest.TLSClientConfig{CAData: ca},
			ExecProvider: &clientcmdapi.ExecConfig{
				APIVersion:      "client.authentication.k8s.io/v1beta1",
				Command:         "aws",
				Args:            []string{"eks", "get-token", "--cluster-name", a.ClusterName, "--region", a.Region, "--output", "json"},
				InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
			},
		}, nil
	}
	return nil, fmt.Errorf("unsupported cluster kind %q", a.Kind)
}

type Clients struct {
	Core    k8s.Interface
	Dynamic dynamic.Interface
}

// Adapter caches clients per cluster.
type Adapter struct {
	mu      sync.Mutex
	clients map[string]*Clients
}

func (ad *Adapter) Clients(a *ClusterAccess) (*Clients, error) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	if ad.clients == nil {
		ad.clients = map[string]*Clients{}
	}
	if c, ok := ad.clients[a.key()]; ok {
		return c, nil
	}
	cfg, err := a.restConfig()
	if err != nil {
		return nil, err
	}
	cfg.Timeout = 30 * time.Second
	core, err := k8s.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	c := &Clients{Core: core, Dynamic: dyn}
	ad.clients[a.key()] = c
	return c, nil
}

// DeleteNamespace removes an application namespace after a teardown; on a
// shared internal cluster nothing else removes it.
func (ad *Adapter) DeleteNamespace(ctx context.Context, a *ClusterAccess, name string) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	err = c.Core.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	deadline := time.Now().Add(3 * time.Minute)
	for {
		_, err := c.Core.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("namespace %s was not deleted within 3m", name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (ad *Adapter) EnsureNamespace(ctx context.Context, a *ClusterAccess, name string, labels map[string]string) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	_, err = c.Core.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// ApplySecret creates or replaces a Secret. Secret values go straight to the
// cluster and never into Git or the IDP database.
func (ad *Adapter) ApplySecret(ctx context.Context, a *ClusterAccess, namespace, name string, labels map[string]string, data map[string][]byte) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels}, Data: data, Type: corev1.SecretTypeOpaque}
	existing, err := c.Core.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.Core.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	secret.ResourceVersion = existing.ResourceVersion
	_, err = c.Core.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

// PruneWorkloadSecrets deletes Secrets of a workload that are not in keep, so a
// workload that no longer needs a secret (or was renamed) leaves none behind.
func (ad *Adapter) PruneWorkloadSecrets(ctx context.Context, a *ClusterAccess, namespace, workloadID string, keep []string) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	list, err := c.Core.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{LabelSelector: "idp.dev/workload-id=" + workloadID})
	if err != nil {
		return err
	}
	for _, s := range list.Items {
		kept := false
		for _, k := range keep {
			if s.Name == k {
				kept = true
			}
		}
		if kept {
			continue
		}
		if err := c.Core.CoreV1().Secrets(namespace).Delete(ctx, s.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (ad *Adapter) DeleteSecretsByLabel(ctx context.Context, a *ClusterAccess, namespace, selector string) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	err = c.Core.CoreV1().Secrets(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: selector})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// WorkloadRef identifies the Deployment of one workload revision.
type WorkloadRef struct {
	WorkloadID string
	Name       string
	Namespace  string
	Image      string
}

// WaitForWorkloadsHealthy waits until every workload's Deployment has rolled
// out the expected image: observed generation caught up, all replicas updated,
// ready and available, and no old replicas left. A pod of a previous revision
// that is still healthy does not count.
func (ad *Adapter) WaitForWorkloadsHealthy(ctx context.Context, a *ClusterAccess, workloads []WorkloadRef, timeout time.Duration) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	pending := append([]WorkloadRef{}, workloads...)
	var lastReason string
	for len(pending) > 0 {
		var next []WorkloadRef
		for _, w := range pending {
			ok, reason, err := rolledOut(ctx, c, w)
			if err != nil {
				return err
			}
			if !ok {
				next = append(next, w)
				lastReason = reason
			}
		}
		pending = next
		if len(pending) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not become healthy within %s: %s", pending[0].Name, timeout, lastReason)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return nil
}

func rolledOut(ctx context.Context, c *Clients, w WorkloadRef) (bool, string, error) {
	d, err := findDeployment(ctx, c, w)
	if err != nil {
		return false, "", err
	}
	if d == nil {
		return false, "deployment not created yet", nil
	}
	image := ""
	for _, ctr := range d.Spec.Template.Spec.Containers {
		if ctr.Name == "main" {
			image = ctr.Image
		}
	}
	if image != w.Image {
		return false, fmt.Sprintf("deployment still has image %s, expected %s", image, w.Image), nil
	}
	want := int32(1)
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}
	st := d.Status
	switch {
	case st.ObservedGeneration < d.Generation:
		return false, "rollout not observed yet", nil
	case st.UpdatedReplicas < want:
		return false, fmt.Sprintf("%d/%d replicas updated; pods: %s", st.UpdatedReplicas, want, podProblem(ctx, c, d)), nil
	case st.Replicas > st.UpdatedReplicas:
		return false, "new revision not available, old replicas still running; pods: " + podProblem(ctx, c, d), nil
	case st.AvailableReplicas < want || st.ReadyReplicas < want:
		return false, podProblem(ctx, c, d) + fmt.Sprintf(" (%d/%d available)", st.AvailableReplicas, want), nil
	}
	return true, "", nil
}

func findDeployment(ctx context.Context, c *Clients, w WorkloadRef) (*appsv1.Deployment, error) {
	list, err := c.Core.AppsV1().Deployments(w.Namespace).List(ctx, metav1.ListOptions{LabelSelector: "idp.dev/workload-id=" + w.WorkloadID})
	if err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, nil
	}
	return &list.Items[0], nil
}

// podProblem summarizes why pods are not ready (for error summaries).
func podProblem(ctx context.Context, c *Clients, d *appsv1.Deployment) string {
	sel := metav1.FormatLabelSelector(d.Spec.Selector)
	pods, err := c.Core.CoreV1().Pods(d.Namespace).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil || len(pods.Items) == 0 {
		return "no pods running"
	}
	var reasons []string
	for _, p := range pods.Items {
		for _, cs := range p.Status.ContainerStatuses {
			switch {
			case cs.State.Waiting != nil:
				reasons = append(reasons, fmt.Sprintf("%s: %s", p.Name, cs.State.Waiting.Reason))
			case cs.State.Terminated != nil:
				reasons = append(reasons, fmt.Sprintf("%s: terminated %s (exit %d)", p.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode))
			case !cs.Ready:
				reasons = append(reasons, fmt.Sprintf("%s: not ready, %d restarts", p.Name, cs.RestartCount))
			}
		}
	}
	sort.Strings(reasons)
	if len(reasons) == 0 {
		return "pods starting"
	}
	return strings.Join(reasons, "; ")
}

// WaitForWorkloadsRemoved waits until no Deployment or Service of the workloads
// is left, so dependencies are destroyed only after the workload is gone.
func (ad *Adapter) WaitForWorkloadsRemoved(ctx context.Context, a *ClusterAccess, namespace string, workloadIDs []string, timeout time.Duration) error {
	c, err := ad.Clients(a)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		left := 0
		for _, id := range workloadIDs {
			sel := metav1.ListOptions{LabelSelector: "idp.dev/workload-id=" + id}
			deps, err := c.Core.AppsV1().Deployments(namespace).List(ctx, sel)
			if err != nil {
				return err
			}
			svcs, err := c.Core.CoreV1().Services(namespace).List(ctx, sel)
			if err != nil {
				return err
			}
			left += len(deps.Items) + len(svcs.Items)
		}
		if left == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%d object(s) of removed workloads are still present after %s", left, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

// ReadWorkloadOutputs reads runtime data of a running workload. The only
// supported output is endpoint: the in-cluster URL of its Service.
func (ad *Adapter) ReadWorkloadOutputs(ctx context.Context, a *ClusterAccess, namespace, workloadID string, outputs []string) (domain.Outputs, error) {
	out := domain.Outputs{}
	if len(outputs) == 0 {
		return out, nil
	}
	c, err := ad.Clients(a)
	if err != nil {
		return nil, err
	}
	for _, name := range outputs {
		switch name {
		case "endpoint":
			svcs, err := c.Core.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: "idp.dev/workload-id=" + workloadID})
			if err != nil {
				return nil, err
			}
			if len(svcs.Items) == 0 || len(svcs.Items[0].Spec.Ports) == 0 {
				return nil, fmt.Errorf("workload %s has no Service to read endpoint from", workloadID)
			}
			s := svcs.Items[0]
			out[name] = domain.Output{Value: fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", s.Name, s.Namespace, s.Spec.Ports[0].Port)}
		default:
			return nil, fmt.Errorf("workload output %q has no reader", name)
		}
	}
	return out, nil
}

// WorkloadStatus is the live status shown when tracking a deployment.
type WorkloadStatus struct {
	Image     string `json:"image"`
	Ready     int32  `json:"ready"`
	Desired   int32  `json:"desired"`
	Available bool   `json:"available"`
	Detail    string `json:"detail,omitempty"`
}

func (ad *Adapter) GetWorkloadStatus(ctx context.Context, a *ClusterAccess, namespace, workloadID string) (*WorkloadStatus, error) {
	c, err := ad.Clients(a)
	if err != nil {
		return nil, err
	}
	d, err := findDeployment(ctx, c, WorkloadRef{WorkloadID: workloadID, Namespace: namespace})
	if err != nil || d == nil {
		return nil, err
	}
	s := &WorkloadStatus{Ready: d.Status.ReadyReplicas, Desired: 1}
	if d.Spec.Replicas != nil {
		s.Desired = *d.Spec.Replicas
	}
	for _, ctr := range d.Spec.Template.Spec.Containers {
		s.Image = ctr.Image
	}
	s.Available = d.Status.AvailableReplicas >= s.Desired
	if !s.Available {
		s.Detail = podProblem(ctx, c, d)
	}
	return s, nil
}
