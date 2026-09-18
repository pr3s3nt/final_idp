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

/**
 * Stable client ID of a draft item, and the stable component ID the backend
 * keeps for a new component. `crypto.randomUUID` exists only in a secure
 * context, so a page served over plain HTTP from an IP address falls back to
 * building the UUID from random bytes.
 */
export function newId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID();
  const bytes = new Uint8Array(16);
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') crypto.getRandomValues(bytes);
  else for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40; // version 4
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80; // variant 10
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
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
