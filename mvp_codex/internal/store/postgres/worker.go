package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/identity"
)

type ClaimedJob struct {
	JobID                   string
	DeploymentID            string
	AcceptedPlanFingerprint string
	Snapshot                domain.DeploymentInputSnapshot
}

func (s *Store) ClaimNext(ctx context.Context, workerRunID string) (ClaimedJob, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return ClaimedJob{}, false, err
	}
	defer tx.Rollback(ctx)
	var job ClaimedJob
	err = tx.QueryRow(ctx, `SELECT job_id::text,deployment_id::text,accepted_plan_fingerprint FROM deployment_execution_job WHERE status='QUEUED' ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&job.JobID, &job.DeploymentID, &job.AcceptedPlanFingerprint)
	if err == pgx.ErrNoRows {
		return ClaimedJob{}, false, nil
	}
	if err != nil {
		return ClaimedJob{}, false, err
	}
	command, err := tx.Exec(ctx, `UPDATE deployment_execution_job SET status='CLAIMED',phase='INFRASTRUCTURE',worker_run_id=$2,started_at=now(),heartbeat_at=now(),updated_at=now() WHERE job_id=$1 AND status='QUEUED'`, job.JobID, workerRunID)
	if err != nil || command.RowsAffected() != 1 {
		return ClaimedJob{}, false, fmt.Errorf("claim job: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment SET status='RUNNING',updated_at=now() WHERE deployment_id=$1 AND status='QUEUED'`, job.DeploymentID); err != nil {
		return ClaimedJob{}, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_record SET status='RUNNING',updated_at=now() WHERE deployment_id=$1 AND status='QUEUED'`, job.DeploymentID); err != nil {
		return ClaimedJob{}, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_step SET status='RUNNING',started_at=now() WHERE deployment_id=$1 AND step_name='INFRASTRUCTURE_READY' AND status='PENDING'`, job.DeploymentID); err != nil {
		return ClaimedJob{}, false, err
	}
	prepared, err := loadPreparedTx(ctx, tx, job.DeploymentID, false)
	if err != nil {
		return ClaimedJob{}, false, err
	}
	job.Snapshot = prepared.Snapshot
	if err := tx.Commit(ctx); err != nil {
		return ClaimedJob{}, false, err
	}
	return job, true, nil
}

