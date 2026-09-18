import { emptyDraft } from './model';
import { draftReducer, type DraftAction } from './reducer';

function apply(...actions: DraftAction[]) {
  return actions.reduce(draftReducer, emptyDraft());
}

test('adds, edits and removes workloads, resources, configuration and dependencies', () => {
  let d = apply({ type: 'addWorkload' }, { type: 'addWorkload' }, { type: 'addResource' });
  const [web, api] = d.workloads;
  const db = d.resources[0];
  if (!web || !api || !db) throw new Error('missing items');
  expect(new Set([web.id, api.id, db.id]).size).toBe(3);

  d = [
    { type: 'updateWorkload', id: api.id, field: 'name', value: 'api' },
    { type: 'updateWorkload', id: api.id, field: 'port', value: '8080' },
    { type: 'addOutput', workloadId: api.id },
    { type: 'updateResource', id: db.id, field: 'type', value: 'PostgreSQL' },
    { type: 'addConfig', workloadId: api.id, group: 'variables' },
    { type: 'addConfig', workloadId: api.id, group: 'secrets' },
    { type: 'addDependency', sourceId: web.id, targetId: api.id },
    { type: 'addDependency', sourceId: api.id, targetId: db.id },
  ].reduce<ReturnType<typeof emptyDraft>>((acc, a) => draftReducer(acc, a as DraftAction), d);

  const apiNow = d.workloads[1]!;
  expect(apiNow).toMatchObject({ name: 'api', port: '8080' });
  expect(apiNow.outputs).toHaveLength(1);
  expect(d.resources[0]!.type).toBe('PostgreSQL');
  const secret = apiNow.secrets[0]!;
  d = draftReducer(d, { type: 'updateConfig', workloadId: api.id, group: 'secrets', id: secret.id, name: 'DB_PASSWORD', required: false });
  expect(d.workloads[1]!.secrets[0]).toEqual({ id: secret.id, name: 'DB_PASSWORD', required: false });
  d = draftReducer(d, { type: 'removeConfig', workloadId: api.id, group: 'variables', id: apiNow.variables[0]!.id });
  expect(d.workloads[1]!.variables).toHaveLength(0);

  const dep = d.dependencies[0]!;
  d = draftReducer(d, { type: 'updateDependency', id: dep.id, targetId: db.id });
  expect(d.dependencies[0]!.targetId).toBe(db.id);

  // Removing a component also removes its dependencies.
  d = draftReducer(d, { type: 'removeResource', id: db.id });
  expect(d.resources).toHaveLength(0);
  expect(d.dependencies).toHaveLength(0);
  d = draftReducer(d, { type: 'addDependency', sourceId: web.id, targetId: api.id });
  d = draftReducer(d, { type: 'removeWorkload', id: api.id });
  expect(d.workloads.map((w) => w.id)).toEqual([web.id]);
  expect(d.dependencies).toHaveLength(0);
});
