package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/identity"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
)

type Confirmation struct {
	TrackingID  string
	PlanChanged bool
	Plan        domain.InfrastructurePlan
}

func (s *Store) PersistPrepared(ctx context.Context, deploymentID string, prepared platform.Prepared) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	snapshot := prepared.Snapshot
	_, err = tx.Exec(ctx, `INSERT INTO deployment(deployment_id,application_id,environment_configuration_id,environment,deployment_target,plan_fingerprint,plan_fingerprint_algo,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,'AWAITING_CONFIRMATION')`, deploymentID, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.ID, snapshot.EnvironmentConfiguration.Environment, snapshot.RenderContext.TargetID, prepared.Plan.Fingerprint, prepared.Plan.Algorithm)
	if err != nil {
		return fmt.Errorf("insert deployment: %w", err)
	}
	appJSON, err := json.Marshal(snapshot.ApplicationDefinition)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(snapshot.EnvironmentConfiguration)
	if err != nil {
		return err
	}
	contextJSON, err := json.Marshal(snapshot.RenderContext)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO deployment_input_snapshot(deployment_id,schema_version,application_definition,environment_configuration,render_context,input_fingerprint,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, deploymentID, snapshot.SchemaVersion, appJSON, configJSON, contextJSON, snapshot.InputFingerprint, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	workloads := make(map[string]domain.Workload, len(snapshot.ApplicationDefinition.Workloads))
	for _, workload := range snapshot.ApplicationDefinition.Workloads {
		workloads[workload.ID] = workload
	}
	if len(snapshot.Images) != len(workloads) {
		return fmt.Errorf("INVALID_IMAGES: expected %d workload images, got %d", len(workloads), len(snapshot.Images))
	}
	for _, image := range snapshot.Images {
		workload, ok := workloads[image.WorkloadID]
		if !ok {
			return fmt.Errorf("INVALID_IMAGE: unknown workload %s", image.WorkloadID)
		}
		id, err := identity.NewUUID()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO workload_deployment(workload_deployment_id,deployment_id,workload_id,image_repository,image_version,image_digest) VALUES($1,$2,$3,$4,$5,$6)`, id, deploymentID, image.WorkloadID, workload.ImageRepository, image.Tag, image.Digest)
		if err != nil {
			return fmt.Errorf("insert workload image: %w", err)
		}
	}
	contextID, err := identity.NewUUID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO deployment_context(deployment_context_id,deployment_id,cloud_provider,region,target_specific_input) VALUES($1,$2,$3,$4,$5)`, contextID, deploymentID, snapshot.RenderContext.CloudProvider, snapshot.RenderContext.Region, contextJSON)
	if err != nil {
		return fmt.Errorf("insert deployment context: %w", err)
	}
	return tx.Commit(ctx)
}

type PreparedDeployment struct {
	ID                string
	Status            string
	StoredFingerprint string
	Snapshot          domain.DeploymentInputSnapshot
}

