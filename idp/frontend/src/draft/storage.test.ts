import { emptyDraft, newWorkload } from './model';
import { DRAFT_SCHEMA_VERSION, draftKey, loadDraft, removeDraft, saveDraft } from './storage';

const APP = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';

describe('draft sessionStorage adapter', () => {
  test('uses a namespaced key per application', () => {
    expect(draftKey(null)).toBe('idp:uc01:application-draft:new');
    expect(draftKey(APP)).toBe(`idp:uc01:application-draft:${APP}`);
  });

  test('saves and restores a draft with its schema version', () => {
    const draft = { ...emptyDraft(), applicationId: APP, baseVersion: 3, name: 'shop-app', workloads: [newWorkload()] };
    saveDraft(draft);
    const raw = JSON.parse(sessionStorage.getItem(draftKey(APP)) ?? '{}');
    expect(raw.schemaVersion).toBe(DRAFT_SCHEMA_VERSION);
    expect(loadDraft(APP)?.draft).toEqual(draft);
    expect(loadDraft(null)).toBeNull();
  });

  test('removes a draft', () => {
    saveDraft(emptyDraft());
    removeDraft(null);
    expect(sessionStorage.getItem(draftKey(null))).toBeNull();
  });

  test.each([
    ['malformed JSON', '{not json'],
    ['another schema version', JSON.stringify({ schemaVersion: 99, savedAt: 'x', draft: emptyDraft() })],
    ['a wrong shape', JSON.stringify({ schemaVersion: 1, savedAt: 'x', draft: { name: 3 } })],
    ['an unexpected field', JSON.stringify({ schemaVersion: 1, savedAt: 'x', draft: { ...emptyDraft(), workloads: [{ ...newWorkload(), secrets: [{ id: 's', name: 'DB_PASSWORD', required: true, value: 'hunter2' }] }] } })],
    ['a draft of another application', JSON.stringify({ schemaVersion: 1, savedAt: 'x', draft: { ...emptyDraft(), applicationId: 'other' } })],
  ])('discards %s', (_, raw) => {
    sessionStorage.setItem(draftKey(null), raw);
    expect(loadDraft(null)).toBeNull();
    expect(sessionStorage.getItem(draftKey(null))).toBeNull();
  });
});
