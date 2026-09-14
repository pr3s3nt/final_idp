package domain

import "time"

const (
	SnapshotSchemaVersion = "mvp-source-v1"
	PlanFingerprintAlgo   = "sha256-mvp-v1"
)

type OutputAvailability string

const (
	OutputPlanTime OutputAvailability = "PLAN_TIME"
	OutputRuntime  OutputAvailability = "RUNTIME"
)

type WorkloadOutputDefinition struct {
	Name         string             `json:"name"`
	Availability OutputAvailability `json:"availability"`
	Port         int                `json:"port,omitempty"`
}

type Workload struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Type            string                     `json:"type"`
	ImageRepository string                     `json:"imageRepository"`
	Port            int                        `json:"port,omitempty"`
	ExposedOutputs  []WorkloadOutputDefinition `json:"exposedOutputs,omitempty"`
}

type ResourceRequirement struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ResourceType string `json:"resourceType"`
}

type DependencyTargetType string

const (
	DependencyWorkload DependencyTargetType = "WORKLOAD"
	DependencyResource DependencyTargetType = "RESOURCE"
)

// Dependency means SourceWorkloadID depends on exactly one target.
type Dependency struct {
	ID                          string               `json:"id"`
	SourceWorkloadID            string               `json:"sourceWorkloadId"`
	TargetType                  DependencyTargetType `json:"targetType"`
	TargetWorkloadID            string               `json:"targetWorkloadId,omitempty"`
	TargetResourceRequirementID string               `json:"targetResourceRequirementId,omitempty"`
}

type ApplicationDefinition struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Workloads            []Workload            `json:"workloads"`
	ResourceRequirements []ResourceRequirement `json:"resourceRequirements,omitempty"`
	Dependencies         []Dependency          `json:"dependencies,omitempty"`
}

type ValueSource string

const (
	ValueDirect         ValueSource = "DIRECT"
	ValueResourceOutput ValueSource = "RESOURCE_OUTPUT"
	ValueWorkloadOutput ValueSource = "WORKLOAD_OUTPUT"
)

type ConfigurationValue struct {
	ID                    string      `json:"id"`
	WorkloadID            string      `json:"workloadId"`
	Name                  string      `json:"name"`
	Source                ValueSource `json:"source"`
	DirectValue           string      `json:"directValue,omitempty"`
	ResourceRequirementID string      `json:"resourceRequirementId,omitempty"`
	ResourceOutputName    string      `json:"resourceOutputName,omitempty"`
	ReferencedWorkloadID  string      `json:"referencedWorkloadId,omitempty"`
	WorkloadOutputName    string      `json:"workloadOutputName,omitempty"`
}

type SecretReference struct {
	ID                    string       `json:"id"`
	WorkloadID            string       `json:"workloadId"`
	Name                  string       `json:"name"`
	Source                SecretSource `json:"source,omitempty"`
	Target                string       `json:"target"`
	Namespace             string       `json:"namespace"`
	SecretName            string       `json:"secretName"`
	UID                   string       `json:"uid,omitempty"`
	Key                   string       `json:"key"`
	ResourceRequirementID string       `json:"resourceRequirementId,omitempty"`
	ResourceOutputName    string       `json:"resourceOutputName,omitempty"`
	RemoteProperty        string       `json:"remoteProperty,omitempty"`
}

type SecretSource string

const (
	SecretKubernetes SecretSource = "KUBERNETES_SECRET"
	SecretResource   SecretSource = "RESOURCE_SECRET"
)

type EnvironmentConfiguration struct {
	ID          string               `json:"id"`
	Environment string               `json:"environment"`
	Values      []ConfigurationValue `json:"values,omitempty"`
	Secrets     []SecretReference    `json:"secrets,omitempty"`
}

type DeploymentContext struct {
	TargetID        string            `json:"targetId"`
	CloudProvider   string            `json:"cloudProvider"`
	Region          string            `json:"region"`
	ClusterIdentity string            `json:"clusterIdentity"`
	Namespace       string            `json:"namespace"`
	NamingPolicy    string            `json:"namingPolicy"`
	RendererVersion string            `json:"rendererVersion"`
	AdapterVersions map[string]string `json:"adapterVersions"`
	// ProvisionerInputs are immutable, target-specific references produced by
	// bootstrap (for example an RDS subnet group and security-group IDs). They
	// are part of the accepted snapshot and must never contain credentials.
	ProvisionerInputs map[string]any `json:"provisionerInputs,omitempty"`
}

