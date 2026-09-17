package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain"
)

// ApplicationRepository reads and writes immutable Application Definition versions.
type ApplicationRepository struct{ DB *DB }

type ApplicationSummary struct {
	ID            string
	Name          string
	Description   string
	LatestVersion int
}

type VersionSummary struct {
	ID        string
	Number    int
	CreatedAt time.Time
}

func (r *ApplicationRepository) ListApplications(ctx context.Context) ([]ApplicationSummary, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT a.application_id, a.name, coalesce(a.description, ''), coalesce(max(v.version_number), 0)
		FROM application_definition a LEFT JOIN application_definition_version v USING (application_id)
		GROUP BY a.application_id ORDER BY a.name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ApplicationSummary, error) {
		var s ApplicationSummary
		err := row.Scan(&s.ID, &s.Name, &s.Description, &s.LatestVersion)
		return s, err
	})
}

func (r *ApplicationRepository) FindApplication(ctx context.Context, idOrName string) (*ApplicationSummary, error) {
	var s ApplicationSummary
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT a.application_id, a.name, coalesce(a.description, ''),
		       coalesce((SELECT max(version_number) FROM application_definition_version v WHERE v.application_id = a.application_id), 0)
		FROM application_definition a
		WHERE a.application_id::text = $1 OR a.name = $1`, idOrName).
		Scan(&s.ID, &s.Name, &s.Description, &s.LatestVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "application %q not found", idOrName)
	}
	return &s, err
}

func (r *ApplicationRepository) ListVersions(ctx context.Context, applicationID string) ([]VersionSummary, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT application_definition_version_id, version_number, created_at
		FROM application_definition_version WHERE application_id = $1 ORDER BY version_number DESC`, applicationID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (VersionSummary, error) {
		var s VersionSummary
		err := row.Scan(&s.ID, &s.Number, &s.CreatedAt)
		return s, err
	})
}

