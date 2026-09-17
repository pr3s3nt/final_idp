package domain_test

import (
	"reflect"
	"testing"

	"sdp/internal/domain"
	dt "sdp/internal/domain/domaintest"
	"sdp/internal/domain/graphbuilder"
	"sdp/internal/domain/infraplanner"
	"sdp/internal/domain/resourceresolver"
	"sdp/internal/domain/waveplanner"
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

func componentNames(list []domain.PlanComponent) []string {
	out := []string{}
	for _, c := range list {
		out = append(out, c.Name)
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
	if res := local.Resolutions[dt.ClusterID]; res.Definition.ManagementMode != domain.Existing {
		t.Fatalf("the internal cluster must resolve to an EXISTING definition, got %s", res.Definition.Name)
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
	p, err := infraplanner.PlanInfrastructureChanges(infraplanner.Input{Kind: domain.KindDeploy, Catalog: dt.CatalogV1, Graph: g, Waves: wp, Environment: domain.Staging,
		Images: images, Instances: instances, Running: running, Configuration: dt.Config(), Resolver: dt.Catalog()})
	if err != nil {
		t.Fatal(err)
	}
	p.PotentialRedeploy = waveplanner.FindPotentialRedeploys(g, p, running)
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

// readyInstance is a READY instance whose last apply used params and the given
// output fingerprints of the resources it requires.
func readyInstance(id, requirement, definition string, params map[string]any, d *domain.ResourceDefinition, required map[string]string) domain.ResourceInstance {
	return domain.ResourceInstance{ID: id, ApplicationID: dt.AppID, Environment: domain.Staging, RequirementID: requirement,
		DefinitionID: definition, Target: "kind-local", Status: domain.RIReady, AppliedOverrides: map[string]any{},
		AppliedInputFingerprint: infraplanner.InputFingerprint(d, params, required)}
}

var onCluster = map[string]string{"k8s-cluster": ""}

func kindInstances() []domain.ResourceInstance {
	cat := dt.Catalog()
	kind, pg, redis := cat.DefinitionByID("d-kind"), cat.DefinitionByID("d-pg-k8s"), cat.DefinitionByID("d-redis-k8s")
	return []domain.ResourceInstance{
		readyInstance("ri-c", dt.ClusterID, "d-kind", nil, kind, nil),
		readyInstance("ri-p", dt.Postgres, "d-pg-k8s", pg.DefaultParameters, pg, onCluster),
		readyInstance("ri-r", dt.Redis, "d-redis-k8s", redis.DefaultParameters, redis, onCluster),
	}
}

func TestInfrastructurePlanLinkReuseUpdate(t *testing.T) {
	v := dt.ShopV1()
	p := plan(t, v, dt.KindLocal, dt.All(v), nil, nil)
	want := map[string]string{"k8s-cluster": "LINK", "postgresql": "CREATE", "redis": "CREATE", "backend": "DEPLOY", "worker": "DEPLOY", "frontend": "DEPLOY"}
	if got := actions(p); !reflect.DeepEqual(got, want) {
		t.Fatalf("first deployment on the internal cluster = %v", got)
	}
	if p.CatalogVersion != 1 || p.CatalogVersionID != dt.CatalogV1.ID {
		t.Fatalf("the plan must carry its catalog version, got %d %q", p.CatalogVersion, p.CatalogVersionID)
	}

	instances := kindInstances()
	instances[2].AppliedInputFingerprint = "stale"
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	got := actions(p)
	if got["k8s-cluster"] != "REUSE" || got["postgresql"] != "REUSE" || got["redis"] != "UPDATE" {
		t.Fatalf("reuse/update actions = %v", got)
	}
	fp1 := p.Fingerprint()
	if fp2 := plan(t, v, dt.KindLocal, dt.All(v), nil, instances).Fingerprint(); fp1 != fp2 {
		t.Fatal("rebuilding the same inputs must give the same fingerprint")
	}

	// A resource applied with older outputs of what it requires is updated.
	instances = kindInstances()
	instances[0].OutputFingerprint = "cluster outputs changed since the last apply"
	got = actions(plan(t, v, dt.KindLocal, dt.All(v), nil, instances))
	if got["postgresql"] != "UPDATE" || got["redis"] != "UPDATE" || got["k8s-cluster"] != "REUSE" {
		t.Fatalf("stale required outputs must update the requiring resources, got %v", got)
	}
}

func TestPotentialRedeployOnlyFromChangingComponents(t *testing.T) {
	v := dt.ShopV1()
	running := dt.Running(dt.V1, 1, dt.Backend, dt.Worker, dt.Frontend)
	instances := kindInstances()

	// Partial worker: nothing is created or updated, but postgresql can still be
	// overridden at confirmation, so backend and frontend may be redone.
	p := plan(t, v, dt.KindLocal, []string{dt.Worker}, running, instances)
	if got := componentNames(p.PotentialRedeploy); !reflect.DeepEqual(got, []string{"backend", "frontend"}) {
		t.Fatalf("partial worker may redo %v", got)
	}
	// Partial frontend reuses everything and nothing depends on frontend.
	p = plan(t, v, dt.KindLocal, []string{dt.Frontend}, running, instances)
	if len(p.PotentialRedeploy) != 0 {
		t.Fatalf("reused components are not a source of redeploys, got %v", componentNames(p.PotentialRedeploy))
	}
}

func TestResourcesToReapplyAfterOutputChanges(t *testing.T) {
	v := dt.ShopV1()
	cat := dt.Catalog()
	net, eks, aurora, cache := cat.DefinitionByID("d-net"), cat.DefinitionByID("d-eks"), cat.DefinitionByID("d-aurora"), cat.DefinitionByID("d-elasticache")
	onNetwork := map[string]string{"network": ""}
	instances := []domain.ResourceInstance{
		readyInstance("ri-n", dt.NetworkID, "d-net", net.DefaultParameters, net, nil),
		readyInstance("ri-c", dt.ClusterID, "d-eks", eks.DefaultParameters, eks, onNetwork),
		readyInstance("ri-p", dt.Postgres, "d-aurora", aurora.DefaultParameters, aurora, onNetwork),
		readyInstance("ri-r", dt.Redis, "d-elasticache", cache.DefaultParameters, cache, onNetwork),
	}
	p := plan(t, v, dt.AWS, dt.All(v), nil, instances)
	for _, it := range p.Items() {
		if it.Kind == domain.NodeResource && it.Action != domain.ActionReuse {
			t.Fatalf("every resource should be reused, %s is %s", it.Name, it.Action)
		}
	}
	g := build(t, v, dt.AWS)
	scope := map[string]bool{}
	for _, it := range p.Items() {
		scope[it.ComponentID] = true
	}
	// With a network update, every reused resource on the network may be redone,
	// including those that also accept overrides.
	instances[0].AppliedInputFingerprint = "stale"
	pu := plan(t, v, dt.AWS, dt.All(v), nil, instances)
	if got := componentNames(pu.PotentialRedeploy); !reflect.DeepEqual(got, []string{"k8s-cluster", "postgresql", "redis"}) {
		t.Fatalf("a network update may redo cluster, aurora and elasticache, got %v", got)
	}
	got := waveplanner.ResourcesToReapply(g, []string{dt.NetworkID}, scope, map[string]bool{dt.NetworkID: true}, p)
	want := []string{dt.ClusterID, dt.Postgres, dt.Redis}
	if !reflect.DeepEqual(sorted(got), sorted(want)) {
		t.Fatalf("a changed network re-applies cluster, aurora and elasticache, got %v", got)
	}
	if got := waveplanner.ResourcesToReapply(g, []string{dt.NetworkID}, scope, map[string]bool{dt.NetworkID: true, dt.ClusterID: true}, p); len(got) != 2 {
		t.Fatalf("resources already executed are not applied again, got %v", got)
	}

	// Partial frontend on the internal cluster: postgresql and redis are outside the
	// scope but require the cluster; backend and worker run on it.
	kind := build(t, v, dt.KindLocal)
	running := dt.Running(dt.V1, 1, dt.Backend, dt.Worker, dt.Frontend)
	wp, err := waveplanner.PlanDeploymentWaves(kind, []string{dt.Frontend}, running)
	if err != nil {
		t.Fatal(err)
	}
	kp := plan(t, v, dt.KindLocal, []string{dt.Frontend}, running, kindInstances())
	if got := waveplanner.ResourcesToReapply(kind, []string{dt.ClusterID}, wp.Scope, map[string]bool{dt.ClusterID: true}, kp); !reflect.DeepEqual(sorted(got), sorted([]string{dt.Postgres, dt.Redis})) {
		t.Fatalf("a changed internal cluster re-applies postgresql and redis, got %v", got)
	}
	var cascaded []string
	for _, c := range waveplanner.PropagateOutputChanges(kind, wp.Scope, []string{dt.ClusterID}, running) {
		cascaded = append(cascaded, c.WorkloadID)
	}
	if !reflect.DeepEqual(sorted(cascaded), sorted([]string{dt.Backend, dt.Worker})) {
		t.Fatalf("workloads on a changed cluster cascade, got %v", cascaded)
	}
}

func sorted(s []string) []string {
	out := append([]string{}, s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestRemovalsForNewVersionAndTeardownOrder(t *testing.T) {
	v2 := dt.ShopV2()
	cat := dt.Catalog()
	running := dt.Running(dt.V1, 1, dt.Backend, dt.Worker, dt.Frontend)
	instances := kindInstances()
	p := plan(t, v2, dt.KindLocal, dt.All(v2), running, instances)
	if len(p.Removals) != 2 || p.Removals[0].Items[0].Action != "REMOVE" || p.Removals[0].Items[0].ComponentID != dt.Worker ||
		p.Removals[1].Items[0].Action != "DESTROY" || p.Removals[1].Items[0].ComponentID != dt.Redis || !p.Removals[1].Items[0].DataLossWarning {
		t.Fatalf("v2 removals = %+v", p.Removals)
	}

	g := graphbuilder.NodesOnly(dt.ShopV1(), dt.KindLocal)
	td, err := infraplanner.PlanInfrastructureChanges(infraplanner.Input{Kind: domain.KindTeardown, Catalog: dt.CatalogV1, Graph: g, Environment: domain.Staging,
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
	want := [][]string{{"REMOVE backend", "REMOVE frontend", "REMOVE worker"}, {"DESTROY postgresql", "DESTROY redis"}, {"UNLINK k8s-cluster"}}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("teardown order = %v, want %v (the internal cluster is only unlinked)", order, want)
	}
}

func TestDefinitionIdentityAcrossCatalogVersions(t *testing.T) {
	v := dt.ShopV1()
	v2 := dt.Catalog()
	for i := range v2.Definitions {
		v2.Definitions[i].ID += "-v2"
		v2.Definitions[i].CatalogVersionID = "cat-2"
	}
	v2.All = append(append([]domain.ResourceDefinition{}, dt.Catalog().Definitions...), v2.Definitions...)
	g, err := graphbuilder.BuildDeploymentGraph(v, dt.Config(), domain.Staging, dt.KindLocal, v2)
	if err != nil {
		t.Fatal(err)
	}
	wp, _ := waveplanner.PlanDeploymentWaves(g, dt.All(v), nil)
	p, err := infraplanner.PlanInfrastructureChanges(infraplanner.Input{Kind: domain.KindDeploy, Catalog: domain.CatalogVersion{ID: "cat-2", Number: 2}, Graph: g, Waves: wp,
		Environment: domain.Staging, Images: map[string]string{dt.Backend: "v1", dt.Worker: "v1", dt.Frontend: "v1"},
		Instances: kindInstances(), Configuration: dt.Config(), Resolver: v2})
	if err != nil {
		t.Fatalf("an instance created with catalog v1 is the same definition in v2: %v", err)
	}
	if got := actions(p); got["postgresql"] != "REUSE" || got["k8s-cluster"] != "REUSE" {
		t.Fatalf("unchanged definitions in a new catalog version are reused, got %v", got)
	}

	renamed := dt.Catalog()
	renamed.Definitions[1].Name = "postgres-k8s-ha"
	g, _ = graphbuilder.BuildDeploymentGraph(v, dt.Config(), domain.Staging, dt.KindLocal, renamed)
	wp, _ = waveplanner.PlanDeploymentWaves(g, dt.All(v), nil)
	renamedInstances := kindInstances()
	renamedInstances[1].DefinitionID = "d-pg-old"
	renamed.All = append(renamed.All, domain.ResourceDefinition{ID: "d-pg-old", Name: "postgres-k8s", ResourceType: "PostgreSQL"})
	_, err = infraplanner.PlanInfrastructureChanges(infraplanner.Input{Kind: domain.KindDeploy, Graph: g, Waves: wp, Environment: domain.Staging,
		Images: map[string]string{dt.Backend: "v1", dt.Worker: "v1", dt.Frontend: "v1"}, Instances: renamedInstances, Configuration: dt.Config(), Resolver: renamed})
	if !domain.HasCode(err, domain.CodeDefinitionChanged) {
		t.Fatalf("switching a running resource to another definition stays rejected (D11), got %v", err)
	}
}

func TestOverridePolicy(t *testing.T) {
	v := dt.ShopV1()
	cat := dt.Catalog()
	pg := cat.DefinitionByID("d-pg-k8s")
	instances := []domain.ResourceInstance{readyInstance("ri-p", dt.Postgres, "d-pg-k8s", pg.DefaultParameters, pg, onCluster)}

	p := plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"postgresql": {"password_revision": 1}}); err != nil {
		t.Fatal(err)
	}
	if it := p.Item(dt.Postgres); it.Action != "UPDATE" || it.NewBaseline["password_revision"] != int64(1) {
		t.Fatalf("override must turn REUSE into UPDATE and join the baseline: %+v", it)
	}

	bad := map[string]map[string]any{
		"postgresql": {"storage_gb": 9, "unknown": 1},
		"frontend":   {"x": 1},
	}
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	err := infraplanner.ApplyInfrastructureOverrides(p, bad)
	if !domain.HasCode(err, domain.CodeInvalidOverride) {
		t.Fatalf("expected invalid override, got %v", err)
	}
	if n := len(err.(*domain.ValidationError).Problems); n != 3 {
		t.Fatalf("expected 3 problems (range, unknown key, workload), got %v", err)
	}
	p = plan(t, v, dt.KindLocal, dt.All(v), nil, instances)
	if err := infraplanner.ApplyInfrastructureOverrides(p, map[string]map[string]any{"k8s-cluster": {"node_count": 2}}); !domain.HasCode(err, domain.CodeInvalidOverride) {
		t.Fatalf("an internal cluster accepts no overrides, got %v", err)
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
		readyInstance("ri-c", dt.ClusterID, "d-eks", map[string]any{"kubernetes_version": "1.34", "node_count": 2.0}, eks, map[string]string{"network": ""}),
		readyInstance("ri-n", dt.NetworkID, "d-net", net.DefaultParameters, net, nil),
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
