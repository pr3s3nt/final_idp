import { vi, type Mock } from 'vitest';
import type { ConfigurationApi } from '../features/environment-configuration/api/client';
import type { ApiResult } from '../shared/api/result';
import type { ConfigurationRequirementsDto } from '../features/environment-configuration/api/types';

export const APP_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
export const BACKEND_ID = '11111111-1111-4111-8111-111111111111';
export const FRONTEND_ID = '22222222-2222-4222-8222-222222222222';
export const DB_ID = '33333333-3333-4333-8333-333333333333';
export const DB_HOST_ID = '44444444-4444-4444-8444-444444444444';
export const SECRET_ID = '55555555-5555-4555-8555-555555555555';
export const BACKEND_URL_ID = '66666666-6666-4666-8666-666666666666';

/** frontend depends on backend; backend depends on postgresql. */
export function requirementsDto(overrides: Partial<ConfigurationRequirementsDto> = {}): ConfigurationRequirementsDto {
  return {
    applicationId: APP_ID,
    applicationName: 'shop-app',
    environment: 'STAGING',
    baseApplicationDefinitionVersion: 2,
    baseConfigurationRevision: 'rev-1',
    workloads: [
      {
        id: BACKEND_ID,
        name: 'backend',
        variables: [{ id: DB_HOST_ID, name: 'DB_HOST', required: true }],
        secrets: [{ id: SECRET_ID, name: 'DB_PASSWORD', required: true }],
        dependsOnResources: [DB_ID],
        dependsOnWorkloads: [],
      },
      {
        id: FRONTEND_ID,
        name: 'frontend',
        variables: [{ id: BACKEND_URL_ID, name: 'BACKEND_URL', required: true }],
        secrets: [],
        dependsOnResources: [],
        dependsOnWorkloads: [BACKEND_ID],
      },
    ],
    resources: [{ id: DB_ID, name: 'postgresql', type: 'PostgreSQL' }],
    catalogVersions: [
      { id: 'cat-2', number: 2, createdAt: '2026-09-01T00:00:00Z' },
      { id: 'cat-1', number: 1, createdAt: '2026-08-01T00:00:00Z' },
    ],
    targets: [
      { target: 'kind-local', cloudProvider: 'local' },
      { target: 'aws', cloudProvider: 'aws', region: 'ap-southeast-1' },
    ],
    configuration: null,
    ...overrides,
  };
}

export interface FakeConfigurationApi extends ConfigurationApi {
  selectEnvironment: Mock<ConfigurationApi['selectEnvironment']>;
  resourceOutputs: Mock<ConfigurationApi['resourceOutputs']>;
  workloadOutputs: Mock<ConfigurationApi['workloadOutputs']>;
  stageSecret: Mock<ConfigurationApi['stageSecret']>;
  saveConfiguration: Mock<ConfigurationApi['saveConfiguration']>;
}

export function fakeConfigurationApi(overrides: Partial<FakeConfigurationApi> = {}): FakeConfigurationApi {
  const api: FakeConfigurationApi = {
    selectEnvironment: vi.fn(async (): Promise<ApiResult<ConfigurationRequirementsDto>> => ({ kind: 'ok', data: requirementsDto() })),
    resourceOutputs: vi.fn(async () => ({
      kind: 'ok' as const,
      data: {
        resourceId: DB_ID,
        resourceName: 'postgresql',
        definitionName: 'postgres-k8s',
        outputs: [
          { name: 'host', sensitive: false },
          { name: 'port', sensitive: false },
          { name: 'password', sensitive: true },
        ],
      },
    })),
    workloadOutputs: vi.fn(async () => ({
      kind: 'ok' as const,
      data: { workloadId: BACKEND_ID, workloadName: 'backend', outputs: [{ name: 'endpoint', sensitive: false }] },
    })),
    stageSecret: vi.fn(async () => ({ kind: 'ok' as const, data: { secretRef: 'idpsecret://shop/staging/db' } })),
    saveConfiguration: vi.fn(async (draft) => ({ kind: 'ok' as const, data: { ...draft, baseConfigurationRevision: 'rev-2' } })),
    ...overrides,
  };
  return api;
}
