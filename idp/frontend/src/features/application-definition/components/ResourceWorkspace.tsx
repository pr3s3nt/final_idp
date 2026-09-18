import type { Dispatch } from 'react';
import type { ResourceDraft } from '../draft/model';
import type { DraftAction } from '../draft/reducer';
import { TextField, type ProblemsByField } from '../../../shared/ui/Field';
import { labelOf } from './labels';

interface Props {
  resource: ResourceDraft;
  dispatch: Dispatch<DraftAction>;
  problems: ProblemsByField;
  onRemove: () => void;
}

export function ResourceWorkspace({ resource, dispatch, problems, onRemove }: Props) {
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
