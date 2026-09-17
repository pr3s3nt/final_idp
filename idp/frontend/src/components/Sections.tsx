import type { Dispatch } from 'react';
import { componentLabel, type ApplicationDraft, type WorkloadDraft } from '../draft/model';
import type { ConfigGroup, DraftAction } from '../draft/reducer';
import { fieldDomId, RowErrors, SelectField, TextField, type ProblemsByField } from './Field';

interface SectionProps {
  draft: ApplicationDraft;
  dispatch: Dispatch<DraftAction>;
  problems: ProblemsByField;
}

function labelOf(kind: string, name: string) {
  return name ? `${kind} ${name}` : `unnamed ${kind.toLowerCase()}`;
}

export function ApplicationInfoSection({ draft, dispatch, problems }: SectionProps) {
  return (
    <section className="card" aria-labelledby="section-application">
      <h2 id="section-application">Application information</h2>
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
    </section>
  );
}

function WorkloadCard({ workload: w, dispatch, problems }: { workload: WorkloadDraft; dispatch: Dispatch<DraftAction>; problems: ProblemsByField }) {
  const base = `workloads.${w.id}`;
  const label = labelOf('Workload', w.name);
  return (
    <li className="item" aria-label={label}>
      <div className="item-header">
        <h3>{label}</h3>
        <button type="button" className="secondary danger" onClick={() => dispatch({ type: 'removeWorkload', id: w.id })}>
          Remove {label.toLowerCase()}
        </button>
      </div>
      <div className="grid">
        <TextField field={`${base}.name`} label="Workload name" required value={w.name} problems={problems} onChange={(value) => dispatch({ type: 'updateWorkload', id: w.id, field: 'name', value })} />
        <TextField field={`${base}.type`} label="Workload type" required list="workload-types" value={w.type} problems={problems} onChange={(value) => dispatch({ type: 'updateWorkload', id: w.id, field: 'type', value })} />
        <TextField
          field={`${base}.imageRepository`}
          label="Image repository"
          required
          value={w.imageRepository}
          problems={problems}
          hint="No tag: the image version is chosen when deploying."
          placeholder="registry.company.local/shop-backend"
          onChange={(value) => dispatch({ type: 'updateWorkload', id: w.id, field: 'imageRepository', value })}
        />
        <TextField field={`${base}.port`} label="Port (optional)" inputMode="numeric" value={w.port} problems={problems} onChange={(value) => dispatch({ type: 'updateWorkload', id: w.id, field: 'port', value })} />
      </div>
      <fieldset className="sublist">
        <legend>Outputs</legend>
        {w.outputs.length === 0 && <p className="muted">No outputs. Add one, for example endpoint, if other workloads use it.</p>}
        <ul>
          {w.outputs.map((o, i) => (
            <li key={o.id} className="row">
              <TextField field={`${base}.outputs.${i}`} label={`Output ${i + 1} name`} value={o.name} problems={problems} onChange={(name) => dispatch({ type: 'updateOutput', workloadId: w.id, outputId: o.id, name })} />
              <button type="button" className="secondary" onClick={() => dispatch({ type: 'removeOutput', workloadId: w.id, outputId: o.id })}>
                Remove output {o.name || i + 1}
              </button>
            </li>
          ))}
        </ul>
        <button type="button" className="secondary" onClick={() => dispatch({ type: 'addOutput', workloadId: w.id })}>
          Add output to {label.toLowerCase()}
        </button>
      </fieldset>
    </li>
  );
}

export function WorkloadsSection({ draft, dispatch, problems }: SectionProps) {
  return (
    <section className="card" aria-labelledby="section-workloads" id={fieldDomId('workloads')} tabIndex={-1}>
      <h2 id="section-workloads">Workloads</h2>
      <RowErrors field="workloads" problems={problems} />
      {draft.workloads.length === 0 && <p className="muted">No workloads yet. An application needs at least one.</p>}
      <ul className="items">
        {draft.workloads.map((w) => (
          <WorkloadCard key={w.id} workload={w} dispatch={dispatch} problems={problems} />
        ))}
      </ul>
      <datalist id="workload-types">
        <option value="Backend Service" />
        <option value="Frontend" />
        <option value="Worker" />
      </datalist>
      <button type="button" onClick={() => dispatch({ type: 'addWorkload' })}>
        Add workload
      </button>
    </section>
  );
}