// FindVersion loads one immutable version with all of its components. version
// may be the version ID or the version number as text.
func (r *ApplicationRepository) FindVersion(ctx context.Context, applicationID, version string) (*domain.ApplicationVersion, error) {
	v := &domain.ApplicationVersion{}
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT a.application_id, a.name, v.application_definition_version_id, v.version_number
		FROM application_definition_version v JOIN application_definition a USING (application_id)
		WHERE a.application_id = $1 AND (v.application_definition_version_id::text = $2 OR v.version_number::text = $2)`,
		applicationID, version).Scan(&v.ApplicationID, &v.ApplicationName, &v.VersionID, &v.VersionNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.Reject(domain.CodeNotFound, "application definition version %q not found", version)
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.DB.Pool.Query(ctx, `
		SELECT workload_id, name, type, image_repository, port, exposed_outputs
		FROM workload WHERE application_definition_version_id = $1 ORDER BY name`, v.VersionID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var w domain.Workload
		var outputs []byte
		if err := rows.Scan(&w.ID, &w.Name, &w.Type, &w.ImageRepository, &w.Port, &outputs); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(outputs, &w.ExposedOutputs); err != nil {
			rows.Close()
			return nil, err
		}
		v.Workloads = append(v.Workloads, w)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	defs := func(table, idCol string, assign func(w *domain.Workload, d domain.ConfigDefinition)) error {
		rows, err := r.DB.Pool.Query(ctx, fmt.Sprintf(`
			SELECT workload_id, %s, name, required FROM %s
			WHERE application_definition_version_id = $1 ORDER BY name`, idCol, table), v.VersionID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var wid string
			var d domain.ConfigDefinition
			if err := rows.Scan(&wid, &d.ID, &d.Name, &d.Required); err != nil {
				return err
			}
			assign(v.Workload(wid), d)
		}
		return rows.Err()
	}
	if err := defs("environment_variable_definition", "variable_definition_id", func(w *domain.Workload, d domain.ConfigDefinition) {
		w.Variables = append(w.Variables, d)
	}); err != nil {
		return nil, err
	}
	if err := defs("secret_definition", "secret_definition_id", func(w *domain.Workload, d domain.ConfigDefinition) {
		w.Secrets = append(w.Secrets, d)
	}); err != nil {
		return nil, err
	}

	rows, err = r.DB.Pool.Query(ctx, `
		SELECT resource_requirement_id, name, resource_type FROM resource_requirement
		WHERE application_definition_version_id = $1 ORDER BY name`, v.VersionID)
	if err != nil {
		return nil, err
	}
	v.Resources, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.ResourceRequirement, error) {
		var rr domain.ResourceRequirement
		err := row.Scan(&rr.ID, &rr.Name, &rr.ResourceType)
		return rr, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = r.DB.Pool.Query(ctx, `
		SELECT dependency_id, source_workload_id, target_type::text,
		       coalesce(target_workload_id, target_resource_requirement_id)
		FROM dependency WHERE application_definition_version_id = $1 ORDER BY dependency_id`, v.VersionID)
	if err != nil {
		return nil, err
	}
	v.Dependencies, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Dependency, error) {
		var d domain.Dependency
		err := row.Scan(&d.ID, &d.SourceWorkloadID, &d.TargetType, &d.TargetID)
		return d, err
	})
	return v, err
}

// VersionInput is a full Application Definition submitted for a new version.
// Components without an ID are new; components with an ID keep it.
type VersionInput struct {
	ApplicationName string
	Description     string
	Workloads       []domain.Workload
	Resources       []domain.ResourceRequirement
	// Dependencies reference components by name (source workload -> target).
	Dependencies []DependencyInput
}

type DependencyInput struct {
	Source string
	Target string
}

// SaveNewVersion inserts a new immutable version (used by the UC-01 fixture
// importer). Existing versions are never updated or deleted. Components are
// matched to earlier versions by explicit ID, otherwise by name, so stable IDs
// survive re-imports.
func (r *ApplicationRepository) SaveNewVersion(ctx context.Context, in VersionInput) (*domain.ApplicationVersion, error) {
	var versionID string
	var appID string
	err := r.DB.InTx(ctx, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		err := tx.QueryRow(ctx, `SELECT application_id FROM application_definition WHERE name = $1 FOR UPDATE`, in.ApplicationName).Scan(&appID)
		if errors.Is(err, pgx.ErrNoRows) {
			appID = uuid.NewString()
			_, err = tx.Exec(ctx, `INSERT INTO application_definition (application_id, name, description, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $4)`, appID, in.ApplicationName, in.Description, now)
		} else if err == nil {
			_, err = tx.Exec(ctx, `UPDATE application_definition SET description = $2, updated_at = $3 WHERE application_id = $1`, appID, in.Description, now)
		}
		if err != nil {
			return err
		}

		// Stable IDs of components from previous versions, looked up by name.
		previous := map[string]string{}
		prow, err := tx.Query(ctx, `
			SELECT 'W:' || w.name, w.workload_id FROM workload w JOIN application_definition_version v USING (application_definition_version_id) WHERE v.application_id = $1
			UNION ALL
			SELECT 'R:' || r.name, r.resource_requirement_id FROM resource_requirement r JOIN application_definition_version v USING (application_definition_version_id) WHERE v.application_id = $1
			UNION ALL
			SELECT 'V:' || ev.workload_id || ':' || ev.name, ev.variable_definition_id FROM environment_variable_definition ev JOIN application_definition_version v USING (application_definition_version_id) WHERE v.application_id = $1
			UNION ALL
			SELECT 'S:' || sd.workload_id || ':' || sd.name, sd.secret_definition_id FROM secret_definition sd JOIN application_definition_version v USING (application_definition_version_id) WHERE v.application_id = $1`, appID)
		if err != nil {
			return err
		}
		for prow.Next() {
			var k, id string
			if err := prow.Scan(&k, &id); err != nil {
				prow.Close()
				return err
			}
			previous[k] = id
		}
		prow.Close()

		ensureComponent := func(id, kind string) error {
			_, err := tx.Exec(ctx, `INSERT INTO application_component (component_id, application_id, component_type, created_at)
				VALUES ($1, $2, $3, $4) ON CONFLICT (component_id) DO NOTHING`, id, appID, kind, now)
			return err
		}
		stableID := func(given, key string) string {
			if given != "" {
				return given
			}
			if id, ok := previous[key]; ok {
				return id
			}
			return uuid.NewString()
		}

		var number int
		if err := tx.QueryRow(ctx, `SELECT coalesce(max(version_number), 0) + 1 FROM application_definition_version WHERE application_id = $1`, appID).Scan(&number); err != nil {
			return err
		}
		versionID = uuid.NewString()
		if _, err := tx.Exec(ctx, `INSERT INTO application_definition_version (application_definition_version_id, application_id, version_number, created_at)
			VALUES ($1, $2, $3, $4)`, versionID, appID, number, now); err != nil {
			return err
		}

		byName := map[string]string{} // component name -> ID within this version
		kindByName := map[string]string{}
		for _, rr := range in.Resources {
			id := stableID(rr.ID, "R:"+rr.Name)
			if err := ensureComponent(id, "RESOURCE_REQUIREMENT"); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO resource_requirement (application_definition_version_id, resource_requirement_id, name, resource_type)
				VALUES ($1, $2, $3, $4)`, versionID, id, rr.Name, rr.ResourceType); err != nil {
				return err
			}
			byName[rr.Name], kindByName[rr.Name] = id, domain.TargetResource
		}
		for _, w := range in.Workloads {
			id := stableID(w.ID, "W:"+w.Name)
			if err := ensureComponent(id, "WORKLOAD"); err != nil {
				return err
			}
			outputs, _ := json.Marshal(nonNil(w.ExposedOutputs))
			if _, err := tx.Exec(ctx, `INSERT INTO workload (application_definition_version_id, workload_id, name, type, image_repository, port, exposed_outputs)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`, versionID, id, w.Name, w.Type, w.ImageRepository, w.Port, outputs); err != nil {
				return err
			}
			if _, dup := byName[w.Name]; dup {
				return domain.Reject(domain.CodeInvalidInput, "component name %q is used twice", w.Name)
			}
			byName[w.Name], kindByName[w.Name] = id, domain.TargetWorkload
			for _, d := range w.Variables {
				did := stableID(d.ID, "V:"+id+":"+d.Name)
				if err := ensureComponent(did, "ENVIRONMENT_VARIABLE_DEFINITION"); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO environment_variable_definition (application_definition_version_id, variable_definition_id, workload_id, name, required)
					VALUES ($1, $2, $3, $4, $5)`, versionID, did, id, d.Name, d.Required); err != nil {
					return err
				}
			}
			for _, d := range w.Secrets {
				did := stableID(d.ID, "S:"+id+":"+d.Name)
				if err := ensureComponent(did, "SECRET_DEFINITION"); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO secret_definition (application_definition_version_id, secret_definition_id, workload_id, name, required)
					VALUES ($1, $2, $3, $4, $5)`, versionID, did, id, d.Name, d.Required); err != nil {
					return err
				}
			}
		}
		for _, d := range in.Dependencies {
			src, ok := byName[d.Source]
			if !ok || kindByName[d.Source] != domain.TargetWorkload {
				return domain.Reject(domain.CodeInvalidInput, "dependency source %q is not a workload of the version", d.Source)
			}
			tgt, ok := byName[d.Target]
			if !ok {
				return domain.Reject(domain.CodeInvalidInput, "dependency target %q does not exist in the version", d.Target)
			}
			var tw, tr *string
			if kindByName[d.Target] == domain.TargetWorkload {
				tw = &tgt
			} else {
				tr = &tgt
			}
			if _, err := tx.Exec(ctx, `INSERT INTO dependency (dependency_id, application_definition_version_id, source_workload_id, target_type, target_workload_id, target_resource_requirement_id)
				VALUES ($1, $2, $3, $4, $5, $6)`, uuid.NewString(), versionID, src, kindByName[d.Target], tw, tr); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.FindVersion(ctx, appID, versionID)
}

// EnsurePlatformComponent inserts the identity row of an implicit platform
// requirement the first time an instance of it is persisted.
func EnsurePlatformComponent(ctx context.Context, q Querier, applicationID, requirementType string) error {
	_, err := q.Exec(ctx, `INSERT INTO application_component (component_id, application_id, component_type, platform_requirement_type, created_at)
		VALUES ($1, $2, 'PLATFORM_REQUIREMENT', $3, now()) ON CONFLICT (component_id) DO NOTHING`,
		domain.PlatformRequirementID(applicationID, requirementType), applicationID, requirementType)
	return err
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
