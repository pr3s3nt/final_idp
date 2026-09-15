package domain_test

import (
	"reflect"
	"testing"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	dt "github.com/pr3s3nt/final_idp/uc03/internal/domain/domaintest"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/graphbuilder"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/infraplanner"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceresolver"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/waveplanner"
)

func names(g *domain.DeploymentGraph, waves [][]string) [][]string {
	out := make([][]string, len(waves))
	for i, w := range waves {
		for _, id := range w {
			out[i] = append(out[i], g.Nodes[id].Name)
		}
	}
	return out
}

func build(t *testing.T, v *domain.ApplicationVersion, ctx domain.DeploymentContext) *domain.DeploymentGraph {
	t.Helper()
	g, err := graphbuilder.BuildDeploymentGraph(v, dt.Config(), domain.Staging, ctx, dt.Catalog())
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	return g
}

func TestResolverPrefersMostSpecificAndRejectsAmbiguity(t *testing.T) {
	r := dt.Catalog()
	scope := resourceresolver.Scope{ApplicationName: "reporting-app", Environment: domain.Staging, Context: dt.KindLocal}
	res, p := r.Resolve("x", "PostgreSQL", scope)
	if p != nil || res.Definition.Name != "postgres-shared-staging" {
		t.Fatalf("reporting-app staging should link the shared definition, got %+v %+v", res, p)
	}
	scope.Environment = domain.Production
	if res, p = r.Resolve("x", "PostgreSQL", scope); p != nil || res.Definition.Name != "postgres-k8s" {
		t.Fatalf("production should use the managed definition, got %+v %+v", res, p)
	}
	if _, p = r.Resolve("x", "PostgreSQL", resourceresolver.Scope{Environment: domain.Staging, Context: domain.DeploymentContext{Target: "gcp"}}); p == nil || p.Code != domain.CodeNoResourceDefinition {
		t.Fatalf("unsupported context must fail, got %+v", p)
	}
	dup := dt.Catalog()
	clone := dup.Definitions[1]
	clone.ID, clone.Name = "d-pg-k8s-2", "postgres-k8s-copy"
	dup.Definitions = append(dup.Definitions, clone)
	if _, p = dup.Resolve("x", "PostgreSQL", resourceresolver.Scope{ApplicationName: "shop-app", Environment: domain.Staging, Context: dt.KindLocal}); p == nil || p.Code != domain.CodeAmbiguousDefinition {
		t.Fatalf("two equally specific definitions must be ambiguous, got %+v", p)
	}
}

func TestGraphAddsClusterAndRequiredNetworkOnAWS(t *testing.T) {
	g := build(t, dt.ShopV1(), dt.AWS)
	if g.Nodes[dt.ClusterID] == nil || g.Nodes[dt.NetworkID] == nil {
		t.Fatalf("aws graph must contain k8s-cluster and network nodes")
	}
	if got := g.Resolutions[dt.ClusterID].Definition.Name; got != "eks-cluster" {
		t.Fatalf("cluster resolved to %s", got)
	}
	if got := g.Nodes[dt.Postgres].DependsOn; !reflect.DeepEqual(got, []string{dt.NetworkID}) {
		t.Fatalf("aurora must require network, got %v", got)
	}
	local := build(t, dt.ShopV1(), dt.KindLocal)
	if local.Nodes[dt.NetworkID] != nil {
		t.Fatalf("kind graph must not contain a network node")
	}
}

