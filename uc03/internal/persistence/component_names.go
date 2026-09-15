package persistence

import "context"

// ComponentNames maps every stable component ID of an application (all
// versions) to its name in the latest version that contains it.
func (r *ApplicationRepository) ComponentNames(ctx context.Context, applicationID string) (map[string]string, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id::text, name FROM (
			SELECT w.workload_id AS id, w.name, v.version_number FROM workload w JOIN application_definition_version v USING (application_definition_version_id) WHERE v.application_id = $1
			UNION ALL
			SELECT rr.resource_requirement_id, rr.name, v.version_number FROM resource_requirement rr JOIN application_definition_version v USING (application_definition_version_id) WHERE v.application_id = $1
		) c ORDER BY version_number`, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}
