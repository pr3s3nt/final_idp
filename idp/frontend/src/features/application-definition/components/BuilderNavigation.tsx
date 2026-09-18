import type { Problem } from '../api/types';
import { componentLabel, type ApplicationDraft } from '../draft/model';

/**
 * Octicon paths (MIT, github/octicons). Inlined rather than added as a
 * dependency, as ADR-018 requires. Shape is redundant with the text label and
 * never replaces it.
 */
function Icon({ name }: { name: 'app' | 'workload' | 'resource' | 'review' }) {
  const path = {
    // apps
    app: 'M2 2h5v5H2V2Zm7 0h5v5H9V2ZM2 9h5v5H2V9Zm7 0h5v5H9V9Z',
    // dot-fill
    workload: 'M8 4a4 4 0 1 1 0 8 4 4 0 0 1 0-8Z',
    // diamond
    resource: 'M8 1.5 14.5 8 8 14.5 1.5 8 8 1.5Z',
    // check
    review: 'M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.75.75 0 0 1 1.06-1.06L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z',
  }[name];
  return (
    <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true" focusable="false">
      <path d={path} />
    </svg>
  );
}

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
  const item = (workspace: Workspace, label: string, symbol: 'app' | 'workload' | 'resource' | 'review') => {
    const key = workspaceKey(workspace);
    return (
      <button type="button" className="builder-nav-item" aria-current={selectedKey === key ? 'page' : undefined} onClick={() => onSelect(workspace)}>
        <span className={`component-symbol ${symbol}-symbol`} aria-hidden="true">
          <Icon name={symbol} />
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
        {item({ kind: 'overview' }, 'Overview', 'app')}
      </div>

      <div className="builder-nav-group">
        <p className="builder-nav-heading">Workloads</p>
        {draft.workloads.map((workload) => (
          <div key={workload.id}>{item({ kind: 'workload', id: workload.id }, componentLabel(draft, workload.id), 'workload')}</div>
        ))}
        <button type="button" className="builder-nav-add" onClick={onAddWorkload}>
          + Add workload
        </button>
      </div>

      <div className="builder-nav-group">
        <p className="builder-nav-heading">Resources</p>
        {draft.resources.map((resource) => (
          <div key={resource.id}>{item({ kind: 'resource', id: resource.id }, componentLabel(draft, resource.id), 'resource')}</div>
        ))}
        <button type="button" className="builder-nav-add" onClick={onAddResource}>
          + Add resource
        </button>
      </div>

      <div className="builder-nav-group builder-nav-review">{item({ kind: 'review' }, 'Review', 'review')}</div>
    </nav>
  );
}
