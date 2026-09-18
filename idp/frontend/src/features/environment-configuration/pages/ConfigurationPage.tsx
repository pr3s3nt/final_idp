import { useMemo, useRef, useState } from 'react';
import type { ConfigurationApi } from '../api/client';
import type { ConfigurationRequirementsDto, OutputOptionDto } from '../api/types';
import type { Problem } from '../../application-definition/api/types';
import { ValidationSummary } from '../../application-definition/components/ValidationSummary';
import { ConfigurationNavigation, workspaceForField, type Workspace } from '../components/ConfigurationNavigation';
import { ConfigurationReviewWorkspace } from '../components/ConfigurationReviewWorkspace';
import { WorkloadConfigurationWorkspace } from '../components/WorkloadConfigurationWorkspace';
import { ENVIRONMENTS, bindingKey, draftToDto, type BindingDraft, type ConfigurationDraft, type Environment } from '../draft/model';
import type { DraftAction } from '../draft/reducer';
import { removeDraft } from '../draft/storage';
import { validateDraft } from '../draft/validation';
import { useConfigurationDraft } from '../hooks/useConfigurationDraft';
import { linkHandler } from '../../../app/router';

interface Props {
  api: ConfigurationApi;
  applicationId: string;
}

export function ConfigurationPage({ api, applicationId }: Props) {
  const [environment, setEnvironment] = useState<Environment>('STAGING');
  const { draft, dispatch, phase, dirty, restored, reload, markSaved } = useConfigurationDraft(api, applicationId, environment);
  // null means "not chosen yet": the page then opens on the first workload
  // that has something to configure.
  const [chosen, setSelected] = useState<Workspace | null>(null);
  const [outputs, setOutputs] = useState<Record<string, OutputOptionDto[]>>({});
  const [stagingKey, setStagingKey] = useState('');
  const [serverProblems, setServerProblems] = useState<Problem[]>([]);
  const [showClientErrors, setShowClientErrors] = useState(false);
  const [saveFailed, setSaveFailed] = useState<Problem[] | null>(null);
  const [conflict, setConflict] = useState<Problem[] | null>(null);
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const summaryRef = useRef<HTMLDivElement>(null);

  const requirements = phase.kind === 'ready' ? phase.requirements : null;
  const selected: Workspace = chosen ?? (requirements ? firstWorkspace(requirements) : { kind: 'review' });

  const clientProblems = useMemo(() => validateDraft(draft), [draft]);
  const shownProblems = showClientErrors ? [...clientProblems, ...serverProblems] : serverProblems;

  const outputsFor = (referenceKind: 'resource' | 'workload', refId: string): OutputOptionDto[] | null =>
    outputs[outputKey(referenceKind, refId, draft)] ?? null;

  const loadOutputs = (referenceKind: 'resource' | 'workload', workloadId: string, refId: string) => {
    const key = outputKey(referenceKind, refId, draft);
    if (outputs[key]) return;
    const request =
      referenceKind === 'resource'
        ? api.resourceOutputs({
            applicationId,
            environment,
            workloadId,
            resourceId: refId,
            catalogVersion: draft.catalogVersion,
            target: draft.deploymentTarget,
          })
        : api.workloadOutputs({ applicationId, environment, workloadId, targetWorkloadId: refId });
    void request.then((result) => {
      if (result.kind === 'ok') setOutputs((current) => ({ ...current, [key]: result.data.outputs }));
      else setServerProblems(result.problems);
    });
  };

  const stageSecret = (binding: BindingDraft) => {
    const key = bindingKey(binding.workloadId, binding.definitionId);
    setStagingKey(key);
    void api
      .stageSecret({
        applicationId,
        environment,
        workloadId: binding.workloadId,
        definitionId: binding.definitionId,
        value: binding.value,
      })
      .then((result) => {
        setStagingKey('');
        if (result.kind === 'ok') dispatch({ type: 'secret-reference', key, secretRef: result.data.secretRef });
        else setServerProblems(result.problems);
      });
  };

  const edit = (action: DraftAction) => {
    dispatch(action);
    setServerProblems([]);
    setSaveFailed(null);
    setSaved(false);
  };

  const showSummary = () => {
    setSelected({ kind: 'review' });
    window.setTimeout(() => summaryRef.current?.focus(), 0);
  };

  const save = async () => {
    setShowClientErrors(true);
    if (clientProblems.length > 0) {
      showSummary();
      return;
    }
    setSaving(true);
    setSaveFailed(null);
    setConflict(null);
    const result = await api.saveConfiguration(draftToDto(draft));
    setSaving(false);
    if (result.kind === 'ok') {
      removeDraft(draft.applicationId, draft.environment);
      markSaved(result.data.baseConfigurationRevision);
      setShowClientErrors(false);
      setServerProblems([]);
      setSaved(true);
      return;
    }
    if (result.kind === 'conflict') {
      setConflict(result.problems);
      return;
    }
    if (result.kind === 'validation') {
      setServerProblems(result.problems);
      showSummary();
      return;
    }
    setSaveFailed(result.problems);
  };

  const clearNotices = () => {
    setOutputs({});
    setServerProblems([]);
    setShowClientErrors(false);
    setSaveFailed(null);
    setConflict(null);
    setSaved(false);
  };

  const reloadWith = (confirmation?: string) => {
    if (confirmation && dirty && !window.confirm(confirmation)) return;
    if (draft.applicationId) removeDraft(draft.applicationId, draft.environment);
    clearNotices();
    reload();
  };

  const changeEnvironment = (next: Environment) => {
    if (next === environment) return;
    if (dirty && !window.confirm('Switch environment and leave the unsaved changes of this one behind?')) return;
    clearNotices();
    setEnvironment(next);
  };

  if (phase.kind === 'loading') {
    return (
      <p role="status" className="muted">
        Loading configuration…
      </p>
    );
  }
  if (phase.kind === 'load-failed' || requirements === null) {
    const problems = phase.kind === 'load-failed' ? phase.problems : [];
    return (
      <div className="banner banner-error" role="alert">
        <p>The configuration could not be loaded: {problems.map((problem) => problem.message).join(' ')}</p>
        <button type="button" className="secondary" onClick={() => reloadWith()}>
          Try again
        </button>{' '}
        <a href="/ui/applications" onClick={linkHandler('/ui/applications')}>
          Back to applications
        </a>
      </div>
    );
  }

  const applicationPath = `/ui/applications/${encodeURIComponent(applicationId)}`;
  const draftStatus = saving
    ? 'Saving…'
    : restored && dirty
      ? 'Restored draft'
      : dirty
        ? 'Unsaved changes · kept in this tab'
        : 'Saved';
  const workload = selected.kind === 'workload' ? requirements.workloads.find((item) => item.id === selected.id) : undefined;
  const nothingToConfigure = draft.variables.length === 0 && draft.secrets.length === 0;

  return (
    <form
      className="editor-shell"
      aria-labelledby="configuration-title"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
    >
      <header className="editor-header">
        <div className="editor-identity">
          <a href={applicationPath} onClick={linkHandler(applicationPath)} className="back-link">
            <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true" focusable="false">
              <path d="M7.78 12.53a.75.75 0 0 1-1.06 0L2.47 8.28a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 1 1 1.06 1.06L4.81 7h8.44a.75.75 0 0 1 0 1.5H4.81l2.97 2.97a.75.75 0 0 1 0 1.06Z" />
            </svg>
            {requirements.applicationName}
          </a>
          <div>
            <h1 id="configuration-title">Configuration · {requirements.applicationName}</h1>
            <p className="editor-meta">
              <span>Version {requirements.baseApplicationDefinitionVersion} requirements</span>
              <span className={dirty ? 'status-dirty' : 'status-saved'}>{draftStatus}</span>
            </p>
          </div>
        </div>
        <div className="header-actions">
          <button
            type="button"
            className="secondary"
            disabled={saving || !dirty}
            onClick={() => reloadWith('Discard all unsaved configuration changes?')}
          >
            Discard
          </button>
          <button type="submit" disabled={saving || nothingToConfigure} aria-busy={saving}>
            {saving ? 'Saving…' : 'Save configuration'}
          </button>
        </div>
      </header>

      <div className="scope-bar">
        <div className="field">
          <label htmlFor="uc02-environment">Environment</label>
          <select id="uc02-environment" value={environment} onChange={(event) => changeEnvironment(event.target.value as Environment)}>
            {ENVIRONMENTS.map((value) => (
              <option key={value} value={value}>
                {value.toLowerCase()}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="uc02-catalog-version">Platform catalog version</label>
          <select
            id="uc02-catalog-version"
            value={draft.catalogVersion}
            onChange={(event) => {
              edit({ type: 'catalog-version', value: event.target.value });
              setOutputs({});
            }}
          >
            {requirements.catalogVersions.map((version) => (
              <option key={version.id} value={String(version.number)}>
                v{version.number}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="uc02-target">Deployment target</label>
          <select
            id="uc02-target"
            value={draft.deploymentTarget}
            onChange={(event) => {
              edit({ type: 'deployment-target', value: event.target.value });
              setOutputs({});
            }}
          >
            {requirements.targets.map((target) => (
              <option key={`${target.target}/${target.region ?? ''}`} value={target.target}>
                {target.target}
                {target.region ? ` (${target.region})` : ''}
              </option>
            ))}
          </select>
        </div>
        <p className="hint scope-hint">
          The catalog version and the deployment target only decide which outputs a resource offers. Neither is saved with the
          configuration, and the deployment checks both again.
        </p>
      </div>

      <div className="editor-notices">
        {saved && !dirty && (
          <div className="banner banner-success" role="status">
            Environment configuration saved. Running environments change only when this application is deployed.
          </div>
        )}
        {restored && (
          <div className="banner banner-info" role="status">
            Restored unsaved changes kept in this browser tab since {new Date(restored.savedAt).toLocaleString()}.
          </div>
        )}
        {conflict && (
          <div className="banner banner-error" role="alert">
            <h2>The configuration changed while you were editing</h2>
            <p>{conflict.map((problem) => problem.message).join(' ')}</p>
            <p>Nothing was saved and your draft is still in this tab. Reload to review and reapply your changes.</p>
            <div className="conflict-actions">
              <button
                type="button"
                onClick={() => reloadWith('Discard your draft and load the current configuration? Your unsaved changes will be lost.')}
              >
                Reload configuration
              </button>
            </div>
          </div>
        )}
        {saveFailed && (
          <div className="banner banner-error" role="alert">
            <h2>Save failed</h2>
            <p>{saveFailed.map((problem) => problem.message).join(' ')}</p>
            <p>Your changes are still here. Use Save configuration to try again.</p>
          </div>
        )}
        {nothingToConfigure && (
          <div className="banner banner-info" role="status">
            Version {requirements.baseApplicationDefinitionVersion} declares no environment variable and no secret. Declare them in
            the application definition first.
          </div>
        )}
      </div>

      <div className="builder-layout">
        <ConfigurationNavigation workloads={requirements.workloads} selected={selected} problems={shownProblems} onSelect={setSelected} />
        <div className="builder-workspace">
          {selected.kind === 'review' && (
            <ValidationSummary
              ref={summaryRef}
              problems={shownProblems}
              title={`${shownProblems.length} ${shownProblems.length === 1 ? 'problem prevents' : 'problems prevent'} saving`}
              onSelectField={(field) => setSelected(workspaceForField(field, requirements.workloads))}
            />
          )}
          {selected.kind === 'workload' && workload && (
            <WorkloadConfigurationWorkspace
              workload={workload}
              draft={draft}
              requirements={requirements}
              dispatch={edit}
              outputsFor={outputsFor}
              loadOutputs={loadOutputs}
              stageSecret={stageSecret}
              stagingKey={stagingKey}
              problems={shownProblems}
            />
          )}
          {selected.kind === 'review' && <ConfigurationReviewWorkspace draft={draft} requirements={requirements} />}
        </div>
      </div>
    </form>
  );
}

function outputKey(referenceKind: 'resource' | 'workload', refId: string, draft: ConfigurationDraft): string {
  return referenceKind === 'resource' ? `resource:${refId}:${draft.catalogVersion}:${draft.deploymentTarget}` : `workload:${refId}`;
}

function firstWorkspace(requirements: ConfigurationRequirementsDto): Workspace {
  const workload = requirements.workloads.find((item) => item.variables.length > 0 || item.secrets.length > 0);
  return workload ? { kind: 'workload', id: workload.id } : { kind: 'review' };
}
