import { draftFromDto, newId, type ApplicationDraft } from './model';
import { validateDraft } from './validation';
import { BACKEND_ID, DB_ID, shopDto } from '../../../test/fixtures';

function shop(): ApplicationDraft {
  return draftFromDto(shopDto());
}

function codesAt(draft: ApplicationDraft) {
  return validateDraft(draft).map((p) => `${p.code}@${p.field}`);
}

test('accepts the specification example', () => {
  expect(validateDraft(shop())).toEqual([]);
});

test.each<[string, (d: ApplicationDraft) => void, string]>([
  ['missing application name', (d) => (d.name = ''), 'MISSING_REQUIRED_FIELD@name'],
  ['invalid application name', (d) => (d.name = 'Shop App'), 'INVALID_NAME@name'],
  ['no workload', (d) => ((d.workloads = []), (d.dependencies = [])), 'NO_WORKLOAD@workloads'],
  ['invalid workload name', (d) => (d.workloads[0]!.name = 'api.v2'), `INVALID_NAME@workloads.${BACKEND_ID}.name`],
  ['workload and resource share a name', (d) => (d.resources[0]!.name = 'backend'), `DUPLICATE_NAME@resources.${DB_ID}.name`],
  ['image tag', (d) => (d.workloads[0]!.imageRepository += ':v1.4.2'), `INVALID_IMAGE_REPOSITORY@workloads.${BACKEND_ID}.imageRepository`],
  ['port out of range', (d) => (d.workloads[0]!.port = '70000'), `INVALID_PORT@workloads.${BACKEND_ID}.port`],
  ['port not a number', (d) => (d.workloads[0]!.port = '80a'), `INVALID_PORT@workloads.${BACKEND_ID}.port`],
  ['invalid variable name', (d) => (d.workloads[0]!.variables[0]!.name = 'DB-HOST'), `INVALID_NAME@workloads.${BACKEND_ID}.variables.44444444-4444-4444-8444-444444444444.name`],
  ['variable and secret share a name', (d) => (d.workloads[0]!.secrets[0]!.name = 'DB_HOST'), `DUPLICATE_NAME@workloads.${BACKEND_ID}.secrets.55555555-5555-4555-8555-555555555555.name`],
  ['missing dependency target', (d) => (d.dependencies[0]!.targetId = newId()), 'DEPENDENCY_UNRESOLVED@dependencies.dep-1'],
  ['resource as dependency source', (d) => (d.dependencies[0] = { id: 'dep-1', sourceId: DB_ID, targetId: BACKEND_ID }), 'INVALID_DEPENDENCY@dependencies.dep-1'],
  ['duplicate dependency', (d) => d.dependencies.push({ id: 'dep-2', sourceId: BACKEND_ID, targetId: DB_ID }), 'DUPLICATE_DEPENDENCY@dependencies.dep-2'],
  ['self dependency', (d) => (d.dependencies[0]!.targetId = BACKEND_ID), 'DEPENDENCY_CYCLE@dependencies.dep-1'],
])('reports %s', (_, mutate, expected) => {
  const d = shop();
  mutate(d);
  expect(codesAt(d)).toContain(expected);
});

test('reports a dependency cycle with workload names', () => {
  const d = shop();
  d.workloads.push({ ...d.workloads[0]!, id: newId(), name: 'frontend', variables: [], secrets: [], outputs: [] });
  const frontend = d.workloads[1]!.id;
  d.dependencies.push({ id: 'a', sourceId: frontend, targetId: BACKEND_ID }, { id: 'b', sourceId: BACKEND_ID, targetId: frontend });
  const cycle = validateDraft(d).find((p) => p.code === 'DEPENDENCY_CYCLE');
  expect(cycle?.field).toBe('dependencies');
  expect(cycle?.message).toMatch(/(frontend → backend → frontend|backend → frontend → backend)/);
});