func loadPreparedTx(ctx context.Context, tx pgx.Tx, deploymentID string, forUpdate bool) (PreparedDeployment, error) {
	query := `SELECT deployment_id::text,status,plan_fingerprint FROM deployment WHERE deployment_id=$1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var result PreparedDeployment
	if err := tx.QueryRow(ctx, query, deploymentID).Scan(&result.ID, &result.Status, &result.StoredFingerprint); err != nil {
		return PreparedDeployment{}, err
	}
	var schemaVersion, inputFingerprint string
	var appJSON, configJSON, contextJSON []byte
	var createdAt time.Time
	err := tx.QueryRow(ctx, `SELECT schema_version,application_definition,environment_configuration,render_context,input_fingerprint,created_at FROM deployment_input_snapshot WHERE deployment_id=$1`, deploymentID).Scan(&schemaVersion, &appJSON, &configJSON, &contextJSON, &inputFingerprint, &createdAt)
	if err != nil {
		return PreparedDeployment{}, err
	}
	result.Snapshot.SchemaVersion = schemaVersion
	result.Snapshot.InputFingerprint = inputFingerprint
	result.Snapshot.CreatedAt = createdAt
	if err := decodeJSON(appJSON, &result.Snapshot.ApplicationDefinition); err != nil {
		return PreparedDeployment{}, err
	}
	if err := decodeJSON(configJSON, &result.Snapshot.EnvironmentConfiguration); err != nil {
		return PreparedDeployment{}, err
	}
	if err := decodeJSON(contextJSON, &result.Snapshot.RenderContext); err != nil {
		return PreparedDeployment{}, err
	}
	rows, err := tx.Query(ctx, `SELECT workload_id::text,image_version,image_digest FROM workload_deployment WHERE deployment_id=$1 ORDER BY workload_id`, deploymentID)
	if err != nil {
		return PreparedDeployment{}, err
	}
	for rows.Next() {
		var image domain.WorkloadImage
		if err := rows.Scan(&image.WorkloadID, &image.Tag, &image.Digest); err != nil {
			rows.Close()
			return PreparedDeployment{}, err
		}
		result.Snapshot.Images = append(result.Snapshot.Images, image)
	}
	rows.Close()
	return result, rows.Err()
}

func (s *Store) Confirm(ctx context.Context, deploymentID, expectedFingerprint, idempotencyKey, requestFingerprint string, engine *platform.Engine) (Confirmation, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Confirmation{}, err
	}
	defer tx.Rollback(ctx)
	if existing, found, err := findAcceptedTx(ctx, tx, deploymentID); err != nil {
		return Confirmation{}, err
	} else if found {
		return acceptedResult(existing, idempotencyKey, requestFingerprint)
	}
	prepared, err := loadPreparedTx(ctx, tx, deploymentID, true)
	if err != nil {
		return Confirmation{}, err
	}
	if existing, found, err := findAcceptedTx(ctx, tx, deploymentID); err != nil {
		return Confirmation{}, err
	} else if found {
		return acceptedResult(existing, idempotencyKey, requestFingerprint)
	}
	if prepared.Status != "AWAITING_CONFIRMATION" {
		return Confirmation{}, fmt.Errorf("ALREADY_ACCEPTED: deployment is %s", prepared.Status)
	}
	snapshot := prepared.Snapshot
	_, err = tx.Exec(ctx, `INSERT INTO deployment_scope_guard(application_id,environment,deployment_target,deployment_id,status) VALUES($1,$2,$3,NULL,'IDLE') ON CONFLICT DO NOTHING`, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, snapshot.RenderContext.TargetID)
	if err != nil {
		return Confirmation{}, err
	}
	var guardStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM deployment_scope_guard WHERE application_id=$1 AND environment=$2 AND deployment_target=$3 FOR UPDATE`, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, snapshot.RenderContext.TargetID).Scan(&guardStatus)
	if err != nil {
		return Confirmation{}, err
	}
	if guardStatus != "IDLE" {
		return Confirmation{}, fmt.Errorf("SCOPE_BUSY: deployment scope is %s", guardStatus)
	}
	definitions, err := loadDefinitionsTx(ctx, tx)
	if err != nil {
		return Confirmation{}, err
	}
	instances, err := loadInstancesTx(ctx, tx, snapshot)
	if err != nil {
		return Confirmation{}, err
	}
	rebuilt, err := engine.Prepare(snapshot, definitions, instances)
	if err != nil {
		return Confirmation{}, err
	}
	if expectedFingerprint != prepared.StoredFingerprint || expectedFingerprint != rebuilt.Plan.Fingerprint {
		_, err = tx.Exec(ctx, `UPDATE deployment SET plan_fingerprint=$2,updated_at=now() WHERE deployment_id=$1 AND status='AWAITING_CONFIRMATION'`, deploymentID, rebuilt.Plan.Fingerprint)
		if err != nil {
			return Confirmation{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Confirmation{}, err
		}
		return Confirmation{PlanChanged: true, Plan: rebuilt.Plan}, nil
	}
	jobID, err := identity.NewUUID()
	if err != nil {
		return Confirmation{}, err
	}
	recordID, err := identity.NewUUID()
	if err != nil {
		return Confirmation{}, err
	}
	command, err := tx.Exec(ctx, `UPDATE deployment SET status='QUEUED',updated_at=now() WHERE deployment_id=$1 AND status='AWAITING_CONFIRMATION' AND plan_fingerprint=$2`, deploymentID, expectedFingerprint)
	if err != nil || command.RowsAffected() != 1 {
		return Confirmation{}, fmt.Errorf("PLAN_CHANGED: deployment was concurrently modified")
	}
	guardCommand, err := tx.Exec(ctx, `UPDATE deployment_scope_guard SET deployment_id=$4,status='EXECUTING',updated_at=now() WHERE application_id=$1 AND environment=$2 AND deployment_target=$3 AND status='IDLE'`, snapshot.ApplicationDefinition.ID, snapshot.EnvironmentConfiguration.Environment, snapshot.RenderContext.TargetID, deploymentID)
	if err != nil {
		return Confirmation{}, err
	}
	if guardCommand.RowsAffected() != 1 {
		return Confirmation{}, fmt.Errorf("SCOPE_BUSY: deployment scope changed while confirming")
	}
	_, err = tx.Exec(ctx, `INSERT INTO deployment_record(deployment_record_id,deployment_id,environment,deployment_target,status,delivery_status) VALUES($1,$2,$3,$4,'QUEUED','NOT_PUBLISHED')`, recordID, deploymentID, snapshot.EnvironmentConfiguration.Environment, snapshot.RenderContext.TargetID)
	if err != nil {
		return Confirmation{}, err
	}
	steps := []string{"INFRASTRUCTURE_READY", "CONFIGURATION_RESOLVED", "MANIFEST_GENERATED"}
	for index, name := range steps {
		stepID, err := identity.NewUUID()
		if err != nil {
			return Confirmation{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO deployment_step(deployment_step_id,deployment_id,deployment_record_id,sequence_number,step_name,status) VALUES($1,$2,$3,$4,$5,'PENDING')`, stepID, deploymentID, recordID, index+1, name)
		if err != nil {
			return Confirmation{}, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO deployment_execution_job(job_id,deployment_id,idempotency_key,request_fingerprint,override_values,accepted_plan_fingerprint,status) VALUES($1,$2,$3,$4,'{}'::jsonb,$5,'QUEUED')`, jobID, deploymentID, idempotencyKey, requestFingerprint, expectedFingerprint)
	if err != nil {
		return Confirmation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Confirmation{}, err
	}
	return Confirmation{TrackingID: jobID}, nil
}

type acceptedJob struct {
	ID                 string
	IdempotencyKey     string
	RequestFingerprint string
}

func findAcceptedTx(ctx context.Context, tx pgx.Tx, deploymentID string) (acceptedJob, bool, error) {
	var job acceptedJob
	err := tx.QueryRow(ctx, `SELECT job_id::text,idempotency_key,request_fingerprint FROM deployment_execution_job WHERE deployment_id=$1`, deploymentID).Scan(&job.ID, &job.IdempotencyKey, &job.RequestFingerprint)
	if err == pgx.ErrNoRows {
		return acceptedJob{}, false, nil
	}
	return job, err == nil, err
}

func acceptedResult(job acceptedJob, key, requestFingerprint string) (Confirmation, error) {
	if job.IdempotencyKey == key && job.RequestFingerprint == requestFingerprint {
		return Confirmation{TrackingID: job.ID}, nil
	}
	if job.IdempotencyKey == key {
		return Confirmation{}, fmt.Errorf("IDEMPOTENCY_KEY_REUSED: key was accepted with another payload")
	}
	return Confirmation{}, fmt.Errorf("ALREADY_ACCEPTED: deployment already has a job")
}
