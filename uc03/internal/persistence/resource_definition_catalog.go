package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
)

// ResourceDefinitionCatalog is the platform-managed catalog. It is the only
// source that decides which infrastructure a deployment creates.
type ResourceDefinitionCatalog struct{ DB *DB }

const definitionColumns = `resource_definition_id, name, resource_type, provisioner_reference, supported_contexts,
	default_parameters, allowed_overrides, exposed_outputs, sensitive_outputs, management_mode::text,
	applicability_conditions, coalesce(existing_resource_reference, ''), requires`

func scanDefinition(row pgx.Row) (domain.ResourceDefinition, error) {
	var d domain.ResourceDefinition
	var contexts, defaults, overrides, exposed, sensitive, applicability, requires []byte
	var mode string
	err := row.Scan(&d.ID, &d.Name, &d.ResourceType, &d.ProvisionerReference, &contexts, &defaults, &overrides,
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

func (c *ResourceDefinitionCatalog) List(ctx context.Context) ([]domain.ResourceDefinition, error) {
	rows, err := c.DB.Pool.Query(ctx, `SELECT `+definitionColumns+` FROM resource_definition ORDER BY name`)
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

func (c *ResourceDefinitionCatalog) FindByID(ctx context.Context, id string) (*domain.ResourceDefinition, error) {
	d, err := scanDefinition(c.DB.Pool.QueryRow(ctx, `SELECT `+definitionColumns+` FROM resource_definition WHERE resource_definition_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "resource definition %s not found", id)
	}
	return &d, err
}

// Upsert inserts or replaces a definition by name (platform administration;
// used by the fixture importer).
func (c *ResourceDefinitionCatalog) Upsert(ctx context.Context, d *domain.ResourceDefinition) error {
	if d.ID == "" {
		_ = c.DB.Pool.QueryRow(ctx, `SELECT resource_definition_id FROM resource_definition WHERE name = $1`, d.Name).Scan(&d.ID)
		if d.ID == "" {
			d.ID = uuid.NewString()
		}
	}
	js := func(v any) []byte { b, _ := json.Marshal(v); return b }
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
	_, err := c.DB.Pool.Exec(ctx, `
		INSERT INTO resource_definition (resource_definition_id, name, resource_type, provisioner_reference, supported_contexts,
			default_parameters, allowed_overrides, exposed_outputs, sensitive_outputs, management_mode,
			applicability_conditions, existing_resource_reference, requires)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (resource_definition_id) DO UPDATE SET name = EXCLUDED.name, resource_type = EXCLUDED.resource_type,
			provisioner_reference = EXCLUDED.provisioner_reference, supported_contexts = EXCLUDED.supported_contexts,
			default_parameters = EXCLUDED.default_parameters, allowed_overrides = EXCLUDED.allowed_overrides,
			exposed_outputs = EXCLUDED.exposed_outputs, sensitive_outputs = EXCLUDED.sensitive_outputs,
			management_mode = EXCLUDED.management_mode, applicability_conditions = EXCLUDED.applicability_conditions,
			existing_resource_reference = EXCLUDED.existing_resource_reference, requires = EXCLUDED.requires`,
		d.ID, d.Name, d.ResourceType, d.ProvisionerReference, js(nonNil(d.SupportedContexts)), js(d.DefaultParameters),
		js(d.AllowedOverrides), js(nonNil(d.ExposedOutputs)), js(nonNil(d.SensitiveOutputs)), string(d.ManagementMode),
		applicability, existing, js(nonNil(d.Requires)))
	return err
}
