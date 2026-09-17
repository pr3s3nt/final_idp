package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"idp/internal/domain"
)

// SaveVersionInput is one validated, complete Application Definition Draft.
// Every component carries its ID: an ID already owned by the application keeps
// its identity (also after a rename), any other ID creates a new component.
type SaveVersionInput struct {
	// ApplicationID is empty when the draft creates a new application.
	ApplicationID string
	// BaseVersion is the version the edit started from; 0 when creating.
	BaseVersion  int
	Name         string
	Description  string
	Workloads    []domain.Workload
	Resources    []domain.ResourceRequirement
	Dependencies []domain.Dependency
}

// SaveNewVersionIfBaseMatches implements saveNewVersionIfBaseMatches() of
// UC-01 (operation contract 1). In one transaction it locks the application,
// compares the latest version with BaseVersion and inserts the next immutable
// version. A stale base returns DRAFT_CONFLICT and writes nothing; rows of
// earlier versions are never updated or deleted.
func (r *ApplicationRepository) SaveNewVersionIfBaseMatches(ctx context.Context, in SaveVersionInput) (*domain.ApplicationVersion, error) {
	var appID, versionID string
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		number := 1
		if in.ApplicationID == "" {
			if err := rejectTakenName(ctx, tx, in.Name, ""); err != nil {
				return err
			}
			appID = uuid.NewString()
			if _, err := tx.Exec(ctx, `INSERT INTO application_definition (application_id, name, description, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $4)`, appID, in.Name, in.Description, now); err != nil {
				return err
			}
		} else {
			parsed, err := uuid.Parse(in.ApplicationID)
			if err != nil {
				return domain.Reject(domain.CodeNotFound, "application %q not found", in.ApplicationID)
			}
			appID = parsed.String()
			var exists bool
			err = tx.QueryRow(ctx, `SELECT true FROM application_definition WHERE application_id = $1 FOR UPDATE`, appID).Scan(&exists)
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.Reject(domain.CodeNotFound, "application %q not found", appID)
			}
			if err != nil {
				return err
			}
			var latest int
			if err := tx.QueryRow(ctx, `SELECT coalesce(max(version_number), 0) FROM application_definition_version WHERE application_id = $1`, appID).Scan(&latest); err != nil {
				return err
			}
			if latest != in.BaseVersion {
				return draftConflict(in.BaseVersion, latest)
			}
			if err := rejectTakenName(ctx, tx, in.Name, appID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE application_definition SET name = $2, description = $3, updated_at = $4 WHERE application_id = $1`,
				appID, in.Name, in.Description, now); err != nil {
				return err
			}
			number = latest + 1
		}

		if err := ensureComponents(ctx, tx, appID, in, now); err != nil {
			return err
		}

		versionID = uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO application_definition_version (application_definition_version_id, application_id, version_number, created_at)
			VALUES ($1, $2, $3, $4)`, versionID, appID, number, now); err != nil {
			return err
		}
		for _, rr := range in.Resources {
			if _, err := tx.Exec(ctx, `INSERT INTO resource_requirement (application_definition_version_id, resource_requirement_id, name, resource_type)
				VALUES ($1, $2, $3, $4)`, versionID, rr.ID, rr.Name, rr.ResourceType); err != nil {
				return err
			}
		}
		for _, w := range in.Workloads {
			outputs, _ := json.Marshal(nonNil(w.ExposedOutputs))
			if _, err := tx.Exec(ctx, `INSERT INTO workload (application_definition_version_id, workload_id, name, type, image_repository, port, exposed_outputs)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`, versionID, w.ID, w.Name, w.Type, w.ImageRepository, w.Port, outputs); err != nil {
				return err
			}
			for _, d := range w.Variables {
				if _, err := tx.Exec(ctx, `INSERT INTO environment_variable_definition (application_definition_version_id, variable_definition_id, workload_id, name, required)
					VALUES ($1, $2, $3, $4, $5)`, versionID, d.ID, w.ID, d.Name, d.Required); err != nil {
					return err
				}
			}
			for _, d := range w.Secrets {
				if _, err := tx.Exec(ctx, `INSERT INTO secret_definition (application_definition_version_id, secret_definition_id, workload_id, name, required)
					VALUES ($1, $2, $3, $4, $5)`, versionID, d.ID, w.ID, d.Name, d.Required); err != nil {
					return err
				}
			}
		}
		for _, d := range in.Dependencies {
			var tw, tr *string
			target := d.TargetID
			if d.TargetType == domain.TargetWorkload {
				tw = &target
			} else {
				tr = &target
			}
			if _, err := tx.Exec(ctx, `INSERT INTO dependency (dependency_id, application_definition_version_id, source_workload_id, target_type, target_workload_id, target_resource_requirement_id)
				VALUES ($1, $2, $3, $4, $5, $6)`, uuid.NewString(), versionID, d.SourceWorkloadID, d.TargetType, tw, tr); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// The row lock serializes saves of one application; the unique keys
		// still guard races the lock cannot see.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.TableName {
			case "application_definition":
				return nil, domain.Reject(domain.CodeApplicationNameTaken, "application name %q is already used", in.Name)
			case "application_definition_version":
				return nil, draftConflict(in.BaseVersion, -1)
			}
		}
		return nil, err
	}
	return r.FindVersion(ctx, appID, versionID)
}

