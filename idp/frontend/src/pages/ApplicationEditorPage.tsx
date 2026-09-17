import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from 'react';
import type { ApplicationApi } from '../api/client';
import type { Problem } from '../api/types';
import { groupProblems } from '../components/Field';
import {
  ApplicationInfoSection,
  ConfigurationSection,
  DependenciesSection,
  OverviewSection,
  ResourcesSection,
  WorkloadsSection,
} from '../components/Sections';
import { ValidationSummary } from '../components/ValidationSummary';
import { draftFromDto, draftToDto, emptyDraft, type ApplicationDraft } from '../draft/model';
import { draftReducer, type DraftAction } from '../draft/reducer';
import { loadDraft, removeDraft, saveDraft } from '../draft/storage';
import { validateDraft } from '../draft/validation';
import { linkHandler, navigate } from '../router';

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

export function ApplicationEditorPage({ api, applicationId, savedVersion }: Props) {
  // A new application starts from an empty draft, or the draft this tab kept.
  const [storedNew] = useState(() => (applicationId === null ? loadDraft(null) : null));
  const [draft, dispatch] = useReducer(draftReducer, null, () => storedNew?.draft ?? emptyDraft());
  const [phase, setPhase] = useState<Phase>(applicationId ? { kind: 'loading' } : { kind: 'ready' });
  const [baseline, setBaseline] = useState(() => fingerprint(emptyDraft()));
  const [restored, setRestored] = useState<Restored | null>(storedNew ? { savedAt: storedNew.savedAt, staleBase: null } : null);
  const [showClientErrors, setShowClientErrors] = useState(false);
  const [serverProblems, setServerProblems] = useState<Problem[]>([]);
  const [saveFailed, setSaveFailed] = useState<Problem[] | null>(null);
  const [conflict, setConflict] = useState<Problem[] | null>(null);
  const [saving, setSaving] = useState(false);
  const [newSource, setNewSource] = useState('');
  const [newTarget, setNewTarget] = useState('');
  const [attempt, setAttempt] = useState(0);
  const summaryRef = useRef<HTMLDivElement>(null);

  // Editing loads the latest version once, then restores this tab's draft if any.
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
      setPhase({ kind: 'ready' });
    });
    return () => {
      cancelled = true;
    };
  }, [api, applicationId, attempt]);

  const dirty = phase.kind === 'ready' && fingerprint(draft) !== baseline;

  // Keep the non-sensitive draft of this tab in sessionStorage while it has changes.
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

  const save = async () => {
    setShowClientErrors(true);
    setServerProblems([]);
    setSaveFailed(null);
    if (clientProblems.length > 0) {
      requestAnimationFrame(() => summaryRef.current?.focus());
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
        return;
      case 'validation':
        setServerProblems(result.problems);
        requestAnimationFrame(() => summaryRef.current?.focus());
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

  const title = applicationId ? `Edit ${draft.name || 'application'}` : 'Create application';

  if (phase.kind === 'loading') {
    return (
      <p role="status" className="muted">
        Loading the latest application version…
      </p>
    );
  }
  if (phase.kind === 'load-failed') {
    return (
      <div className="banner banner-error" role="alert">
        <h1>{phase.notFound ? 'Application not found' : 'The application could not be loaded'}</h1>
        <p>{phase.problems.map((p) => p.message).join(' ')}</p>
        {!phase.notFound && (
          <button
            type="button"
            onClick={() => {
              setPhase({ kind: 'loading' });
              setAttempt((n) => n + 1);
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

  return (
    <form
      className="editor"
      aria-labelledby="editor-title"
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        void save();
      }}
    >
      <div className="page-header">
        <div>
          <a href="/ui/applications" onClick={linkHandler('/ui/applications')}>
            ← Applications
          </a>
          <h1 id="editor-title">{title}</h1>
          <p className="muted">{draft.baseVersion ? `Editing from version ${draft.baseVersion}. Saving creates version ${draft.baseVersion + 1}.` : 'Saving creates version 1.'}</p>
        </div>
      </div>

      {savedVersion !== undefined && !dirty && (
        <div className="banner banner-success" role="status">
          Saved version {savedVersion}. The draft was cleared. Running environments change only when this version is deployed.
        </div>
      )}
      {restored && (
        <div className="banner banner-info" role="status">
          Restored unsaved changes kept in this browser tab since {new Date(restored.savedAt).toLocaleString()}.
          {restored.staleBase && (
            <strong>
              {' '}
              The draft started from version {restored.staleBase.draftBase}, but version {restored.staleBase.latest} is now the latest. Saving will report a conflict.
            </strong>
          )}
        </div>
      )}
      {conflict && (
        <div className="banner banner-error" role="alert">
          <h2>Someone saved a newer version first</h2>
          <p>{conflict.map((p) => p.message).join(' ')}</p>
          <p>Nothing was saved and your changes are still here. Note them, then load the latest version and apply them again.</p>
          <button type="button" onClick={() => reload('Discard your draft and load the latest version? Your unsaved changes will be lost.')}>
            Discard my draft and load the latest version
          </button>
        </div>
      )}
      {saveFailed && (
        <div className="banner banner-error" role="alert">
          <h2>Save failed</h2>
          <p>{saveFailed.map((p) => p.message).join(' ')}</p>
          <p>Your changes are still here. Try saving again.</p>
        </div>
      )}
      <ValidationSummary ref={summaryRef} problems={shownProblems} title={`${shownProblems.length} ${shownProblems.length === 1 ? 'problem prevents' : 'problems prevent'} saving`} />

      <ApplicationInfoSection draft={draft} dispatch={edit} problems={problemsByField} />
      <WorkloadsSection draft={draft} dispatch={edit} problems={problemsByField} />
      <ResourcesSection draft={draft} dispatch={edit} problems={problemsByField} />
      <ConfigurationSection draft={draft} dispatch={edit} problems={problemsByField} />
      <DependenciesSection draft={draft} dispatch={edit} problems={problemsByField} newSource={newSource} newTarget={newTarget} setNewSource={setNewSource} setNewTarget={setNewTarget} />
      <OverviewSection draft={draft} />

      <div className="actions">
        <span role="status" className="muted">
          {saving ? 'Saving…' : dirty ? 'Unsaved changes (kept in this browser tab)' : 'No unsaved changes'}
        </span>
        <button type="button" className="secondary" disabled={saving || !dirty} onClick={() => reload('Discard all unsaved changes?')}>
          Discard changes
        </button>
        <button type="submit" disabled={saving} aria-busy={saving}>
          {saving ? 'Saving…' : 'Save application'}
        </button>
      </div>
    </form>
  );
}
