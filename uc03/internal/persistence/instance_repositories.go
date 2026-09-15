package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
)

// ResourceInstanceRepository stores the durable state of provisioned or linked
// infrastructure per owner: application + environment + requirement + target.
type ResourceInstanceRepository struct{ DB *DB }

const resourceInstanceColumns = `resource_instance_id, application_id, environment::text, resource_requirement_id,
	resource_definition_id, deployment_target, infrastructure_reference, coalesce(provider_state_reference, ''),
	status::text, coalesce(output_fingerprint, ''), applied_overrides, coalesce(applied_input_fingerprint, ''),
	created_at, updated_at`

func scanResourceInstance(row pgx.Row) (domain.ResourceInstance, error) {
	var ri domain.ResourceInstance
	var env, status string
	var overrides []byte
	err := row.Scan(&ri.ID, &ri.ApplicationID, &env, &ri.RequirementID, &ri.DefinitionID, &ri.Target,
		&ri.InfrastructureReference, &ri.ProviderStateReference, &status, &ri.OutputFingerprint, &overrides,
		&ri.AppliedInputFingerprint, &ri.CreatedAt, &ri.UpdatedAt)
	if err != nil {
		return ri, err
	}
	ri.Environment, ri.Status = domain.Environment(env), domain.ResourceInstanceStatus(status)
	err = json.Unmarshal(overrides, &ri.AppliedOverrides)
	return ri, err
}

func (r *ResourceInstanceRepository) collect(rows pgx.Rows, err error) ([]domain.ResourceInstance, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ResourceInstance
	for rows.Next() {
		ri, err := scanResourceInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ri)
	}
	return out, rows.Err()
}

// FindResourceInstances returns every non-terminal instance of the owner
// application + environment + target, including requirements no longer in the
// deployed version, so removals can be planned.
func (r *ResourceInstanceRepository) FindResourceInstances(ctx context.Context, applicationID string, env domain.Environment, target string) ([]domain.ResourceInstance, error) {
	return r.collect(r.DB.Pool.Query(ctx, `SELECT `+resourceInstanceColumns+` FROM resource_instance
		WHERE application_id = $1 AND environment = $2 AND deployment_target = $3
		  AND status NOT IN ('DESTROYED', 'UNLINKED') ORDER BY created_at`, applicationID, string(env), target))
}

func (r *ResourceInstanceRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.ResourceInstance, error) {
	return r.collect(r.DB.Pool.Query(ctx, `SELECT `+resourceInstanceColumns+` FROM resource_instance
		WHERE resource_instance_id = ANY($1::uuid[]) ORDER BY created_at`, ids))
}