export function ResourcesSection({ draft, dispatch, problems }: SectionProps) {
  return (
    <section className="card" aria-labelledby="section-resources">
      <h2 id="section-resources">Resources</h2>
      <p className="muted">Describe logical resources only, for example PostgreSQL or Redis. How they are provisioned is decided at deployment.</p>
      {draft.resources.length === 0 && <p className="muted">No resources. An application may need none.</p>}
      <ul className="items">
        {draft.resources.map((r) => {
          const label = labelOf('Resource', r.name);
          return (
            <li key={r.id} className="item" aria-label={label}>
              <div className="grid">
                <TextField field={`resources.${r.id}.name`} label="Resource name" required value={r.name} problems={problems} onChange={(value) => dispatch({ type: 'updateResource', id: r.id, field: 'name', value })} />
                <TextField field={`resources.${r.id}.type`} label="Resource type" required list="resource-types" value={r.type} problems={problems} onChange={(value) => dispatch({ type: 'updateResource', id: r.id, field: 'type', value })} />
              </div>
              <button type="button" className="secondary danger" onClick={() => dispatch({ type: 'removeResource', id: r.id })}>
                Remove {label.toLowerCase()}
              </button>
            </li>
          );
        })}
      </ul>
      <datalist id="resource-types">
        <option value="PostgreSQL" />
        <option value="Redis" />
      </datalist>
      <button type="button" onClick={() => dispatch({ type: 'addResource' })}>
        Add resource
      </button>
    </section>
  );
}

function ConfigList({ workload: w, group, dispatch, problems }: { workload: WorkloadDraft; group: ConfigGroup; dispatch: Dispatch<DraftAction>; problems: ProblemsByField }) {
  const kind = group === 'variables' ? 'Environment Variable' : 'Secret';
  const workloadLabel = labelOf('workload', w.name);
  return (
    <fieldset className="sublist">
      <legend>{group === 'variables' ? 'Environment Variables' : 'Secrets'}</legend>
      {group === 'secrets' && <p className="hint">Declare the Secret name only. Its value is provided per environment later, never in this editor.</p>}
      {w[group].length === 0 && <p className="muted">None declared.</p>}
      <ul>
        {w[group].map((c) => (
          <li key={c.id} className="row">
            <TextField
              field={`workloads.${w.id}.${group}.${c.id}.name`}
              label={`${kind} name`}
              value={c.name}
              problems={problems}
              onChange={(name) => dispatch({ type: 'updateConfig', workloadId: w.id, group, id: c.id, name })}
            />
            <label className="checkbox">
              <input type="checkbox" checked={c.required} onChange={(e) => dispatch({ type: 'updateConfig', workloadId: w.id, group, id: c.id, required: e.target.checked })} />
              Required
            </label>
            <button type="button" className="secondary" onClick={() => dispatch({ type: 'removeConfig', workloadId: w.id, group, id: c.id })}>
              Remove {kind} {c.name || 'unnamed'}
            </button>
          </li>
        ))}
      </ul>
      <button type="button" className="secondary" onClick={() => dispatch({ type: 'addConfig', workloadId: w.id, group })}>
        Add {kind} to {workloadLabel}
      </button>
    </fieldset>
  );
}

export function ConfigurationSection({ draft, dispatch, problems }: SectionProps) {
  return (
    <section className="card" aria-labelledby="section-configuration">
      <h2 id="section-configuration">Configuration requirements</h2>
      <p className="muted">Declare which Environment Variables and Secrets each workload needs. Values are set per environment in UC-02.</p>
      {draft.workloads.length === 0 && <p className="muted">Add a workload first.</p>}
      {draft.workloads.map((w) => (
        <div key={w.id} className="item" role="group" aria-label={`Configuration of ${labelOf('workload', w.name)}`}>
          <h3>{labelOf('Workload', w.name)}</h3>
          <div className="grid">
            <ConfigList workload={w} group="variables" dispatch={dispatch} problems={problems} />
            <ConfigList workload={w} group="secrets" dispatch={dispatch} problems={problems} />
          </div>
        </div>
      ))}
    </section>
  );
}

