import type { ValueSource } from '../api/types';
import type { BindingDraft, ConfigurationDraft } from './model';
import { bindingKey } from './model';

// Every edit of the UC-02 draft happens here, in the browser. Only loading,
// catalog queries, Secret staging and Save call the backend (ADR-016).

export type BindingKind = 'variable' | 'secret';

export type DraftAction =
  | { type: 'replace'; draft: ConfigurationDraft }
  | { type: 'catalog-version'; value: string }
  | { type: 'deployment-target'; value: string }
  | { type: 'source'; kind: BindingKind; key: string; source: ValueSource | '' }
  | { type: 'direct-value'; kind: BindingKind; key: string; value: string }
  | { type: 'secret-reference'; key: string; secretRef: string }
  | { type: 'reference'; kind: BindingKind; key: string; refId: string; outputName: string };

function mapBinding(list: BindingDraft[], key: string, change: (b: BindingDraft) => BindingDraft): BindingDraft[] {
  return list.map((b) => (bindingKey(b.workloadId, b.definitionId) === key ? change(b) : b));
}

function edit(draft: ConfigurationDraft, kind: BindingKind, key: string, change: (b: BindingDraft) => BindingDraft): ConfigurationDraft {
  if (kind === 'variable') return { ...draft, variables: mapBinding(draft.variables, key, change) };
  return { ...draft, secrets: mapBinding(draft.secrets, key, change) };
}

export function draftReducer(draft: ConfigurationDraft, action: DraftAction): ConfigurationDraft {
  switch (action.type) {
    case 'replace':
      return action.draft;

    // Changing either selection can change which outputs exist, so bindings to
    // a Resource Output are cleared rather than silently kept.
    case 'catalog-version':
      return clearResourceOutputs({ ...draft, catalogVersion: action.value });
    case 'deployment-target':
      return clearResourceOutputs({ ...draft, deploymentTarget: action.value });

    case 'source':
      return edit(draft, action.kind, action.key, (b) => ({
        ...b,
        source: action.source,
        // Switching source drops the payload of the previous one.
        value: '',
        secretRef: '',
        refId: '',
        outputName: '',
      }));

    case 'direct-value':
      return edit(draft, action.kind, action.key, (b) => ({ ...b, value: action.value, secretRef: '' }));

    case 'secret-reference':
      return edit(draft, 'secret', action.key, (b) => ({ ...b, secretRef: action.secretRef, value: '' }));

    case 'reference':
      return edit(draft, action.kind, action.key, (b) => ({ ...b, refId: action.refId, outputName: action.outputName }));

    default:
      return draft;
  }
}

function clearResourceOutputs(draft: ConfigurationDraft): ConfigurationDraft {
  const clear = (b: BindingDraft): BindingDraft =>
    b.source === 'RESOURCE_OUTPUT' ? { ...b, refId: '', outputName: '' } : b;
  return { ...draft, variables: draft.variables.map(clear), secrets: draft.secrets.map(clear) };
}
