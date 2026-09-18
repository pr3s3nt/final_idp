import type { ApplicationDraft, ConfigRequirementDraft, DependencyDraft, OutputDraft, ResourceDraft, WorkloadDraft } from './model';

// sessionStorage adapter for the UC-01 draft (ADR-016, ADR-017). The draft
// holds only non-sensitive definition data: names, types, repositories and
// relations. UC-01 has no Secret value field, so none can be stored.

export const DRAFT_SCHEMA_VERSION = 1;
const KEY_PREFIX = 'idp:uc01:application-draft:';

export interface StoredDraft {
  schemaVersion: typeof DRAFT_SCHEMA_VERSION;
  savedAt: string;
  draft: ApplicationDraft;
}

/** Key for an existing application, or for the one new-application draft of this tab. */
export function draftKey(applicationId: string | null): string {
  return KEY_PREFIX + (applicationId ?? 'new');
}

function storage(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

export function saveDraft(draft: ApplicationDraft): void {
  const value: StoredDraft = { schemaVersion: DRAFT_SCHEMA_VERSION, savedAt: new Date().toISOString(), draft };
  try {
    storage()?.setItem(draftKey(draft.applicationId), JSON.stringify(value));
  } catch {
    // Storage full or blocked: the draft stays in memory for this page.
  }
}

/**
 * Returns the stored draft for the key, or null. A value that does not parse,
 * uses another schema version, has the wrong shape or belongs to another
 * application is removed.
 */
export function loadDraft(applicationId: string | null): StoredDraft | null {
  const store = storage();
  const key = draftKey(applicationId);
  const raw = store?.getItem(key);
  if (raw == null) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    parsed = null;
  }
  if (!isStoredDraft(parsed) || parsed.draft.applicationId !== applicationId) {
    store?.removeItem(key);
    return null;
  }
  return parsed;
}

export function removeDraft(applicationId: string | null): void {
  try {
    storage()?.removeItem(draftKey(applicationId));
  } catch {
    // Nothing stored.
  }
}

type Guard = (v: unknown) => boolean;

const isString: Guard = (v) => typeof v === 'string';
const isObject = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v);
const arrayOf = (guard: Guard): Guard => (v) => Array.isArray(v) && v.every(guard);

/** Checks exact keys so extra fields (for example a value) are never restored. */
function shape(fields: Record<string, Guard>): Guard {
  return (v) =>
    isObject(v) &&
    Object.keys(v).length === Object.keys(fields).length &&
    Object.entries(fields).every(([k, guard]) => k in v && guard(v[k]));
}

const isConfig = shape({ id: isString, name: isString, required: (v) => typeof v === 'boolean' }) as (v: unknown) => v is ConfigRequirementDraft;
const isOutput = shape({ id: isString, name: isString }) as (v: unknown) => v is OutputDraft;
const isWorkload = shape({
  id: isString,
  name: isString,
  type: isString,
  imageRepository: isString,
  port: isString,
  outputs: arrayOf(isOutput),
  variables: arrayOf(isConfig),
  secrets: arrayOf(isConfig),
}) as (v: unknown) => v is WorkloadDraft;
const isResource = shape({ id: isString, name: isString, type: isString }) as (v: unknown) => v is ResourceDraft;
const isDependency = shape({ id: isString, sourceId: isString, targetId: isString }) as (v: unknown) => v is DependencyDraft;
const isDraft = shape({
  applicationId: (v) => v === null || typeof v === 'string',
  baseVersion: (v) => v === null || (typeof v === 'number' && Number.isInteger(v) && v > 0),
  name: isString,
  description: isString,
  workloads: arrayOf(isWorkload),
  resources: arrayOf(isResource),
  dependencies: arrayOf(isDependency),
});

function isStoredDraft(v: unknown): v is StoredDraft {
  return isObject(v) && v.schemaVersion === DRAFT_SCHEMA_VERSION && typeof v.savedAt === 'string' && isDraft(v.draft);
}
