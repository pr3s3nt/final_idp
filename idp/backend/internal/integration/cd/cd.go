// Package cd is the CD Integration / CD Provider Interface and its Argo CD +
// GitOps implementation. Nothing outside this package knows about Argo CD.
package cd

import (
	"context"
	"time"

	"deploy/internal/domain"
	"deploy/internal/integration/kubernetes"
)

// SecretObject is materialized secret data applied directly to the cluster,
// never written to Git.
type SecretObject struct {
	Name   string
	Labels map[string]string
	Data   map[string][]byte
}

// WorkloadState is the desired state of one workload.
type WorkloadState struct {
	WorkloadID string
	Manifests  []byte // multi-document YAML without secret values
	Secrets    []SecretObject
}

// DesiredState is what one publish changes for an application + environment +
// target: upserted workloads and, in the final publish of a removal, workloads
// to drop. Workloads not mentioned are left untouched.
type DesiredState struct {
	Cluster       *kubernetes.ClusterAccess
	Target        string
	ApplicationID string // owns the delivery repository the state is written to
	Application   string
	Environment   string
	Namespace     string
	DeploymentID  string
	Wave          int
	Upsert        []WorkloadState
	Remove        []string
}

// Status is the CD view of an application, in neutral terms.
type Status struct {
	Revision string `json:"revision"`
	Sync     string `json:"sync"`   // SYNCED | SYNCING | OUT_OF_SYNC | UNKNOWN
	Health   string `json:"health"` // HEALTHY | PROGRESSING | DEGRADED | MISSING | UNKNOWN
	Message  string `json:"message,omitempty"`
}

// DeliveryRepositoryStore is the Delivery Repository Registry as the CD
// integration uses it: find where an application keeps its desired state, and
// record it the first time the IDP prepares it.
type DeliveryRepositoryStore interface {
	Find(ctx context.Context, applicationID string) (*domain.DeliveryRepository, error)
	Save(ctx context.Context, r domain.DeliveryRepository) (*domain.DeliveryRepository, error)
}

type Integration interface {
	// PublishDesiredDeploymentState returns a delivery reference once the CD
	// system accepted the new desired state; it does not mean workloads are ready.
	PublishDesiredDeploymentState(ctx context.Context, ds DesiredState) (string, error)
	// WaitForDelivery waits until the CD system has applied the delivery reference.
	WaitForDelivery(ctx context.Context, ds DesiredState, deliveryReference string, timeout time.Duration) error
	// RemoveApplication removes the CD application of an app + environment
	// (teardown), after its workloads are gone.
	RemoveApplication(ctx context.Context, ds DesiredState) error
	GetCDStatus(ctx context.Context, ds DesiredState) (*Status, error)
}
