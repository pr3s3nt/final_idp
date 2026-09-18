import type { Dispatch } from 'react';
import { componentLabel, type ApplicationDraft, type ResourceDraft, type WorkloadDraft } from '../draft/model';
import type { ConfigGroup, DraftAction } from '../draft/reducer';
import { fieldDomId, RowErrors, SelectField, TextField, type ProblemsByField } from '../../../shared/ui/Field';

interface WorkspaceProps {
  draft: ApplicationDraft;
  dispatch: Dispatch<DraftAction>;
  problems: ProblemsByField;
}

function labelOf(kind: string, name: string) {
  return name ? `${kind} ${name}` : `unnamed ${kind.toLowerCase()}`;
}

export function OverviewWorkspace({ draft, dispatch, problems }: WorkspaceProps) {
  return (
    <section className="workspace-panel" aria-labelledby="workspace-overview-title">
      <div className="workspace-heading">
        <div>
          <p className="eyebrow">Application</p>
          <h2 id="workspace-overview-title">Overview</h2>
          <p className="muted">Name this application and describe what it provides. Environments and image versions are chosen later.</p>
        </div>
      </div>
      <div className="form-section">
        <h3>Application information</h3>
        <div className="grid">
          <TextField
            field="name"
            label="Application name"
            required
            value={draft.name}
            problems={problems}
            hint="Lowercase letters, digits and '-', for example shop-app."
            onChange={(value) => dispatch({ type: 'setApplication', field: 'name', value })}
          />
          <TextField
            field="description"
            label="Description"
            multiline
            value={draft.description}
            problems={problems}
            className="wide"
            onChange={(value) => dispatch({ type: 'setApplication', field: 'description', value })}
          />
        </div>
      </div>
      <div className="boundary-note">
        <strong>This defines the application, not a deployment.</strong>
        <p>UC-01 records workloads, logical resources and configuration requirements. It does not choose an environment, image tag or Secret value.</p>
      </div>
    </section>
  );
}

