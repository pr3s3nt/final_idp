package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"idp/internal/domain"
)

// EnvironmentConfigurationRepository stores value sources and references per
// application + environment; it never stores resolved output values.
type EnvironmentConfigurationRepository struct{ DB *DB }

func (r *EnvironmentConfigurationRepository) FindByApplicationAndEnvironment(ctx context.Context, applicationID string, env domain.Environment) (*domain.EnvironmentConfiguration, error) {
	var id string
	err := r.DB.Pool.QueryRow(ctx, `SELECT environment_configuration_id FROM environment_configuration
		WHERE application_id = $1 AND environment = $2`, applicationID, string(env)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *EnvironmentConfigurationRepository) FindByID(ctx context.Context, id string) (*domain.EnvironmentConfiguration, error) {
	c := &domain.EnvironmentConfiguration{}
	var env string
	err := r.DB.Pool.QueryRow(ctx, `SELECT environment_configuration_id, application_id, environment
		FROM environment_configuration WHERE environment_configuration_id = $1`, id).Scan(&c.ID, &c.ApplicationID, &env)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "environment configuration %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	c.Environment = domain.Environment(env)

	rows, err := r.DB.Pool.Query(ctx, `
		SELECT ev.environment_variable_id, ev.workload_id, ev.variable_definition_id, ev.variable_name,
		       cv.value_source::text, coalesce(cv.direct_value, ''),
		       coalesce(cv.resource_requirement_id, cv.workload_id)::text,
		       coalesce(cv.resource_output_name, cv.workload_output_name, '')
		FROM environment_variable ev JOIN configuration_value cv USING (environment_variable_id)
		WHERE ev.environment_configuration_id = $1 ORDER BY ev.variable_name`, id)
	if err != nil {
		return nil, err
	}
	c.Variables, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.ConfiguredValue, error) {
		var v domain.ConfiguredValue
		var ref *string
		err := row.Scan(&v.ID, &v.WorkloadID, &v.DefinitionID, &v.Name, &v.Source, &v.DirectValue, &ref, &v.OutputName)
		if ref != nil {
			v.RefID = *ref
		}
		return v, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = r.DB.Pool.Query(ctx, `
		SELECT secret_id, workload_id, secret_definition_id, secret_name, value_source::text,
		       coalesce(secret_ref, ''), coalesce(resource_requirement_id::text, ''), coalesce(resource_output_name, '')
		FROM secret WHERE environment_configuration_id = $1 ORDER BY secret_name`, id)
	if err != nil {
		return nil, err
	}
	c.Secrets, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.ConfiguredValue, error) {
		var v domain.ConfiguredValue
		err := row.Scan(&v.ID, &v.WorkloadID, &v.DefinitionID, &v.Name, &v.Source, &v.SecretRef, &v.RefID, &v.OutputName)
		return v, err
	})
	return c, err
}

// Revision is the opaque baseConfigurationRevision of UC-02. It is derived
// from updated_at, so no column is added for it; an empty revision means the
// environment has no configuration yet.
func revisionOf(updatedAt time.Time) string { return updatedAt.UTC().Format(time.RFC3339Nano) }

// FindRevision returns the current opaque revision of one application +
// environment, or an empty string when no configuration exists.
func (r *EnvironmentConfigurationRepository) FindRevision(ctx context.Context, applicationID string, env domain.Environment) (string, error) {
	var updatedAt time.Time
	err := r.DB.Pool.QueryRow(ctx, `SELECT updated_at FROM environment_configuration
		WHERE application_id = $1 AND environment = $2`, applicationID, string(env)).Scan(&updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return revisionOf(updatedAt), nil
}

// SaveConfigurationInput is one complete client-owned draft plus the two
// concurrency bases it was started from (UC-02 operation contract 3).
type SaveConfigurationInput struct {
	Configuration *domain.EnvironmentConfiguration
	// BaseApplicationDefinitionVersion is the latest version the requirements
	// were loaded from.
	BaseApplicationDefinitionVersion int
	// BaseConfigurationRevision is empty when the draft creates the
	// configuration of this environment.
	BaseConfigurationRevision string
}

// SaveIfBasesMatch implements saveIfBasesMatch() of UC-02. Both base
// comparisons and the durable write happen in one transaction, so two Saves
// that start from the same bases cannot both succeed. A stale base returns
// DRAFT_CONFLICT and writes nothing.
func (r *EnvironmentConfigurationRepository) SaveIfBasesMatch(ctx context.Context, in SaveConfigurationInput) (id string, revision string, err error) {
	c := in.Configuration
	err = r.DB.InTx(ctx, func(tx pgx.Tx) error {
		var latest int
		if err := tx.QueryRow(ctx, `SELECT coalesce(max(version_number), 0) FROM application_definition_version
			WHERE application_id = $1`, c.ApplicationID).Scan(&latest); err != nil {
			return err
		}
		if latest != in.BaseApplicationDefinitionVersion {
			return draftConflict(in.BaseApplicationDefinitionVersion, latest)
		}

		var current string
		var updatedAt time.Time
		err := tx.QueryRow(ctx, `SELECT environment_configuration_id, updated_at FROM environment_configuration
			WHERE application_id = $1 AND environment = $2 FOR UPDATE`, c.ApplicationID, string(c.Environment)).Scan(&c.ID, &updatedAt)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c.ID = ""
		case err != nil:
			return err
		default:
			current = revisionOf(updatedAt)
		}
		if current != in.BaseConfigurationRevision {
			return configurationConflict(in.BaseConfigurationRevision, current)
		}

		now := time.Now().UTC()
		revision = revisionOf(now)
		return writeConfiguration(ctx, tx, c, now)
	})
	if err != nil {
		return "", "", err
	}
	return c.ID, revision, nil
}

