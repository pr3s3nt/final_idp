import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from 'react';
import type { ApplicationApi } from '../api/client';
import type { Problem } from '../api/types';
import { BuilderNavigation, workspaceForField, type Workspace } from '../components/BuilderNavigation';
import { groupProblems } from '../../../shared/ui/Field';
import { OverviewWorkspace, ResourceWorkspace, ReviewWorkspace, WorkloadWorkspace } from '../components/Sections';
import { ValidationSummary } from '../components/ValidationSummary';
import { draftFromDto, draftToDto, emptyDraft, newResource, newWorkload, type ApplicationDraft } from '../draft/model';
import { draftReducer, type DraftAction } from '../draft/reducer';
import { loadDraft, removeDraft, saveDraft } from '../draft/storage';
import { validateDraft } from '../draft/validation';
import { linkHandler, navigate } from '../../../app/router';

interface Props {
  api: ApplicationApi;
  /** null creates a new application. */
  applicationId: string | null;
  savedVersion?: number;
}

type Phase = { kind: 'loading' } | { kind: 'load-failed'; problems: Problem[]; notFound: boolean } | { kind: 'ready' };

interface Restored {
  savedAt: string;
  staleBase: { draftBase: number; latest: number } | null;
}

/** Comparable form of a draft; ignores client-only keys such as output IDs. */
function fingerprint(draft: ApplicationDraft): string {
  return JSON.stringify(draftToDto(draft));
}

function draftJson(draft: ApplicationDraft): string {
  return JSON.stringify(draftToDto(draft), null, 2);
}

export function ApplicationEditorPage({ api, applicationId, savedVersion }: Props) {
  const [storedNew] = useState(() => (applicationId === null ? loadDraft(null) : null));
  const [draft, dispatch] = useReducer(draftReducer, null, () => storedNew?.draft ?? emptyDraft());
  const [phase, setPhase] = useState<Phase>(applicationId ? { kind: 'loading' } : { kind: 'ready' });
  const [baseline, setBaseline] = useState(() => fingerprint(emptyDraft()));
  const [restored, setRestored] = useState<Restored | null>(storedNew ? { savedAt: storedNew.savedAt, staleBase: null } : null);
  const [selected, setSelected] = useState<Workspace>({ kind: 'overview' });
  const [showClientErrors, setShowClientErrors] = useState(false);
  const [serverProblems, setServerProblems] = useState<Problem[]>([]);
  const [saveFailed, setSaveFailed] = useState<Problem[] | null>(null);
  const [conflict, setConflict] = useState<Problem[] | null>(null);
  const [copyStatus, setCopyStatus] = useState('');
  const [saving, setSaving] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const summaryRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (applicationId === null) return;
    let cancelled = false;
    api.loadApplication(applicationId).then((result) => {
      if (cancelled) return;
      if (result.kind !== 'ok') {
        setPhase({ kind: 'load-failed', problems: result.problems, notFound: result.kind === 'not-found' });
        return;
      }
      const loaded = draftFromDto(result.data);
      const latest = result.data.baseVersion;
      const stored = loadDraft(applicationId);
      if (stored) {
        const draftBase = stored.draft.baseVersion;
        dispatch({ type: 'replace', draft: stored.draft });
        setRestored({
          savedAt: stored.savedAt,
          staleBase: latest !== null && draftBase !== null && draftBase !== latest ? { draftBase, latest } : null,
        });
      } else {
        dispatch({ type: 'replace', draft: loaded });
      }
      setBaseline(fingerprint(loaded));
      setSelected({ kind: 'overview' });
      setPhase({ kind: 'ready' });
    });
    return () => {
      cancelled = true;
    };
  }, [api, applicationId, attempt]);

  const dirty = phase.kind === 'ready' && fingerprint(draft) !== baseline;

  useEffect(() => {
    if (phase.kind !== 'ready') return;
    if (dirty) saveDraft(draft);
    else removeDraft(applicationId);
  }, [draft, dirty, phase.kind, applicationId]);

  const edit = useCallback((action: DraftAction) => {
    dispatch(action);
    setServerProblems([]);
    setSaveFailed(null);
  }, []);

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
        setCopyStatus('');
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

  const copyDraft = async () => {
    try {
      if (!navigator.clipboard?.writeText) throw new Error('Clipboard unavailable');
      await navigator.clipboard.writeText(draftJson(draft));
      setCopyStatus('Draft JSON copied.');
    } catch {
      setCopyStatus('Copy was not available. Download the draft instead.');
    }
  };

  const downloadDraft = () => {
    const url = URL.createObjectURL(new Blob([draftJson(draft)], { type: 'application/json' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `${draft.name || 'application'}-draft.json`;
    anchor.click();
    URL.revokeObjectURL(url);
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
          <button
            type="button"
            onClick={() => {
              setPhase({ kind: 'loading' });
              setAttempt((value) => value + 1);
            }}
          >
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
              {typeof URL.createObjectURL === 'function' && <button type="button" className="secondary" onClick={downloadDraft}>Download draft JSON</button>}
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
          {selected.kind === 'resource' && resource && <ResourceWorkspace draft={draft} resource={resource} dispatch={edit} problems={problemsByField} onRemove={() => removeResource(resource.id)} />}
          {selected.kind === 'review' && <ReviewWorkspace draft={draft} />}
        </div>
      </div>
    </form>
  );
}
