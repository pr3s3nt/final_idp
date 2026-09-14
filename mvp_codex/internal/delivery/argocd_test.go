package delivery

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/worker"
)

func TestArgoCreatesDigestPinnedApplication(t *testing.T) {
	client := fake.NewSimpleDynamicClient(runtime.NewScheme())
	adapter := NewArgoCD("argocd")
	target := domain.DeploymentContext{TargetID: "aws-demo", Namespace: "idp-demo-dev", AdapterVersions: map[string]string{"argoDestinationServer": "https://eks.example"}}
	artifact := worker.Artifact{URI: "oci://registry.example/idp/manifests", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	ack, err := adapter.applyWithClient(context.Background(), client, "idp-notes-dev", target, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Digest != artifact.Digest || !strings.HasPrefix(ack.Reference, "argocd://argocd/idp-notes-dev") {
		t.Fatalf("ack = %#v", ack)
	}
	application, err := client.Resource(applicationGVR).Namespace("argocd").Get(context.Background(), "idp-notes-dev", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _, _ := unstructured.NestedString(application.Object, "spec", "source", "targetRevision")
	destination, _, _ := unstructured.NestedString(application.Object, "spec", "destination", "server")
	if digest != artifact.Digest || destination != "https://eks.example" {
		t.Fatalf("application = %#v", application.Object)
	}
}

func TestArgoRejectsUnownedApplication(t *testing.T) {
	existing := applicationObject("argocd", "idp-notes-dev", domain.DeploymentContext{TargetID: "another-target", Namespace: "other"}, worker.Artifact{URI: "oci://old/repo", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, "https://other")
	existing.SetResourceVersion("1")
	existing.SetUID(types.UID("existing-uid"))
	client := fake.NewSimpleDynamicClient(runtime.NewScheme(), existing)
	adapter := NewArgoCD("argocd")
	target := domain.DeploymentContext{TargetID: "aws-demo", Namespace: "idp-demo-dev", AdapterVersions: map[string]string{"argoDestinationServer": "https://eks.example"}}
	_, err := adapter.applyWithClient(context.Background(), client, "idp-notes-dev", target, worker.Artifact{URI: "oci://new/repo", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err == nil || !strings.Contains(err.Error(), "DELIVERY_OWNERSHIP_CONFLICT") {
		t.Fatalf("error = %v", err)
	}
}
