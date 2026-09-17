import type { ApplicationDefinitionDto, ApplicationListItem, Problem } from './types';

export type ApiResult<T> =
  | { kind: 'ok'; data: T }
  | { kind: 'validation'; problems: Problem[] }
  | { kind: 'conflict'; problems: Problem[] }
  | { kind: 'not-found'; problems: Problem[] }
  | { kind: 'error'; problems: Problem[] };

/** The only backend calls UC-01 makes: list, load for edit, and Save. */
export interface ApplicationApi {
  listApplications(): Promise<ApiResult<ApplicationListItem[]>>;
  loadApplication(applicationId: string): Promise<ApiResult<ApplicationDefinitionDto>>;
  saveApplication(draft: ApplicationDefinitionDto): Promise<ApiResult<ApplicationDefinitionDto>>;
}

const base = '/api/application-definitions';

async function call<T>(input: string, init?: RequestInit): Promise<ApiResult<T>> {
  let response: Response;
  try {
    response = await fetch(input, { ...init, headers: { Accept: 'application/json', ...init?.headers } });
  } catch {
    return { kind: 'error', problems: [{ code: 'NETWORK_ERROR', message: 'The IDP backend could not be reached.' }] };
  }
  const body: unknown = await response.json().catch(() => null);
  if (response.ok) {
    return { kind: 'ok', data: body as T };
  }
  const problems = problemsOf(body, response.status);
  switch (response.status) {
    case 404:
      return { kind: 'not-found', problems };
    case 409:
      return { kind: 'conflict', problems };
    case 422:
      return { kind: 'validation', problems };
    default:
      return { kind: 'error', problems };
  }
}

function problemsOf(body: unknown, status: number): Problem[] {
  if (body && typeof body === 'object' && Array.isArray((body as { problems?: unknown }).problems)) {
    return (body as { problems: Problem[] }).problems;
  }
  return [{ code: 'HTTP_' + status, message: `The request failed with HTTP status ${status}.` }];
}

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
    return call<ApplicationDefinitionDto>(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(draft),
    });
  },
};