func configurationConflict(base, current string) error {
	v := &domain.ValidationError{}
	switch {
	case base == "":
		v.Add(domain.CodeDraftConflict, "this draft creates the configuration of the environment, but a configuration already exists; reload it and reapply your changes")
	case current == "":
		v.Add(domain.CodeDraftConflict, "the configuration this draft is based on no longer exists; reload the environment and reapply your changes")
	default:
		v.Add(domain.CodeDraftConflict, "the configuration changed while this draft was saved; reload it and reapply your changes")
	}
	return v
}

// Save replaces the binding set of one application + environment without a
// base check. It exists for fixture/import loading; the UC-02 Save path is
// SaveIfBasesMatch.
func (r *EnvironmentConfigurationRepository) Save(ctx context.Context, c *domain.EnvironmentConfiguration) (string, error) {
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		err := tx.QueryRow(ctx, `SELECT environment_configuration_id FROM environment_configuration
			WHERE application_id = $1 AND environment = $2 FOR UPDATE`, c.ApplicationID, string(c.Environment)).Scan(&c.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			c.ID = ""
		} else if err != nil {
			return err
		}
		return writeConfiguration(ctx, tx, c, now)
	})
	return c.ID, err
}

// writeConfiguration inserts or updates the configuration row and replaces its
// bindings. An empty c.ID means the configuration does not exist yet. Only
// references and opaque secret references are written.
func writeConfiguration(ctx context.Context, tx pgx.Tx, c *domain.EnvironmentConfiguration, now time.Time) error {
	var err error
	if c.ID == "" {
		c.ID = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO environment_configuration (environment_configuration_id, application_id, environment, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $4)`, c.ID, c.ApplicationID, string(c.Environment), now)
	} else {
		_, err = tx.Exec(ctx, `UPDATE environment_configuration SET updated_at = $2 WHERE environment_configuration_id = $1`, c.ID, now)
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM environment_variable WHERE environment_configuration_id = $1`, c.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM secret WHERE environment_configuration_id = $1`, c.ID); err != nil {
		return err
	}
	for _, v := range c.Variables {
		evID := uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO environment_variable (environment_variable_id, environment_configuration_id, workload_id, variable_definition_id, variable_name)
			VALUES ($1, $2, $3, $4, $5)`, evID, c.ID, v.WorkloadID, v.DefinitionID, v.Name); err != nil {
			return err
		}
		var direct, resID, resOut, wlID, wlOut *string
		switch v.Source {
		case domain.SourceDirect:
			direct = &v.DirectValue
		case domain.SourceResourceOutput:
			resID, resOut = &v.RefID, &v.OutputName
		case domain.SourceWorkloadOutput:
			wlID, wlOut = &v.RefID, &v.OutputName
		}
		if _, err := tx.Exec(ctx, `INSERT INTO configuration_value (configuration_value_id, environment_configuration_id, environment_variable_id,
			value_source, direct_value, resource_requirement_id, resource_output_name, workload_id, workload_output_name)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, uuid.NewString(), c.ID, evID, v.Source, direct, resID, resOut, wlID, wlOut); err != nil {
			return err
		}
	}
	for _, s := range c.Secrets {
		var ref, resID, resOut *string
		switch s.Source {
		case domain.SourceSecretRef:
			ref = &s.SecretRef
		case domain.SourceResourceOutput:
			resID, resOut = &s.RefID, &s.OutputName
		}
		if _, err := tx.Exec(ctx, `INSERT INTO secret (secret_id, environment_configuration_id, workload_id, secret_definition_id, secret_name,
			value_source, secret_ref, resource_requirement_id, resource_output_name)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, uuid.NewString(), c.ID, s.WorkloadID, s.DefinitionID, s.Name, s.Source, ref, resID, resOut); err != nil {
			return err
		}
	}
	return nil
}
