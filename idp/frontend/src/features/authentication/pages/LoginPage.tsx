import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { httpAuthenticationApi, type AuthenticationApi, type Problem } from '../api/client';

interface Props {
  api?: AuthenticationApi;
}

export function LoginPage({ api = httpAuthenticationApi }: Props) {
  const query = new URLSearchParams(window.location.search);
  const requestedReturnTo = query.get('return_to') ?? '';
  const [csrfToken, setCSRFToken] = useState('');
  const [returnTo, setReturnTo] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [problems, setProblems] = useState<Problem[]>([]);
  const [formExpired, setFormExpired] = useState(false);
  const [retryAfter, setRetryAfter] = useState<number | null>(null);
  const errorRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    api.loadLoginContext(requestedReturnTo).then((result) => {
      if (cancelled) return;
      if (result.kind !== 'ok') {
        setProblems(result.problems);
        setFormExpired(true);
        setLoading(false);
        return;
      }
      if (result.data.redirectTo) {
        window.location.assign(result.data.redirectTo);
        return;
      }
      setCSRFToken(result.data.csrfToken ?? '');
      setReturnTo(result.data.returnTo ?? '');
      setLoading(false);
    });
    return () => { cancelled = true; };
  }, [api, requestedReturnTo]);

  useEffect(() => {
    if (problems.length > 0) errorRef.current?.focus();
  }, [problems]);

  useEffect(() => {
    if (retryAfter === null) return;
    const timer = window.setTimeout(() => setRetryAfter((seconds) => seconds === null || seconds <= 1 ? null : seconds - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [retryAfter]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (submitting || retryAfter !== null || !csrfToken) return;
    setSubmitting(true);
    setProblems([]);
    setFormExpired(false);
    setRetryAfter(null);
    const result = await api.signIn(username, password, returnTo, csrfToken);
    setPassword('');
    setSubmitting(false);
    if (result.kind === 'ok') {
      window.location.assign(result.data.redirectTo);
      return;
    }
    setProblems(result.problems);
    if (result.kind === 'form-expired') setFormExpired(true);
    if (result.kind === 'rate-limited') setRetryAfter(result.retryAfter);
  };

  const reason = query.get('reason');
  return (
    <div className="login-page">
      <header className="login-brand">IDP</header>
      <main className="login-main">
        <form className="login-panel" onSubmit={(event) => void submit(event)} aria-labelledby="login-title" noValidate>
          <h1 id="login-title">Sign in to IDP</h1>
          <p className="muted">Use the local account provided by your Platform Operator.</p>
          {reason === 'logged-out' && <div className="banner banner-info" role="status">You have signed out.</div>}
          {reason === 'session-expired' && <div className="banner banner-info" role="status">Your session has expired. Sign in again.</div>}
          {problems.length > 0 && (
            <div className="banner banner-error" role="alert" tabIndex={-1} ref={errorRef}>
              <p>{problems.map((problem) => problem.message).join(' ')}</p>
              {retryAfter !== null && <p>Try again in about {retryAfter} seconds.</p>}
              {formExpired && <button type="button" className="secondary" onClick={() => window.location.reload()}>Reload sign-in form</button>}
            </div>
          )}
          {loading ? (
            <p role="status" className="muted">Loading sign-in form…</p>
          ) : (
            <>
              <label htmlFor="username">Username</label>
              <input id="username" name="username" type="text" autoComplete="username" autoFocus value={username} onChange={(event) => setUsername(event.target.value)} disabled={submitting || formExpired} required />
              <label htmlFor="password">Password</label>
              <input id="password" name="password" type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} disabled={submitting || formExpired} required />
              <button type="submit" disabled={submitting || retryAfter !== null || formExpired || !username || !password} aria-busy={submitting}>
                {submitting ? 'Signing in…' : 'Sign in'}
              </button>
            </>
          )}
        </form>
      </main>
    </div>
  );
}
