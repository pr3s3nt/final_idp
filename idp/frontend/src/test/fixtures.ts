import { vi, type Mock } from 'vitest';
import type { ApiResult, ApplicationApi } from '../features/application-definition/api/client';
import type { ApplicationDefinitionDto, ApplicationListItem } from '../features/application-definition/api/types';

export const APP_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
export const BACKEND_ID = '11111111-1111-4111-8111-111111111111';
export const DB_ID = '33333333-3333-4333-8333-333333333333';

export function shopDto(version = 2): ApplicationDefinitionDto {
  return {
    applicationId: APP_ID,
    baseVersion: version,
    name: 'shop-app',
    description: 'Demo shop',
    workloads: [
      {
        id: BACKEND_ID,
        name: 'backend',
        type: 'Backend Service',
        imageRepository: 'registry.company.local/shop-backend',
        port: 8080,
        outputs: ['endpoint'],
        variables: [{ id: '44444444-4444-4444-8444-444444444444', name: 'DB_HOST', required: true }],
        secrets: [{ id: '55555555-5555-4555-8555-555555555555', name: 'DB_PASSWORD', required: true }],
      },
    ],
    resources: [{ id: DB_ID, name: 'postgresql', type: 'PostgreSQL' }],
    dependencies: [{ id: 'dep-1', sourceId: BACKEND_ID, targetId: DB_ID }],
  };
}

export interface FakeApi extends ApplicationApi {
  listApplications: Mock<ApplicationApi['listApplications']>;
  loadApplication: Mock<ApplicationApi['loadApplication']>;
  saveApplication: Mock<ApplicationApi['saveApplication']>;
}

export function fakeApi(overrides: Partial<FakeApi> = {}): FakeApi {
  const api: FakeApi = {
    listApplications: vi.fn(async (): Promise<ApiResult<ApplicationListItem[]>> => ({
      kind: 'ok',
      data: [{ applicationId: APP_ID, name: 'shop-app', description: 'Demo shop', latestVersion: 2 }],
    })),
    loadApplication: vi.fn(async (): Promise<ApiResult<ApplicationDefinitionDto>> => ({ kind: 'ok', data: shopDto() })),
    saveApplication: vi.fn(async (draft: ApplicationDefinitionDto): Promise<ApiResult<ApplicationDefinitionDto>> => ({
      kind: 'ok',
      data: { ...draft, applicationId: draft.applicationId ?? APP_ID, baseVersion: (draft.baseVersion ?? 0) + 1 },
    })),
    ...overrides,
  };
  return api;
}
