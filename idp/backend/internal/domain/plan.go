package domain

import "sort"

// ---------------------------------------------------------------------------
// Deployment Graph (transient, rebuilt for every create/confirm/execution)

type NodeKind string

const (
	NodeWorkload NodeKind = "WORKLOAD"
	NodeResource NodeKind = "RESOURCE"
)

type GraphNode struct {
	ID           string
	Kind         NodeKind
	Name         string
	ResourceType string // resources only
	Platform     bool   // implicit platform requirement (k8s-cluster, network)
	DependsOn    []string
}

type DeploymentGraph struct {
	Version     *ApplicationVersion
	Context     DeploymentContext
	Nodes       map[string]*GraphNode
	Resolutions map[string]*ResourceResolution // resource node ID -> resolution
	// ConfigurationReferenceIDs are the IDs of bindings that reference outputs.
	ConfigurationReferenceIDs []string
	// ComponentNames maps stable component IDs of every version to their
	// latest name, so components no longer in this version can be named.
	ComponentNames map[string]string
}

// ResourceResolution records which Resource Definition was selected for a
// resource node and why.
type ResourceResolution struct {
	RequirementID string
	Definition    *ResourceDefinition
	Reason        string
}

// SortedNodeIDs returns node IDs ordered by kind (resources first) then name,
// giving every consumer a deterministic iteration order.
func (g *DeploymentGraph) SortedNodeIDs() []string {
	ids := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := g.Nodes[ids[i]], g.Nodes[ids[j]]
		if a.Kind != b.Kind {
			return a.Kind == NodeResource
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	return ids
}

// Dependents returns the IDs of nodes that directly depend on id.
func (g *DeploymentGraph) Dependents(id string) []string {
	var out []string
	for _, nid := range g.SortedNodeIDs() {
		for _, d := range g.Nodes[nid].DependsOn {
			if d == id {
				out = append(out, nid)
				break
			}
		}
	}
	return out
}

// WavePlan is the scope and ordered waves produced by the Wave Planner.
type WavePlan struct {
	SelectedWorkloads []string
	Scope             map[string]bool
	Waves             [][]string
	// OutOfScopeDependencies are workloads the scope depends on but does not
	// redeploy; their outputs come from the running instances.
	OutOfScopeDependencies []string
	FullDeployment         bool
}

// ---------------------------------------------------------------------------
// Infrastructure / Deployment Plan (transient; only its fingerprint persists)

const (
	ActionCreate  = "CREATE"
	ActionUpdate  = "UPDATE"
	ActionReuse   = "REUSE"
	ActionLink    = "LINK"
	ActionDestroy = "DESTROY"
	ActionUnlink  = "UNLINK"
	ActionDeploy  = "DEPLOY"
	ActionRemove  = "REMOVE"
)

const FingerprintAlgo = "sha256-v1"

type Plan struct {
	Kind                DeploymentKind    `json:"kind"`
	ApplicationID       string            `json:"applicationId"`
	ApplicationName     string            `json:"applicationName"`
	VersionID           string            `json:"versionId"`
	VersionNumber       int               `json:"versionNumber"`
	CatalogVersionID    string            `json:"catalogVersionId"`
	CatalogVersion      int               `json:"catalogVersion"`
	Environment         Environment       `json:"environment"`
	Target              string            `json:"target"`
	Context             DeploymentContext `json:"context"`
	FullDeployment      bool              `json:"fullDeployment"`
	Waves               []PlanWave        `json:"waves"`
	Removals            []PlanWave        `json:"removals"`
	PotentialRedeploy   []PlanComponent   `json:"potentialRedeploy"`
	ConfigurationDigest string            `json:"configurationDigest"`
}

type PlanWave struct {
	Number int        `json:"number"`
	Items  []PlanItem `json:"items"`
}

type PlanComponent struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Kind NodeKind `json:"kind"`
}

type PlanItem struct {
	ComponentID string   `json:"componentId"`
	Name        string   `json:"name"`
	Kind        NodeKind `json:"kind"`
	Action      string   `json:"action"`
	DependsOn   []string `json:"dependsOn"`

	// Resources
	ResourceType         string                  `json:"resourceType,omitempty"`
	Platform             bool                    `json:"platform,omitempty"`
	DefinitionID         string                  `json:"definitionId,omitempty"`
	DefinitionName       string                  `json:"definitionName,omitempty"`
	DefinitionDigest     string                  `json:"definitionDigest,omitempty"`
	ManagementMode       ManagementMode          `json:"managementMode,omitempty"`
	ProvisionerReference string                  `json:"provisionerReference,omitempty"`
	ExistingReference    string                  `json:"existingReference,omitempty"`
	ResourceInstanceID   string                  `json:"resourceInstanceId,omitempty"`
	Parameters           map[string]any          `json:"parameters,omitempty"`
	AllowedOverrides     map[string]OverrideRule `json:"allowedOverrides,omitempty"`
	DataLossWarning      bool                    `json:"dataLossWarning,omitempty"`

	// Workloads
	ImageRepository string `json:"imageRepository,omitempty"`
	ImageVersion    string `json:"imageVersion,omitempty"`
	InclusionReason string `json:"inclusionReason,omitempty"`

	// Execution-only values; excluded from the fingerprint.
	FinalParameters map[string]any `json:"-"`
	NewBaseline     map[string]any `json:"-"`
}

// Fingerprint returns the sha256-v1 fingerprint of the plan. Developer-selected
// override values are not part of Plan, so they never change it.
func (p *Plan) Fingerprint() string { return HashCanonical(p) }

// Items returns every item of every wave in order.
func (p *Plan) Items() []*PlanItem {
	var out []*PlanItem
	for wi := range p.Waves {
		for ii := range p.Waves[wi].Items {
			out = append(out, &p.Waves[wi].Items[ii])
		}
	}
	return out
}

// Item finds the wave item for a component.
func (p *Plan) Item(componentID string) *PlanItem {
	for _, it := range p.Items() {
		if it.ComponentID == componentID {
			return it
		}
	}
	return nil
}

// WaveOf returns the wave number of a component, or -1.
func (p *Plan) WaveOf(componentID string) int {
	for _, w := range p.Waves {
		for _, it := range w.Items {
			if it.ComponentID == componentID {
				return w.Number
			}
		}
	}
	return -1
}

// ResourceItems returns every resource item of the deployment waves keyed by
// component ID.
func (p *Plan) ResourceItems() map[string]*PlanItem {
	out := map[string]*PlanItem{}
	for _, it := range p.Items() {
		if it.Kind == NodeResource {
			out[it.ComponentID] = it
		}
	}
	return out
}
