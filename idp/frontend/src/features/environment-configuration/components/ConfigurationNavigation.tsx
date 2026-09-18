import type { Problem } from '../../application-definition/api/types';
import type { ConfigurationWorkloadDto } from '../api/types';
import { workloadOfField } from '../draft/validation';

/**
 * Octicon paths (MIT, github/octicons), inlined as ADR-018 requires. The shape
 * is redundant with the text label and never replaces it.
 */
function Icon({ name }: { name: 'workload' | 'review' }) {
  const path = {
    workload: 'M8 4a4 4 0 1 1 0 8 4 4 0 0 1 0-8Z',
    review: 'M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.75.75 0 0 1 1.06-1.06L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z',
  }[name];
  return (
    <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true" focusable="false">
      <path d={path} />
    </svg>
  );
}

export type Workspace = { kind: 'workload'; id: string } | { kind: 'review' };

export function workspaceKey(workspace: Workspace): string {
  return workspace.kind === 'workload' ? `workload:${workspace.id}` : 'review';
}

/** Workspace that owns a problem; a problem without a workload lands on Review. */
export function workspaceForField(field: string | undefined, workloads: ConfigurationWorkloadDto[]): Workspace {
  const workloadId = workloadOfField(field);
  if (workloadId && workloads.some((w) => w.id === workloadId)) return { kind: 'workload', id: workloadId };
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
  workloads: ConfigurationWorkloadDto[];
  selected: Workspace;
  problems: Problem[];
  onSelect: (workspace: Workspace) => void;
}

export function ConfigurationNavigation({ workloads, selected, problems, onSelect }: Props) {
  const selectedKey = workspaceKey(selected);
  const countFor = (workspace: Workspace) =>
    workspace.kind === 'review'
      ? problems.length
      : problems.filter((problem) => workloadOfField(problem.field) === workspace.id).length;

  const item = (workspace: Workspace, label: string, symbol: 'workload' | 'review') => {
    const key = workspaceKey(workspace);
    return (
      <button
        type="button"
        className="builder-nav-item"
        aria-current={selectedKey === key ? 'page' : undefined}
        onClick={() => onSelect(workspace)}
      >
        <span className={`component-symbol ${symbol === 'workload' ? 'workload-symbol' : ''}`} aria-hidden="true">
          <Icon name={symbol} />
        </span>
        <span className="builder-nav-label">{label}</span>
        <ProblemCount count={countFor(workspace)} />
      </button>
    );
  };

  return (
    <nav className="builder-nav" aria-label="Configuration">
      <div className="builder-nav-group">
        <p className="builder-nav-heading">Workloads</p>
        {workloads.map((workload) => (
          <div key={workload.id}>{item({ kind: 'workload', id: workload.id }, workload.name, 'workload')}</div>
        ))}
      </div>
      <div className="builder-nav-group builder-nav-review">{item({ kind: 'review' }, 'Review', 'review')}</div>
    </nav>
  );
}
