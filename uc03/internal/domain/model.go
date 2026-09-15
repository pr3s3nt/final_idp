// Package domain holds the UC-03 domain objects from 02_domain_model, including
// the transient execution objects (graph, resolution, plan, outputs).
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Environment string

const (
	Staging    Environment = "STAGING"
	Production Environment = "PRODUCTION"
)

func (e Environment) Valid() bool { return e == Staging || e == Production }

// ---------------------------------------------------------------------------
// Application Definition (immutable versions, stable component IDs)

type ApplicationVersion struct {
	ApplicationID   string
	ApplicationName string
	VersionID       string
	VersionNumber   int
	Workloads       []Workload
	Resources       []ResourceRequirement
	Dependencies    []Dependency
}

type Workload struct {
	ID              string
	Name            string
	Type            string
	ImageRepository string
	Port            *int
	ExposedOutputs  []string
	Variables       []ConfigDefinition
	Secrets         []ConfigDefinition
}

type ConfigDefinition struct {
	ID       string
	Name     string
	Required bool
}

type ResourceRequirement struct {
	ID           string
	Name         string
	ResourceType string
}

const (
	TargetWorkload = "WORKLOAD"
	TargetResource = "RESOURCE"
)

type Dependency struct {
	ID               string
	SourceWorkloadID string
	TargetType       string
	TargetID         string
}

func (v *ApplicationVersion) Workload(id string) *Workload {
	for i := range v.Workloads {
		if v.Workloads[i].ID == id {
			return &v.Workloads[i]
		}
	}
	return nil
}

func (v *ApplicationVersion) Resource(id string) *ResourceRequirement {
	for i := range v.Resources {
		if v.Resources[i].ID == id {
			return &v.Resources[i]
		}
	}
	return nil
}

// DependsOn reports whether workload source declares depends-on target.
func (v *ApplicationVersion) DependsOn(sourceWorkloadID, targetID string) bool {
	for _, d := range v.Dependencies {
		if d.SourceWorkloadID == sourceWorkloadID && d.TargetID == targetID {
			return true
		}
	}
	return false
}

// PlatformRequirementID is the stable component ID of an implicit platform
// requirement (k8s-cluster, network) of an application. It is derived, so
// planning never needs to write an identity row.
func PlatformRequirementID(applicationID, requirementType string) string {
	ns := uuid.MustParse(applicationID)
	return uuid.NewSHA1(ns, []byte("platform-requirement:"+requirementType)).String()
}

// ---------------------------------------------------------------------------
// Environment Configuration (not versioned, references stable IDs)

const (
	SourceDirect         = "DIRECT"
	SourceResourceOutput = "RESOURCE_OUTPUT"
	SourceWorkloadOutput = "WORKLOAD_OUTPUT"
	SourceSecretRef      = "SECRET_REF"
)

type EnvironmentConfiguration struct {
	ID            string
	ApplicationID string
	Environment   Environment
	Variables     []ConfiguredValue
	Secrets       []ConfiguredValue
}

// ConfiguredValue is an Environment Variable or Secret binding with exactly one
// value source.
type ConfiguredValue struct {
	ID           string
	WorkloadID   string
	DefinitionID string
	Name         string
	Source       string
	DirectValue  string // DIRECT (variables only)
	SecretRef    string // SECRET_REF (secrets only)
	RefID        string // resource requirement ID or workload ID
	OutputName   string
}

// ---------------------------------------------------------------------------
// Platform Resource Definition catalog

type ManagementMode string

const (
	Managed  ManagementMode = "MANAGED"
	Existing ManagementMode = "EXISTING"
)

// Implicit platform requirement types.
const (
	ResourceTypeCluster = "k8s-cluster"
	ResourceTypeNetwork = "network"
)

type ResourceDefinition struct {
	ID                        string
	Name                      string
	ResourceType              string
	ProvisionerReference      string
	SupportedContexts         []map[string]string
	DefaultParameters         map[string]any
	AllowedOverrides          map[string]OverrideRule
	ExposedOutputs            []string
	SensitiveOutputs          []string
	ManagementMode            ManagementMode
	ApplicabilityConditions   map[string][]string
	ExistingResourceReference string
	Requires                  []string
}

// OverrideRule is the policy for one parameter a Developer may override.
type OverrideRule struct {
	Type          string   `json:"type" yaml:"type"` // string | integer | number | enum | cidr
	Enum          []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Min           *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max           *float64 `json:"max,omitempty" yaml:"max,omitempty"`
	Immutable     bool     `json:"immutable,omitempty" yaml:"immutable,omitempty"`
	IncreaseOnly  bool     `json:"increaseOnly,omitempty" yaml:"increaseOnly,omitempty"`
	MaxStep       *float64 `json:"maxStep,omitempty" yaml:"maxStep,omitempty"` // for enum: max index step upward
	CIDRPrefixLen *int     `json:"cidrPrefixLength,omitempty" yaml:"cidrPrefixLength,omitempty"`
	CIDRWithin    string   `json:"cidrWithin,omitempty" yaml:"cidrWithin,omitempty"`
}

func (d *ResourceDefinition) HasOutput(name string) bool {
	for _, o := range d.ExposedOutputs {
		if o == name {
			return true
		}
	}
	return d.IsSensitiveOutput(name)
}

