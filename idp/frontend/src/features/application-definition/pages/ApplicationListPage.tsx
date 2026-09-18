import { useEffect, useState } from 'react';
import type { ApiResult, ApplicationApi } from '../api/client';
import type { ApplicationListItem } from '../api/types';
import { loadDraft } from '../draft/storage';
import { linkHandler } from '../../../app/router';

export function ApplicationListPage({ api }: { api: ApplicationApi }) {
  const [result, setResult] = useState<ApiResult<ApplicationListItem[]> | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    api.listApplications().then((r) => {
      if (!cancelled) setResult(r);
    });
    return () => {
      cancelled = true;
    };
  }, [api, attempt]);

  const load = () => {
    setResult(null);
    setAttempt((n) => n + 1);
  };

  const hasNewDraft = loadDraft(null) !== null;

  return (
    <>
      <div className="page-header">
        <h1>Application definitions</h1>
        <a className="button" href="/ui/applications/new" onClick={linkHandler('/ui/applications/new')}>
          {hasNewDraft ? 'Continue draft' : 'Create application'}
        </a>
      </div>
      {result === null && (
        <p role="status" className="muted">
          Loading applications…
        </p>
      )}
      {result && result.kind !== 'ok' && (
        <div className="banner banner-error" role="alert">
          <p>The application list could not be loaded: {result.problems.map((p) => p.message).join(' ')}</p>
          <button type="button" className="secondary" onClick={load}>
            Try again
          </button>
        </div>
      )}
      {result?.kind === 'ok' && result.data.length === 0 && (
        <div className="card">
          <p>No application definitions yet. Create the first application to declare its workloads and resources.</p>
        </div>
      )}
      {result?.kind === 'ok' && result.data.length > 0 && (
        <div className="card table-wrap">
          <table>
            <thead>
              <tr>
                <th scope="col">Application</th>
                <th scope="col">Latest version</th>
                <th scope="col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {result.data.map((a) => {
                const path = `/ui/applications/${encodeURIComponent(a.applicationId)}`;
                return (
                  <tr key={a.applicationId}>
                    <td>
                      <strong className="component-name">{a.name}</strong>
                      {a.description && <div className="muted">{a.description}</div>}
                      {loadDraft(a.applicationId) && <span className="badge badge-draft">Unsaved draft in this tab</span>}
                    </td>
                    <td>v{a.latestVersion}</td>
                    <td>
                      <a href={path} onClick={linkHandler(path)} aria-label={`Edit ${a.name}`}>
                        Edit
                      </a>
                      {' · '}
                      <a href={`${path}/configuration`} onClick={linkHandler(`${path}/configuration`)} aria-label={`Configure ${a.name}`}>
                        Configure
                      </a>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
