import { afterEach, describe, expect, test, vi } from 'vitest';
import { apiFetch, sessionCSRFToken } from './http';

afterEach(() => {
  vi.unstubAllGlobals();
  document.cookie = 'idp_csrf=; Max-Age=0; Path=/';
});

describe('shared API transport', () => {
  test('attaches the session CSRF cookie to unsafe same-origin requests', async () => {
    document.cookie = 'idp_csrf=csrf%20value; Path=/';
    const fetch = vi.fn(async (_input: string, _init?: RequestInit) => new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetch);

    await apiFetch('/api/application-definitions', { method: 'POST', body: '{}' });

    const [input, init = {}] = fetch.mock.calls[0]!;
    expect(input).toBe('/api/application-definitions');
    expect(init.credentials).toBe('same-origin');
    expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('csrf value');
    expect(sessionCSRFToken()).toBe('csrf value');
  });

  test('does not add a CSRF header to safe requests', async () => {
    document.cookie = 'idp_csrf=csrf-value; Path=/';
    const fetch = vi.fn(async (_input: string, _init?: RequestInit) => new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetch);

    await apiFetch('/api/application-definitions');

    const [, init = {}] = fetch.mock.calls[0]!;
    expect(new Headers(init.headers).has('X-CSRF-Token')).toBe(false);
  });
});
