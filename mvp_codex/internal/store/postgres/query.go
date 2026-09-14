package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type DeploymentDetail struct {
	DeploymentID    string           `json:"deploymentId"`
	ApplicationID   string           `json:"applicationId"`
	Environment     string           `json:"environment"`
	TargetID        string           `json:"targetId"`
	LifecycleStatus string           `json:"lifecycleStatus"`
	PlanFingerprint string           `json:"planFingerprint"`
	Algorithm       string           `json:"algorithm"`
	Delivery        *DeliveryDetail  `json:"delivery,omitempty"`
	Execution       *ExecutionDetail `json:"execution,omitempty"`
	Steps           []DeploymentStep `json:"steps"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type DeliveryDetail struct {
	Status                  string `json:"status"`
	Reference               string `json:"reference,omitempty"`
	ArtifactURI             string `json:"artifactUri,omitempty"`
	ArtifactDigest          string `json:"artifactDigest,omitempty"`
	ExpectedApplicationName string `json:"expectedApplicationName,omitempty"`
	Error                   string `json:"error,omitempty"`
}

type ExecutionDetail struct {
	TrackingID  string `json:"trackingId"`
	Status      string `json:"status"`
	Phase       string `json:"phase,omitempty"`
	FailureCode string `json:"failureCode,omitempty"`
	Error       string `json:"error,omitempty"`
}

type DeploymentStep struct {
	Sequence    int        `json:"sequence"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (s *Store) GetDeploymentDetail(ctx context.Context, deploymentID string) (DeploymentDetail, error) {
	var detail DeploymentDetail
	err := s.pool.QueryRow(ctx, `SELECT deployment_id::text,application_id::text,environment,deployment_target,status,plan_fingerprint,plan_fingerprint_algo,created_at,updated_at FROM deployment WHERE deployment_id=$1`, deploymentID).Scan(
		&detail.DeploymentID, &detail.ApplicationID, &detail.Environment, &detail.TargetID, &detail.LifecycleStatus, &detail.PlanFingerprint, &detail.Algorithm, &detail.CreatedAt, &detail.UpdatedAt,
	)
	if err != nil {
		return DeploymentDetail{}, err
	}
	var delivery DeliveryDetail
	err = s.pool.QueryRow(ctx, `SELECT delivery_status,COALESCE(delivery_reference,''),COALESCE(artifact_uri,''),COALESCE(artifact_digest,''),COALESCE(expected_application_name,''),COALESCE(error_summary,'') FROM deployment_record WHERE deployment_id=$1`, deploymentID).Scan(
		&delivery.Status, &delivery.Reference, &delivery.ArtifactURI, &delivery.ArtifactDigest, &delivery.ExpectedApplicationName, &delivery.Error,
	)
	if err == nil {
		detail.Delivery = &delivery
	} else if err != pgx.ErrNoRows {
		return DeploymentDetail{}, err
	}
	var execution ExecutionDetail
	err = s.pool.QueryRow(ctx, `SELECT job_id::text,status,COALESCE(phase::text,''),COALESCE(failure_code,''),COALESCE(last_error,'') FROM deployment_execution_job WHERE deployment_id=$1`, deploymentID).Scan(
		&execution.TrackingID, &execution.Status, &execution.Phase, &execution.FailureCode, &execution.Error,
	)
	if err == nil {
		detail.Execution = &execution
	} else if err != pgx.ErrNoRows {
		return DeploymentDetail{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT sequence_number,step_name,status,COALESCE(error_summary,''),started_at,completed_at FROM deployment_step WHERE deployment_id=$1 ORDER BY sequence_number`, deploymentID)
	if err != nil {
		return DeploymentDetail{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var step DeploymentStep
		if err := rows.Scan(&step.Sequence, &step.Name, &step.Status, &step.Error, &step.StartedAt, &step.CompletedAt); err != nil {
			return DeploymentDetail{}, err
		}
		detail.Steps = append(detail.Steps, step)
	}
	if detail.Steps == nil {
		detail.Steps = []DeploymentStep{}
	}
	return detail, rows.Err()
}
