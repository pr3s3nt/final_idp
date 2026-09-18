import type { Problem } from '../api/types';
import type { ApplicationDraft } from './model';

// Client-side checks for fast feedback. They follow the UC-01 specification
// rules; the backend validates the complete draft again and is authoritative.
// Field paths match the backend: workloads.<id>.name, dependencies.<id>, ...

const componentName = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
const definitionName = /^[A-Za-z_][A-Za-z0-9_]*$/;
const pathComponent = '[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*';
const hostComponent = '(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])';
const imageRepository = new RegExp(
  `^(?:${hostComponent}(?:\\.${hostComponent})*(?::[0-9]+)?/)?${pathComponent}(?:/${pathComponent})*$`,
);

export function validateDraft(draft: ApplicationDraft): Problem[] {
  const problems: Problem[] = [];
  const add = (field: string, code: string, message: string) => problems.push({ field, code, message });

  const checkComponentName = (field: string, label: string, name: string) => {
    if (name === '') add(field, 'MISSING_REQUIRED_FIELD', `${label} is required.`);
    else if (name.length > 63 || !componentName.test(name))
      add(field, 'INVALID_NAME', `${label} "${name}" must use lowercase letters, digits and "-", start and end with a letter or digit, and have at most 63 characters.`);
  };
  const checkType = (field: string, label: string, value: string) => {
    if (value.trim() === '') add(field, 'MISSING_REQUIRED_FIELD', `${label} is required.`);
    else if (value.length > 100 || value.trim() !== value)
      add(field, 'INVALID_INPUT', `${label} must have at most 100 characters and no leading or trailing spaces.`);
  };

  checkComponentName('name', 'Application name', draft.name);
  if (draft.workloads.length === 0) add('workloads', 'NO_WORKLOAD', 'An application needs at least one workload.');

  const names = new Map<string, string>();
  const checkUnique = (field: string, label: string, name: string) => {
    if (name === '') return;
    const other = names.get(name);
    if (other) add(field, 'DUPLICATE_NAME', `${label} name "${name}" is already used by ${other}.`);
    else names.set(name, `${label.toLowerCase()} ${name}`);
  };

  for (const w of draft.workloads) {
    const base = `workloads.${w.id}`;
    checkComponentName(`${base}.name`, 'Workload name', w.name);
    checkUnique(`${base}.name`, 'Workload', w.name);
    checkType(`${base}.type`, 'Workload type', w.type);

    const repo = w.imageRepository;
    const lastSegment = repo.slice(repo.lastIndexOf('/') + 1);
    if (repo === '') add(`${base}.imageRepository`, 'MISSING_REQUIRED_FIELD', 'Image repository is required.');
    else if (repo.includes('@') || lastSegment.includes(':'))
      add(`${base}.imageRepository`, 'INVALID_IMAGE_REPOSITORY', `Image repository "${repo}" must not include a tag or digest; the image version is chosen at deployment.`);
    else if (repo.length > 1024 || !imageRepository.test(repo))
      add(`${base}.imageRepository`, 'INVALID_IMAGE_REPOSITORY', `Image repository "${repo}" is not valid, for example registry.company.local/shop-backend.`);

    const port = w.port.trim();
    if (port !== '' && (!/^\d+$/.test(port) || Number(port) < 1 || Number(port) > 65535))
      add(`${base}.port`, 'INVALID_PORT', `Port "${w.port}" must be a whole number from 1 to 65535.`);

    const outputs = new Set<string>();
    w.outputs.forEach((o, i) => {
      const field = `${base}.outputs.${i}`;
      checkComponentName(field, 'Output name', o.name);
      if (o.name !== '' && outputs.has(o.name)) add(field, 'DUPLICATE_NAME', `Output "${o.name}" is declared twice.`);
      outputs.add(o.name);
    });

    const configNames = new Map<string, string>();
    const checkConfig = (group: 'variables' | 'secrets', kind: string) => {
      for (const c of w[group]) {
        const field = `${base}.${group}.${c.id}.name`;
        if (c.name === '') add(field, 'MISSING_REQUIRED_FIELD', `${kind} name is required.`);
        else if (c.name.length > 255 || !definitionName.test(c.name))
          add(field, 'INVALID_NAME', `${kind} name "${c.name}" must use letters, digits and "_" and must not start with a digit.`);
        if (c.name === '') continue;
        const other = configNames.get(c.name);
        if (other) add(field, 'DUPLICATE_NAME', `This workload already declares ${other} "${c.name}".`);
        else configNames.set(c.name, kind.toLowerCase());
      }
    };
    checkConfig('variables', 'Environment Variable');
    checkConfig('secrets', 'Secret');
  }

  for (const r of draft.resources) {
    const base = `resources.${r.id}`;
    checkComponentName(`${base}.name`, 'Resource name', r.name);
    checkUnique(`${base}.name`, 'Resource', r.name);
    checkType(`${base}.type`, 'Resource type', r.type);
  }

  const workloadIds = new Set(draft.workloads.map((w) => w.id));
  const resourceIds = new Set(draft.resources.map((r) => r.id));
  const nameOf = (id: string) => draft.workloads.find((w) => w.id === id)?.name ?? draft.resources.find((r) => r.id === id)?.name ?? id;
  const pairs = new Set<string>();
  const edges = new Map<string, string[]>();
  for (const d of draft.dependencies) {
    const field = `dependencies.${d.id}`;
    if (d.sourceId === '' || d.targetId === '') {
      add(field, 'MISSING_REQUIRED_FIELD', 'A dependency needs a source and a target.');
      continue;
    }
    if (!workloadIds.has(d.sourceId)) {
      if (resourceIds.has(d.sourceId)) add(field, 'INVALID_DEPENDENCY', `Resource "${nameOf(d.sourceId)}" cannot depend on another component; only workloads can.`);
      else add(field, 'DEPENDENCY_UNRESOLVED', 'The dependency source does not exist in the application.');
      continue;
    }
    if (!workloadIds.has(d.targetId) && !resourceIds.has(d.targetId)) {
      add(field, 'DEPENDENCY_UNRESOLVED', 'The dependency target does not exist in the application.');
      continue;
    }
    if (d.sourceId === d.targetId) {
      add(field, 'DEPENDENCY_CYCLE', `Workload "${nameOf(d.sourceId)}" cannot depend on itself.`);
      continue;
    }
    const pair = `${d.sourceId}>${d.targetId}`;
    if (pairs.has(pair)) {
      add(field, 'DUPLICATE_DEPENDENCY', `"${nameOf(d.sourceId)}" already depends on "${nameOf(d.targetId)}".`);
      continue;
    }
    pairs.add(pair);
    if (workloadIds.has(d.targetId)) edges.set(d.sourceId, [...(edges.get(d.sourceId) ?? []), d.targetId]);
  }
  const cycle = findCycle(draft.workloads.map((w) => w.id), edges);
  if (cycle) add('dependencies', 'DEPENDENCY_CYCLE', `Depends-on relations form a cycle: ${cycle.map(nameOf).join(' → ')}.`);

  return problems;
}

function findCycle(ids: string[], edges: Map<string, string[]>): string[] | null {
  const state = new Map<string, 'active' | 'done'>();
  const stack: string[] = [];
  const visit = (id: string): string[] | null => {
    state.set(id, 'active');
    stack.push(id);
    for (const next of edges.get(id) ?? []) {
      if (state.get(next) === 'active') return [...stack.slice(stack.indexOf(next)), next];
      if (!state.has(next)) {
        const found = visit(next);
        if (found) return found;
      }
    }
    stack.pop();
    state.set(id, 'done');
    return null;
  };
  for (const id of ids) {
    if (!state.has(id)) {
      const found = visit(id);
      if (found) return found;
    }
  }
  return null;
}