func (s *Store) PlanningData(ctx context.Context, snapshot domain.DeploymentInputSnapshot) ([]domain.ResourceDefinition, map[string]domain.ResourceInstance, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	definitions, err := loadDefinitionsTx(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	instances, err := loadInstancesTx(ctx, tx, snapshot)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return definitions, instances, nil
}

func (s *Store) ReserveResource(ctx context.Context, jobID, workerRunID string, snapshot domain.DeploymentInputSnapshot, item domain.InfrastructurePlanItem) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var deploymentID, recordID string
	err = tx.QueryRow(ctx, `SELECT j.deployment_id::text,r.deployment_record_id::text FROM deployment_execution_job j JOIN deployment_record r ON r.deployment_id=j.deployment_id WHERE j.job_id=$1 AND j.status='CLAIMED' AND j.worker_run_id=$2 FOR UPDATE OF j`, jobID, workerRunID).Scan(&deploymentID, &recordID)
	if err != nil {
		return fmt.Errorf("WORKER_FENCE_REJECTED: %w", err)
	}
	if item.Action == domain.PlanCreate {
		_, err = tx.Exec(ctx, `INSERT INTO resource_instance(resource_instance_id,resource_definition_id,owner_application_id,environment,resource_requirement_id,deployment_target,provider_state_reference,definition_fingerprint,parameters_fingerprint,version,recovery_verified,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,1,false,'PLANNED') ON CONFLICT(resource_instance_id) DO NOTHING`, item.ReferencedResourceInstanceID, item.ResourceDefinitionID, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, item.ResourceRequirementID, snapshot.RenderContext.TargetID, item.ProviderStateReference, item.DefinitionFingerprint, item.ParametersFingerprint)
		if err != nil {
			return fmt.Errorf("reserve resource: %w", err)
		}
		bindingID, _ := identity.NewUUID()
		_, err = tx.Exec(ctx, `INSERT INTO resource_instance_binding(resource_instance_binding_id,resource_instance_id,application_id,environment,resource_requirement_id,deployment_target,binding_role) VALUES($1,$2,$3,$4,$5,$6,'OWNER') ON CONFLICT DO NOTHING`, bindingID, item.ReferencedResourceInstanceID, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, item.ResourceRequirementID, snapshot.RenderContext.TargetID)
		if err != nil {
			return err
		}
	} else {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM resource_instance WHERE resource_instance_id=$1 FOR UPDATE`, item.ReferencedResourceInstanceID).Scan(&status); err != nil {
			return err
		}
		if status != "READY" {
			return fmt.Errorf("RESOURCE_RECOVERY_REQUIRED: reuse instance is %s", status)
		}
	}
	associationID, _ := identity.NewUUID()
	_, err = tx.Exec(ctx, `INSERT INTO deployment_record_resource_instance(deployment_record_resource_instance_id,deployment_record_id,resource_instance_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, associationID, recordID, item.ReferencedResourceInstanceID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkProvisioning(ctx context.Context, jobID, workerRunID, instanceID string) error {
	command, err := s.pool.Exec(ctx, `UPDATE resource_instance ri SET status='PROVISIONING',recovery_verified=false,updated_at=now() FROM deployment_execution_job j WHERE ri.resource_instance_id=$1 AND j.job_id=$2 AND j.worker_run_id=$3 AND j.status='CLAIMED' AND ri.status='PLANNED'`, instanceID, jobID, workerRunID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("WORKER_FENCE_REJECTED: cannot mark resource provisioning")
	}
	return nil
}

func (s *Store) CheckpointReady(ctx context.Context, jobID, workerRunID, instanceID, infrastructureReference string) error {
	command, err := s.pool.Exec(ctx, `UPDATE resource_instance ri SET status='READY',infrastructure_reference=$4,version=version+1,updated_at=now() FROM deployment_execution_job j WHERE ri.resource_instance_id=$1 AND j.job_id=$2 AND j.worker_run_id=$3 AND j.status='CLAIMED' AND ri.status='PROVISIONING'`, instanceID, jobID, workerRunID, infrastructureReference)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("WORKER_FENCE_REJECTED: cannot checkpoint resource")
	}
	return nil
}

func (s *Store) AdvanceStep(ctx context.Context, jobID, workerRunID, currentStep, nextStep, nextPhase string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var deploymentID string
	if err := tx.QueryRow(ctx, `SELECT deployment_id::text FROM deployment_execution_job WHERE job_id=$1 AND worker_run_id=$2 AND status='CLAIMED' FOR UPDATE`, jobID, workerRunID).Scan(&deploymentID); err != nil {
		return fmt.Errorf("WORKER_FENCE_REJECTED: %w", err)
	}
	command, err := tx.Exec(ctx, `UPDATE deployment_step SET status='SUCCEEDED',completed_at=now() WHERE deployment_id=$1 AND step_name=$2 AND status='RUNNING'`, deploymentID, currentStep)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("INVALID_EXECUTION_STATE: step %s is not RUNNING", currentStep)
	}
	if nextStep != "" {
		command, err = tx.Exec(ctx, `UPDATE deployment_step SET status='RUNNING',started_at=now() WHERE deployment_id=$1 AND step_name=$2 AND status='PENDING'`, deploymentID, nextStep)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return fmt.Errorf("INVALID_EXECUTION_STATE: step %s is not PENDING", nextStep)
		}
	}
	command, err = tx.Exec(ctx, `UPDATE deployment_execution_job SET phase=$3,updated_at=now(),heartbeat_at=now() WHERE job_id=$1 AND worker_run_id=$2 AND status='CLAIMED'`, jobID, workerRunID, nextPhase)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("WORKER_FENCE_REJECTED: cannot advance job phase")
	}
	return tx.Commit(ctx)
}

