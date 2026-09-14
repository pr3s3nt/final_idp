package platform

import (
	"sort"
	"time"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/catalog"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fingerprint"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/graph"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/planner"
)

type Engine struct {
	graph   *graph.Builder
	catalog *catalog.Resolver
	planner *planner.Planner
}

type Prepared struct {
	Snapshot    domain.DeploymentInputSnapshot `json:"snapshot"`
	Graph       domain.DeploymentGraph         `json:"graph"`
	Resolutions []domain.ResourceResolution    `json:"resolutions"`
	Plan        domain.InfrastructurePlan      `json:"plan"`
}

func NewEngine(stateRoot string) *Engine {
	return &Engine{graph: graph.NewBuilder(), catalog: catalog.NewResolver(), planner: planner.New(stateRoot)}
}

func (e *Engine) Prepare(snapshot domain.DeploymentInputSnapshot, definitions []domain.ResourceDefinition, instances map[string]domain.ResourceInstance) (Prepared, error) {
	normalizeSnapshot(&snapshot)
	snapshot.InputFingerprint = ""
	fp, err := snapshotFingerprint(snapshot)
	if err != nil {
		return Prepared{}, err
	}
	snapshot.InputFingerprint = fp
	deploymentGraph, err := e.graph.Build(snapshot)
	if err != nil {
		return Prepared{}, err
	}
	resolutions, err := e.catalog.Resolve(snapshot, deploymentGraph, definitions)
	if err != nil {
		return Prepared{}, err
	}
	plan, err := e.planner.Plan(snapshot, resolutions, instances)
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Snapshot: snapshot, Graph: deploymentGraph, Resolutions: resolutions, Plan: plan}, nil
}

func snapshotFingerprint(snapshot domain.DeploymentInputSnapshot) (string, error) {
	snapshot.InputFingerprint = ""
	snapshot.CreatedAt = time.Time{}
	return fingerprint.Sum(snapshot)
}

func normalizeSnapshot(snapshot *domain.DeploymentInputSnapshot) {
	if snapshot.SchemaVersion == "" {
		snapshot.SchemaVersion = domain.SnapshotSchemaVersion
	}
	sort.Slice(snapshot.ApplicationDefinition.Workloads, func(i, j int) bool {
		return snapshot.ApplicationDefinition.Workloads[i].ID < snapshot.ApplicationDefinition.Workloads[j].ID
	})
	sort.Slice(snapshot.ApplicationDefinition.ResourceRequirements, func(i, j int) bool {
		return snapshot.ApplicationDefinition.ResourceRequirements[i].ID < snapshot.ApplicationDefinition.ResourceRequirements[j].ID
	})
	sort.Slice(snapshot.ApplicationDefinition.Dependencies, func(i, j int) bool {
		return snapshot.ApplicationDefinition.Dependencies[i].ID < snapshot.ApplicationDefinition.Dependencies[j].ID
	})
	sort.Slice(snapshot.EnvironmentConfiguration.Values, func(i, j int) bool {
		return snapshot.EnvironmentConfiguration.Values[i].ID < snapshot.EnvironmentConfiguration.Values[j].ID
	})
	sort.Slice(snapshot.EnvironmentConfiguration.Secrets, func(i, j int) bool {
		return snapshot.EnvironmentConfiguration.Secrets[i].ID < snapshot.EnvironmentConfiguration.Secrets[j].ID
	})
	sort.Slice(snapshot.Images, func(i, j int) bool { return snapshot.Images[i].WorkloadID < snapshot.Images[j].WorkloadID })
	for i := range snapshot.ApplicationDefinition.Workloads {
		sort.Slice(snapshot.ApplicationDefinition.Workloads[i].ExposedOutputs, func(a, b int) bool {
			return snapshot.ApplicationDefinition.Workloads[i].ExposedOutputs[a].Name < snapshot.ApplicationDefinition.Workloads[i].ExposedOutputs[b].Name
		})
	}
}
