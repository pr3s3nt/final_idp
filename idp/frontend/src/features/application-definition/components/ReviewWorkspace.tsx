import { componentLabel, type ApplicationDraft } from '../draft/model';

/** Octicon paths (MIT, github/octicons); see BuilderNavigation for the rule. */
function NodeIcon({ kind }: { kind: 'workload' | 'resource' }) {
  const path =
    kind === 'workload'
      ? 'M8 4a4 4 0 1 1 0 8 4 4 0 0 1 0-8Z'
      : 'M8 1.5 14.5 8 8 14.5 1.5 8 8 1.5Z';
  return (
    <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true" focusable="false">
      <path d={path} />
    </svg>
  );
}

function Topology({ draft }: { draft: ApplicationDraft }) {
  if (draft.workloads.length === 0 && draft.resources.length === 0) return <p className="empty-inline">Add a workload or resource to see the topology.</p>;
  return (
    <div className="topology" aria-label="Read-only application topology">
      <div className="topology-nodes">
        {draft.workloads.map((workload) => (
          <div className="topology-node workload-node" key={workload.id}>
            <span className="component-symbol workload-symbol" aria-hidden="true"><NodeIcon kind="workload" /></span>
            <span><small>Workload</small><strong>{workload.name || 'unnamed'}</strong></span>
          </div>
        ))}
        {draft.resources.map((resource) => (
          <div className="topology-node resource-node" key={resource.id}>
            <span className="component-symbol resource-symbol" aria-hidden="true"><NodeIcon kind="resource" /></span>
            <span><small>Resource</small><strong>{resource.name || 'unnamed'}</strong></span>
          </div>
        ))}
      </div>
      <div className="topology-relations">
        <h4>Relationships</h4>
        {draft.dependencies.length === 0 ? (
          <p className="empty-inline">Components are independent.</p>
        ) : (
          <ul>
            {draft.dependencies.map((dependency) => (
              <li key={dependency.id}><strong>{componentLabel(draft, dependency.sourceId)}</strong> <span>depends on →</span> <strong>{componentLabel(draft, dependency.targetId)}</strong></li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

export function ReviewWorkspace({ draft }: { draft: ApplicationDraft }) {
  return (
    <section className="workspace-panel" aria-labelledby="workspace-review-title">
      <div className="workspace-heading">
        <div>
          <p className="eyebrow">Final check</p>
          <h2 id="workspace-review-title" className="plain-title">Review application</h2>
          <p className="muted">Confirm the structure before saving a new immutable version.</p>
        </div>
      </div>
      <div className="form-section">
        <h3>Topology <span className="read-only-badge">Read only</span></h3>
        <Topology draft={draft} />
      </div>
      <div className="form-section">
        <h3>Component summary</h3>
        <p className="summary-counts">{draft.workloads.length} {draft.workloads.length === 1 ? 'workload' : 'workloads'} · {draft.resources.length} {draft.resources.length === 1 ? 'resource' : 'resources'} · {draft.dependencies.length} {draft.dependencies.length === 1 ? 'dependency' : 'dependencies'}</p>
        {draft.workloads.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead><tr><th scope="col">Workload</th><th scope="col">Runtime</th><th scope="col">Outputs</th><th scope="col">Configuration</th></tr></thead>
              <tbody>
                {draft.workloads.map((workload) => (
                  <tr key={workload.id}>
                    <td><strong className="component-name">{workload.name || 'unnamed'}</strong></td>
                    <td>{workload.type || '—'}{workload.port ? ` · port ${workload.port}` : ''}</td>
                    <td>{workload.outputs.map((output) => output.name || 'unnamed').join(', ') || '—'}</td>
                    <td>{workload.variables.length} variables · {workload.secrets.length} secrets</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </section>
  );
}
