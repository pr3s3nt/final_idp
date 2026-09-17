package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"deploy/internal/domain"
)

// ResourceDefinitionCatalog is the platform-managed, versioned catalog. It is
// the only source that decides which infrastructure a deployment creates. The
// IDP only reads it; CreateVersion is platform administration (fixtures).
type ResourceDefinitionCatalog struct{ DB *DB }

const definitionColumns = `resource_definition_id, catalog_version_id, name, resource_type, provisioner_reference, supported_contexts,
	default_parameters, allowed_overrides, exposed_outputs, sensitive_outputs, management_mode::text,
	applicability_conditions, coalesce(existing_resource_reference, ''), requires`

func scanDefinition(row pgx.Row) (domain.ResourceDefinition, error) {
	var d domain.ResourceDefinition
	var contexts, defaults, overrides, exposed, sensitive, applicability, requires []byte
	var mode string
	err := row.Scan(&d.ID, &d.CatalogVersionID, &d.Name, &d.ResourceType, &d.ProvisionerReference, &contexts, &defaults, &overrides,
		&exposed, &sensitive, &mode, &applicability, &d.ExistingResourceReference, &requires)
	if err != nil {
		return d, err
	}
	d.ManagementMode = domain.ManagementMode(mode)
	for _, p := range []struct {
		raw []byte
		dst any
	}{{contexts, &d.SupportedContexts}, {defaults, &d.DefaultParameters}, {overrides, &d.AllowedOverrides},
		{exposed, &d.ExposedOutputs}, {sensitive, &d.SensitiveOutputs}, {applicability, &d.ApplicabilityConditions}, {requires, &d.Requires}} {
		if len(p.raw) == 0 {
			continue
		}
		if err := json.Unmarshal(p.raw, p.dst); err != nil {
			return d, err
		}
	}
	return d, nil
}

func (c *ResourceDefinitionCatalog) definitions(ctx context.Context, where string, args ...any) ([]domain.ResourceDefinition, error) {
	rows, err := c.DB.Pool.Query(ctx, `SELECT `+definitionColumns+` FROM resource_definition `+where+` ORDER BY name, resource_definition_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ResourceDefinition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDefinitions returns the definitions of one catalog version.
func (c *ResourceDefinitionCatalog) ListDefinitions(ctx context.Context, catalogVersionID string) ([]domain.ResourceDefinition, error) {
	return c.definitions(ctx, `WHERE catalog_version_id = $1`, catalogVersionID)
}

// ListAll returns the definitions of every catalog version.
func (c *ResourceDefinitionCatalog) ListAll(ctx context.Context) ([]domain.ResourceDefinition, error) {
	return c.definitions(ctx, ``)
}

func (c *ResourceDefinitionCatalog) FindByID(ctx context.Context, id string) (*domain.ResourceDefinition, error) {
	d, err := scanDefinition(c.DB.Pool.QueryRow(ctx, `SELECT `+definitionColumns+` FROM resource_definition WHERE resource_definition_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "resource definition %s not found", id)
	}
	return &d, err
}

// ListVersions returns catalog versions, newest first.
func (c *ResourceDefinitionCatalog) ListVersions(ctx context.Context) ([]domain.CatalogVersion, error) {
	rows, err := c.DB.Pool.Query(ctx, `SELECT catalog_version_id, version_number, created_at FROM catalog_version ORDER BY version_number DESC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.CatalogVersion, error) {
		var v domain.CatalogVersion
		err := row.Scan(&v.ID, &v.Number, &v.CreatedAt)
		return v, err
	})
}

// FindVersion finds a catalog version by number ("2") or ID.
func (c *ResourceDefinitionCatalog) FindVersion(ctx context.Context, key string) (*domain.CatalogVersion, error) {
	v := &domain.CatalogVersion{}
	var err error
	if n, convErr := strconv.Atoi(key); convErr == nil {
		err = c.DB.Pool.QueryRow(ctx, `SELECT catalog_version_id, version_number, created_at FROM catalog_version WHERE version_number = $1`, n).
			Scan(&v.ID, &v.Number, &v.CreatedAt)
	} else if _, parseErr := uuid.Parse(key); parseErr == nil {
		err = c.DB.Pool.QueryRow(ctx, `SELECT catalog_version_id, version_number, created_at FROM catalog_version WHERE catalog_version_id = $1`, key).
			Scan(&v.ID, &v.Number, &v.CreatedAt)
	} else {
		err = pgx.ErrNoRows
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "catalog version %s does not exist", key)
	}
	return v, err
}

// CreateVersion stores a new immutable catalog version with its definitions.
// An existing version number is never changed; created is false then.
func (c *ResourceDefinitionCatalog) CreateVersion(ctx context.Context, number int, defs []domain.ResourceDefinition) (created bool, err error) {
	err = c.DB.InTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM catalog_version WHERE version_number = $1)`, number).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return nil
		}
		versionID := uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO catalog_version (catalog_version_id, version_number, created_at) VALUES ($1, $2, $3)`,
			versionID, number, time.Now().UTC()); err != nil {
			return err
		}
		js := func(v any) []byte { b, _ := json.Marshal(v); return b }
		for i := range defs {
			d := &defs[i]
			d.ID, d.CatalogVersionID = uuid.NewString(), versionID
			var applicability, existing any
			if len(d.ApplicabilityConditions) > 0 {
				applicability = js(d.ApplicabilityConditions)
			}
			if d.ExistingResourceReference != "" {
				existing = d.ExistingResourceReference
			}
			if d.DefaultParameters == nil {
				d.DefaultParameters = map[string]any{}
			}
			if d.AllowedOverrides == nil {
				d.AllowedOverrides = map[string]domain.OverrideRule{}
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO resource_definition (resource_definition_id, catalog_version_id, name, resource_type, provisioner_reference,
					supported_contexts, default_parameters, allowed_overrides, exposed_outputs, sensitive_outputs, management_mode,
					applicability_conditions, existing_resource_reference, requires)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
				d.ID, versionID, d.Name, d.ResourceType, d.ProvisionerReference, js(nonNil(d.SupportedContexts)), js(d.DefaultParameters),
				js(d.AllowedOverrides), js(nonNil(d.ExposedOutputs)), js(nonNil(d.SensitiveOutputs)), string(d.ManagementMode),
				applicability, existing, js(nonNil(d.Requires))); err != nil {
				return err
			}
		}
		created = true
		return nil
	})
	return created, err
}
