package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain"
)

// DeploymentRepository stores the Deployment aggregate: deployment, context,
// workload deployments, execution job, record and steps.
type DeploymentRepository struct{ DB *DB }

// PersistDeployment stores a validated deployment in AWAITING_CONFIRMATION in
// one transaction. It re-checks the in-progress guard under the application
// row lock so two creates cannot race past it.
func (r *DeploymentRepository) PersistDeployment(ctx context.Context, d *domain.Deployment) error {
	return r.DB.InTx(ctx, func(tx pgx.Tx) error {
		if err := lockApplication(ctx, tx, d.ApplicationID); err != nil {
			return err
		}
		if err := checkNotInProgress(ctx, tx, d.ApplicationID, d.Environment, d.Target, ""); err != nil {
			return err
		}
		now := time.Now().UTC()
		d.ID, d.CreatedAt, d.UpdatedAt = uuid.NewString(), now, now
		d.Status = domain.AwaitingConfirmation
		if _, err := tx.Exec(ctx, `INSERT INTO deployment (deployment_id, application_id, application_definition_version_id, catalog_version_id,
			environment_configuration_id, environment, deployment_target, kind, plan_fingerprint, plan_fingerprint_algo, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)`,
			d.ID, d.ApplicationID, d.VersionID, d.CatalogVersionID, d.EnvironmentConfigurationID, string(d.Environment), d.Target, string(d.Kind),
			d.PlanFingerprint, d.PlanFingerprintAlgo, string(d.Status), now); err != nil {
			return err
		}
		input, _ := json.Marshal(nonNilStringMap(d.Context.TargetSpecificInput))
		if _, err := tx.Exec(ctx, `INSERT INTO deployment_context (deployment_context_id, deployment_id, cloud_provider, region, target_specific_input)
			VALUES ($1, $2, nullif($3, ''), nullif($4, ''), $5)`, uuid.NewString(), d.ID, d.Context.CloudProvider, d.Context.Region, input); err != nil {
			return err
		}
		for i := range d.WorkloadDeployments {
			wd := &d.WorkloadDeployments[i]
			wd.ID = uuid.NewString()
			if err := insertWorkloadDeployment(ctx, tx, d.ID, wd); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertWorkloadDeployment(ctx context.Context, q Querier, deploymentID string, wd *domain.WorkloadDeployment) error {
	_, err := q.Exec(ctx, `INSERT INTO workload_deployment (workload_deployment_id, deployment_id, workload_id, image_repository,
		image_version, inclusion_reason, wave_number) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		wd.ID, deploymentID, wd.WorkloadID, wd.ImageRepository, wd.ImageVersion, wd.InclusionReason, wd.WaveNumber)
	return err
}

func lockApplication(ctx context.Context, tx pgx.Tx, applicationID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT application_id FROM application_definition WHERE application_id = $1 FOR UPDATE`, applicationID).Scan(&id)
}

func checkNotInProgress(ctx context.Context, q Querier, applicationID string, env domain.Environment, target, excludeID string) error {
	var otherID string
	err := q.QueryRow(ctx, `SELECT deployment_id FROM deployment
		WHERE application_id = $1 AND environment = $2 AND deployment_target = $3
		  AND status IN ('CONFIRMED', 'DEPLOYING') AND deployment_id::text <> $4 LIMIT 1`,
		applicationID, string(env), target, excludeID).Scan(&otherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return domain.Reject(domain.CodeDeploymentInProgress,
		"deployment %s is still running on this environment and target; wait for it to finish", otherID)
}

// CheckNotInProgress is the A1-12 guard used before planning.
func (r *DeploymentRepository) CheckNotInProgress(ctx context.Context, applicationID string, env domain.Environment, target string) error {
	return checkNotInProgress(ctx, r.DB.Pool, applicationID, env, target, "")
}

// FindByIDWithPersistedInputs loads the deployment, its context and workload
// deployments (images).
func (r *DeploymentRepository) FindByIDWithPersistedInputs(ctx context.Context, id string) (*domain.Deployment, error) {
	d := &domain.Deployment{}
	var env, kind, status string
	var cloud, region *string
	var input []byte
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT d.deployment_id, d.application_id, d.application_definition_version_id, d.catalog_version_id, cv.version_number,
		       d.environment_configuration_id, d.environment::text, d.deployment_target, d.kind::text, d.plan_fingerprint,
		       d.plan_fingerprint_algo, d.status::text, d.created_at, d.updated_at, c.cloud_provider, c.region, c.target_specific_input
		FROM deployment d JOIN deployment_context c USING (deployment_id) JOIN catalog_version cv USING (catalog_version_id)
		WHERE d.deployment_id = $1`, id).
		Scan(&d.ID, &d.ApplicationID, &d.VersionID, &d.CatalogVersionID, &d.CatalogVersionNumber, &d.EnvironmentConfigurationID, &env, &d.Target, &kind,
			&d.PlanFingerprint, &d.PlanFingerprintAlgo, &status, &d.CreatedAt, &d.UpdatedAt, &cloud, &region, &input)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "deployment %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	d.Environment, d.Kind, d.Status = domain.Environment(env), domain.DeploymentKind(kind), domain.DeploymentStatus(status)
	d.Context.Target = d.Target
	if cloud != nil {
		d.Context.CloudProvider = *cloud
	}
	if region != nil {
		d.Context.Region = *region
	}
	if err := json.Unmarshal(input, &d.Context.TargetSpecificInput); err != nil {
		return nil, err
	}
	rows, err := r.DB.Pool.Query(ctx, `SELECT workload_deployment_id, workload_id, image_repository, image_version,
		inclusion_reason::text, wave_number FROM workload_deployment WHERE deployment_id = $1 ORDER BY wave_number, workload_id`, id)
	if err != nil {
		return nil, err
	}
	d.WorkloadDeployments, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.WorkloadDeployment, error) {
		var wd domain.WorkloadDeployment
		err := row.Scan(&wd.ID, &wd.WorkloadID, &wd.ImageRepository, &wd.ImageVersion, &wd.InclusionReason, &wd.WaveNumber)
		return wd, err
	})
	return d, err
}

// CompareAndSetPlanFingerprint refreshes the stored fingerprint after a
// PLAN_CHANGED rebuild, only if nobody changed it meanwhile.
func (r *DeploymentRepository) CompareAndSetPlanFingerprint(ctx context.Context, id string, expectedStatus domain.DeploymentStatus, expected, updated string) (bool, error) {
	tag, err := r.DB.Pool.Exec(ctx, `UPDATE deployment SET plan_fingerprint = $4, updated_at = now()
		WHERE deployment_id = $1 AND status = $2 AND plan_fingerprint = $3`, id, string(expectedStatus), expected, updated)
	return tag.RowsAffected() == 1, err
}

// ConfirmDeploymentAndCreateJob atomically moves AWAITING_CONFIRMATION to
// CONFIRMED and creates the execution job with the selected overrides. It
// returns false when the compare-and-swap lost.
func (r *DeploymentRepository) ConfirmDeploymentAndCreateJob(ctx context.Context, d *domain.Deployment, overrides map[string]map[string]any) (bool, error) {
	confirmed := false
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		if err := lockApplication(ctx, tx, d.ApplicationID); err != nil {
			return err
		}
		if err := checkNotInProgress(ctx, tx, d.ApplicationID, d.Environment, d.Target, d.ID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE deployment SET status = 'CONFIRMED', updated_at = now()
			WHERE deployment_id = $1 AND status = 'AWAITING_CONFIRMATION'`, d.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return nil
		}
		if overrides == nil {
			overrides = map[string]map[string]any{}
		}
		body, _ := json.Marshal(overrides)
		if _, err := tx.Exec(ctx, `INSERT INTO deployment_execution_job (job_id, deployment_id, selected_overrides, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'QUEUED', now(), now())`, uuid.NewString(), d.ID, body); err != nil {
			return err
		}
		confirmed = true
		return nil
	})
	return confirmed, err
}

type ClaimedJob struct {
	JobID             string
	DeploymentID      string
	RecordID          string
	SelectedOverrides map[string]map[string]any
}

// ClaimNextExecutionJob takes the oldest queued job, marks it RUNNING, moves
// its deployment CONFIRMED -> DEPLOYING and creates the Deployment Record, all
// in one transaction. SKIP LOCKED keeps two workers from taking the same job.
func (r *DeploymentRepository) ClaimNextExecutionJob(ctx context.Context) (*ClaimedJob, error) {
	var job *ClaimedJob
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		j := &ClaimedJob{}
		var overrides []byte
		err := tx.QueryRow(ctx, `SELECT job_id, deployment_id, selected_overrides FROM deployment_execution_job
			WHERE status = 'QUEUED' ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&j.JobID, &j.DeploymentID, &overrides)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := json.Unmarshal(overrides, &j.SelectedOverrides); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE deployment_execution_job SET status = 'RUNNING', updated_at = now() WHERE job_id = $1`, j.JobID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE deployment SET status = 'DEPLOYING', updated_at = now() WHERE deployment_id = $1 AND status = 'CONFIRMED'`, j.DeploymentID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("claimed job whose deployment is not CONFIRMED")
		}
		j.RecordID = uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO deployment_record (deployment_record_id, deployment_id, environment, deployment_target,
			removed_components, status, created_at, updated_at)
			SELECT $1, deployment_id, environment, deployment_target, '[]', 'DEPLOYING', now(), now() FROM deployment WHERE deployment_id = $2`,
			j.RecordID, j.DeploymentID); err != nil {
			return err
		}
		job = j
		return nil
	})
	return job, err
}