func (d *ResourceDefinition) IsSensitiveOutput(name string) bool {
	for _, o := range d.SensitiveOutputs {
		if o == name {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Deployment context and targets

// DeploymentContext is what resolves Resource Definitions: the selected target
// plus cloud provider, region and target-specific input.
type DeploymentContext struct {
	Target              string            `json:"target"`
	CloudProvider       string            `json:"cloudProvider,omitempty"`
	Region              string            `json:"region,omitempty"`
	TargetSpecificInput map[string]string `json:"targetSpecificInput,omitempty"`
}

// Values exposes the context as match keys for supported_contexts.
func (c DeploymentContext) Values() map[string]string {
	v := map[string]string{"target": c.Target}
	if c.CloudProvider != "" {
		v["cloudProvider"] = c.CloudProvider
	}
	if c.Region != "" {
		v["region"] = c.Region
	}
	return v
}

// ---------------------------------------------------------------------------
// Instances (durable current state per environment + target)

type ResourceInstanceStatus string

const (
	RIPlanned      ResourceInstanceStatus = "PLANNED"
	RIProvisioning ResourceInstanceStatus = "PROVISIONING"
	RIReady        ResourceInstanceStatus = "READY"
	RIFailed       ResourceInstanceStatus = "FAILED"
	RIDestroyed    ResourceInstanceStatus = "DESTROYED"
	RIUnlinked     ResourceInstanceStatus = "UNLINKED"
)

type ResourceInstance struct {
	ID                      string
	ApplicationID           string
	Environment             Environment
	RequirementID           string
	DefinitionID            string
	Target                  string
	InfrastructureReference string
	ProviderStateReference  string
	Status                  ResourceInstanceStatus
	OutputFingerprint       string
	AppliedOverrides        map[string]any
	AppliedInputFingerprint string
	CreatedAt, UpdatedAt    time.Time
}

type WorkloadInstanceStatus string

const (
	WIDeploying WorkloadInstanceStatus = "DEPLOYING"
	WIHealthy   WorkloadInstanceStatus = "HEALTHY"
	WIFailed    WorkloadInstanceStatus = "FAILED"
	WIRemoved   WorkloadInstanceStatus = "REMOVED"
)

type WorkloadInstance struct {
	ID                          string
	ApplicationID               string
	WorkloadID                  string
	Environment                 Environment
	Target                      string
	CurrentWorkloadDeploymentID string
	Status                      WorkloadInstanceStatus
	OutputFingerprint           string
	// Joined from the current workload deployment.
	RunningVersionID     string
	RunningVersionNumber int
	ImageRepository      string
	ImageVersion         string
	DeploymentID         string
}

// ---------------------------------------------------------------------------
// Deployment aggregate

type DeploymentStatus string

const (
	AwaitingConfirmation DeploymentStatus = "AWAITING_CONFIRMATION"
	Confirmed            DeploymentStatus = "CONFIRMED"
	Deploying            DeploymentStatus = "DEPLOYING"
	Succeeded            DeploymentStatus = "SUCCEEDED"
	Failed               DeploymentStatus = "FAILED"
)

type DeploymentKind string

const (
	KindDeploy   DeploymentKind = "DEPLOY"
	KindTeardown DeploymentKind = "TEARDOWN"
)

const (
	Selected = "SELECTED"
	Cascaded = "CASCADED"
)

type Deployment struct {
	ID                         string
	ApplicationID              string
	VersionID                  string
	EnvironmentConfigurationID string
	Environment                Environment
	Target                     string
	Kind                       DeploymentKind
	PlanFingerprint            string
	PlanFingerprintAlgo        string
	Status                     DeploymentStatus
	Context                    DeploymentContext
	WorkloadDeployments        []WorkloadDeployment
	CreatedAt, UpdatedAt       time.Time
}

type WorkloadDeployment struct {
	ID              string
	WorkloadID      string
	ImageRepository string
	ImageVersion    string
	InclusionReason string
	WaveNumber      int
}

// ImageRef is the full image reference used in manifests.
func (w WorkloadDeployment) ImageRef() string { return w.ImageRepository + ":" + w.ImageVersion }

type StepName string

const (
	StepPlanVerified          StepName = "PLAN_VERIFIED"
	StepInfrastructureReady   StepName = "INFRASTRUCTURE_READY"
	StepConfigurationResolved StepName = "CONFIGURATION_RESOLVED"
	StepManifestGenerated     StepName = "MANIFEST_GENERATED"
	StepCDSynced              StepName = "CD_SYNCED"
	StepApplicationReady      StepName = "APPLICATION_READY"
	StepRemoved               StepName = "REMOVED"
	StepDestroyed             StepName = "DESTROYED"
	StepUnlinked              StepName = "UNLINKED"
)

type StepStatus string

const (
	StepPending   StepStatus = "PENDING"
	StepRunning   StepStatus = "RUNNING"
	StepSucceeded StepStatus = "SUCCEEDED"
	StepFailed    StepStatus = "FAILED"
	StepSkipped   StepStatus = "SKIPPED"
)

// ---------------------------------------------------------------------------
// Transient execution objects

// Output is a runtime Resource Output or Workload Output value. It is never
// persisted; only fingerprints of output sets are.
type Output struct {
	Value     string
	Sensitive bool
}

type Outputs map[string]Output

// Fingerprint hashes the canonical (sorted) output set.
func (o Outputs) Fingerprint() string {
	return HashCanonical(o.canonical())
}

func (o Outputs) canonical() map[string]any {
	m := make(map[string]any, len(o))
	for k, v := range o {
		m[k] = v.Value
	}
	return m
}

// HashCanonical returns the SHA-256 hex of the canonical JSON of v.
func HashCanonical(v any) string {
	sum := sha256.Sum256(CanonicalJSON(v))
	return hex.EncodeToString(sum[:])
}
