import { useCallback, useEffect, useReducer, useState } from 'react';
import type { ApplicationApi } from '../api/client';
import type { Problem } from '../api/types';
import { draftFromDto, draftToDto, emptyDraft, type ApplicationDraft } from '../draft/model';
import { draftReducer, type DraftAction } from '../draft/reducer';
import { loadDraft, removeDraft, saveDraft } from '../draft/storage';

export type Phase =
  | { kind: 'loading' }
  | { kind: 'load-failed'; problems: Problem[]; notFound: boolean }
  | { kind: 'ready' };

export interface Restored {
  savedAt: string;
  staleBase: { draftBase: number; latest: number } | null;
}

/** Comparable form of a draft; ignores client-only keys such as output IDs. */
function fingerprint(draft: ApplicationDraft): string {
  return JSON.stringify(draftToDto(draft));
}

export interface ApplicationDraftState {
  draft: ApplicationDraft;
  dispatch: (action: DraftAction) => void;
  phase: Phase;
  /** True when the draft differs from the version it was loaded from. */
  dirty: boolean;
  restored: Restored | null;
  /** Load the latest version again after a failed load. */
  retry: () => void;
}

/**
 * Owns the UC-01 draft for one editor session: the initial load, the
 * sessionStorage draft restored on top of it (ADR-016), and the dirty flag
 * that decides whether the draft is kept or cleared.
 */
export function useApplicationDraft(api: ApplicationApi, applicationId: string | null): ApplicationDraftState {
  const [storedNew] = useState(() => (applicationId === null ? loadDraft(null) : null));
  const [draft, dispatch] = useReducer(draftReducer, null, () => storedNew?.draft ?? emptyDraft());
  const [phase, setPhase] = useState<Phase>(applicationId ? { kind: 'loading' } : { kind: 'ready' });
  const [baseline, setBaseline] = useState(() => fingerprint(emptyDraft()));
  const [restored, setRestored] = useState<Restored | null>(storedNew ? { savedAt: storedNew.savedAt, staleBase: null } : null);
  const [attempt, setAttempt] = useState(0);

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

  useEffect(() => {
    if (phase.kind !== 'ready') return;
    if (dirty) saveDraft(draft);
    else removeDraft(applicationId);
  }, [draft, dirty, phase.kind, applicationId]);

  const retry = useCallback(() => {
    setPhase({ kind: 'loading' });
    setAttempt((value) => value + 1);
  }, []);

  return { draft, dispatch, phase, dirty, restored, retry };
}