// InsertCascadedWorkloadDeployment records a workload added by output-change
// propagation, with the image it is currently running.
func (r *DeploymentRepository) InsertCascadedWorkloadDeployment(ctx context.Context, deploymentID string, wd *domain.WorkloadDeployment) error {
	wd.ID = uuid.NewString()
	wd.InclusionReason = domain.Cascaded
	return insertWorkloadDeployment(ctx, r.DB.Pool, deploymentID, wd)
}

// UpdateWaveNumbers stores re-computed waves after the scope grew.
func (r *DeploymentRepository) UpdateWaveNumbers(ctx context.Context, deploymentID string, waves map[string]int) error {
	for workloadID, wave := range waves {
		if _, err := r.DB.Pool.Exec(ctx, `UPDATE workload_deployment SET wave_number = $3 WHERE deployment_id = $1 AND workload_id = $2`,
			deploymentID, workloadID, wave); err != nil {
			return err
		}
	}
	return nil
}

// StartStep records a RUNNING step and returns its ID.
func (r *DeploymentRepository) StartStep(ctx context.Context, deploymentID, recordID string, wave int, name domain.StepName, component string) (string, error) {
	id := uuid.NewString()
	_, err := r.DB.Pool.Exec(ctx, `INSERT INTO deployment_step (deployment_step_id, deployment_id, deployment_record_id, sequence_number,
		wave_number, step_name, status, related_component_reference, started_at)
		VALUES ($1, $2, $3, (SELECT coalesce(max(sequence_number), 0) + 1 FROM deployment_step WHERE deployment_record_id = $3),
		        $4, $5, 'RUNNING', nullif($6, ''), now())`, id, deploymentID, recordID, wave, string(name), component)
	return id, err
}

