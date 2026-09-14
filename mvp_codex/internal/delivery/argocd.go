package delivery

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/worker"
)

const managedBy = "mvp-codex"

var applicationGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}

type ArgoCD struct {
	Namespace string
}

func NewArgoCD(namespace string) *ArgoCD {
	if namespace == "" {
		namespace = "argocd"
	}
	return &ArgoCD{Namespace: namespace}
}

func (a *ArgoCD) Apply(ctx context.Context, applicationName string, target domain.DeploymentContext, artifact worker.Artifact) (worker.Acknowledgment, error) {
	kubeconfigPath := target.AdapterVersions["argoKubeconfigPath"]
	kubeContext := target.AdapterVersions["argoKubeContext"]
	if kubeconfigPath == "" || kubeContext == "" {
		return worker.Acknowledgment{}, fmt.Errorf("INVALID_ARGO_TARGET: explicit argoKubeconfigPath and argoKubeContext are required")
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeconfigPath
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{CurrentContext: kubeContext}).ClientConfig()
	if err != nil {
		return worker.Acknowledgment{}, fmt.Errorf("load Argo CD Kubernetes client config: %w", err)
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return worker.Acknowledgment{}, fmt.Errorf("create Argo CD Kubernetes client: %w", err)
	}
	return a.applyWithClient(ctx, client, applicationName, target, artifact)
}

func (a *ArgoCD) applyWithClient(ctx context.Context, client dynamic.Interface, applicationName string, target domain.DeploymentContext, artifact worker.Artifact) (worker.Acknowledgment, error) {
	if applicationName == "" || target.TargetID == "" || target.Namespace == "" {
		return worker.Acknowledgment{}, fmt.Errorf("INVALID_ARGO_APPLICATION: application name and target identity are required")
	}
	if !strings.HasPrefix(artifact.URI, "oci://") || !strings.HasPrefix(artifact.Digest, "sha256:") {
		return worker.Acknowledgment{}, fmt.Errorf("INVALID_ARTIFACT: Argo CD requires an OCI URI and sha256 digest")
	}
	destinationServer := target.AdapterVersions["argoDestinationServer"]
	if destinationServer == "" {
		return worker.Acknowledgment{}, fmt.Errorf("INVALID_ARGO_TARGET: explicit registered destination server is required")
	}

	desired := applicationObject(a.Namespace, applicationName, target, artifact, destinationServer)
	applications := client.Resource(applicationGVR).Namespace(a.Namespace)
	existing, err := applications.Get(ctx, applicationName, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		existing, err = applications.Create(ctx, desired, metav1.CreateOptions{})
	case err != nil:
		return worker.Acknowledgment{}, fmt.Errorf("read Argo CD Application: %w", err)
	default:
		if !ownedByScope(existing, applicationName, target.TargetID) {
			return worker.Acknowledgment{}, fmt.Errorf("DELIVERY_OWNERSHIP_CONFLICT: Application %s/%s is not owned by this scope", a.Namespace, applicationName)
		}
		desired.SetResourceVersion(existing.GetResourceVersion())
		desired.SetUID(existing.GetUID())
		existing, err = applications.Update(ctx, desired, metav1.UpdateOptions{})
	}
	if err != nil {
		return worker.Acknowledgment{}, fmt.Errorf("upsert Argo CD Application: %w", err)
	}
	reference := fmt.Sprintf("argocd://%s/%s", a.Namespace, applicationName)
	if existing.GetUID() != "" {
		reference += "?uid=" + string(existing.GetUID())
	}
	return worker.Acknowledgment{Reference: reference, Digest: artifact.Digest}, nil
}

func applicationObject(namespace, applicationName string, target domain.DeploymentContext, artifact worker.Artifact, destinationServer string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]any{
			"name":      applicationName,
			"namespace": namespace,
			"labels": map[string]any{
				"app.kubernetes.io/managed-by": managedBy,
				"idp.application-name":         applicationName,
			},
			"annotations": map[string]any{
				"idp.deployment-target": target.TargetID,
			},
		},
		"spec": map[string]any{
			"project": "default",
			"source": map[string]any{
				"repoURL":        artifact.URI,
				"targetRevision": artifact.Digest,
				"path":           ".",
			},
			"destination": map[string]any{
				"server":    destinationServer,
				"namespace": target.Namespace,
			},
			"syncPolicy": map[string]any{
				"automated":   map[string]any{"prune": true, "selfHeal": true},
				"syncOptions": []any{"CreateNamespace=false"},
			},
		},
	}}
}

func ownedByScope(application *unstructured.Unstructured, name, targetID string) bool {
	labels := application.GetLabels()
	annotations := application.GetAnnotations()
	return labels["app.kubernetes.io/managed-by"] == managedBy &&
		labels["idp.application-name"] == name &&
		annotations["idp.deployment-target"] == targetID
}
