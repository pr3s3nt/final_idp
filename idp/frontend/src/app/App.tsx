import { httpApplicationApi, type ApplicationApi } from '../features/application-definition/api/client';
import { ErrorBoundary } from '../shared/ui/ErrorBoundary';
import { ApplicationEditorPage } from '../features/application-definition/pages/ApplicationEditorPage';
import { ApplicationListPage } from '../features/application-definition/pages/ApplicationListPage';
import { linkHandler, parseRoute, useLocation } from './router';

export function App({ api = httpApplicationApi }: { api?: ApplicationApi }) {
  const location = useLocation();
  const route = parseRoute(location.pathname);
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
  return (
    <>
      <header className="topbar">
        <a href="/">IDP · Deploy Application</a>
        <a href="/ui/applications" onClick={linkHandler('/ui/applications')} aria-current={route.name === 'list' ? 'page' : undefined}>
          Application definitions
        </a>
      </header>
      <main className="page">
        <ErrorBoundary>{page}</ErrorBoundary>
      </main>
    </>
  );
}