// FinishStep closes a step. errorSummary must never contain secret values.
func (r *DeploymentRepository) FinishStep(ctx context.Context, stepID string, status domain.StepStatus, errorSummary string, detail map[string]any) error {
	body, _ := json.Marshal(nonNilMap(detail))
	_, err := r.DB.Pool.Exec(ctx, `UPDATE deployment_step SET status = $2, error_summary = nullif($3, ''), detail = $4, completed_at = now()
		WHERE deployment_step_id = $1`, stepID, string(status), errorSummary, body)
	return err
}

type FinishInput struct {
	DeploymentID        string
	JobID               string
	RecordID            string
	Succeeded           bool
	DeliveryReference   string
	RemovedComponents   []string
	ErrorSummary        string
	ResourceInstanceIDs []string
}

// FinishDeployment writes the Deployment Record result, the final deployment
// status (CAS from DEPLOYING), the job status and the record's infrastructure
// references in one transaction.
func (r *DeploymentRepository) FinishDeployment(ctx context.Context, in FinishInput) error {
	status, jobStatus := domain.Succeeded, "COMPLETED"
	if !in.Succeeded {
		status, jobStatus = domain.Failed, "FAILED"
	}
	return r.DB.InTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE deployment SET status = $2, updated_at = now() WHERE deployment_id = $1 AND status = 'DEPLOYING'`,
			in.DeploymentID, string(status))
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("finish deployment: deployment is not DEPLOYING")
		}
		removed, _ := json.Marshal(nonNil(in.RemovedComponents))
		if _, err := tx.Exec(ctx, `UPDATE deployment_record SET delivery_reference = nullif($2, ''), removed_components = $3,
			status = $4, error_summary = nullif($5, ''), updated_at = now() WHERE deployment_record_id = $1`,
			in.RecordID, in.DeliveryReference, removed, string(status), in.ErrorSummary); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE deployment_step SET status = 'SKIPPED', completed_at = now()
			WHERE deployment_record_id = $1 AND status IN ('PENDING', 'RUNNING')`, in.RecordID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM deployment_record_resource_instance WHERE deployment_record_id = $1`, in.RecordID); err != nil {
			return err
		}
		for _, riID := range in.ResourceInstanceIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO deployment_record_resource_instance (deployment_record_resource_instance_id, deployment_record_id, resource_instance_id)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, uuid.NewString(), in.RecordID, riID); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `UPDATE deployment_execution_job SET status = $2, updated_at = now() WHERE job_id = $1`, in.JobID, jobStatus)
		return err
	})
}

// FailOrphanedJob moves a job stuck in RUNNING (its worker died) to FAILED.
// Worker recovery is deferred (D6); this is the manual escape hatch.
func (r *DeploymentRepository) FailOrphanedJob(ctx context.Context, deploymentID, reason string) error {
	var jobID, recordID string
	err := r.DB.Pool.QueryRow(ctx, `SELECT j.job_id, rec.deployment_record_id FROM deployment_execution_job j
		JOIN deployment_record rec USING (deployment_id) WHERE j.deployment_id = $1 AND j.status = 'RUNNING'`, deploymentID).Scan(&jobID, &recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reject(domain.CodeNotFound, "no RUNNING job for deployment %s", deploymentID)
	}
	if err != nil {
		return err
	}
	return r.FinishDeployment(ctx, FinishInput{DeploymentID: deploymentID, JobID: jobID, RecordID: recordID, ErrorSummary: reason})
}

// ---------------------------------------------------------------------------
// Queries for result tracking (UC-04 subset)

type DeploymentSummary struct {
	ID             string
	Kind           string
	Environment    string
	Target         string
	VersionNumber  int
	CatalogVersion int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r *DeploymentRepository) FindByApplication(ctx context.Context, applicationID string) ([]DeploymentSummary, error) {
	rows, err := r.DB.Pool.Query(ctx, `SELECT d.deployment_id, d.kind::text, d.environment::text, d.deployment_target, v.version_number,
		cv.version_number, d.status::text, d.created_at, d.updated_at
		FROM deployment d JOIN application_definition_version v USING (application_definition_version_id)
		JOIN catalog_version cv USING (catalog_version_id)
		WHERE d.application_id = $1 ORDER BY d.created_at DESC`, applicationID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (DeploymentSummary, error) {
		var s DeploymentSummary
		err := row.Scan(&s.ID, &s.Kind, &s.Environment, &s.Target, &s.VersionNumber, &s.CatalogVersion, &s.Status, &s.CreatedAt, &s.UpdatedAt)
		return s, err
	})
}

type StepView struct {
	Sequence     int
	Wave         int
	Name         string
	Status       string
	Component    string
	ErrorSummary string
	Detail       map[string]any
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

type RecordView struct {
	DeliveryReference   string
	RemovedComponents   []string
	Status              string
	ErrorSummary        string
	ResourceInstanceIDs []string
	JobStatus           string
}

func (r *DeploymentRepository) GetDeploymentProgress(ctx context.Context, deploymentID string) ([]StepView, error) {
	rows, err := r.DB.Pool.Query(ctx, `SELECT sequence_number, wave_number, step_name::text, status::text,
		coalesce(related_component_reference, ''), coalesce(error_summary, ''), detail, started_at, completed_at
		FROM deployment_step WHERE deployment_id = $1 ORDER BY sequence_number`, deploymentID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StepView, error) {
		var s StepView
		var detail []byte
		err := row.Scan(&s.Sequence, &s.Wave, &s.Name, &s.Status, &s.Component, &s.ErrorSummary, &detail, &s.StartedAt, &s.CompletedAt)
		if err == nil {
			err = json.Unmarshal(detail, &s.Detail)
		}
		return s, err
	})
}

func (r *DeploymentRepository) GetDeploymentRecord(ctx context.Context, deploymentID string) (*RecordView, error) {
	v := &RecordView{}
	var removed []byte
	var recordID string
	err := r.DB.Pool.QueryRow(ctx, `SELECT rec.deployment_record_id, coalesce(rec.delivery_reference, ''), rec.removed_components,
		rec.status::text, coalesce(rec.error_summary, ''), coalesce(j.status::text, '')
		FROM deployment_record rec LEFT JOIN deployment_execution_job j USING (deployment_id)
		WHERE rec.deployment_id = $1`, deploymentID).Scan(&recordID, &v.DeliveryReference, &removed, &v.Status, &v.ErrorSummary, &v.JobStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(removed, &v.RemovedComponents); err != nil {
		return nil, err
	}
	rows, err := r.DB.Pool.Query(ctx, `SELECT resource_instance_id FROM deployment_record_resource_instance WHERE deployment_record_id = $1`, recordID)
	if err != nil {
		return nil, err
	}
	v.ResourceInstanceIDs, err = pgx.CollectRows(rows, pgx.RowTo[string])
	return v, err
}

// JobStatus returns the execution job status of a deployment ("" when none).
func (r *DeploymentRepository) JobStatus(ctx context.Context, deploymentID string) (string, int, error) {
	var status string
	var count int
	err := r.DB.Pool.QueryRow(ctx, `SELECT coalesce(max(status::text), ''), count(*) FROM deployment_execution_job WHERE deployment_id = $1`, deploymentID).Scan(&status, &count)
	return status, count, err
}

func nonNilStringMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