func TestGraphDetectsCycle(t *testing.T) {
	v := dt.ShopV1()
	v.Dependencies = append(v.Dependencies, domain.Dependency{ID: "cyc", SourceWorkloadID: dt.Backend, TargetType: domain.TargetWorkload, TargetID: dt.Frontend})
	_, err := graphbuilder.BuildDeploymentGraph(v, dt.Config(), domain.Staging, dt.KindLocal, dt.Catalog())
	if !domain.HasCode(err, domain.CodeDependencyCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestFullDeploymentWaves(t *testing.T) {
	cases := map[string]struct {
		ctx  domain.DeploymentContext
		want [][]string
	}{
		"kind": {dt.KindLocal, [][]string{{"k8s-cluster"}, {"postgresql", "redis"}, {"backend", "worker"}, {"frontend"}}},
		"aws":  {dt.AWS, [][]string{{"network"}, {"k8s-cluster", "postgresql", "redis"}, {"backend", "worker"}, {"frontend"}}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v := dt.ShopV1()
			g := build(t, v, c.ctx)
			wp, err := waveplanner.PlanDeploymentWaves(g, dt.All(v), nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := names(g, wp.Waves); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("waves = %v, want %v", got, c.want)
			}
			if !wp.FullDeployment {
				t.Fatal("selecting every workload is a full deployment")
			}
		})
	}
}

func TestPartialDeploymentScopeAndOutOfScopeDependency(t *testing.T) {
	v := dt.ShopV1()
	g := build(t, v, dt.KindLocal)
	if _, err := waveplanner.PlanDeploymentWaves(g, []string{dt.Frontend}, nil); !domain.HasCode(err, domain.CodeDependencyNotHealthy) {
		t.Fatalf("frontend alone needs a running backend, got %v", err)
	}
	running := dt.Running(dt.V1, 1, dt.Backend, dt.Worker, dt.Frontend)
	wp, err := waveplanner.PlanDeploymentWaves(g, []string{dt.Frontend}, running)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(g, wp.Waves); !reflect.DeepEqual(got, [][]string{{"k8s-cluster"}, {"frontend"}}) {
		t.Fatalf("partial frontend waves = %v", got)
	}
	if !reflect.DeepEqual(wp.OutOfScopeDependencies, []string{dt.Backend}) {
		t.Fatalf("backend is an out-of-scope dependency, got %v", wp.OutOfScopeDependencies)
	}

	wp, err = waveplanner.PlanDeploymentWaves(g, []string{dt.Worker}, running)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(g, wp.Waves); !reflect.DeepEqual(got, [][]string{{"k8s-cluster"}, {"postgresql", "redis"}, {"worker"}}) {
		t.Fatalf("partial worker waves = %v", got)
	}
	// postgresql is in scope and backend depends on it, frontend on backend.
	if !reflect.DeepEqual(wp.PotentialRedeploy, []string{dt.Backend, dt.Frontend}) {
		t.Fatalf("potential redeploy = %v", wp.PotentialRedeploy)
	}
	cands := waveplanner.PropagateOutputChanges(g, wp.Scope, []string{dt.Postgres}, running)
	if len(cands) != 1 || cands[0].WorkloadID != dt.Backend {
		t.Fatalf("a changed postgresql output cascades to backend only, got %+v", cands)
	}
}

func plan(t *testing.T, v *domain.ApplicationVersion, ctx domain.DeploymentContext, selected []string, running []domain.WorkloadInstance, instances []domain.ResourceInstance) *domain.Plan {
	t.Helper()
	g := build(t, v, ctx)
	wp, err := waveplanner.PlanDeploymentWaves(g, selected, running)
	if err != nil {
		t.Fatal(err)
	}
	images := map[string]string{}
	for _, id := range selected {
		images[id] = "v1"
	}
	p, err := infraplanner.PlanInfrastructureChanges(infraplanner.Input{Kind: domain.KindDeploy, Graph: g, Waves: wp, Environment: domain.Staging,
		Images: images, Instances: instances, Running: running, Configuration: dt.Config(), Resolver: dt.Catalog()})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func actions(p *domain.Plan) map[string]string {
	out := map[string]string{}
	for _, it := range p.Items() {
		out[it.Name] = it.Action
	}
	for _, w := range p.Removals {
		for _, it := range w.Items {
			out["remove:"+it.Name] = it.Action
		}
	}
	return out
}

func readyInstance(id, requirement, definition string, params map[string]any, d *domain.ResourceDefinition) domain.ResourceInstance {
	return domain.ResourceInstance{ID: id, ApplicationID: dt.AppID, Environment: domain.Staging, RequirementID: requirement,
		DefinitionID: definition, Target: "kind-local", Status: domain.RIReady, AppliedOverrides: map[string]any{},
		AppliedInputFingerprint: infraplanner.InputFingerprint(d, params)}
}

func TestInfrastructurePlanCreateReuseUpdate(t *testing.T) {
	v := dt.ShopV1()
	p := plan(t, v, dt.KindLocal, dt.All(v), nil, nil)
	want := map[string]string{"k8s-cluster": "CREATE", "postgresql": "CREATE", "redis": "CREATE", "backend": "DEPLOY", "worker": "DEPLOY", "frontend": "DEPLOY"}
	if got := actions(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("first deployment actions = %v", got)
	}

	cat := dt.Catalog()
	kind, pg, redis := cat.DefinitionByID("d-kind"), cat.DefinitionByID("d-pg-k8s"), cat.DefinitionByID("d-redis-k8s")
	instances := []domain.ResourceInstance{
		readyInstance("ri-c", dt.ClusterID, "d-kind", kind.DefaultParameters, kind),
		readyInstance("ri-p", dt.Postgres, "d-pg-k8s", pg.DefaultParameters, pg),
		readyInstance("ri-r", dt.Redis, "d-redis-k8s", map[string]any{"stale": true}, redis),
	}
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	got := actions(p)
	if got["k8s-cluster"] != "REUSE" || got["postgresql"] != "REUSE" || got["redis"] != "UPDATE" {
		t.Fatalf("reuse/update actions = %v", got)
	}
	fp1 := p.Fingerprint()
	if fp2 := plan(t, v, dt.KindLocal, dt.All(v), nil, instances).Fingerprint(); fp1 != fp2 {
		t.Fatal("rebuilding the same inputs must give the same fingerprint")
	}
}

func TestRemovalsForNewVersionAndTeardownOrder(t *testing.T) {
	v2 := dt.ShopV2()
	cat := dt.Catalog()
	running := dt.Running(dt.V1, 1, dt.Backend, dt.Worker, dt.Frontend)
	instances := []domain.ResourceInstance{
		readyInstance("ri-c", dt.ClusterID, "d-kind", nil, cat.DefinitionByID("d-kind")),
		readyInstance("ri-p", dt.Postgres, "d-pg-k8s", nil, cat.DefinitionByID("d-pg-k8s")),
		readyInstance("ri-r", dt.Redis, "d-redis-k8s", nil, cat.DefinitionByID("d-redis-k8s")),
	}
	p := plan(t, v2, dt.KindLocal, dt.All(v2), running, instances)
	if len(p.Removals) != 2 || p.Removals[0].Items[0].Action != "REMOVE" || p.Removals[0].Items[0].ComponentID != dt.Worker ||
		p.Removals[1].Items[0].Action != "DESTROY" || p.Removals[1].Items[0].ComponentID != dt.Redis || !p.Removals[1].Items[0].DataLossWarning {
		t.Fatalf("v2 removals = %+v", p.Removals)
	}

	g := graphbuilder.NodesOnly(dt.ShopV1(), dt.KindLocal)
	td, err := infraplanner.PlanInfrastructureChanges(infraplanner.Input{Kind: domain.KindTeardown, Graph: g, Environment: domain.Staging,
		Instances: instances, Running: running, Resolver: cat})
	if err != nil {
		t.Fatal(err)
	}
	var order [][]string
	for _, w := range td.Removals {
		var names []string
		for _, it := range w.Items {
			names = append(names, it.Action+" "+it.Name)
		}
		order = append(order, names)
	}
	want := [][]string{{"REMOVE backend", "REMOVE frontend", "REMOVE worker"}, {"DESTROY postgresql", "DESTROY redis"}, {"DESTROY k8s-cluster"}}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("teardown order = %v, want %v", order, want)
	}
}

func TestOverridePolicy(t *testing.T) {
	v := dt.ShopV1()
	cat := dt.Catalog()
	pg := cat.DefinitionByID("d-pg-k8s")
	instances := []domain.ResourceInstance{readyInstance("ri-p", dt.Postgres, "d-pg-k8s", pg.DefaultParameters, pg)}

	p := plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"postgresql": {"password_revision": 1}}); err != nil {
		t.Fatal(err)
	}
	if it := p.Item(dt.Postgres); it.Action != "UPDATE" || it.NewBaseline["password_revision"] != int64(1) {
		t.Fatalf("override must turn REUSE into UPDATE and join the baseline: %+v", it)
	}

	bad := map[string]map[string]any{
		"postgresql":  {"storage_gb": 9, "unknown": 1},
		"k8s-cluster": {"node_count": 2}, // immutable only once the cluster exists; here it is CREATE
		"frontend":    {"x": 1},
	}
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	err := infraplanner.ApplyInfrastructureOverrides(p, bad)
	if !domain.HasCode(err, domain.CodeInvalidOverride) {
		t.Fatalf("expected invalid override, got %v", err)
	}
	if n := len(err.(*domain.ValidationError).Problems); n != 3 {
		t.Fatalf("expected 3 problems (range, unknown key, workload), got %v", err)
	}

	kind := cat.DefinitionByID("d-kind")
	instances = append(instances, readyInstance("ri-c", dt.ClusterID, "d-kind", kind.DefaultParameters, kind))
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"k8s-cluster": {"node_count": 2}}); !domain.HasCode(err, domain.CodeImmutableParameter) {
		t.Fatalf("changing an immutable parameter of an existing cluster must fail, got %v", err)
	}
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	pgInstance := p.Item(dt.Postgres)
	pgInstance.Parameters["password_revision"] = 3.0
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"postgresql": {"password_revision": 2}}); err == nil {
		t.Fatal("decreasing an increase-only parameter must fail")
	}
}

