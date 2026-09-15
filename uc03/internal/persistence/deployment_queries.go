package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
)

// LastDeploymentOfOwner returns the most recent deployment that reached
// execution for application + environment + target, used to plan a teardown
// against the version and context that built the current infrastructure.
func (r *DeploymentRepository) LastDeploymentOfOwner(ctx context.Context, applicationID string, env domain.Environment, target string) (*domain.Deployment, error) {
	var id string
	err := r.DB.Pool.QueryRow(ctx, `SELECT deployment_id FROM deployment
		WHERE application_id = $1 AND environment = $2 AND deployment_target = $3 AND kind = 'DEPLOY'
		  AND status IN ('DEPLOYING', 'SUCCEEDED', 'FAILED')
		ORDER BY created_at DESC LIMIT 1`, applicationID, string(env), target).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindByIDWithPersistedInputs(ctx, id)
}