func (r *ResourceInstanceRepository) FindByID(ctx context.Context, id string) (*domain.ResourceInstance, error) {
	ri, err := scanResourceInstance(r.DB.Pool.QueryRow(ctx, `SELECT `+resourceInstanceColumns+` FROM resource_instance WHERE resource_instance_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "resource instance %s not found", id)
	}
	return &ri, err
}

// Create inserts a new instance for an owner. The partial unique index rejects
// a second active instance for the same owner.
func (r *ResourceInstanceRepository) Create(ctx context.Context, ri *domain.ResourceInstance, platformType string) error {
	return r.DB.InTx(ctx, func(tx pgx.Tx) error {
		if platformType != "" {
			if err := EnsurePlatformComponent(ctx, tx, ri.ApplicationID, platformType); err != nil {
				return err
			}
		}
		if ri.ID == "" {
			ri.ID = uuid.NewString()
		}
		overrides, _ := json.Marshal(nonNilMap(ri.AppliedOverrides))
		now := time.Now().UTC()
		ri.CreatedAt, ri.UpdatedAt = now, now
		_, err := tx.Exec(ctx, `INSERT INTO resource_instance (resource_instance_id, application_id, environment, resource_requirement_id,
			resource_definition_id, deployment_target, infrastructure_reference, provider_state_reference, status,
			output_fingerprint, applied_overrides, applied_input_fingerprint, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, nullif($8, ''), $9, nullif($10, ''), $11, nullif($12, ''), $13, $13)`,
			ri.ID, ri.ApplicationID, string(ri.Environment), ri.RequirementID, ri.DefinitionID, ri.Target,
			ri.InfrastructureReference, ri.ProviderStateReference, string(ri.Status), ri.OutputFingerprint, overrides,
			ri.AppliedInputFingerprint, now)
		return err
	})
}

// SaveState updates status, references and the applied override baseline after
// a provisioner call.
func (r *ResourceInstanceRepository) SaveState(ctx context.Context, ri *domain.ResourceInstance) error {
	overrides, _ := json.Marshal(nonNilMap(ri.AppliedOverrides))
	_, err := r.DB.Pool.Exec(ctx, `UPDATE resource_instance SET infrastructure_reference = $2,
		provider_state_reference = nullif($3, ''), status = $4, applied_overrides = $5,
		applied_input_fingerprint = nullif($6, ''), updated_at = now() WHERE resource_instance_id = $1`,
		ri.ID, ri.InfrastructureReference, ri.ProviderStateReference, string(ri.Status), overrides, ri.AppliedInputFingerprint)
	return err
}

func (r *ResourceInstanceRepository) UpdateStatus(ctx context.Context, id string, status domain.ResourceInstanceStatus) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE resource_instance SET status = $2, updated_at = now() WHERE resource_instance_id = $1`, id, string(status))
	return err
}

func (r *ResourceInstanceRepository) UpdateOutputFingerprint(ctx context.Context, id, fingerprint string) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE resource_instance SET output_fingerprint = $2, updated_at = now() WHERE resource_instance_id = $1`, id, fingerprint)
	return err
}

// WorkloadInstanceRepository stores the current state of each workload on an
// environment + target.
type WorkloadInstanceRepository struct{ DB *DB }

const workloadInstanceSelect = `
	SELECT wi.workload_instance_id, wi.application_id, wi.workload_id, wi.environment::text, wi.deployment_target,
	       wi.current_workload_deployment_id, wi.status::text, coalesce(wi.output_fingerprint, ''),
	       d.application_definition_version_id, v.version_number, wd.image_repository, wd.image_version, d.deployment_id
	FROM workload_instance wi
	JOIN workload_deployment wd ON wd.workload_deployment_id = wi.current_workload_deployment_id
	JOIN deployment d ON d.deployment_id = wd.deployment_id
	JOIN application_definition_version v ON v.application_definition_version_id = d.application_definition_version_id`

func scanWorkloadInstance(row pgx.Row) (domain.WorkloadInstance, error) {
	var wi domain.WorkloadInstance
	var env, status string
	err := row.Scan(&wi.ID, &wi.ApplicationID, &wi.WorkloadID, &env, &wi.Target, &wi.CurrentWorkloadDeploymentID,
		&status, &wi.OutputFingerprint, &wi.RunningVersionID, &wi.RunningVersionNumber, &wi.ImageRepository,
		&wi.ImageVersion, &wi.DeploymentID)
	wi.Environment, wi.Status = domain.Environment(env), domain.WorkloadInstanceStatus(status)
	return wi, err
}

// FindWorkloadInstances returns the non-removed workload instances of an
// application on an environment + target.
func (r *WorkloadInstanceRepository) FindWorkloadInstances(ctx context.Context, applicationID string, env domain.Environment, target string) ([]domain.WorkloadInstance, error) {
	rows, err := r.DB.Pool.Query(ctx, workloadInstanceSelect+`
		WHERE wi.application_id = $1 AND wi.environment = $2 AND wi.deployment_target = $3 AND wi.status <> 'REMOVED'
		ORDER BY wi.created_at`, applicationID, string(env), target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.WorkloadInstance
	for rows.Next() {
		wi, err := scanWorkloadInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wi)
	}
	return out, rows.Err()
}

// SaveDeploying points the workload's active instance at a new workload
// deployment and marks it DEPLOYING, creating the instance on first deploy.
func (r *WorkloadInstanceRepository) SaveDeploying(ctx context.Context, applicationID, workloadID string, env domain.Environment, target, workloadDeploymentID string) (string, error) {
	var id string
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT workload_instance_id FROM workload_instance
			WHERE workload_id = $1 AND environment = $2 AND deployment_target = $3 AND status <> 'REMOVED' FOR UPDATE`,
			workloadID, string(env), target).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			id = uuid.NewString()
			_, err = tx.Exec(ctx, `INSERT INTO workload_instance (workload_instance_id, application_id, workload_id, environment,
				deployment_target, current_workload_deployment_id, status, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'DEPLOYING', now(), now())`, id, applicationID, workloadID, string(env), target, workloadDeploymentID)
			return err
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE workload_instance SET current_workload_deployment_id = $2, status = 'DEPLOYING', updated_at = now()
			WHERE workload_instance_id = $1`, id, workloadDeploymentID)
		return err
	})
	return id, err
}

func (r *WorkloadInstanceRepository) UpdateStatus(ctx context.Context, id string, status domain.WorkloadInstanceStatus, fingerprint *string) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE workload_instance SET status = $2, output_fingerprint = coalesce($3, output_fingerprint),
		updated_at = now() WHERE workload_instance_id = $1`, id, string(status), fingerprint)
	return err
}

func nonNilMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
