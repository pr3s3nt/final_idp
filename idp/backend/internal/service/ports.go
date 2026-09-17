package service

import (
	"context"
	"time"

	"deploy/internal/domain"
	"deploy/internal/integration/kubernetes"
)

// WorkloadStatusProvider is the Workload Status Provider abstraction used by
// the worker and the query service; kubernetes.Adapter implements it.
type WorkloadStatusProvider interface {
	WaitForWorkloadsHealthy(ctx context.Context, a *kubernetes.ClusterAccess, workloads []kubernetes.WorkloadRef, timeout time.Duration) error
	WaitForWorkloadsRemoved(ctx context.Context, a *kubernetes.ClusterAccess, namespace string, workloadIDs []string, timeout time.Duration) error
	ReadWorkloadOutputs(ctx context.Context, a *kubernetes.ClusterAccess, namespace, workloadID string, outputs []string) (domain.Outputs, error)
	DeleteSecretsByLabel(ctx context.Context, a *kubernetes.ClusterAccess, namespace, selector string) error
	GetWorkloadStatus(ctx context.Context, a *kubernetes.ClusterAccess, namespace, workloadID string) (*kubernetes.WorkloadStatus, error)
	DeleteNamespace(ctx context.Context, a *kubernetes.ClusterAccess, namespace string) error
}

var _ WorkloadStatusProvider = (*kubernetes.Adapter)(nil)
