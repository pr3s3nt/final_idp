import type { Dispatch } from 'react';
import type { ApplicationDraft } from '../draft/model';
import type { DraftAction } from '../draft/reducer';
import { TextField, type ProblemsByField } from '../../../shared/ui/Field';

interface Props {
  draft: ApplicationDraft;
  dispatch: Dispatch<DraftAction>;
  problems: ProblemsByField;
}

export function OverviewWorkspace({ draft, dispatch, problems }: Props) {
  return (
    <section className="workspace-panel" aria-labelledby="workspace-overview-title">
      <div className="workspace-heading">
        <div>
          <p className="eyebrow">Application</p>
          <h2 id="workspace-overview-title" className="plain-title">Overview</h2>
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
