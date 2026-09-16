package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
)

// DeliveryRepositoryRegistry records where the desired state of each
// application lives. It holds the repository URL, the branch and secret
// references to the key pair of that application; keys and Git hosting
// credentials are never stored here.
type DeliveryRepositoryRegistry struct{ DB *DB }

const deliveryRepositoryColumns = `delivery_repository_id, application_id, repository_url, branch,
	write_key_reference, read_key_reference, created_at, updated_at`

func scanDeliveryRepository(row pgx.Row) (*domain.DeliveryRepository, error) {
	var r domain.DeliveryRepository
	err := row.Scan(&r.ID, &r.ApplicationID, &r.RepositoryURL, &r.Branch,
		&r.WriteKeyReference, &r.ReadKeyReference, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Find returns the delivery repository of an application, or nil when the
// application has none yet.
func (reg *DeliveryRepositoryRegistry) Find(ctx context.Context, applicationID string) (*domain.DeliveryRepository, error) {
	r, err := scanDeliveryRepository(reg.DB.Pool.QueryRow(ctx,
		`SELECT `+deliveryRepositoryColumns+` FROM delivery_repository WHERE application_id = $1`, applicationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// Save stores the delivery repository of an application. An application keeps
// exactly one row: a second worker that prepared the same repository updates
// the row instead of creating another one.
func (reg *DeliveryRepositoryRegistry) Save(ctx context.Context, r domain.DeliveryRepository) (*domain.DeliveryRepository, error) {
	now := time.Now().UTC()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return scanDeliveryRepository(reg.DB.Pool.QueryRow(ctx, `
		INSERT INTO delivery_repository (delivery_repository_id, application_id, repository_url, branch,
			write_key_reference, read_key_reference, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (application_id) DO UPDATE SET repository_url = EXCLUDED.repository_url, branch = EXCLUDED.branch,
			write_key_reference = EXCLUDED.write_key_reference, read_key_reference = EXCLUDED.read_key_reference,
			updated_at = EXCLUDED.updated_at
		RETURNING `+deliveryRepositoryColumns,
		r.ID, r.ApplicationID, r.RepositoryURL, r.Branch, r.WriteKeyReference, r.ReadKeyReference, now))
}
