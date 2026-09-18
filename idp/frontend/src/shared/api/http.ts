const unsafeMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

export function cookieValue(name: string): string {
  const prefix = `${name}=`;
  for (const part of document.cookie.split(';')) {
    const value = part.trim();
    if (value.startsWith(prefix)) return decodeURIComponent(value.slice(prefix.length));
  }
  return '';
}

export function sessionCSRFToken(): string {
  return cookieValue('__Host-idp_csrf') || cookieValue('idp_csrf');
}

interface Options {
  redirectOnUnauthorized?: boolean;
  csrfToken?: string;
}

export async function apiFetch(input: string, init: RequestInit = {}, options: Options = {}): Promise<Response> {
  const method = (init.method ?? 'GET').toUpperCase();
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (unsafeMethods.has(method) && !headers.has('X-CSRF-Token')) {
    const token = options.csrfToken ?? sessionCSRFToken();
    if (token) headers.set('X-CSRF-Token', token);
  }
  const response = await fetch(input, { ...init, headers, credentials: 'same-origin' });
  if (response.status === 401 && options.redirectOnUnauthorized !== false && window.location.pathname !== '/ui/login') {
    const returnTo = window.location.pathname + window.location.search;
    window.location.assign(`/ui/login?return_to=${encodeURIComponent(returnTo)}`);
  }
  return response;
}
