import { useCallback, useMemo, useRef, useState } from 'react';
import type { ApplicationApi } from '../api/client';
import type { Problem } from '../api/types';
import { BuilderNavigation, workspaceForField, type Workspace } from '../components/BuilderNavigation';
import { groupProblems } from '../../../shared/ui/Field';
import { OverviewWorkspace } from '../components/OverviewWorkspace';
import { ResourceWorkspace } from '../components/ResourceWorkspace';
import { ReviewWorkspace } from '../components/ReviewWorkspace';
import { WorkloadWorkspace } from '../components/WorkloadWorkspace';
import { ValidationSummary } from '../components/ValidationSummary';
import { draftToDto, newResource, newWorkload } from '../draft/model';
import type { DraftAction } from '../draft/reducer';
import { removeDraft } from '../draft/storage';
import { validateDraft } from '../draft/validation';
import { useApplicationDraft } from '../hooks/useApplicationDraft';
import { useDraftExport } from '../hooks/useDraftExport';
import { linkHandler, navigate } from '../../../app/router';

interface Props {
  api: ApplicationApi;
  /** null creates a new application. */
  applicationId: string | null;
  savedVersion?: number;
}

export function ApplicationEditorPage({ api, applicationId, savedVersion }: Props) {
  const { draft, dispatch, phase, dirty, restored, retry } = useApplicationDraft(api, applicationId);
  const { copyStatus, clearStatus, copyDraft, downloadDraft, canDownload } = useDraftExport(draft);
  const [selected, setSelected] = useState<Workspace>({ kind: 'overview' });
  const [showClientErrors, setShowClientErrors] = useState(false);
  const [serverProblems, setServerProblems] = useState<Problem[]>([]);
  const [saveFailed, setSaveFailed] = useState<Problem[] | null>(null);
  const [conflict, setConflict] = useState<Problem[] | null>(null);
  const [saving, setSaving] = useState(false);
  const summaryRef = useRef<HTMLDivElement>(null);

  const edit = useCallback((action: DraftAction) => {
    dispatch(action);
    setServerProblems([]);
    setSaveFailed(null);
  }, [dispatch]);

  const clientProblems = useMemo(() => validateDraft(draft), [draft]);
  const shownProblems = [...(showClientErrors ? clientProblems : []), ...serverProblems];
  const problemsByField = groupProblems(shownProblems);

  const openReviewAndFocus = () => {
    setSelected({ kind: 'review' });
    window.setTimeout(() => summaryRef.current?.focus(), 0);
  };

  const save = async () => {
    setShowClientErrors(true);
    setServerProblems([]);
    setSaveFailed(null);
    if (clientProblems.length > 0) {
      openReviewAndFocus();
      return;
    }
    setSaving(true);
    const result = await api.saveApplication(draftToDto(draft));
    setSaving(false);
    switch (result.kind) {
      case 'ok': {
        removeDraft(applicationId);
        const id = result.data.applicationId ?? '';
        navigate(`/ui/applications/${encodeURIComponent(id)}`, { savedVersion: result.data.baseVersion ?? undefined });
        return;
      }
      case 'conflict':
        setConflict(result.problems);
        clearStatus();
        return;
      case 'validation':
        setServerProblems(result.problems);
        openReviewAndFocus();
        return;
      default:
        setSaveFailed(result.problems);
    }
  };

  const reload = (message: string) => {
    if (dirty && !window.confirm(message)) return;
    removeDraft(applicationId);
    navigate(applicationId ? `/ui/applications/${encodeURIComponent(applicationId)}` : '/ui/applications/new');
  };

  const addWorkload = () => {
    const workload = newWorkload();
    edit({ type: 'addWorkload', workload });
    setSelected({ kind: 'workload', id: workload.id });
  };

  const addResource = () => {
    const resource = newResource();
    edit({ type: 'addResource', resource });
    setSelected({ kind: 'resource', id: resource.id });
  };

  const removeWorkload = (id: string) => {
    const workload = draft.workloads.find((item) => item.id === id);
    if (!window.confirm(`Remove workload ${workload?.name || 'unnamed'} and its dependencies?`)) return;
    edit({ type: 'removeWorkload', id });
    const next = draft.workloads.find((item) => item.id !== id);
    setSelected(next ? { kind: 'workload', id: next.id } : { kind: 'overview' });
  };

  const removeResource = (id: string) => {
    const resource = draft.resources.find((item) => item.id === id);
    if (!window.confirm(`Remove resource ${resource?.name || 'unnamed'} and its dependencies?`)) return;
    edit({ type: 'removeResource', id });
    const next = draft.resources.find((item) => item.id !== id);
    setSelected(next ? { kind: 'resource', id: next.id } : { kind: 'overview' });
  };

  if (phase.kind === 'loading') {
    return (
      <p role="status" className="muted loading-state">
        Loading the latest application version…
      </p>
    );
  }
  if (phase.kind === 'load-failed') {
    return (
      <div className="banner banner-error" role="alert">
        <h1>{phase.notFound ? 'Application not found' : 'The application could not be loaded'}</h1>
        <p>{phase.problems.map((problem) => problem.message).join(' ')}</p>
        {!phase.notFound && (
          <button type="button" onClick={retry}>
            Try again
          </button>
        )}{' '}
        <a href="/ui/applications" onClick={linkHandler('/ui/applications')}>
          Back to applications
        </a>
      </div>
    );
  }

  const workload = selected.kind === 'workload' ? draft.workloads.find((item) => item.id === selected.id) : undefined;
  const resource = selected.kind === 'resource' ? draft.resources.find((item) => item.id === selected.id) : undefined;
  const draftStatus = saving ? 'Saving…' : restored && dirty ? 'Restored draft' : dirty ? 'Unsaved changes · kept in this tab' : 'Saved';
  const versionStatus = draft.baseVersion ? `Version ${draft.baseVersion} · next save creates version ${draft.baseVersion + 1}` : 'New application · first save creates version 1';

  return (
    <form
      className="editor-shell"
      aria-labelledby="editor-title"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
    >
      <header className="editor-header">
        <div className="editor-identity">
          <a href="/ui/applications" onClick={linkHandler('/ui/applications')} className="back-link">
            ← Applications
          </a>
          <div>
            <h1 id="editor-title">{draft.name || (applicationId ? 'Unnamed application' : 'New application')}</h1>
            <p className="editor-meta"><span>{versionStatus}</span><span className={dirty ? 'status-dirty' : 'status-saved'}>{draftStatus}</span></p>
          </div>
        </div>
        <div className="header-actions">
          <button type="button" className="secondary" disabled={saving || !dirty} onClick={() => reload('Discard all unsaved changes?')}>
            Discard
          </button>
          <button type="submit" disabled={saving} aria-busy={saving}>
            {saving ? 'Saving…' : 'Save application'}
          </button>
        </div>
      </header>

      <div className="editor-notices">
        {savedVersion !== undefined && !dirty && (
          <div className="banner banner-success" role="status">
            Saved version {savedVersion}. The draft was cleared. Running environments change only when this version is deployed.
          </div>
        )}
        {restored && (
          <div className="banner banner-info" role="status">
            Restored unsaved changes kept in this browser tab since {new Date(restored.savedAt).toLocaleString()}.
            {restored.staleBase && (
              <strong> The draft started from version {restored.staleBase.draftBase}, but version {restored.staleBase.latest} is now the latest. Saving will report a conflict.</strong>
            )}
          </div>
        )}
        {conflict && (
          <div className="banner banner-error conflict-panel" role="alert">
            <h2>Someone saved a newer version first</h2>
            <p>{conflict.map((problem) => problem.message).join(' ')}</p>
            <p>Nothing was saved and your draft is still in this tab. Keep a copy before loading the latest version if you need to reapply these changes.</p>
            <div className="conflict-actions">
              <button type="button" className="secondary" onClick={() => void copyDraft()}>Copy draft JSON</button>
              {canDownload && <button type="button" className="secondary" onClick={downloadDraft}>Download draft JSON</button>}
              <button type="button" onClick={() => reload('Discard your draft and load the latest version? Your unsaved changes will be lost.')}>Load latest version</button>
            </div>
            {copyStatus && <p role="status" className="copy-status">{copyStatus}</p>}
          </div>
        )}
        {saveFailed && (
          <div className="banner banner-error" role="alert">
            <h2>Save failed</h2>
            <p>{saveFailed.map((problem) => problem.message).join(' ')}</p>
            <p>Your changes are still here. Use Save application to try again.</p>
          </div>
        )}
      </div>

      <div className="builder-layout">
        <BuilderNavigation draft={draft} selected={selected} problems={shownProblems} onSelect={setSelected} onAddWorkload={addWorkload} onAddResource={addResource} />
        <div className="builder-workspace">
          {selected.kind === 'review' && (
            <ValidationSummary
              ref={summaryRef}
              problems={shownProblems}
              title={`${shownProblems.length} ${shownProblems.length === 1 ? 'problem prevents' : 'problems prevent'} saving`}
              onSelectField={(field) => setSelected(workspaceForField(draft, field))}
            />
          )}
          {selected.kind === 'overview' && <OverviewWorkspace draft={draft} dispatch={edit} problems={problemsByField} />}
          {selected.kind === 'workload' && workload && <WorkloadWorkspace draft={draft} workload={workload} dispatch={edit} problems={problemsByField} onRemove={() => removeWorkload(workload.id)} />}
          {selected.kind === 'resource' && resource && <ResourceWorkspace resource={resource} dispatch={edit} problems={problemsByField} onRemove={() => removeResource(resource.id)} />}
          {selected.kind === 'review' && <ReviewWorkspace draft={draft} />}
        </div>
      </div>
    </form>
  );
}
