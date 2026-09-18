import { useCallback, useEffect, useReducer, useRef, useState } from 'react';
import type { ConfigurationApi } from '../api/client';
import type {
  ConfigurationRequirementsDto,
  ConfigurationWorkloadDto,
  OutputOptionDto,
  ValueSource,
} from '../api/types';
import type { Problem } from '../../application-definition/api/types';
import {
  ENVIRONMENTS,
  bindingKey,
  draftFromRequirements,
  draftToDto,
  pendingSecrets,
  unsetRequired,
  type BindingDraft,
  type ConfigurationDraft,
  type Environment,
} from '../draft/model';
import { draftReducer, type BindingKind, type DraftAction } from '../draft/reducer';
import { loadDraft, removeDraft, saveDraft } from '../draft/storage';
import { linkHandler } from '../../../app/router';

// The phase carries the session it belongs to, so a phase from the previous
// environment is never rendered for the current one.
type Phase =
  | { kind: 'loading'; session: string }
  | { kind: 'load-failed'; session: string; problems: Problem[] }
  | { kind: 'ready'; session: string; requirements: ConfigurationRequirementsDto };

type SaveState =
  | { kind: 'idle' }
  | { kind: 'saving' }
  | { kind: 'saved' }
  | { kind: 'rejected'; problems: Problem[] }
  | { kind: 'conflict'; problems: Problem[] };

const emptyDraft: ConfigurationDraft = {
  applicationId: '',
  environment: 'STAGING',
  baseApplicationDefinitionVersion: 0,
  baseConfigurationRevision: '',
  catalogVersion: '',
  deploymentTarget: '',
  variables: [],
  secrets: [],
};

const sourceLabels: Record<ValueSource, string> = {
  DIRECT: 'Environment value',
  RESOURCE_OUTPUT: 'Resource output',
  WORKLOAD_OUTPUT: 'Workload output',
  SECRET_REF: 'Secret value',
};

