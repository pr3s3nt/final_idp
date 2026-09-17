// Package waveplanner is the Deployment Wave Planner: it determines the
// deployment scope, orders it into waves by dependency, checks dependencies
// outside the scope and propagates output changes into later waves.
package waveplanner

import (
	"sort"

	"sdp/internal/domain"
)

// PlanDeploymentWaves computes scope and waves for the selected workloads.
// Scope = selected workloads + resources they depend on directly + the
// infrastructure those resources require (transitively). Workloads the selected
// ones depend on but that are not selected must be running HEALTHY.
func PlanDeploymentWaves(g *domain.DeploymentGraph, selected []string, running []domain.WorkloadInstance) (*domain.WavePlan, error) {
	problems := &domain.ValidationError{}
	wp := &domain.WavePlan{Scope: map[string]bool{}}

	selectedSet := map[string]bool{}
	for _, id := range selected {
		n, ok := g.Nodes[id]
		if !ok || n.Kind != domain.NodeWorkload {
			problems.Add(domain.CodeInvalidInput, "selected workload %s is not in version %d", id, g.Version.VersionNumber)
			continue
		}
		selectedSet[id] = true
	}
	if len(selectedSet) == 0 && len(problems.Problems) == 0 {
		problems.Add(domain.CodeInvalidInput, "select at least one workload to deploy")
	}
	if err := problems.OrNil(); err != nil {
		return nil, err
	}
	wp.SelectedWorkloads = sortedKeys(selectedSet)
	wp.FullDeployment = len(selectedSet) == len(g.Version.Workloads)

	runningByWorkload := map[string]domain.WorkloadInstance{}
	for _, wi := range running {
		runningByWorkload[wi.WorkloadID] = wi
	}

	outside := map[string]bool{}
	for _, id := range wp.SelectedWorkloads {
		wp.Scope[id] = true
		for _, dep := range g.Nodes[id].DependsOn {
			switch g.Nodes[dep].Kind {
			case domain.NodeResource:
				addWithRequires(g, wp.Scope, dep)
			case domain.NodeWorkload:
				if !selectedSet[dep] {
					outside[dep] = true
				}
			}
		}
	}
	for _, id := range sortedKeys(outside) {
		wi, ok := runningByWorkload[id]
		if !ok || wi.Status != domain.WIHealthy {
			problems.Add(domain.CodeDependencyNotHealthy, "%s depends on %s, which is not selected and is not running healthy on this environment and target",
				namesDependingOn(g, wp.SelectedWorkloads, id), g.Nodes[id].Name)
			continue
		}
		if wi.RunningVersionID != g.Version.VersionID {
			problems.Add(domain.CodeDependencyNotHealthy, "%s runs version %d, not the selected version %d", g.Nodes[id].Name, wi.RunningVersionNumber, g.Version.VersionNumber)
			continue
		}
		wp.OutOfScopeDependencies = append(wp.OutOfScopeDependencies, id)
	}
	if err := problems.OrNil(); err != nil {
		return nil, err
	}

	wp.Waves = Layer(g, wp.Scope)
	return wp, nil
}

func addWithRequires(g *domain.DeploymentGraph, scope map[string]bool, id string) {
	if scope[id] {
		return
	}
	scope[id] = true
	for _, dep := range g.Nodes[id].DependsOn {
		if g.Nodes[dep].Kind == domain.NodeResource {
			addWithRequires(g, scope, dep)
		}
	}
}

// Layer orders the given nodes into waves: wave 0 has nodes with no dependency
// inside the set, every later wave only depends on earlier waves.
func Layer(g *domain.DeploymentGraph, set map[string]bool) [][]string {
	wave := map[string]int{}
	var depth func(id string) int
	depth = func(id string) int {
		if w, ok := wave[id]; ok {
			return w
		}
		w := 0
		for _, dep := range g.Nodes[id].DependsOn {
			if set[dep] {
				if d := depth(dep) + 1; d > w {
					w = d
				}
			}
		}
		wave[id] = w
		return w
	}
	maxWave := -1
	for id := range set {
		if d := depth(id); d > maxWave {
			maxWave = d
		}
	}
	waves := make([][]string, maxWave+1)
	for _, id := range g.SortedNodeIDs() {
		if set[id] {
			waves[wave[id]] = append(waves[wave[id]], id)
		}
	}
	return waves
}

// Candidate is a workload that output-change propagation adds to the deployment.
type Candidate struct {
	WorkloadID string
	Running    domain.WorkloadInstance
}

// PropagateOutputChanges returns the workloads to add because an output of one
// of changed components differs from its previous fingerprint: direct
// dependents that are outside the scope, running HEALTHY with the deployed
// version. Transitive propagation happens when those workloads' own outputs are
// compared after their wave.
func PropagateOutputChanges(g *domain.DeploymentGraph, scope map[string]bool, changed []string, running []domain.WorkloadInstance) []Candidate {
	byWorkload := map[string]domain.WorkloadInstance{}
	for _, wi := range running {
		byWorkload[wi.WorkloadID] = wi
	}
	added := map[string]bool{}
	var out []Candidate
	for _, id := range changed {
		for _, dependent := range g.Dependents(id) {
			n := g.Nodes[dependent]
			if n.Kind != domain.NodeWorkload || scope[dependent] || added[dependent] {
				continue
			}
			wi, ok := byWorkload[dependent]
			if !ok || wi.Status != domain.WIHealthy || wi.RunningVersionID != g.Version.VersionID {
				continue
			}
			added[dependent] = true
			out = append(out, Candidate{WorkloadID: dependent, Running: wi})
		}
	}
	sort.Slice(out, func(i, j int) bool { return g.Nodes[out[i].WorkloadID].Name < g.Nodes[out[j].WorkloadID].Name })
	return out
}

func namesDependingOn(g *domain.DeploymentGraph, selected []string, target string) string {
	for _, id := range selected {
		for _, dep := range g.Nodes[id].DependsOn {
			if dep == target {
				return g.Nodes[id].Name
			}
		}
	}
	return "a selected workload"
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
