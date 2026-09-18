import { useState } from 'react';
import { httpApplicationApi, type ApplicationApi } from '../features/application-definition/api/client';
import { ErrorBoundary } from '../shared/ui/ErrorBoundary';
import { ApplicationEditorPage } from '../features/application-definition/pages/ApplicationEditorPage';
import { ApplicationListPage } from '../features/application-definition/pages/ApplicationListPage';
import { LoginPage } from '../features/authentication/pages/LoginPage';
import { httpAuthenticationApi, type AuthenticationApi } from '../features/authentication/api/client';
import { linkHandler, parseRoute, useLocation } from './router';

export function App({ api = httpApplicationApi, authenticationApi = httpAuthenticationApi }: { api?: ApplicationApi; authenticationApi?: AuthenticationApi }) {
  const location = useLocation();
  const [signingOut, setSigningOut] = useState(false);
  const [signOutError, setSignOutError] = useState('');
  const route = parseRoute(location.pathname);
  if (route.name === 'login') return <LoginPage api={authenticationApi} />;
  let page;
  switch (route.name) {
    case 'list':
      page = <ApplicationListPage key={location.key} api={api} />;
      break;
    case 'new':
      page = <ApplicationEditorPage key={location.key} api={api} applicationId={null} />;
      break;
    case 'edit':
      page = <ApplicationEditorPage key={location.key} api={api} applicationId={route.applicationId} savedVersion={location.state.savedVersion} />;
      break;
    default:
      page = (
        <p>
          Page not found.{' '}
          <a href="/ui/applications" onClick={linkHandler('/ui/applications')}>
            Go to applications
          </a>
        </p>
      );
  }
  const signOut = async () => {
    if (signingOut) return;
    setSigningOut(true);
    setSignOutError('');
    const result = await authenticationApi.signOut();
    if (result.kind === 'ok') window.location.assign('/ui/login?reason=logged-out');
    else if (result.kind === 'form-expired') window.location.assign('/ui/login?reason=session-expired');
    else {
      setSignOutError(result.problems.map((problem) => problem.message).join(' '));
      setSigningOut(false);
    }
  };
  return (
    <>
      <header className="topbar">
        <a href="/" className="topbar-brand">IDP · Deploy Application</a>
        <a className="topbar-nav" href="/ui/applications" onClick={linkHandler('/ui/applications')} aria-current={route.name === 'list' ? 'page' : undefined}>
          Application definitions
        </a>
        <button type="button" className="topbar-signout" disabled={signingOut} onClick={() => void signOut()}>
          {signingOut ? 'Signing out…' : 'Sign out'}
        </button>
      </header>
      {signOutError && <div className="banner banner-error shell-message" role="alert">{signOutError}</div>}
      <main className="page">
        <ErrorBoundary>{page}</ErrorBoundary>
      </main>
    </>
  );
}