func (s *Store) FailExecution(ctx context.Context, jobID, workerRunID, code, message string, recoveryRequired bool) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var deploymentID string
	if err := tx.QueryRow(ctx, `SELECT deployment_id::text FROM deployment_execution_job WHERE job_id=$1 AND worker_run_id=$2 AND status='CLAIMED' FOR UPDATE`, jobID, workerRunID).Scan(&deploymentID); err != nil {
		return err
	}
	if len(message) > 512 {
		message = message[:512]
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_execution_job SET status='FAILED',failure_code=$3,last_error=$4,completed_at=now(),updated_at=now() WHERE job_id=$1 AND worker_run_id=$2`, jobID, workerRunID, code, message); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment SET status='FAILED',updated_at=now() WHERE deployment_id=$1`, deploymentID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_record SET status='FAILED',error_summary=$2,updated_at=now() WHERE deployment_id=$1`, deploymentID, message); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_step SET status=CASE WHEN status='RUNNING' THEN 'FAILED'::step_status ELSE 'SKIPPED'::step_status END,error_summary=CASE WHEN status='RUNNING' THEN $2 ELSE error_summary END,completed_at=CASE WHEN status='RUNNING' THEN now() ELSE completed_at END WHERE deployment_id=$1 AND status IN ('RUNNING','PENDING')`, deploymentID, message); err != nil {
		return err
	}
	guard := "IDLE"
	var owner any = nil
	if recoveryRequired {
		guard = "RECOVERY_REQUIRED"
		owner = deploymentID
	}
	_, err = tx.Exec(ctx, `UPDATE deployment_scope_guard SET status=$2,deployment_id=$3,updated_at=now() WHERE deployment_id=$1`, deploymentID, guard, owner)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SavePublicationIntent(ctx context.Context, jobID, workerRunID, uri, digest, applicationName string) error {
	command, err := s.pool.Exec(ctx, `UPDATE deployment_record r SET artifact_uri=$3,artifact_digest=$4,expected_application_name=$5,updated_at=now() FROM deployment_execution_job j WHERE j.job_id=$1 AND j.worker_run_id=$2 AND j.status='CLAIMED' AND j.phase='PUBLISH' AND r.deployment_id=j.deployment_id AND r.artifact_digest IS NULL`, jobID, workerRunID, uri, digest, applicationName)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("WORKER_FENCE_REJECTED: cannot save publication intent")
	}
	return nil
}

func (s *Store) CompleteExecution(ctx context.Context, jobID, workerRunID, deliveryReference, artifactDigest string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var deploymentID, expectedDigest string
	if err := tx.QueryRow(ctx, `SELECT j.deployment_id::text,r.artifact_digest FROM deployment_execution_job j JOIN deployment_record r ON r.deployment_id=j.deployment_id WHERE j.job_id=$1 AND j.worker_run_id=$2 AND j.status='CLAIMED' AND j.phase='PUBLISH' FOR UPDATE OF j,r`, jobID, workerRunID).Scan(&deploymentID, &expectedDigest); err != nil {
		return fmt.Errorf("WORKER_FENCE_REJECTED: %w", err)
	}
	if expectedDigest != artifactDigest {
		return fmt.Errorf("DELIVERY_ACK_MISMATCH: expected %s", expectedDigest)
	}
	var successful int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM deployment_step WHERE deployment_id=$1 AND status='SUCCEEDED'`, deploymentID).Scan(&successful); err != nil {
		return err
	}
	if successful != 3 {
		return fmt.Errorf("INVALID_EXECUTION_STATE: %d steps succeeded", successful)
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_record SET status='SUBMITTED',delivery_status='ACCEPTED',delivery_reference=$2,updated_at=now() WHERE deployment_id=$1`, deploymentID, deliveryReference); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment SET status='SUBMITTED',updated_at=now() WHERE deployment_id=$1`, deploymentID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_execution_job SET status='SUCCEEDED',phase='COMPLETE',completed_at=now(),updated_at=now() WHERE job_id=$1 AND worker_run_id=$2`, jobID, workerRunID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_scope_guard SET status='IDLE',deployment_id=NULL,updated_at=now() WHERE deployment_id=$1`, deploymentID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
