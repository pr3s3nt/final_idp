import type { Dispatch } from 'react';
import { componentLabel, type ApplicationDraft, type WorkloadDraft } from '../draft/model';
import type { ConfigGroup, DraftAction } from '../draft/reducer';
import { fieldDomId, RowErrors, SelectField, TextField, type ProblemsByField } from '../../../shared/ui/Field';
import { labelOf } from './labels';

interface Props {
  draft: ApplicationDraft;
  workload: WorkloadDraft;
  dispatch: Dispatch<DraftAction>;
  problems: ProblemsByField;
  onRemove: () => void;
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

export function WorkloadWorkspace({ draft, workload, dispatch, problems, onRemove }: Props) {
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