export function ConfigurationPage({ api, applicationId }: { api: ConfigurationApi; applicationId: string }) {
  const [environment, setEnvironment] = useState<Environment>('STAGING');
  const [attempt, setAttempt] = useState(0);
  const session = `${applicationId}:${environment}:${attempt}`;
  const [loadedPhase, setPhase] = useState<Phase>({ kind: 'loading', session });
  const [draft, dispatch] = useReducer(draftReducer, emptyDraft);
  const [save, setSave] = useState<SaveState>({ kind: 'idle' });
  const [outputs, setOutputs] = useState<Record<string, OutputOptionDto[]>>({});
  const loaded = useRef(false);
  // The last state that is durable (loaded or saved). The draft is kept in
  // sessionStorage only while it differs from it; Save and Discard clear it.
  const baseline = useRef('');
  const phase: Phase = loadedPhase.session === session ? loadedPhase : { kind: 'loading', session };

  useEffect(() => {
    let cancelled = false;
    loaded.current = false;
    api.selectEnvironment(applicationId, environment).then((result) => {
      if (cancelled) return;
      if (result.kind !== 'ok') {
        setPhase({ kind: 'load-failed', session, problems: result.problems });
        return;
      }
      const dto = result.data;
      const catalogVersion = String(dto.catalogVersions[0]?.number ?? '');
      const target = dto.targets[0]?.target ?? '';
      const stored = loadDraft(dto.applicationId, dto.environment);
      const fresh = draftFromRequirements(dto, catalogVersion, target);
      dispatch({
        type: 'replace',
        draft:
          stored && stored.draft.baseApplicationDefinitionVersion === dto.baseApplicationDefinitionVersion ? stored.draft : fresh,
      });
      baseline.current = JSON.stringify(draftToDto(fresh));
      setPhase({ kind: 'ready', session, requirements: dto });
      loaded.current = true;
    });
    return () => {
      cancelled = true;
    };
  }, [api, applicationId, environment, session]);

  // Switching environment or reloading starts a new session with no carried
  // over save result or output list.
  const startSession = (next: Environment, retry: boolean) => {
    setSave({ kind: 'idle' });
    setOutputs({});
    if (retry) setAttempt((n) => n + 1);
    else setEnvironment(next);
  };

  // The draft belongs to this tab; the backend keeps none between requests.
  useEffect(() => {
    if (phase.kind !== 'ready' || !loaded.current || !draft.applicationId) return;
    if (JSON.stringify(draftToDto(draft)) === baseline.current) removeDraft(draft.applicationId, draft.environment);
    else saveDraft(draft);
  }, [draft, phase.kind]);

  const outputKey = (kind: 'resource' | 'workload', workloadId: string, refId: string) =>
    kind === 'resource' ? `resource:${workloadId}:${refId}:${draft.catalogVersion}:${draft.deploymentTarget}` : `workload:${workloadId}:${refId}`;

  const loadOutputs = useCallback(
    async (kind: 'resource' | 'workload', workloadId: string, refId: string) => {
      const key = kind === 'resource' ? `resource:${workloadId}:${refId}:${draft.catalogVersion}:${draft.deploymentTarget}` : `workload:${workloadId}:${refId}`;
      const result =
        kind === 'resource'
          ? await api.resourceOutputs({ applicationId, environment, workloadId, resourceId: refId, catalogVersion: draft.catalogVersion, target: draft.deploymentTarget })
          : await api.workloadOutputs({ applicationId, environment, workloadId, targetWorkloadId: refId });
      setOutputs((current) => ({ ...current, [key]: result.kind === 'ok' ? result.data.outputs : [] }));
      if (result.kind !== 'ok') setSave({ kind: 'rejected', problems: result.problems });
    },
    [api, applicationId, environment, draft.catalogVersion, draft.deploymentTarget],
  );

  const stageSecret = async (binding: BindingDraft) => {
    const result = await api.stageSecret({
      applicationId,
      environment,
      workloadId: binding.workloadId,
      definitionId: binding.definitionId,
      value: binding.value,
    });
    if (result.kind !== 'ok') {
      setSave({ kind: 'rejected', problems: result.problems });
      return;
    }
    dispatch({ type: 'secret-reference', key: bindingKey(binding.workloadId, binding.definitionId), secretRef: result.data.secretRef });
  };

  const submit = async () => {
    setSave({ kind: 'saving' });
    const result = await api.saveConfiguration(draftToDto(draft));
    if (result.kind === 'ok') {
      const saved = { ...draft, baseConfigurationRevision: result.data.baseConfigurationRevision };
      baseline.current = JSON.stringify(draftToDto(saved));
      removeDraft(draft.applicationId, draft.environment);
      dispatch({ type: 'replace', draft: saved });
      setSave({ kind: 'saved' });
      return;
    }
    setSave(result.kind === 'conflict' ? { kind: 'conflict', problems: result.problems } : { kind: 'rejected', problems: result.problems });
  };

  const discard = () => {
    removeDraft(draft.applicationId, draft.environment);
    startSession(environment, true);
  };

  if (phase.kind === 'loading') {
    return (
      <p role="status" className="muted">
        Loading configuration…
      </p>
    );
  }
  if (phase.kind === 'load-failed') {
    return (
      <div className="banner banner-error" role="alert">
        <p>The configuration could not be loaded: {phase.problems.map((p) => p.message).join(' ')}</p>
        <button type="button" className="secondary" onClick={() => startSession(environment, true)}>
          Try again
        </button>
      </div>
    );
  }

  const requirements = phase.requirements;
  const missing = unsetRequired(draft);
  const pending = pendingSecrets(draft);
  const blocked = missing.length > 0 || pending.length > 0;

  return (
    <>
      <div className="page-header">
        <h1>Configuration · {requirements.applicationName}</h1>
        <a href={`/ui/applications/${encodeURIComponent(applicationId)}`} onClick={linkHandler(`/ui/applications/${encodeURIComponent(applicationId)}`)}>
          Back to the application definition
        </a>
      </div>

      <div className="card">
        <div className="field">
          <label htmlFor="uc02-environment">Environment</label>
          <select id="uc02-environment" value={environment} onChange={(e) => startSession(e.target.value as Environment, false)}>
            {ENVIRONMENTS.map((value) => (
              <option key={value} value={value}>
                {value.toLowerCase()}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="uc02-catalog-version">Platform catalog version</label>
          <select id="uc02-catalog-version" value={draft.catalogVersion} onChange={(e) => dispatch({ type: 'catalog-version', value: e.target.value })}>
            {requirements.catalogVersions.map((v) => (
              <option key={v.id} value={String(v.number)}>
                v{v.number}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="uc02-target">Deployment target</label>
          <select id="uc02-target" value={draft.deploymentTarget} onChange={(e) => dispatch({ type: 'deployment-target', value: e.target.value })}>
            {requirements.targets.map((t) => (
              <option key={`${t.target}/${t.region ?? ''}`} value={t.target}>
                {t.target}
                {t.region ? ` (${t.region})` : ''}
              </option>
            ))}
          </select>
        </div>
        <p className="hint">
          The catalog version and the deployment target decide which resource definition applies, so they decide which outputs a
          resource offers. Neither is saved with the configuration; the deployment checks both again.
        </p>
      </div>

      {requirements.workloads.map((workload) => (
        <WorkloadSection
          key={workload.id}
          workload={workload}
          requirements={requirements}
          draft={draft}
          dispatch={dispatch}
          outputs={outputs}
          outputKey={outputKey}
          loadOutputs={loadOutputs}
          stageSecret={stageSecret}
        />
      ))}

      {save.kind === 'saved' && (
        <div className="banner banner-success" role="status">
          Environment configuration saved. It does not deploy the application.
        </div>
      )}
      {(save.kind === 'rejected' || save.kind === 'conflict') && (
        <div className="banner banner-error" role="alert">
          <p>
            {save.kind === 'conflict'
              ? 'The configuration changed while you were editing. Reload it, then review and reapply your changes.'
              : 'The configuration was not saved:'}
          </p>
          <ul>
            {save.problems.map((p, i) => (
              <li key={i}>{p.message}</li>
            ))}
          </ul>
          {save.kind === 'conflict' && (
            <button type="button" className="secondary" onClick={() => startSession(environment, true)}>
              Reload the configuration
            </button>
          )}
        </div>
      )}
      {blocked && (
        <div className="banner banner-warning" role="status">
          {missing.length > 0 && <p>{missing.length} required value(s) still have no source.</p>}
          {pending.length > 0 && <p>{pending.length} secret value(s) were typed but not stored yet.</p>}
        </div>
      )}

      <div className="form-actions">
        <button type="button" onClick={() => void submit()} disabled={save.kind === 'saving' || blocked}>
          {save.kind === 'saving' ? 'Saving…' : 'Save configuration'}
        </button>
        <button type="button" className="secondary" onClick={discard}>
          Discard draft
        </button>
      </div>
    </>
  );
}

interface SectionProps {
  workload: ConfigurationWorkloadDto;
  requirements: ConfigurationRequirementsDto;
  draft: ConfigurationDraft;
  dispatch: (action: DraftAction) => void;
  outputs: Record<string, OutputOptionDto[]>;
  outputKey: (kind: 'resource' | 'workload', workloadId: string, refId: string) => string;
  loadOutputs: (kind: 'resource' | 'workload', workloadId: string, refId: string) => Promise<void>;
  stageSecret: (binding: BindingDraft) => Promise<void>;
}

function WorkloadSection(props: SectionProps) {
  const { workload, draft } = props;
  const variables = draft.variables.filter((b) => b.workloadId === workload.id);
  const secrets = draft.secrets.filter((b) => b.workloadId === workload.id);
  if (variables.length === 0 && secrets.length === 0) return null;
  return (
    <section className="card">
      <h2 className="component-name">{workload.name}</h2>
      {variables.map((binding) => (
        <BindingRow key={binding.definitionId} kind="variable" binding={binding} {...props} />
      ))}
      {secrets.map((binding) => (
        <BindingRow key={binding.definitionId} kind="secret" binding={binding} {...props} />
      ))}
    </section>
  );
}

function BindingRow({ kind, binding, ...props }: SectionProps & { kind: BindingKind; binding: BindingDraft }) {
  const { workload, requirements, dispatch, outputs, outputKey, loadOutputs, stageSecret } = props;
  const key = bindingKey(binding.workloadId, binding.definitionId);
  const id = `uc02-${kind}-${binding.definitionId}`;
  const sources: ValueSource[] = kind === 'secret' ? ['SECRET_REF', 'RESOURCE_OUTPUT'] : ['DIRECT', 'RESOURCE_OUTPUT', 'WORKLOAD_OUTPUT'];

  const referenceOptions =
    binding.source === 'WORKLOAD_OUTPUT'
      ? workload.dependsOnWorkloads.map((refId) => ({ refId, label: requirements.workloads.find((w) => w.id === refId)?.name ?? refId }))
      : workload.dependsOnResources.map((refId) => ({ refId, label: requirements.resources.find((r) => r.id === refId)?.name ?? refId }));
  const referenceKind = binding.source === 'WORKLOAD_OUTPUT' ? 'workload' : 'resource';
  const available = binding.refId ? (outputs[outputKey(referenceKind, workload.id, binding.refId)] ?? null) : null;
  const selectable = kind === 'secret' ? (available ?? []).filter((o) => o.sensitive) : (available ?? []).filter((o) => !o.sensitive);

  return (
    <div className="binding">
      <div className="field">
        <label htmlFor={`${id}-source`}>
          {binding.name}
          {binding.required && <span aria-hidden="true"> *</span>}
        </label>
        <select
          id={`${id}-source`}
          value={binding.source}
          onChange={(e) => dispatch({ type: 'source', kind, key, source: e.target.value as ValueSource | '' })}
        >
          <option value="">Choose a value source</option>
          {sources.map((source) => (
            <option key={source} value={source}>
              {sourceLabels[source]}
            </option>
          ))}
        </select>
      </div>

      {binding.source === 'DIRECT' && (
        <div className="field">
          <label htmlFor={`${id}-value`}>Value</label>
          <input
            id={`${id}-value`}
            type="text"
            autoComplete="off"
            value={binding.value}
            onChange={(e) => dispatch({ type: 'direct-value', kind, key, value: e.target.value })}
          />
        </div>
      )}

      {binding.source === 'SECRET_REF' && (
        <div className="field">
          <label htmlFor={`${id}-secret`}>Secret value</label>
          <input
            id={`${id}-secret`}
            type="password"
            autoComplete="new-password"
            value={binding.value}
            onChange={(e) => dispatch({ type: 'direct-value', kind: 'secret', key, value: e.target.value })}
          />
          <button type="button" className="secondary" disabled={binding.value === ''} onClick={() => void stageSecret(binding)}>
            Store secret
          </button>
          <p className="hint">
            {binding.secretRef
              ? 'Stored. The IDP keeps only a reference; the value is never shown again.'
              : 'The value is sent to the Secret Store and replaced by a reference.'}
          </p>
        </div>
      )}

      {(binding.source === 'RESOURCE_OUTPUT' || binding.source === 'WORKLOAD_OUTPUT') && (
        <>
          <div className="field">
            <label htmlFor={`${id}-ref`}>{binding.source === 'WORKLOAD_OUTPUT' ? 'Workload' : 'Resource'}</label>
            <select
              id={`${id}-ref`}
              value={binding.refId}
              onChange={(e) => {
                dispatch({ type: 'reference', kind, key, refId: e.target.value, outputName: '' });
                if (e.target.value) void loadOutputs(referenceKind, workload.id, e.target.value);
              }}
            >
              <option value="">
                {referenceOptions.length === 0
                  ? `${workload.name} depends on nothing of this kind; add the dependency in UC-01 first`
                  : 'Choose a component this workload depends on'}
              </option>
              {referenceOptions.map((o) => (
                <option key={o.refId} value={o.refId}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>
          {binding.refId && (
            <div className="field">
              <label htmlFor={`${id}-output`}>Output</label>
              <select
                id={`${id}-output`}
                value={binding.outputName}
                onChange={(e) => dispatch({ type: 'reference', kind, key, refId: binding.refId, outputName: e.target.value })}
              >
                <option value="">{available === null ? 'Loading outputs…' : 'Choose an output'}</option>
                {selectable.map((o) => (
                  <option key={o.name} value={o.name}>
                    {o.name}
                  </option>
                ))}
              </select>
            </div>
          )}
        </>
      )}
    </div>
  );
}
