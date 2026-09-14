package graph

import (
	"fmt"
	"sort"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

func WorkloadKey(id string) string { return "workload:" + id }
func ResourceKey(id string) string { return "resource:" + id }

func (b *Builder) Build(snapshot domain.DeploymentInputSnapshot) (domain.DeploymentGraph, error) {
	app := snapshot.ApplicationDefinition
	if app.ID == "" || len(app.Workloads) == 0 {
		return domain.DeploymentGraph{}, fmt.Errorf("INVALID_APPLICATION: application id and at least one workload are required")
	}

	nodes := make(map[string]domain.GraphNode)
	workloads := make(map[string]domain.Workload)
	resources := make(map[string]domain.ResourceRequirement)
	for _, workload := range app.Workloads {
		if workload.ID == "" || workload.Name == "" {
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_WORKLOAD: id and name are required")
		}
		if _, exists := workloads[workload.ID]; exists {
			return domain.DeploymentGraph{}, fmt.Errorf("DUPLICATE_WORKLOAD: %s", workload.ID)
		}
		workloads[workload.ID] = workload
		key := WorkloadKey(workload.ID)
		nodes[key] = domain.GraphNode{Key: key, Kind: domain.GraphNodeWorkload, ID: workload.ID, Name: workload.Name}
	}
	for _, requirement := range app.ResourceRequirements {
		if requirement.ID == "" || requirement.Name == "" || requirement.ResourceType == "" {
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_RESOURCE_REQUIREMENT: id, name and type are required")
		}
		if _, exists := resources[requirement.ID]; exists {
			return domain.DeploymentGraph{}, fmt.Errorf("DUPLICATE_RESOURCE_REQUIREMENT: %s", requirement.ID)
		}
		resources[requirement.ID] = requirement
		key := ResourceKey(requirement.ID)
		nodes[key] = domain.GraphNode{Key: key, Kind: domain.GraphNodeResource, ID: requirement.ID, Name: requirement.Name}
	}

	edges := make(map[string]domain.GraphEdge)
	addEdge := func(from, to, via string) {
		// Preserve both explicit dependencies and configuration-derived edges.
		// They can connect the same two nodes but remain distinct traceability facts.
		key := from + "->" + to + ":" + via
		edges[key] = domain.GraphEdge{From: from, To: to, Via: via}
	}

	for _, dependency := range app.Dependencies {
		if _, ok := workloads[dependency.SourceWorkloadID]; !ok {
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_DEPENDENCY: %s has unknown source workload %s", dependency.ID, dependency.SourceWorkloadID)
		}
		to := WorkloadKey(dependency.SourceWorkloadID)
		switch dependency.TargetType {
		case domain.DependencyWorkload:
			if dependency.TargetWorkloadID == "" || dependency.TargetResourceRequirementID != "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_DEPENDENCY: %s must select exactly one workload target", dependency.ID)
			}
			if _, ok := workloads[dependency.TargetWorkloadID]; !ok {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_DEPENDENCY: %s has unknown workload target %s", dependency.ID, dependency.TargetWorkloadID)
			}
			addEdge(WorkloadKey(dependency.TargetWorkloadID), to, "dependency:"+dependency.ID)
		case domain.DependencyResource:
			if dependency.TargetResourceRequirementID == "" || dependency.TargetWorkloadID != "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_DEPENDENCY: %s must select exactly one resource target", dependency.ID)
			}
			if _, ok := resources[dependency.TargetResourceRequirementID]; !ok {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_DEPENDENCY: %s has unknown resource target %s", dependency.ID, dependency.TargetResourceRequirementID)
			}
			addEdge(ResourceKey(dependency.TargetResourceRequirementID), to, "dependency:"+dependency.ID)
		default:
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_DEPENDENCY: %s has unsupported target type %s", dependency.ID, dependency.TargetType)
		}
	}

	refs := make([]string, 0, len(snapshot.EnvironmentConfiguration.Values)+len(snapshot.EnvironmentConfiguration.Secrets))
	for _, value := range snapshot.EnvironmentConfiguration.Values {
		if _, ok := workloads[value.WorkloadID]; !ok {
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: %s has unknown destination workload %s", value.ID, value.WorkloadID)
		}
		refs = append(refs, value.ID)
		switch value.Source {
		case domain.ValueDirect:
			if value.ResourceRequirementID != "" || value.ReferencedWorkloadID != "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: direct value %s contains an output reference", value.ID)
			}
		case domain.ValueResourceOutput:
			if _, ok := resources[value.ResourceRequirementID]; !ok || value.ResourceOutputName == "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: %s has an invalid resource output", value.ID)
			}
			addEdge(ResourceKey(value.ResourceRequirementID), WorkloadKey(value.WorkloadID), "configuration:"+value.ID)
		case domain.ValueWorkloadOutput:
			referenced, ok := workloads[value.ReferencedWorkloadID]
			if !ok || value.WorkloadOutputName == "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: %s has an invalid workload output", value.ID)
			}
			if !hasPlanTimeOutput(referenced, value.WorkloadOutputName) {
				return domain.DeploymentGraph{}, fmt.Errorf("RUNTIME_OUTPUT_UNSUPPORTED: %s.%s is not a PLAN_TIME output", referenced.Name, value.WorkloadOutputName)
			}
			addEdge(WorkloadKey(value.ReferencedWorkloadID), WorkloadKey(value.WorkloadID), "configuration:"+value.ID)
		default:
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: %s has unsupported source %s", value.ID, value.Source)
		}
	}
	for _, secret := range snapshot.EnvironmentConfiguration.Secrets {
		if _, ok := workloads[secret.WorkloadID]; !ok {
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_SECRET_REFERENCE: %s has unknown destination workload %s", secret.ID, secret.WorkloadID)
		}
		refs = append(refs, secret.ID)
		source := secret.Source
		if source == "" {
			source = domain.SecretKubernetes
		}
		switch source {
		case domain.SecretKubernetes:
			if secret.ResourceRequirementID != "" || secret.ResourceOutputName != "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_SECRET_REFERENCE: Kubernetes secret %s contains a resource output", secret.ID)
			}
		case domain.SecretResource:
			if _, ok := resources[secret.ResourceRequirementID]; !ok || secret.ResourceOutputName == "" || secret.RemoteProperty == "" {
				return domain.DeploymentGraph{}, fmt.Errorf("INVALID_SECRET_REFERENCE: %s has an invalid resource secret output", secret.ID)
			}
			addEdge(ResourceKey(secret.ResourceRequirementID), WorkloadKey(secret.WorkloadID), "secret:"+secret.ID)
		default:
			return domain.DeploymentGraph{}, fmt.Errorf("INVALID_SECRET_REFERENCE: %s has unsupported source %s", secret.ID, source)
		}
	}

	nodeList := make([]domain.GraphNode, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	sort.Slice(nodeList, func(i, j int) bool { return nodeList[i].Key < nodeList[j].Key })
	edgeList := make([]domain.GraphEdge, 0, len(edges))
	for _, edge := range edges {
		edgeList = append(edgeList, edge)
	}
	sort.Slice(edgeList, func(i, j int) bool {
		if edgeList[i].From != edgeList[j].From {
			return edgeList[i].From < edgeList[j].From
		}
		if edgeList[i].To != edgeList[j].To {
			return edgeList[i].To < edgeList[j].To
		}
		return edgeList[i].Via < edgeList[j].Via
	})
	order, err := topologicalOrder(nodeList, edgeList)
	if err != nil {
		return domain.DeploymentGraph{}, err
	}
	sort.Strings(refs)
	return domain.DeploymentGraph{Nodes: nodeList, Edges: edgeList, TopologicalOrder: order, ConfigurationReferenceIDs: refs}, nil
}

func hasPlanTimeOutput(workload domain.Workload, name string) bool {
	for _, output := range workload.ExposedOutputs {
		if output.Name == name && output.Availability == domain.OutputPlanTime {
			return true
		}
	}
	return false
}

func topologicalOrder(nodes []domain.GraphNode, edges []domain.GraphEdge) ([]string, error) {
	indegree := make(map[string]int, len(nodes))
	adjacency := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		indegree[node.Key] = 0
	}
	for _, edge := range edges {
		indegree[edge.To]++
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}
	for key := range adjacency {
		sort.Strings(adjacency[key])
	}
	ready := make([]string, 0)
	for key, degree := range indegree {
		if degree == 0 {
			ready = append(ready, key)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(nodes))
	for len(ready) > 0 {
		key := ready[0]
		ready = ready[1:]
		order = append(order, key)
		for _, next := range adjacency[key] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				sort.Strings(ready)
			}
		}
	}
	if len(order) != len(nodes) {
		return nil, fmt.Errorf("DEPENDENCY_CYCLE: deployment graph is cyclic")
	}
	return order, nil
}