function ConfigList({ workload, group, dispatch, problems }: { workload: WorkloadDraft; group: ConfigGroup; dispatch: Dispatch<DraftAction>; problems: ProblemsByField }) {
  const kind = group === 'variables' ? 'Environment Variable' : 'Secret';
  return (
    <div className="definition-list">
      <div className="subsection-heading">
        <div>
          <h4>{group === 'variables' ? 'Environment variables' : 'Secrets'}</h4>
          {group === 'secrets' && <p className="hint">Names only. Values are provided per environment in UC-02.</p>}
        </div>
        <button type="button" className="secondary compact" onClick={() => dispatch({ type: 'addConfig', workloadId: workload.id, group })}>
          + Add {group === 'variables' ? 'variable' : 'secret'}
        </button>
      </div>
      {workload[group].length === 0 && <p className="empty-inline">None declared.</p>}
      <ul className="compact-list">
        {workload[group].map((requirement) => (
          <li key={requirement.id} className="definition-row">
            <TextField
              field={`workloads.${workload.id}.${group}.${requirement.id}.name`}
              label={`${kind} name`}
              value={requirement.name}
              problems={problems}
              onChange={(name) => dispatch({ type: 'updateConfig', workloadId: workload.id, group, id: requirement.id, name })}
            />
            <label className="checkbox">
              <input
                type="checkbox"
                checked={requirement.required}
                onChange={(event) => dispatch({ type: 'updateConfig', workloadId: workload.id, group, id: requirement.id, required: event.target.checked })}
              />
              Required
            </label>
            <button type="button" className="icon-button danger" aria-label={`Remove ${kind} ${requirement.name || 'unnamed'}`} onClick={() => dispatch({ type: 'removeConfig', workloadId: workload.id, group, id: requirement.id })}>
              Remove
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function WorkloadDependencies({ draft, workload, dispatch, problems }: { draft: ApplicationDraft; workload: WorkloadDraft; dispatch: Dispatch<DraftAction>; problems: ProblemsByField }) {
  const dependencies = draft.dependencies.filter((dependency) => dependency.sourceId === workload.id);
  const targets = [
    ...draft.workloads.filter((item) => item.id !== workload.id).map((item) => ({ value: item.id, label: labelOf('Workload', item.name) })),
    ...draft.resources.map((item) => ({ value: item.id, label: labelOf('Resource', item.name) })),
  ];
  const usedTargets = new Set(dependencies.map((dependency) => dependency.targetId));
  const hasIncompleteDependency = dependencies.some((dependency) => dependency.targetId === '');
  return (
    <div>
      <p className="section-help">Dependencies determine deployment order and which component outputs this workload may use.</p>
      <RowErrors field="dependencies" problems={problems} />
      {dependencies.length === 0 && <p className="empty-inline">No dependencies. This workload is independent.</p>}
      <ul className="compact-list dependency-list">
        {dependencies.map((dependency) => (
          <li key={dependency.id} id={fieldDomId(`dependencies.${dependency.id}`)} tabIndex={-1} className="dependency-row">
            <span className="dependency-source">{workload.name || 'unnamed workload'}</span>
            <span className="dependency-arrow">depends on</span>
            <SelectField
              field={`dependencies.${dependency.id}`}
              label="Component"
              value={dependency.targetId}
              options={targets}
              placeholder="Choose a component"
              problems={problems}
              onChange={(targetId) => dispatch({ type: 'updateDependency', id: dependency.id, targetId })}
            />
            <button type="button" className="icon-button danger" aria-label={`Remove dependency ${componentLabel(draft, dependency.sourceId)} to ${componentLabel(draft, dependency.targetId)}`} onClick={() => dispatch({ type: 'removeDependency', id: dependency.id })}>
              Remove
            </button>
            <RowErrors field={`dependencies.${dependency.id}`} problems={problems} />
          </li>
        ))}
      </ul>
      <button
        type="button"
        className="secondary compact"
        disabled={hasIncompleteDependency || targets.every((target) => usedTargets.has(target.value))}
        onClick={() => dispatch({ type: 'addDependency', sourceId: workload.id, targetId: '' })}
      >
        + Add dependency
      </button>
      {targets.length === 0 && <span className="hint action-hint"> Add another workload or resource first.</span>}
    </div>
  );
}

export function WorkloadWorkspace({ draft, workload, dispatch, problems, onRemove }: WorkspaceProps & { workload: WorkloadDraft; onRemove: () => void }) {
  const base = `workloads.${workload.id}`;
  const label = labelOf('Workload', workload.name);
  return (
    <section className="workspace-panel" aria-labelledby="workspace-workload-title" aria-label={label}>
      <div className="workspace-heading">
        <div>
          <p className="eyebrow">Workload</p>
          <h2 id="workspace-workload-title">{workload.name || 'Unnamed workload'}</h2>
        </div>
        <button type="button" className="secondary danger" onClick={onRemove}>
          Remove workload
        </button>
      </div>

      <div className="form-section">
        <h3>General</h3>
        <div className="grid">
          <TextField field={`${base}.name`} label="Workload name" required value={workload.name} problems={problems} onChange={(value) => dispatch({ type: 'updateWorkload', id: workload.id, field: 'name', value })} />
          <TextField field={`${base}.type`} label="Workload type" required list="workload-types" value={workload.type} problems={problems} onChange={(value) => dispatch({ type: 'updateWorkload', id: workload.id, field: 'type', value })} />
          <TextField field={`${base}.imageRepository`} label="Image repository" required value={workload.imageRepository} problems={problems} hint="Repository only. The image version is chosen when deploying." placeholder="registry.company.local/shop-backend" onChange={(value) => dispatch({ type: 'updateWorkload', id: workload.id, field: 'imageRepository', value })} />
          <TextField field={`${base}.port`} label="Application port (optional)" inputMode="numeric" value={workload.port} problems={problems} onChange={(value) => dispatch({ type: 'updateWorkload', id: workload.id, field: 'port', value })} />
        </div>
        <datalist id="workload-types">
          <option value="Backend Service" />
          <option value="Frontend" />
          <option value="Worker" />
        </datalist>
      </div>

      <div className="form-section">
        <div className="subsection-heading">
          <div>
            <h3>Outputs</h3>
            <p className="section-help">Names this workload exposes to workloads that depend on it.</p>
          </div>
          <button type="button" className="secondary compact" onClick={() => dispatch({ type: 'addOutput', workloadId: workload.id })}>
            + Add output
          </button>
        </div>
        {workload.outputs.length === 0 && <p className="empty-inline">No outputs declared.</p>}
        <ul className="compact-list output-list">
          {workload.outputs.map((output, index) => (
            <li key={output.id} className="definition-row">
              <TextField field={`${base}.outputs.${index}`} label={`Output ${index + 1} name`} value={output.name} problems={problems} onChange={(name) => dispatch({ type: 'updateOutput', workloadId: workload.id, outputId: output.id, name })} />
              <button type="button" className="icon-button danger" aria-label={`Remove output ${output.name || index + 1}`} onClick={() => dispatch({ type: 'removeOutput', workloadId: workload.id, outputId: output.id })}>
                Remove
              </button>
            </li>
          ))}
        </ul>
      </div>

      <div className="form-section">
        <h3>Configuration requirements</h3>
        <p className="section-help">Declare what this workload needs. Values are supplied per environment in UC-02.</p>
        <div className="configuration-grid">
          <ConfigList workload={workload} group="variables" dispatch={dispatch} problems={problems} />
          <ConfigList workload={workload} group="secrets" dispatch={dispatch} problems={problems} />
        </div>
      </div>

      <div className="form-section">
        <h3>Dependencies</h3>
        <WorkloadDependencies draft={draft} workload={workload} dispatch={dispatch} problems={problems} />
      </div>
    </section>
  );
}

export function ResourceWorkspace({ resource, dispatch, problems, onRemove }: WorkspaceProps & { resource: ResourceDraft; onRemove: () => void }) {
  const label = labelOf('Resource', resource.name);
  return (
    <section className="workspace-panel" aria-labelledby="workspace-resource-title" aria-label={label}>
      <div className="workspace-heading">
        <div>
          <p className="eyebrow">Logical resource</p>
          <h2 id="workspace-resource-title">{resource.name || 'Unnamed resource'}</h2>
          <p className="muted">Describe the capability the application needs. Provisioning is decided at deployment.</p>
        </div>
        <button type="button" className="secondary danger" onClick={onRemove}>
          Remove resource
        </button>
      </div>
      <div className="form-section">
        <h3>Resource information</h3>
        <div className="grid">
          <TextField field={`resources.${resource.id}.name`} label="Resource name" required value={resource.name} problems={problems} onChange={(value) => dispatch({ type: 'updateResource', id: resource.id, field: 'name', value })} />
          <TextField field={`resources.${resource.id}.type`} label="Resource type" required list="resource-types" value={resource.type} problems={problems} onChange={(value) => dispatch({ type: 'updateResource', id: resource.id, field: 'type', value })} />
        </div>
        <datalist id="resource-types">
          <option value="PostgreSQL" />
          <option value="Redis" />
        </datalist>
      </div>
    </section>
  );
}

function Topology({ draft }: { draft: ApplicationDraft }) {
  if (draft.workloads.length === 0 && draft.resources.length === 0) return <p className="empty-inline">Add a workload or resource to see the topology.</p>;
  return (
    <div className="topology" aria-label="Read-only application topology">
      <div className="topology-nodes">
        {draft.workloads.map((workload) => (
          <div className="topology-node workload-node" key={workload.id}>
            <span className="component-symbol" aria-hidden="true">●</span>
            <span><small>Workload</small><strong>{workload.name || 'unnamed'}</strong></span>
          </div>
        ))}
        {draft.resources.map((resource) => (
          <div className="topology-node resource-node" key={resource.id}>
            <span className="component-symbol" aria-hidden="true">◆</span>
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
          <h2 id="workspace-review-title">Review application</h2>
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
                    <td><strong>{workload.name || 'unnamed'}</strong></td>
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
