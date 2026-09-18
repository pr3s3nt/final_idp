import type { Problem } from '../../features/application-definition/api/types';
import { apiFetch } from './http';

// Shared result shape of the JSON APIs. A call never throws: a transport error
// and an HTTP error both become a result the page can render.

export type ApiResult<T> =
  | { kind: 'ok'; data: T }
  | { kind: 'validation'; problems: Problem[] }
  | { kind: 'conflict'; problems: Problem[] }
  | { kind: 'not-found'; problems: Problem[] }
  | { kind: 'error'; problems: Problem[] };

export async function call<T>(input: string, init?: RequestInit): Promise<ApiResult<T>> {
  let response: Response;
  try {
    response = await apiFetch(input, init);
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

export function problemsOf(body: unknown, status: number): Problem[] {
  if (body && typeof body === 'object' && Array.isArray((body as { problems?: unknown }).problems)) {
    return (body as { problems: Problem[] }).problems;
  }
  return [{ code: 'HTTP_' + status, message: `The request failed with HTTP status ${status}.` }];
}

/** Sends a JSON body with the method the API expects. */
export function jsonRequest(method: 'POST' | 'PUT', body: unknown): RequestInit {
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) };
}
