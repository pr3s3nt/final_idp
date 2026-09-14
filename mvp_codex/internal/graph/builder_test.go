package graph_test

import (
	"strings"
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/graph"
)

func TestBuildOrdersResourceBackendFrontend(t *testing.T) {
	snapshot := testSnapshot()
	got, err := graph.NewBuilder().Build(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"resource:db", "workload:backend", "workload:frontend"}
	if strings.Join(got.TopologicalOrder, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got.TopologicalOrder, want)
	}
}

func TestBuildRejectsCycle(t *testing.T) {
	snapshot := testSnapshot()
	snapshot.ApplicationDefinition.Dependencies = append(snapshot.ApplicationDefinition.Dependencies, domain.Dependency{ID: "cycle", SourceWorkloadID: "backend", TargetType: domain.DependencyWorkload, TargetWorkloadID: "frontend"})
	_, err := graph.NewBuilder().Build(snapshot)
	if err == nil || !strings.Contains(err.Error(), "DEPENDENCY_CYCLE") {
		t.Fatalf("err = %v", err)
	}
}

func testSnapshot() domain.DeploymentInputSnapshot {
	return domain.DeploymentInputSnapshot{
		ApplicationDefinition: domain.ApplicationDefinition{ID: "app", Workloads: []domain.Workload{
			{ID: "frontend", Name: "frontend"},
			{ID: "backend", Name: "backend", ExposedOutputs: []domain.WorkloadOutputDefinition{{Name: "url", Availability: domain.OutputPlanTime}}},
		}, ResourceRequirements: []domain.ResourceRequirement{{ID: "db", Name: "database", ResourceType: "postgres"}}, Dependencies: []domain.Dependency{
			{ID: "backend-db", SourceWorkloadID: "backend", TargetType: domain.DependencyResource, TargetResourceRequirementID: "db"},
			{ID: "frontend-backend", SourceWorkloadID: "frontend", TargetType: domain.DependencyWorkload, TargetWorkloadID: "backend"},
		}},
		EnvironmentConfiguration: domain.EnvironmentConfiguration{Environment: "dev"},
	}
}
