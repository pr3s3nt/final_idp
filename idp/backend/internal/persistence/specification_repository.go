package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SpecificationRepository is the Specification Repository of UC-01. Each
// Application Definition Version has at most one generated specification.
type SpecificationRepository struct{ DB *DB }

// Save stores the specification generated for one version. A version that
// already has a specification keeps it (UNIQUE application_definition_version_id).
func (r *SpecificationRepository) Save(ctx context.Context, applicationID, versionID, format, content, version string) error {
	_, err := r.DB.Pool.Exec(ctx, `INSERT INTO application_specification
		(specification_id, application_id, application_definition_version_id, format, content, version, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (application_definition_version_id) DO NOTHING`,
		uuid.NewString(), applicationID, versionID, format, content, version, time.Now().UTC())
	return err
}

// Find returns the format and content of the specification of one version.
func (r *SpecificationRepository) Find(ctx context.Context, versionID string) (format, content string, err error) {
	err = r.DB.Pool.QueryRow(ctx, `SELECT format, content FROM application_specification
		WHERE application_definition_version_id = $1`, versionID).Scan(&format, &content)
	return format, content, err
}
