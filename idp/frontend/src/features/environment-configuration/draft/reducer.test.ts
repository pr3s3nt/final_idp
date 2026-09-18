import { bindingKey, draftFromRequirements, draftToDto } from './model';
import { validateDraft } from './validation';
import { draftReducer } from './reducer';
import { requirementsDto, BACKEND_ID, DB_HOST_ID, DB_ID, SECRET_ID } from '../../../test/configuration-fixtures';

const key = bindingKey(BACKEND_ID, DB_HOST_ID);
const secretKey = bindingKey(BACKEND_ID, SECRET_ID);

function draft() {
  return draftFromRequirements(requirementsDto(), '2', 'kind-local');
}

describe('UC-02 draft', () => {
  test('lists every declared variable and secret with no source yet', () => {
    const d = draft();
    expect(d.variables.map((b) => b.name)).toEqual(['DB_HOST', 'BACKEND_URL']);
    expect(d.secrets.map((b) => b.name)).toEqual(['DB_PASSWORD']);
    expect(validateDraft(d).filter((problem) => problem.code === 'MISSING_REQUIRED_CONFIGURATION')).toHaveLength(3);
  });

  test('prefills the values already stored for the environment', () => {
    const dto = requirementsDto({
      configuration: {
        applicationId: 'a',
        environment: 'STAGING',
        baseApplicationDefinitionVersion: 2,
        baseConfigurationRevision: 'rev-1',
        catalogVersion: '',
        deploymentTarget: '',
        variables: [{ workloadId: BACKEND_ID, definitionId: DB_HOST_ID, name: 'DB_HOST', source: 'RESOURCE_OUTPUT', refId: DB_ID, outputName: 'host' }],
        secrets: [],
      },
    });
    const d = draftFromRequirements(dto, '2', 'kind-local');
    const binding = d.variables.find((b) => b.definitionId === DB_HOST_ID);
    expect(binding).toMatchObject({ source: 'RESOURCE_OUTPUT', refId: DB_ID, outputName: 'host' });
  });

  test('a direct value is kept in the draft only', () => {
    const d = draftReducer(draftReducer(draft(), { type: 'source', kind: 'variable', key, source: 'DIRECT' }), {
      type: 'direct-value',
      kind: 'variable',
      key,
      value: 'INFO',
    });
    expect(draftToDto(d).variables).toEqual([{ workloadId: BACKEND_ID, definitionId: DB_HOST_ID, name: 'DB_HOST', source: 'DIRECT', value: 'INFO' }]);
  });

  test('switching the source drops the payload of the previous one', () => {
    let d = draftReducer(draft(), { type: 'source', kind: 'variable', key, source: 'RESOURCE_OUTPUT' });
    d = draftReducer(d, { type: 'reference', kind: 'variable', key, refId: DB_ID, outputName: 'host' });
    d = draftReducer(d, { type: 'source', kind: 'variable', key, source: 'DIRECT' });
    expect(d.variables.find((b) => b.definitionId === DB_HOST_ID)).toMatchObject({ refId: '', outputName: '', value: '' });
  });

  test('changing the catalog version or the target clears resource output bindings', () => {
    let d = draftReducer(draft(), { type: 'source', kind: 'variable', key, source: 'RESOURCE_OUTPUT' });
    d = draftReducer(d, { type: 'reference', kind: 'variable', key, refId: DB_ID, outputName: 'host' });

    const afterVersion = draftReducer(d, { type: 'catalog-version', value: '1' });
    expect(afterVersion.catalogVersion).toBe('1');
    expect(afterVersion.variables.find((b) => b.definitionId === DB_HOST_ID)).toMatchObject({ refId: '', outputName: '' });

    const afterTarget = draftReducer(d, { type: 'deployment-target', value: 'aws' });
    expect(afterTarget.deploymentTarget).toBe('aws');
    expect(afterTarget.variables.find((b) => b.definitionId === DB_HOST_ID)).toMatchObject({ refId: '', outputName: '' });
  });

  test('a typed Secret counts as pending until it is staged', () => {
    let d = draftReducer(draft(), { type: 'source', kind: 'secret', key: secretKey, source: 'SECRET_REF' });
    d = draftReducer(d, { type: 'direct-value', kind: 'secret', key: secretKey, value: 'hunter2' });
    expect(validateDraft(d).filter((problem) => problem.code === 'SECRET_NOT_STORED')).toHaveLength(1);

    d = draftReducer(d, { type: 'secret-reference', key: secretKey, secretRef: 'idpsecret://x' });
    expect(validateDraft(d).filter((problem) => problem.code === 'SECRET_NOT_STORED')).toHaveLength(0);
    expect(d.secrets.find((b) => b.definitionId === SECRET_ID)?.value).toBe('');
  });

  test('the Save request carries only the reference of a Secret', () => {
    let d = draftReducer(draft(), { type: 'source', kind: 'secret', key: secretKey, source: 'SECRET_REF' });
    d = draftReducer(d, { type: 'direct-value', kind: 'secret', key: secretKey, value: 'hunter2' });
    d = draftReducer(d, { type: 'secret-reference', key: secretKey, secretRef: 'idpsecret://x' });

    const dto = draftToDto(d);
    expect(dto.secrets).toEqual([{ workloadId: BACKEND_ID, definitionId: SECRET_ID, name: 'DB_PASSWORD', source: 'SECRET_REF', secretRef: 'idpsecret://x' }]);
    expect(JSON.stringify(dto)).not.toContain('hunter2');
  });

  test('the Save request carries both concurrency bases and the catalog selection', () => {
    const dto = draftToDto(draft());
    expect(dto).toMatchObject({ baseApplicationDefinitionVersion: 2, baseConfigurationRevision: 'rev-1', catalogVersion: '2', deploymentTarget: 'kind-local' });
  });

  test('a binding without a source is left out of the Save request', () => {
    expect(draftToDto(draft()).variables).toEqual([]);
  });
});
