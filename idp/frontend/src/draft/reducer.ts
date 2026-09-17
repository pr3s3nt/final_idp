import {
  newId,
  newResource,
  newWorkload,
  type ApplicationDraft,
  type ConfigRequirementDraft,
  type ResourceDraft,
  type WorkloadDraft,
} from './model';

export type ConfigGroup = 'variables' | 'secrets';

// Field and component edits only change the client draft; none of them call
// the backend (ADR-016).
export type DraftAction =
  | { type: 'replace'; draft: ApplicationDraft }
  | { type: 'setApplication'; field: 'name' | 'description'; value: string }
  | { type: 'addWorkload'; workload?: WorkloadDraft }
  | { type: 'updateWorkload'; id: string; field: 'name' | 'type' | 'imageRepository' | 'port'; value: string }
  | { type: 'removeWorkload'; id: string }
  | { type: 'addOutput'; workloadId: string }
  | { type: 'updateOutput'; workloadId: string; outputId: string; name: string }
  | { type: 'removeOutput'; workloadId: string; outputId: string }
  | { type: 'addResource'; resource?: ResourceDraft }
  | { type: 'updateResource'; id: string; field: 'name' | 'type'; value: string }
  | { type: 'removeResource'; id: string }
  | { type: 'addConfig'; workloadId: string; group: ConfigGroup; requirement?: ConfigRequirementDraft }
  | { type: 'updateConfig'; workloadId: string; group: ConfigGroup; id: string; name?: string; required?: boolean }
  | { type: 'removeConfig'; workloadId: string; group: ConfigGroup; id: string }
  | { type: 'addDependency'; sourceId: string; targetId: string }
  | { type: 'updateDependency'; id: string; sourceId?: string; targetId?: string }
  | { type: 'removeDependency'; id: string };

function mapWorkload(draft: ApplicationDraft, id: string, change: (w: WorkloadDraft) => WorkloadDraft): ApplicationDraft {
  return { ...draft, workloads: draft.workloads.map((w) => (w.id === id ? change(w) : w)) };
}

/** Removing a component also removes dependencies that point to or from it. */
function withoutDependenciesOf(draft: ApplicationDraft, id: string): ApplicationDraft {
  return { ...draft, dependencies: draft.dependencies.filter((d) => d.sourceId !== id && d.targetId !== id) };
}

export function draftReducer(draft: ApplicationDraft, action: DraftAction): ApplicationDraft {
  switch (action.type) {
    case 'replace':
      return action.draft;
    case 'setApplication':
      return { ...draft, [action.field]: action.value };
    case 'addWorkload':
      return { ...draft, workloads: [...draft.workloads, action.workload ?? newWorkload()] };
    case 'updateWorkload':
      return mapWorkload(draft, action.id, (w) => ({ ...w, [action.field]: action.value }));
    case 'removeWorkload':
      return withoutDependenciesOf({ ...draft, workloads: draft.workloads.filter((w) => w.id !== action.id) }, action.id);
    case 'addOutput':
      return mapWorkload(draft, action.workloadId, (w) => ({ ...w, outputs: [...w.outputs, { id: newId(), name: '' }] }));
    case 'updateOutput':
      return mapWorkload(draft, action.workloadId, (w) => ({
        ...w,
        outputs: w.outputs.map((o) => (o.id === action.outputId ? { ...o, name: action.name } : o)),
      }));
    case 'removeOutput':
      return mapWorkload(draft, action.workloadId, (w) => ({ ...w, outputs: w.outputs.filter((o) => o.id !== action.outputId) }));
    case 'addResource':
      return { ...draft, resources: [...draft.resources, action.resource ?? newResource()] };
    case 'updateResource':
      return { ...draft, resources: draft.resources.map((r) => (r.id === action.id ? { ...r, [action.field]: action.value } : r)) };
    case 'removeResource':
      return withoutDependenciesOf({ ...draft, resources: draft.resources.filter((r) => r.id !== action.id) }, action.id);
    case 'addConfig':
      return mapWorkload(draft, action.workloadId, (w) => ({
        ...w,
        [action.group]: [...w[action.group], action.requirement ?? { id: newId(), name: '', required: true }],
      }));
    case 'updateConfig':
      return mapWorkload(draft, action.workloadId, (w) => ({
        ...w,
        [action.group]: w[action.group].map((c) =>
          c.id === action.id
            ? { ...c, ...(action.name !== undefined && { name: action.name }), ...(action.required !== undefined && { required: action.required }) }
            : c,
        ),
      }));
    case 'removeConfig':
      return mapWorkload(draft, action.workloadId, (w) => ({ ...w, [action.group]: w[action.group].filter((c) => c.id !== action.id) }));
    case 'addDependency':
      return { ...draft, dependencies: [...draft.dependencies, { id: newId(), sourceId: action.sourceId, targetId: action.targetId }] };
    case 'updateDependency':
      return {
        ...draft,
        dependencies: draft.dependencies.map((d) =>
          d.id === action.id
            ? { ...d, ...(action.sourceId !== undefined && { sourceId: action.sourceId }), ...(action.targetId !== undefined && { targetId: action.targetId }) }
            : d,
        ),
      };
    case 'removeDependency':
      return { ...draft, dependencies: draft.dependencies.filter((d) => d.id !== action.id) };
  }
}