// LatestVersion returns the latest version of an application, or NOT_FOUND.
func (r *ApplicationRepository) LatestVersion(ctx context.Context, applicationID string) (*domain.ApplicationVersion, error) {
	if uuid.Validate(applicationID) != nil {
		return nil, domain.Reject(domain.CodeNotFound, "application %q not found", applicationID)
	}
	var latest int
	err := r.DB.Pool.QueryRow(ctx, `SELECT max(v.version_number) FROM application_definition a
		JOIN application_definition_version v USING (application_id) WHERE a.application_id = $1 GROUP BY a.application_id`, applicationID).Scan(&latest)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "application %q not found", applicationID)
	}
	if err != nil {
		return nil, err
	}
	return r.FindVersion(ctx, applicationID, strconv.Itoa(latest))
}

func draftConflict(base, latest int) error {
	v := &domain.ValidationError{}
	if latest < 0 {
		v.Add(domain.CodeDraftConflict, "the application changed while this draft based on version %d was saved; reload the latest version and reapply your changes", base)
	} else {
		v.Add(domain.CodeDraftConflict, "this draft is based on version %d but version %d is the latest; reload the latest version and reapply your changes", base, latest)
	}
	return v
}

func rejectTakenName(ctx context.Context, tx pgx.Tx, name, ownID string) error {
	var other string
	err := tx.QueryRow(ctx, `SELECT application_id FROM application_definition WHERE name = $1`, name).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && other == ownID) {
		return nil
	}
	if err != nil {
		return err
	}
	v := &domain.ValidationError{}
	v.AddField("name", domain.CodeApplicationNameTaken, "application name %q is already used by another application", name)
	return v
}

// ensureComponents checks that every submitted ID is either new or a
// component of this application with the same type, then inserts the identity
// rows of new components.
func ensureComponents(ctx context.Context, tx pgx.Tx, appID string, in SaveVersionInput, now time.Time) error {
	type component struct{ id, kind, field string }
	var all []component
	for _, rr := range in.Resources {
		all = append(all, component{rr.ID, "RESOURCE_REQUIREMENT", "resources." + rr.ID + ".id"})
	}
	for _, w := range in.Workloads {
		all = append(all, component{w.ID, "WORKLOAD", "workloads." + w.ID + ".id"})
		for _, d := range w.Variables {
			all = append(all, component{d.ID, "ENVIRONMENT_VARIABLE_DEFINITION", "workloads." + w.ID + ".variables." + d.ID + ".id"})
		}
		for _, d := range w.Secrets {
			all = append(all, component{d.ID, "SECRET_DEFINITION", "workloads." + w.ID + ".secrets." + d.ID + ".id"})
		}
	}
	ids := make([]string, len(all))
	for i, c := range all {
		ids[i] = c.id
	}
	rows, err := tx.Query(ctx, `SELECT component_id::text, application_id::text, component_type::text
		FROM application_component WHERE component_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return err
	}
	type stored struct{ app, kind string }
	existing := map[string]stored{}
	for rows.Next() {
		var id string
		var s stored
		if err := rows.Scan(&id, &s.app, &s.kind); err != nil {
			rows.Close()
			return err
		}
		existing[id] = s
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	v := &domain.ValidationError{}
	for _, c := range all {
		s, ok := existing[strings.ToLower(c.id)]
		switch {
		case !ok:
			if _, err := tx.Exec(ctx, `INSERT INTO application_component (component_id, application_id, component_type, created_at)
				VALUES ($1, $2, $3, $4)`, c.id, appID, c.kind, now); err != nil {
				return err
			}
		case s.app != appID:
			v.AddField(c.field, domain.CodeInvalidComponentID, "component ID %s belongs to another application", c.id)
		case s.kind != c.kind:
			v.AddField(c.field, domain.CodeInvalidComponentID, "component ID %s identifies a %s, not a %s", c.id, s.kind, c.kind)
		}
	}
	return v.OrNil()
}