type WorkloadImage struct {
	WorkloadID string `json:"workloadId"`
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`
}

type DeploymentInputSnapshot struct {
	SchemaVersion            string                   `json:"schemaVersion"`
	ApplicationDefinition    ApplicationDefinition    `json:"applicationDefinition"`
	EnvironmentConfiguration EnvironmentConfiguration `json:"environmentConfiguration"`
	RenderContext            DeploymentContext        `json:"renderContext"`
	Images                   []WorkloadImage          `json:"images"`
	InputFingerprint         string                   `json:"inputFingerprint"`
	CreatedAt                time.Time                `json:"createdAt"`
}

type GraphNodeKind string

const (
	GraphNodeWorkload GraphNodeKind = "WORKLOAD"
	GraphNodeResource GraphNodeKind = "RESOURCE"
)

type GraphNode struct {
	Key  string        `json:"key"`
	Kind GraphNodeKind `json:"kind"`
	ID   string        `json:"id"`
	Name string        `json:"name"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Via  string `json:"via"`
}

type DeploymentGraph struct {
	DeploymentID              string      `json:"deploymentId,omitempty"`
	Nodes                     []GraphNode `json:"nodes"`
	Edges                     []GraphEdge `json:"edges"`
	TopologicalOrder          []string    `json:"topologicalOrder"`
	ConfigurationReferenceIDs []string    `json:"configurationReferenceIds,omitempty"`
}

type ContextSelector struct {
	CloudProvider string `json:"cloudProvider"`
	Region        string `json:"region,omitempty"`
}

// ResourceDefinition is platform-managed catalog data. The provisioner URI is
// resolved by the worker's registry; orchestration code never switches on a
// concrete resource such as PostgreSQL.
type ResourceDefinition struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	ResourceType         string            `json:"resourceType"`
	ProvisionerReference string            `json:"provisionerReference"`
	SupportedContexts    []ContextSelector `json:"supportedContexts"`
	DefaultParameters    map[string]any    `json:"defaultParameters"`
	AllowedOverrides     map[string]any    `json:"allowedOverrides"`
	ExposedOutputs       []string          `json:"exposedOutputs"`
	SensitiveOutputs     []string          `json:"sensitiveOutputs,omitempty"`
	Retired              bool              `json:"retired"`
}

type ResourceResolution struct {
	Requirement ResourceRequirement `json:"requirement"`
	Definition  ResourceDefinition  `json:"definition"`
}

type ResourceStatus string

const (
	ResourcePlanned      ResourceStatus = "PLANNED"
	ResourceProvisioning ResourceStatus = "PROVISIONING"
	ResourceReady        ResourceStatus = "READY"
	ResourceFailed       ResourceStatus = "FAILED"
	ResourceRetired      ResourceStatus = "RETIRED"
)

type ResourceInstance struct {
	ID                      string         `json:"id"`
	ResourceDefinitionID    string         `json:"resourceDefinitionId"`
	OwnerApplicationID      string         `json:"ownerApplicationId"`
	Environment             string         `json:"environment"`
	ResourceRequirementID   string         `json:"resourceRequirementId"`
	DeploymentTarget        string         `json:"deploymentTarget"`
	ProviderStateReference  string         `json:"providerStateReference"`
	InfrastructureReference string         `json:"infrastructureReference,omitempty"`
	DefinitionFingerprint   string         `json:"definitionFingerprint"`
	ParametersFingerprint   string         `json:"parametersFingerprint"`
	Version                 int64          `json:"version"`
	RecoveryVerified        bool           `json:"recoveryVerified"`
	Status                  ResourceStatus `json:"status"`
}

type PlanAction string

const (
	PlanCreate PlanAction = "CREATE"
	PlanReuse  PlanAction = "REUSE"
)

type InfrastructurePlanItem struct {
	ResourceRequirementID        string         `json:"resourceRequirementId"`
	ResourceDefinitionID         string         `json:"resourceDefinitionId"`
	ProvisionerReference         string         `json:"provisionerReference"`
	Action                       PlanAction     `json:"action"`
	ResolvedParameters           map[string]any `json:"resolvedParameters"`
	ExposedOutputs               []string       `json:"exposedOutputs"`
	ReferencedResourceInstanceID string         `json:"referencedResourceInstanceId"`
	ReferencedInstanceVersion    int64          `json:"referencedInstanceVersion,omitempty"`
	ProviderStateReference       string         `json:"providerStateReference"`
	DefinitionFingerprint        string         `json:"definitionFingerprint"`
	ParametersFingerprint        string         `json:"parametersFingerprint"`
}

type InfrastructurePlan struct {
	ApplicationID    string                   `json:"applicationId"`
	Environment      string                   `json:"environment"`
	DeploymentTarget string                   `json:"deploymentTarget"`
	InputFingerprint string                   `json:"inputFingerprint"`
	Items            []InfrastructurePlanItem `json:"items"`
	Fingerprint      string                   `json:"fingerprint"`
	Algorithm        string                   `json:"algorithm"`
}
