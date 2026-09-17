// Package graphbuilder is the Deployment Graph Builder. It builds the
// dependency/resource graph of one Application Definition version for a
// deployment context: declared workloads, resources and dependencies, the
// implicit k8s-cluster every workload runs on, and the extra infrastructure
// nodes that resolved Resource Definitions require.
package graphbuilder

import (
	"sort"
	"strings"

	"deploy/internal/domain"
	"deploy/internal/domain/resourceresolver"
)

// BuildDeploymentGraph returns the graph with a Resource Resolution for every
// resource node, or an A1 ValidationError (unresolved dependency, cycle, no or
// ambiguous definition).
func BuildDeploymentGraph(version *domain.ApplicationVersion, configuration *domain.EnvironmentConfiguration,
	env domain.Environment, ctx domain.DeploymentContext, resolver *resourceresolver.Resolver) (*domain.DeploymentGraph, error) {

	problems := &domain.ValidationError{}
	g := &domain.DeploymentGraph{
		Version:     version,
		Context:     ctx,
		Nodes:       map[string]*domain.GraphNode{},
		Resolutions: map[string]*domain.ResourceResolution{},
	}
	clusterID := domain.PlatformRequirementID(version.ApplicationID, domain.ResourceTypeCluster)

	for _, r := range version.Resources {
		g.Nodes[r.ID] = &domain.GraphNode{ID: r.ID, Kind: domain.NodeResource, Name: r.Name, ResourceType: r.ResourceType}
	}
	for _, w := range version.Workloads {
		g.Nodes[w.ID] = &domain.GraphNode{ID: w.ID, Kind: domain.NodeWorkload, Name: w.Name, DependsOn: []string{clusterID}}
	}
	g.Nodes[clusterID] = &domain.GraphNode{ID: clusterID, Kind: domain.NodeResource, Name: domain.ResourceTypeCluster,
		ResourceType: domain.ResourceTypeCluster, Platform: true}

	for _, d := range version.Dependencies {
		src, ok := g.Nodes[d.SourceWorkloadID]
		if !ok || src.Kind != domain.NodeWorkload {
			problems.Add(domain.CodeDependencyUnresolved, "dependency %s has a source that is not a workload of version %d", d.ID, version.VersionNumber)
			continue
		}
		tgt, ok := g.Nodes[d.TargetID]
		if !ok || tgt.Platform {
			problems.Add(domain.CodeDependencyUnresolved, "%s depends on a component that is not in version %d", src.Name, version.VersionNumber)
			continue
		}
		wantKind := domain.NodeWorkload
		if d.TargetType == domain.TargetResource {
			wantKind = domain.NodeResource
		}
		if tgt.Kind != wantKind {
			problems.Add(domain.CodeDependencyUnresolved, "dependency %s -> %s has target type %s but the target is a %s", src.Name, tgt.Name, d.TargetType, tgt.Kind)
			continue
		}
		src.DependsOn = append(src.DependsOn, tgt.ID)
	}

	// Resolve resource definitions; a definition's `requires` adds platform
	// nodes until nothing new appears.
	scope := resourceresolver.Scope{ApplicationID: version.ApplicationID, ApplicationName: version.ApplicationName, Environment: env, Context: ctx}
	for {
		pending := unresolved(g)
		if len(pending) == 0 {
			break
		}
		for _, id := range pending {
			node := g.Nodes[id]
			res, problem := resolver.Resolve(id, node.ResourceType, scope)
			if problem != nil {
				problems.Problems = append(problems.Problems, *problem)
				// Record a nil resolution so the loop does not retry it.
				g.Resolutions[id] = nil
				continue
			}
			g.Resolutions[id] = res
			for _, requiredType := range res.Definition.Requires {
				reqID := requiredNode(g, version.ApplicationID, requiredType)
				if reqID == id {
					problems.Add(domain.CodeDependencyCycle, "resource definition %s requires its own type %s", res.Definition.Name, requiredType)
					continue
				}
				node.DependsOn = append(node.DependsOn, reqID)
			}
		}
	}

	for _, n := range g.Nodes {
		sort.Strings(n.DependsOn)
		n.DependsOn = dedupe(n.DependsOn)
	}
	if cycle := findCycle(g); cycle != nil {
		names := make([]string, len(cycle))
		for i, id := range cycle {
			names[i] = g.Nodes[id].Name
		}
		problems.Add(domain.CodeDependencyCycle, "depends-on relations form a cycle: %s", strings.Join(names, " -> "))
	}

	if configuration != nil {
		for _, v := range append(append([]domain.ConfiguredValue{}, configuration.Variables...), configuration.Secrets...) {
			if (v.Source == domain.SourceResourceOutput || v.Source == domain.SourceWorkloadOutput) && version.Workload(v.WorkloadID) != nil {
				g.ConfigurationReferenceIDs = append(g.ConfigurationReferenceIDs, v.ID)
			}
		}
		sort.Strings(g.ConfigurationReferenceIDs)
	}

	for id, res := range g.Resolutions {
		if res == nil {
			delete(g.Resolutions, id)
		}
	}
	return g, problems.OrNil()
}

// NodesOnly builds a graph of a version's declared components without resolving
// definitions. A teardown only needs names; it removes what instances exist.
func NodesOnly(version *domain.ApplicationVersion, ctx domain.DeploymentContext) *domain.DeploymentGraph {
	g := &domain.DeploymentGraph{Version: version, Context: ctx, Nodes: map[string]*domain.GraphNode{}, Resolutions: map[string]*domain.ResourceResolution{}}
	for _, r := range version.Resources {
		g.Nodes[r.ID] = &domain.GraphNode{ID: r.ID, Kind: domain.NodeResource, Name: r.Name, ResourceType: r.ResourceType}
	}
	for _, w := range version.Workloads {
		g.Nodes[w.ID] = &domain.GraphNode{ID: w.ID, Kind: domain.NodeWorkload, Name: w.Name}
	}
	return g
}

func unresolved(g *domain.DeploymentGraph) []string {
	var ids []string
	for _, id := range g.SortedNodeIDs() {
		n := g.Nodes[id]
		if _, done := g.Resolutions[id]; n.Kind == domain.NodeResource && !done {
			ids = append(ids, id)
		}
	}
	return ids
}

// requiredNode returns the node that satisfies a required resource type,
// creating the implicit platform requirement node when it does not exist yet.
func requiredNode(g *domain.DeploymentGraph, applicationID, requiredType string) string {
	id := domain.PlatformRequirementID(applicationID, requiredType)
	if _, ok := g.Nodes[id]; !ok {
		g.Nodes[id] = &domain.GraphNode{ID: id, Kind: domain.NodeResource, Name: requiredType, ResourceType: requiredType, Platform: true}
	}
	return id
}

func dedupe(sorted []string) []string {
	out := sorted[:0]
	for i, s := range sorted {
		if i == 0 || s != sorted[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// findCycle returns one depends-on cycle as a node path, or nil.
func findCycle(g *domain.DeploymentGraph) []string {
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var stack []string
	var cycle []string
	var visit func(id string) bool
	visit = func(id string) bool {
		color[id] = grey
		stack = append(stack, id)
		for _, dep := range g.Nodes[id].DependsOn {
			if _, ok := g.Nodes[dep]; !ok {
				continue
			}
			switch color[dep] {
			case grey:
				for i, s := range stack {
					if s == dep {
						cycle = append(append([]string{}, stack[i:]...), dep)
						return true
					}
				}
			case white:
				if visit(dep) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		return false
	}
	for _, id := range g.SortedNodeIDs() {
		if color[id] == white && visit(id) {
			return cycle
		}
	}
	return nil
}
