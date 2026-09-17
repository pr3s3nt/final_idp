package waveplanner

import (
	"sort"

	"deploy/internal/domain"
)

// FindPotentialRedeploys lists the components that may be redone automatically
// when outputs change. Only components the plan creates, updates, links or
// deploys can change outputs, plus reused managed resources the Developer may
// still override at confirmation; other reused ones cannot. From those, it follows
// dependents transitively: managed resources that are not already applied by
// the plan, and running workloads outside the plan.
func FindPotentialRedeploys(g *domain.DeploymentGraph, p *domain.Plan, running []domain.WorkloadInstance) []domain.PlanComponent {
	healthy := map[string]bool{}
	for _, wi := range running {
		if wi.Status == domain.WIHealthy {
			healthy[wi.WorkloadID] = true
		}
	}
	// applied: components the plan already applies with fresh inputs. origins:
	// those plus reused managed resources that may still be overridden.
	applied, origins := map[string]bool{}, map[string]bool{}
	for _, it := range p.Items() {
		switch it.Action {
		case domain.ActionDeploy, domain.ActionCreate, domain.ActionUpdate, domain.ActionLink:
			applied[it.ComponentID], origins[it.ComponentID] = true, true
		case domain.ActionReuse:
			if it.ManagementMode == domain.Managed && len(it.AllowedOverrides) > 0 {
				origins[it.ComponentID] = true
			}
		}
	}
	affected := map[string]bool{}
	var visit func(id string)
	visit = func(id string) {
		for _, dep := range g.Dependents(id) {
			if affected[dep] || applied[dep] {
				continue
			}
			n := g.Nodes[dep]
			switch n.Kind {
			case domain.NodeWorkload:
				if p.Item(dep) != nil || !healthy[dep] {
					continue
				}
			case domain.NodeResource:
				res := g.Resolutions[dep]
				if res == nil || res.Definition.ManagementMode != domain.Managed {
					continue
				}
			}
			affected[dep] = true
			visit(dep)
		}
	}
	for _, id := range sortedKeys(origins) {
		visit(id)
	}
	out := []domain.PlanComponent{}
	for id := range affected {
		out = append(out, domain.PlanComponent{ID: id, Name: g.Nodes[id].Name, Kind: g.Nodes[id].Kind})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == domain.NodeResource
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ResourcesToReapply returns the managed resources that require one of the
// changed components and must be applied again in this deployment: resources
// outside the scope, and resources in the scope the plan would only reuse.
// Resources already executed are never returned.
func ResourcesToReapply(g *domain.DeploymentGraph, changed []string, scope, done map[string]bool, p *domain.Plan) []string {
	set := map[string]bool{}
	for _, id := range changed {
		for _, dep := range g.Dependents(id) {
			n := g.Nodes[dep]
			res := g.Resolutions[dep]
			if n.Kind != domain.NodeResource || done[dep] || res == nil || res.Definition.ManagementMode != domain.Managed {
				continue
			}
			if scope[dep] {
				if it := p.Item(dep); it == nil || it.Action != domain.ActionReuse {
					continue
				}
			}
			set[dep] = true
		}
	}
	return sortedKeys(set)
}
