import type { BindingDraft, ConfigurationDraft, Environment } from './model';

// sessionStorage adapter for the UC-02 draft (ADR-016). Only non-sensitive
// draft state is stored: a typed Secret value is dropped before writing, so a
// plaintext Secret can never be restored. A staged Secret keeps its opaque
// reference, which is not the value.

export const DRAFT_SCHEMA_VERSION = 1;
const KEY_PREFIX = 'idp:uc02:configuration-draft:';

export interface StoredDraft {
  schemaVersion: typeof DRAFT_SCHEMA_VERSION;
  savedAt: string;
  draft: ConfigurationDraft;
}

/** Key of one application + environment draft in this tab. */
export function draftKey(applicationId: string, environment: string): string {
  return `${KEY_PREFIX}${applicationId}:${environment.toUpperCase()}`;
}

function storage(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

/** Removes the typed Secret value; everything else may be restored. */
export function withoutSecretValues(draft: ConfigurationDraft): ConfigurationDraft {
  return { ...draft, secrets: draft.secrets.map((b) => ({ ...b, value: '' })) };
}

export function saveDraft(draft: ConfigurationDraft): void {
  const value: StoredDraft = { schemaVersion: DRAFT_SCHEMA_VERSION, savedAt: new Date().toISOString(), draft: withoutSecretValues(draft) };
  try {
    storage()?.setItem(draftKey(draft.applicationId, draft.environment), JSON.stringify(value));
  } catch {
    // Storage full or blocked: the draft stays in memory for this page.
  }
}

/**
 * Returns the stored draft for the key, or null. A value that does not parse,
 * uses another schema version, has the wrong shape or belongs to another
 * application or environment is removed.
 */
export function loadDraft(applicationId: string, environment: string): StoredDraft | null {
  const store = storage();
  const key = draftKey(applicationId, environment);
  const raw = store?.getItem(key);
  if (raw == null) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    parsed = null;
  }
  if (
    !isStoredDraft(parsed) ||
    parsed.draft.applicationId !== applicationId ||
    parsed.draft.environment !== environment.toUpperCase()
  ) {
    store?.removeItem(key);
    return null;
  }
  // A restored draft never carries a plaintext Secret, whatever was stored.
  return { ...parsed, draft: withoutSecretValues(parsed.draft) };
}

export function removeDraft(applicationId: string, environment: string): void {
  try {
    storage()?.removeItem(draftKey(applicationId, environment));
  } catch {
    // Nothing stored.
  }
}

type Guard = (v: unknown) => boolean;

const isString: Guard = (v) => typeof v === 'string';
const isObject = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v);
const arrayOf = (guard: Guard): Guard => (v) => Array.isArray(v) && v.every(guard);

/** Checks exact keys so an extra field is never restored. */
function shape(fields: Record<string, Guard>): Guard {
  return (v) =>
    isObject(v) &&
    Object.keys(v).length === Object.keys(fields).length &&
    Object.entries(fields).every(([k, guard]) => k in v && guard(v[k]));
}

const isBinding = shape({
  workloadId: isString,
  definitionId: isString,
  name: isString,
  required: (v) => typeof v === 'boolean',
  source: isString,
  value: isString,
  secretRef: isString,
  refId: isString,
  outputName: isString,
}) as (v: unknown) => v is BindingDraft;

const isDraft = shape({
  applicationId: isString,
  environment: (v) => v === 'STAGING' || v === 'PRODUCTION',
  baseApplicationDefinitionVersion: (v) => typeof v === 'number' && Number.isInteger(v),
  baseConfigurationRevision: isString,
  catalogVersion: isString,
  deploymentTarget: isString,
  variables: arrayOf(isBinding),
  secrets: arrayOf(isBinding),
}) as (v: unknown) => v is ConfigurationDraft & { environment: Environment };

function isStoredDraft(v: unknown): v is StoredDraft {
  return isObject(v) && v.schemaVersion === DRAFT_SCHEMA_VERSION && typeof v.savedAt === 'string' && isDraft(v.draft);
}
