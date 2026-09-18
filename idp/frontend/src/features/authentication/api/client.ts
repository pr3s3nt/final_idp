import { apiFetch } from '../../../shared/api/http';

export interface Problem {
  code: string;
  message: string;
}

export interface LoginContext {
  csrfToken?: string;
  returnTo?: string;
  redirectTo?: string;
}

export type AuthResult<T> =
  | { kind: 'ok'; data: T }
  | { kind: 'invalid'; problems: Problem[] }
  | { kind: 'rate-limited'; problems: Problem[]; retryAfter: number | null }
  | { kind: 'form-expired'; problems: Problem[] }
  | { kind: 'error'; problems: Problem[] };

export interface AuthenticationApi {
  loadLoginContext(returnTo: string): Promise<AuthResult<LoginContext>>;
  signIn(username: string, password: string, returnTo: string, csrfToken: string): Promise<AuthResult<{ redirectTo: string }>>;
  signOut(): Promise<AuthResult<null>>;
}

async function problems(response: Response): Promise<Problem[]> {
  const body: unknown = await response.json().catch(() => null);
  if (body && typeof body === 'object' && Array.isArray((body as { problems?: unknown }).problems)) {
    return (body as { problems: Problem[] }).problems;
  }
  return [{ code: `HTTP_${response.status}`, message: `The request failed with HTTP status ${response.status}.` }];
}

export const httpAuthenticationApi: AuthenticationApi = {
  async loadLoginContext(returnTo) {
    try {
      const query = returnTo ? `?return_to=${encodeURIComponent(returnTo)}` : '';
      const response = await apiFetch(`/api/auth/login-context${query}`, {}, { redirectOnUnauthorized: false });
      if (response.ok) return { kind: 'ok', data: (await response.json()) as LoginContext };
      return { kind: 'error', problems: await problems(response) };
    } catch {
      return { kind: 'error', problems: [{ code: 'NETWORK_ERROR', message: 'The IDP backend could not be reached.' }] };
    }
  },

  async signIn(username, password, returnTo, csrfToken) {
    try {
      const response = await apiFetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password, returnTo }),
      }, { redirectOnUnauthorized: false, csrfToken });
      if (response.ok) return { kind: 'ok', data: (await response.json()) as { redirectTo: string } };
      const body = await problems(response);
      if (response.status === 401) return { kind: 'invalid', problems: body };
      if (response.status === 403) return { kind: 'form-expired', problems: body };
      if (response.status === 429) {
        const raw = response.headers.get('Retry-After');
        return { kind: 'rate-limited', problems: body, retryAfter: raw && /^\d+$/.test(raw) ? Number(raw) : null };
      }
      return { kind: 'error', problems: body };
    } catch {
      return { kind: 'error', problems: [{ code: 'NETWORK_ERROR', message: 'The IDP backend could not be reached.' }] };
    }
  },

  async signOut() {
    try {
      const response = await apiFetch('/api/auth/logout', { method: 'POST' }, { redirectOnUnauthorized: false });
      if (response.ok) return { kind: 'ok', data: null };
      if (response.status === 401 || response.status === 403) return { kind: 'form-expired', problems: await problems(response) };
      return { kind: 'error', problems: await problems(response) };
    } catch {
      return { kind: 'error', problems: [{ code: 'NETWORK_ERROR', message: 'The IDP backend could not be reached.' }] };
    }
  },
};