export function DependenciesSection({ draft, dispatch, problems, newSource, newTarget, setNewSource, setNewTarget }: SectionProps & {
  newSource: string;
  newTarget: string;
  setNewSource: (v: string) => void;
  setNewTarget: (v: string) => void;
}) {
  const sources = draft.workloads.map((w) => ({ value: w.id, label: labelOf('Workload', w.name) }));
  const targets = [
    ...draft.workloads.map((w) => ({ value: w.id, label: labelOf('Workload', w.name) })),
    ...draft.resources.map((r) => ({ value: r.id, label: labelOf('Resource', r.name) })),
  ];
  const add = () => {
    dispatch({ type: 'addDependency', sourceId: newSource, targetId: newTarget });
    setNewTarget('');
  };
  return (
    <section className="card" aria-labelledby="section-dependencies" id={fieldDomId('dependencies')} tabIndex={-1}>
      <h2 id="section-dependencies">Dependencies</h2>
      <p className="muted">A workload depends on another workload or a resource. Deployment follows this order, and a workload may only use outputs of what it depends on.</p>
      <RowErrors field="dependencies" problems={problems} />
      {draft.dependencies.length === 0 && <p className="muted">No dependencies. Components without a dependency are independent.</p>}
      <ul className="items">
        {draft.dependencies.map((d) => (
          <li key={d.id} className="row item" id={fieldDomId(`dependencies.${d.id}`)} tabIndex={-1} aria-label={`${componentLabel(draft, d.sourceId)} depends on ${componentLabel(draft, d.targetId)}`}>
            <SelectField field={`dependencies.${d.id}`} label="Workload" value={d.sourceId} options={sources} placeholder="Choose a workload" problems={problems} onChange={(sourceId) => dispatch({ type: 'updateDependency', id: d.id, sourceId })} />
            <span aria-hidden="true" className="arrow">
              depends on
            </span>
            <SelectField field={`dependencies.${d.id}`} label="Depends on" value={d.targetId} options={targets} placeholder="Choose a component" problems={problems} onChange={(targetId) => dispatch({ type: 'updateDependency', id: d.id, targetId })} />
            <button type="button" className="secondary" onClick={() => dispatch({ type: 'removeDependency', id: d.id })}>
              Remove dependency {componentLabel(draft, d.sourceId)} → {componentLabel(draft, d.targetId)}
            </button>
            <RowErrors field={`dependencies.${d.id}`} problems={problems} />
          </li>
        ))}
      </ul>
      <div className="row add-dependency" role="group" aria-label="New dependency">
        <SelectField field="new-dependency" label="Workload" value={newSource} options={sources} placeholder="Choose a workload" problems={problems} onChange={setNewSource} />
        <span aria-hidden="true" className="arrow">
          depends on
        </span>
        <SelectField field="new-dependency" label="Depends on" value={newTarget} options={targets.filter((t) => t.value !== newSource)} placeholder="Choose a component" problems={problems} onChange={setNewTarget} />
        <button type="button" disabled={!newSource || !newTarget} onClick={add}>
          Add dependency
        </button>
      </div>
    </section>
  );
}

/** Topology and configuration overview of the current draft. */
export function OverviewSection({ draft }: { draft: ApplicationDraft }) {
  return (
    <section className="card" aria-labelledby="section-overview">
      <h2 id="section-overview">Overview</h2>
      {draft.workloads.length === 0 ? (
        <p className="muted">Nothing to show yet.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th scope="col">Workload</th>
              <th scope="col">Depends on</th>
              <th scope="col">Environment Variables</th>
              <th scope="col">Secrets</th>
            </tr>
          </thead>
          <tbody>
            {draft.workloads.map((w) => (
              <tr key={w.id}>
                <td>{w.name || 'unnamed'}</td>
                <td>
                  {draft.dependencies
                    .filter((d) => d.sourceId === w.id)
                    .map((d) => componentLabel(draft, d.targetId))
                    .join(', ') || '—'}
                </td>
                <td>{w.variables.map((c) => c.name).join(', ') || '—'}</td>
                <td>{w.secrets.map((c) => c.name).join(', ') || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
