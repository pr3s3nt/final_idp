package render_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/configuration"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/render"
)

func TestScoreGeneratesPinnedAnnotatedDeployment(t *testing.T) {
	if _, err := exec.LookPath("score-k8s"); err != nil {
		t.Skip("score-k8s is not installed")
	}
	snapshot := domain.DeploymentInputSnapshot{ApplicationDefinition: domain.ApplicationDefinition{ID: "app", Name: "notes", Workloads: []domain.Workload{{ID: "backend", Name: "backend", ImageRepository: "registry.example/backend", Port: 8080}}}, EnvironmentConfiguration: domain.EnvironmentConfiguration{Environment: "dev"}, RenderContext: domain.DeploymentContext{Namespace: "idp-demo-dev", NamingPolicy: "mvp-v1"}, Images: []domain.WorkloadImage{{WorkloadID: "backend", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}
	resolved := configuration.Resolved{Workloads: map[string]configuration.WorkloadConfiguration{"backend": {Environment: map[string]string{"PORT": "8080"}}}}
	manifest, err := render.NewScore().Generate(context.Background(), "deployment-1", snapshot, resolved)
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	for _, expected := range []string{"kind: Deployment", "namespace: idp-demo-dev", "idp.deployment-id: deployment-1", "registry.example/backend@sha256:aaaaaaaa"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("manifest does not contain %q:\n%s", expected, text)
		}
	}
}

func TestScoreAddsExternalSecretReferenceWithoutCredentialValue(t *testing.T) {
	if _, err := exec.LookPath("score-k8s"); err != nil {
		t.Skip("score-k8s is not installed")
	}
	snapshot := domain.DeploymentInputSnapshot{
		ApplicationDefinition:    domain.ApplicationDefinition{ID: "app", Name: "notes", Workloads: []domain.Workload{{ID: "backend", Name: "backend", ImageRepository: "registry.example/backend", Port: 8080}}},
		EnvironmentConfiguration: domain.EnvironmentConfiguration{Environment: "dev"},
		RenderContext: domain.DeploymentContext{
			CloudProvider: "aws", Namespace: "idp-demo-dev", NamingPolicy: "mvp-v1",
			AdapterVersions: map[string]string{"externalSecretsApiVersion": "external-secrets.io/v1beta1", "clusterSecretStore": "aws-secrets-manager"},
		},
		Images: []domain.WorkloadImage{{WorkloadID: "backend", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
	}
	arn := "arn:aws:secretsmanager:ap-southeast-1:123456789012:secret:rds!cluster-demo"
	resolved := configuration.Resolved{
		Workloads: map[string]configuration.WorkloadConfiguration{"backend": {
			Environment: map[string]string{},
			Secrets:     []configuration.MaterializedSecret{{Name: "DB_PASSWORD", SecretName: "notes-db", Key: "password"}},
		}},
		Materializations: []configuration.SecretMaterialization{{Provider: "aws", Namespace: "idp-demo-dev", SecretName: "notes-db", SecretKey: "password", RemoteReference: arn, RemoteProperty: "password"}},
	}
	manifest, err := render.NewScore().Generate(context.Background(), "deployment-1", snapshot, resolved)
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	for _, expected := range []string{"kind: ExternalSecret", "name: aws-secrets-manager", arn, "property: password", "secretKeyRef:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("manifest does not contain %q:\n%s", expected, text)
		}
	}
}