func TestEKSVersionOverrideStepAndCIDR(t *testing.T) {
	v := dt.ShopV1()
	cat := dt.Catalog()
	eks, net := cat.DefinitionByID("d-eks"), cat.DefinitionByID("d-net")
	instances := []domain.ResourceInstance{
		readyInstance("ri-c", dt.ClusterID, "d-eks", map[string]any{"kubernetes_version": "1.34", "node_count": 2.0}, eks),
		readyInstance("ri-n", dt.NetworkID, "d-net", net.DefaultParameters, net),
	}
	instances[0].AppliedOverrides = map[string]any{"kubernetes_version": "1.34"}
	p := plan(t, v, dt.AWS, dt.All(v), nil, instances)
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"k8s-cluster": {"kubernetes_version": "1.36"}}); err == nil {
		t.Fatal("upgrading two minor versions at once must fail")
	}
	p = plan(t, v, dt.AWS, dt.All(v), nil, instances)
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"network": {"vpc_cidr": "10.70.0.0/16"}}); !domain.HasCode(err, domain.CodeImmutableParameter) {
		t.Fatalf("vpc_cidr is immutable after creation, got %v", err)
	}
	fresh := plan(t, v, dt.AWS, dt.All(v), nil, nil)
	if err := infraplanner.ApplyInfrastructureOverrides(fresh, map[string]map[string]any{"network": {"vpc_cidr": "192.168.0.0/16"}}); err == nil {
		t.Fatal("a CIDR outside 10.0.0.0/8 must fail")
	}
	fresh = plan(t, v, dt.AWS, dt.All(v), nil, nil)
	if err := infraplanner.ApplyInfrastructureOverrides(fresh, map[string]map[string]any{"network": {"vpc_cidr": "10.70.0.0/16"}}); err != nil {
		t.Fatalf("a valid CIDR on CREATE must pass: %v", err)
	}
}
