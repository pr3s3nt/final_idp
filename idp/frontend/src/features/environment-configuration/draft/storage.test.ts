import { draftFromRequirements } from './model';
import { DRAFT_SCHEMA_VERSION, draftKey, loadDraft, removeDraft, saveDraft } from './storage';
import { requirementsDto, BACKEND_ID, SECRET_ID, APP_ID } from '../../../test/configuration-fixtures';

function draft() {
  return draftFromRequirements(requirementsDto(), '1', 'kind-local');
}

describe('UC-02 draft sessionStorage adapter', () => {
  test('uses a namespaced key per application and environment', () => {
    expect(draftKey(APP_ID, 'STAGING')).toBe(`idp:uc02:configuration-draft:${APP_ID}:STAGING`);
    expect(draftKey(APP_ID, 'production')).toBe(`idp:uc02:configuration-draft:${APP_ID}:PRODUCTION`);
  });

  test('saves and restores a draft with its schema version', () => {
    const d = draft();
    saveDraft(d);
    const raw = JSON.parse(sessionStorage.getItem(draftKey(APP_ID, 'STAGING')) ?? '{}');
    expect(raw.schemaVersion).toBe(DRAFT_SCHEMA_VERSION);
    expect(loadDraft(APP_ID, 'STAGING')?.draft).toEqual(d);
    expect(loadDraft(APP_ID, 'PRODUCTION')).toBeNull();
  });

  test('never writes a typed Secret value', () => {
    const d = draft();
    const secret = d.secrets.find((b) => b.definitionId === SECRET_ID);
    expect(secret).toBeDefined();
    secret!.source = 'SECRET_REF';
    secret!.value = 'hunter2';
    saveDraft(d);

    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).not.toContain('hunter2');
    expect(loadDraft(APP_ID, 'STAGING')?.draft.secrets.find((b) => b.definitionId === SECRET_ID)?.value).toBe('');
  });

  test('keeps the opaque reference of a staged Secret', () => {
    const d = draft();
    const secret = d.secrets.find((b) => b.definitionId === SECRET_ID)!;
    secret.source = 'SECRET_REF';
    secret.secretRef = 'idpsecret://shop/staging/db';
    saveDraft(d);

    expect(loadDraft(APP_ID, 'STAGING')?.draft.secrets.find((b) => b.definitionId === SECRET_ID)?.secretRef).toBe(
      'idpsecret://shop/staging/db',
    );
  });

  test('drops a Secret value that another tab wrote into storage anyway', () => {
    const d = draft();
    const secret = d.secrets.find((b) => b.definitionId === SECRET_ID)!;
    secret.source = 'SECRET_REF';
    secret.value = 'hunter2';
    sessionStorage.setItem(
      draftKey(APP_ID, 'STAGING'),
      JSON.stringify({ schemaVersion: DRAFT_SCHEMA_VERSION, savedAt: 'now', draft: d }),
    );

    expect(loadDraft(APP_ID, 'STAGING')?.draft.secrets.find((b) => b.definitionId === SECRET_ID)?.value).toBe('');
  });

  test('removes a draft', () => {
    saveDraft(draft());
    removeDraft(APP_ID, 'STAGING');
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).toBeNull();
  });

  test.each([
    ['malformed JSON', '{not json'],
    ['another schema version', JSON.stringify({ schemaVersion: 99, savedAt: 'x', draft: draft() })],
    ['a wrong shape', JSON.stringify({ schemaVersion: 1, savedAt: 'x', draft: { applicationId: 3 } })],
    [
      'another environment',
      JSON.stringify({ schemaVersion: 1, savedAt: 'x', draft: { ...draft(), environment: 'PRODUCTION' } }),
    ],
  ])('discards %s', (_name, raw) => {
    sessionStorage.setItem(draftKey(APP_ID, 'STAGING'), raw);
    expect(loadDraft(APP_ID, 'STAGING')).toBeNull();
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).toBeNull();
  });

  test('a binding with an extra field is not restored', () => {
    const d = draft();
    const broken = { ...d, variables: [{ ...d.variables[0], leaked: 'hunter2' }] };
    sessionStorage.setItem(
      draftKey(APP_ID, 'STAGING'),
      JSON.stringify({ schemaVersion: DRAFT_SCHEMA_VERSION, savedAt: 'x', draft: broken }),
    );
    expect(loadDraft(APP_ID, 'STAGING')).toBeNull();
  });

  test('BACKEND_ID owns the restored bindings', () => {
    saveDraft(draft());
    expect(loadDraft(APP_ID, 'STAGING')?.draft.variables[0]?.workloadId).toBe(BACKEND_ID);
  });
});
