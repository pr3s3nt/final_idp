import { useCallback, useEffect, useReducer, useState } from 'react';
import type { ConfigurationApi } from '../api/client';
import type { ConfigurationRequirementsDto } from '../api/types';
import type { Problem } from '../../application-definition/api/types';
import { draftFromRequirements, draftToDto, emptyDraft, type ConfigurationDraft, type Environment } from '../draft/model';
import { draftReducer, type DraftAction } from '../draft/reducer';
import { loadDraft, removeDraft, saveDraft } from '../draft/storage';

export type Phase =
  | { kind: 'loading' }
  | { kind: 'load-failed'; problems: Problem[] }
  | { kind: 'ready'; requirements: ConfigurationRequirementsDto };

export interface Restored {
  savedAt: string;
}

export interface ConfigurationDraftState {
  draft: ConfigurationDraft;
  dispatch: (action: DraftAction) => void;
  phase: Phase;
  /** True while the draft differs from the state last loaded or saved. */
  dirty: boolean;
  restored: Restored | null;
  /** Load the environment again, discarding whatever is in memory. */
  reload: () => void;
  /** Record a successful Save: the submitted draft becomes the durable state. */
  markSaved: (revision: string) => void;
}

/**
 * Owns the UC-02 draft of one application + environment: the initial load, the
 * sessionStorage draft restored on top of it (ADR-016), and the dirty flag that
 * decides whether the draft is kept or cleared.
 */
export function useConfigurationDraft(api: ConfigurationApi, applicationId: string, environment: Environment): ConfigurationDraftState {
  const [draft, dispatch] = useReducer(draftReducer, emptyDraft(environment));
  const [phase, setPhase] = useState<Phase>({ kind: 'loading' });
  const [restored, setRestored] = useState<Restored | null>(null);
  const [attempt, setAttempt] = useState(0);
  // Comparable form of the state last loaded or saved; empty until the first
  // load succeeds, so an empty placeholder draft never counts as dirty.
  const [baseline, setBaseline] = useState('');

  useEffect(() => {
    let cancelled = false;
    api.selectEnvironment(applicationId, environment).then((result) => {
      if (cancelled) return;
      if (result.kind !== 'ok') {
        setPhase({ kind: 'load-failed', problems: result.problems });
        return;
      }
      const dto = result.data;
      const catalogVersion = String(dto.catalogVersions[0]?.number ?? '');
      const target = dto.targets[0]?.target ?? '';
      const fresh = draftFromRequirements(dto, catalogVersion, target);
      setBaseline(JSON.stringify(draftToDto(fresh)));

      // A stored draft is only restored when it was started from the same
      // Application Definition version; otherwise the requirements moved on.
      const stored = loadDraft(dto.applicationId, dto.environment);
      if (stored && stored.draft.baseApplicationDefinitionVersion === dto.baseApplicationDefinitionVersion) {
        dispatch({ type: 'replace', draft: stored.draft });
        setRestored({ savedAt: stored.savedAt });
      } else {
        if (stored) removeDraft(dto.applicationId, dto.environment);
        dispatch({ type: 'replace', draft: fresh });
        setRestored(null);
      }
      setPhase({ kind: 'ready', requirements: dto });
    });
    return () => {
      cancelled = true;
    };
  }, [api, applicationId, environment, attempt]);

  const dirty = phase.kind === 'ready' && baseline !== '' && JSON.stringify(draftToDto(draft)) !== baseline;

  useEffect(() => {
    if (phase.kind !== 'ready' || baseline === '' || draft.applicationId === '') return;
    if (dirty) saveDraft(draft);
    else removeDraft(draft.applicationId, draft.environment);
  }, [draft, dirty, phase.kind, baseline]);

  const markSaved = useCallback(
    (revision: string) => {
      setBaseline(JSON.stringify(draftToDto({ ...draft, baseConfigurationRevision: revision })));
      dispatch({ type: 'saved', revision });
    },
    [draft],
  );

  const reload = useCallback(() => {
    setBaseline('');
    setRestored(null);
    setPhase({ kind: 'loading' });
    setAttempt((value) => value + 1);
  }, []);

  return { draft, dispatch, phase, dirty, restored, reload, markSaved };
}
