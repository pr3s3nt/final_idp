package platform_test

import (
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
)

func TestPrepareIsStableAndMapsRequirementToTerraform(t *testing.T) {
	snapshot := domain.DeploymentInputSnapshot{
		ApplicationDefinition:    domain.ApplicationDefinition{ID: "app", Name: "notes", Workloads: []domain.Workload{{ID: "backend", Name: "backend"}}, ResourceRequirements: []domain.ResourceRequirement{{ID: "db", Name: "database", ResourceType: "postgres"}}, Dependencies: []domain.Dependency{{ID: "backend-db", SourceWorkloadID: "backend", TargetType: domain.DependencyResource, TargetResourceRequirementID: "db"}}},
		EnvironmentConfiguration: domain.EnvironmentConfiguration{ID: "config", Environment: "dev"},
		RenderContext:            domain.DeploymentContext{TargetID: "aws-demo", CloudProvider: "aws", Region: "ap-southeast-1", Namespace: "idp-demo-dev", AdapterVersions: map[string]string{"terraform": "1.9.8"}},
		Images:                   []domain.WorkloadImage{{WorkloadID: "backend", Tag: "mvp", Digest: "sha256:abc"}},
	}
	definitions := []domain.ResourceDefinition{
		{ID: "postgres-aurora-v1", Name: "postgres-aurora", ResourceType: "postgres", ProvisionerReference: "terraform://modules/postgres-aurora@v1", SupportedContexts: []domain.ContextSelector{{CloudProvider: "aws"}}, DefaultParameters: map[string]any{"databaseName": "notes"}, AllowedOverrides: map[string]any{}, ExposedOutputs: []string{"port", "host"}},
		{ID: "postgres-kind-v1", Name: "postgres-kind", ResourceType: "postgres", ProvisionerReference: "terraform://modules/postgres-kubernetes@v1", SupportedContexts: []domain.ContextSelector{{CloudProvider: "local"}}, DefaultParameters: map[string]any{"storageMi": int64(1024)}, AllowedOverrides: map[string]any{}, ExposedOutputs: []string{"port", "host"}},
	}
	engine := platform.NewEngine("/durable/resources")
	first, err := engine.Prepare(snapshot, definitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.Prepare(snapshot, definitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Plan.Fingerprint != second.Plan.Fingerprint {
		t.Fatalf("unstable plan fingerprint: %s != %s", first.Plan.Fingerprint, second.Plan.Fingerprint)
	}
	if len(first.Plan.Items) != 1 || first.Plan.Items[0].Action != domain.PlanCreate {
		t.Fatalf("plan = %#v", first.Plan)
	}
	if first.Plan.Items[0].ProvisionerReference != "terraform://modules/postgres-aurora@v1" {
		t.Fatalf("provisioner = %q", first.Plan.Items[0].ProvisionerReference)
	}
}

func TestPrepareMapsKindToKubernetesPostgres(t *testing.T) {
	snapshot := domain.DeploymentInputSnapshot{
		ApplicationDefinition:    domain.ApplicationDefinition{ID: "app", Name: "notes", Workloads: []domain.Workload{{ID: "backend", Name: "backend"}}, ResourceRequirements: []domain.ResourceRequirement{{ID: "db", Name: "database", ResourceType: "postgres"}}, Dependencies: []domain.Dependency{{ID: "backend-db", SourceWorkloadID: "backend", TargetType: domain.DependencyResource, TargetResourceRequirementID: "db"}}},
		EnvironmentConfiguration: domain.EnvironmentConfiguration{ID: "config", Environment: "dev"},
		RenderContext:            domain.DeploymentContext{TargetID: "kind-dev", CloudProvider: "local", Namespace: "idp-demo-dev", AdapterVersions: map[string]string{"terraform": "1.9.8"}},
	}
	definitions := []domain.ResourceDefinition{
		{ID: "aurora", Name: "aurora", ResourceType: "postgres", ProvisionerReference: "terraform://modules/postgres-aurora@v1", SupportedContexts: []domain.ContextSelector{{CloudProvider: "aws"}}, DefaultParameters: map[string]any{}, AllowedOverrides: map[string]any{}, ExposedOutputs: []string{"host"}},
		{ID: "kind", Name: "kind", ResourceType: "postgres", ProvisionerReference: "terraform://modules/postgres-kubernetes@v1", SupportedContexts: []domain.ContextSelector{{CloudProvider: "local"}}, DefaultParameters: map[string]any{}, AllowedOverrides: map[string]any{}, ExposedOutputs: []string{"host"}},
	}
	prepared, err := platform.NewEngine("/durable/resources").Prepare(snapshot, definitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := prepared.Plan.Items[0].ProvisionerReference; got != "terraform://modules/postgres-kubernetes@v1" {
		t.Fatalf("provisioner = %q", got)
	}
}
