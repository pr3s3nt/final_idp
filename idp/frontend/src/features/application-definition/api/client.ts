import type { ApplicationDefinitionDto, ApplicationListItem } from './types';
import { call, jsonRequest, type ApiResult } from '../../../shared/api/result';

export type { ApiResult };

/** The only backend calls UC-01 makes: list, load for edit, and Save. */
export interface ApplicationApi {
  listApplications(): Promise<ApiResult<ApplicationListItem[]>>;
  loadApplication(applicationId: string): Promise<ApiResult<ApplicationDefinitionDto>>;
  saveApplication(draft: ApplicationDefinitionDto): Promise<ApiResult<ApplicationDefinitionDto>>;
}

const base = '/api/application-definitions';

export const httpApplicationApi: ApplicationApi = {
  async listApplications() {
    const result = await call<{ applications: ApplicationListItem[] }>(base);
    return result.kind === 'ok' ? { kind: 'ok', data: result.data.applications } : result;
  },
  loadApplication(applicationId) {
    return call<ApplicationDefinitionDto>(`${base}/${encodeURIComponent(applicationId)}`);
  },
  saveApplication(draft) {
    const url = draft.applicationId ? `${base}/${encodeURIComponent(draft.applicationId)}/versions` : base;
    return call<ApplicationDefinitionDto>(url, jsonRequest('POST', draft));
  },
};
