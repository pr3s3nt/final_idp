import type { ApplicationDefinitionDto } from '../api/types';

// ApplicationDefinitionDraft as held by the browser tab (ADR-016). Every
// repeated item has a stable client ID: saved components keep their server ID,
// new components get a UUID that the backend keeps as their stable ID.

export interface ConfigRequirementDraft {
  id: string;
  name: string;
  required: boolean;
}

export interface OutputDraft {
  /** Client-only key; outputs are stored by name. */
  id: string;
  name: string;
}

export interface WorkloadDraft {
  id: string;
  name: string;
  type: string;
  imageRepository: string;
  /** Kept as typed text so an invalid entry can be shown and corrected. */
  port: string;
  outputs: OutputDraft[];
  variables: ConfigRequirementDraft[];
  secrets: ConfigRequirementDraft[];
}

export interface ResourceDraft {
  id: string;
  name: string;
  type: string;
}

export interface DependencyDraft {
  id: string;
  sourceId: string;
  targetId: string;
}

export interface ApplicationDraft {
  applicationId: string | null;
  baseVersion: number | null;
  name: string;
  description: string;
  workloads: WorkloadDraft[];
  resources: ResourceDraft[];
  dependencies: DependencyDraft[];
}

export function newId(): string {
  return crypto.randomUUID();
}

export function emptyDraft(): ApplicationDraft {
  return { applicationId: null, baseVersion: null, name: '', description: '', workloads: [], resources: [], dependencies: [] };
}

export function newWorkload(): WorkloadDraft {
  return { id: newId(), name: '', type: '', imageRepository: '', port: '', outputs: [], variables: [], secrets: [] };
}

export function newResource(): ResourceDraft {
  return { id: newId(), name: '', type: '' };
}

export function draftFromDto(dto: ApplicationDefinitionDto): ApplicationDraft {
  return {
    applicationId: dto.applicationId ?? null,
    baseVersion: dto.baseVersion,
    name: dto.name,
    description: dto.description,
    workloads: dto.workloads.map((w) => ({
      id: w.id,
      name: w.name,
      type: w.type,
      imageRepository: w.imageRepository,
      port: w.port === null ? '' : String(w.port),
      outputs: w.outputs.map((name) => ({ id: newId(), name })),
      variables: w.variables.map((c) => ({ ...c })),
      secrets: w.secrets.map((c) => ({ ...c })),
    })),
    resources: dto.resources.map((r) => ({ ...r })),
    dependencies: dto.dependencies.map((d) => ({ ...d })),
  };
}

/** Builds the complete Save request. Call only after client validation passed. */
export function draftToDto(draft: ApplicationDraft): ApplicationDefinitionDto {
  const dto: ApplicationDefinitionDto = {
    baseVersion: draft.baseVersion,
    name: draft.name,
    description: draft.description,
    workloads: draft.workloads.map((w) => ({
      id: w.id,
      name: w.name,
      type: w.type,
      imageRepository: w.imageRepository,
      port: w.port.trim() === '' ? null : Number(w.port),
      outputs: w.outputs.map((o) => o.name),
      variables: w.variables.map(({ id, name, required }) => ({ id, name, required })),
      secrets: w.secrets.map(({ id, name, required }) => ({ id, name, required })),
    })),
    resources: draft.resources.map(({ id, name, type }) => ({ id, name, type })),
    dependencies: draft.dependencies.map(({ id, sourceId, targetId }) => ({ id, sourceId, targetId })),
  };
  if (draft.applicationId) dto.applicationId = draft.applicationId;
  return dto;
}

/** Name to show for a workload or resource ID in dependency lists. */
export function componentLabel(draft: ApplicationDraft, id: string): string {
  const w = draft.workloads.find((x) => x.id === id);
  if (w) return w.name || 'unnamed workload';
  const r = draft.resources.find((x) => x.id === id);
  if (r) return r.name || 'unnamed resource';
  return 'missing component';
}
