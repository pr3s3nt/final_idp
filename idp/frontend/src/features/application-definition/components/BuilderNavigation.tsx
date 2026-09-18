import type { Problem } from '../api/types';
import { componentLabel, type ApplicationDraft } from '../draft/model';

export type Workspace =
  | { kind: 'overview' }
  | { kind: 'workload'; id: string }
  | { kind: 'resource'; id: string }
  | { kind: 'review' };

export function workspaceKey(workspace: Workspace): string {
  return workspace.kind === 'workload' || workspace.kind === 'resource' ? `${workspace.kind}:${workspace.id}` : workspace.kind;
}

export function workspaceForField(draft: ApplicationDraft, field?: string): Workspace {
  if (!field || field === 'workloads' || field === 'dependencies') return { kind: 'review' };
  if (field === 'name' || field === 'description') return { kind: 'overview' };

  const workload = /^workloads\.([^.]+)/.exec(field)?.[1];
  if (workload && draft.workloads.some((item) => item.id === workload)) return { kind: 'workload', id: workload };

  const resource = /^resources\.([^.]+)/.exec(field)?.[1];
  if (resource && draft.resources.some((item) => item.id === resource)) return { kind: 'resource', id: resource };

  const dependency = /^dependencies\.([^.]+)/.exec(field)?.[1];
  if (dependency) {
    const sourceId = draft.dependencies.find((item) => item.id === dependency)?.sourceId;
    if (sourceId && draft.workloads.some((item) => item.id === sourceId)) return { kind: 'workload', id: sourceId };
  }
  return { kind: 'review' };
}

function ProblemCount({ count }: { count: number }) {
  if (count === 0) return null;
  return (
    <span className="nav-problem-count" aria-label={`${count} ${count === 1 ? 'problem' : 'problems'}`}>
      {count}
    </span>
  );
}

interface Props {
  draft: ApplicationDraft;
  selected: Workspace;
  problems: Problem[];
  onSelect: (workspace: Workspace) => void;
  onAddWorkload: () => void;
  onAddResource: () => void;
}

export function BuilderNavigation({ draft, selected, problems, onSelect, onAddWorkload, onAddResource }: Props) {
  const selectedKey = workspaceKey(selected);
  const countFor = (workspace: Workspace) => {
    if (workspace.kind === 'review') return problems.length;
    const key = workspaceKey(workspace);
    return problems.filter((problem) => workspaceKey(workspaceForField(draft, problem.field)) === key).length;
  };
  const item = (workspace: Workspace, label: string, symbol: string) => {
    const key = workspaceKey(workspace);
    return (
      <button type="button" className="builder-nav-item" aria-current={selectedKey === key ? 'page' : undefined} onClick={() => onSelect(workspace)}>
        <span className="component-symbol" aria-hidden="true">
          {symbol}
        </span>
        <span className="builder-nav-label">{label}</span>
        <ProblemCount count={countFor(workspace)} />
      </button>
    );
  };

  return (
    <nav className="builder-nav" aria-label="Application builder">
      <div className="builder-nav-group">
        <p className="builder-nav-heading">Application</p>
        {item({ kind: 'overview' }, 'Overview', 'A')}
      </div>

      <div className="builder-nav-group">
        <p className="builder-nav-heading">Workloads</p>
        {draft.workloads.map((workload) => (
          <div key={workload.id}>{item({ kind: 'workload', id: workload.id }, componentLabel(draft, workload.id), '●')}</div>
        ))}
        <button type="button" className="builder-nav-add" onClick={onAddWorkload}>
          + Add workload
        </button>
      </div>

      <div className="builder-nav-group">
        <p className="builder-nav-heading">Resources</p>
        {draft.resources.map((resource) => (
          <div key={resource.id}>{item({ kind: 'resource', id: resource.id }, componentLabel(draft, resource.id), '◆')}</div>
        ))}
        <button type="button" className="builder-nav-add" onClick={onAddResource}>
          + Add resource
        </button>
      </div>

      <div className="builder-nav-group builder-nav-review">{item({ kind: 'review' }, 'Review', '✓')}</div>
    </nav>
  );
}
