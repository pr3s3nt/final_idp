import type { Problem } from '../../application-definition/api/types';
import type { ConfigurationRequirementsDto, ConfigurationWorkloadDto, OutputOptionDto, ValueSource } from '../api/types';
import { bindingKey, type BindingDraft } from '../draft/model';
import { fieldPath, type BindingKind } from '../draft/validation';
import type { DraftAction } from '../draft/reducer';
import { fieldDomId } from '../../../shared/ui/Field';

const sourceLabels: Record<ValueSource, string> = {
  DIRECT: 'Environment value',
  RESOURCE_OUTPUT: 'Resource output',
  WORKLOAD_OUTPUT: 'Workload output',
  SECRET_REF: 'Secret value',
};

export interface BindingFieldProps {
  kind: BindingKind;
  binding: BindingDraft;
  workload: ConfigurationWorkloadDto;
  requirements: ConfigurationRequirementsDto;
  dispatch: (action: DraftAction) => void;
  /** Outputs already loaded for a referenced component, or null while loading. */
  outputsFor: (referenceKind: 'resource' | 'workload', refId: string) => OutputOptionDto[] | null;
  loadOutputs: (referenceKind: 'resource' | 'workload', workloadId: string, refId: string) => void;
  stageSecret: (binding: BindingDraft) => void;
  staging: boolean;
  problems: Problem[];
}

export function BindingField(props: BindingFieldProps) {
  const { kind, binding, workload, requirements, dispatch, outputsFor, loadOutputs, stageSecret, staging, problems } = props;
  const key = bindingKey(binding.workloadId, binding.definitionId);
  const field = fieldPath(kind, binding);
  const id = fieldDomId(field);
  const errors = problems.filter((problem) => problem.field === field);
  const sources: ValueSource[] = kind === 'secret' ? ['SECRET_REF', 'RESOURCE_OUTPUT'] : ['DIRECT', 'RESOURCE_OUTPUT', 'WORKLOAD_OUTPUT'];

  const referenceKind = binding.source === 'WORKLOAD_OUTPUT' ? 'workload' : 'resource';
  const referenceOptions =
    referenceKind === 'workload'
      ? workload.dependsOnWorkloads.map((refId) => ({ refId, label: requirements.workloads.find((w) => w.id === refId)?.name ?? refId }))
      : workload.dependsOnResources.map((refId) => ({ refId, label: requirements.resources.find((r) => r.id === refId)?.name ?? refId }));
  const available = binding.refId ? outputsFor(referenceKind, binding.refId) : null;
  const selectable = (available ?? []).filter((output) => (kind === 'secret' ? output.sensitive : !output.sensitive));
  const referenceLabel = referenceKind === 'workload' ? 'Workload' : 'Resource';
  const errorId = errors.length > 0 ? `${id}-errors` : undefined;

  return (
    <fieldset className="binding" id={`${id}-row`}>
      <legend className="binding-header">
        <span className="binding-name">
          {binding.name}
          {binding.required && <span className="binding-required"> *</span>}
        </span>
        {kind === 'secret' && <span className="binding-tag">secret</span>}
      </legend>

      <div className="field">
        <label htmlFor={id}>Source</label>
        <select
          id={id}
          value={binding.source}
          aria-invalid={errors.length > 0 || undefined}
          aria-describedby={errorId}
          onChange={(event) => dispatch({ type: 'source', kind, key, source: event.target.value as ValueSource | '' })}
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
            className="mono"
            type="text"
            autoComplete="off"
            spellCheck={false}
            value={binding.value}
            onChange={(event) => dispatch({ type: 'direct-value', kind, key, value: event.target.value })}
          />
        </div>
      )}

      {binding.source === 'SECRET_REF' && (
        <div className="field">
          <label htmlFor={`${id}-secret`}>{binding.secretRef ? 'Replace the secret value' : 'Secret value'}</label>
          <div className="binding-actions">
            <input
              id={`${id}-secret`}
              type="password"
              autoComplete="new-password"
              value={binding.value}
              onChange={(event) => dispatch({ type: 'direct-value', kind: 'secret', key, value: event.target.value })}
            />
            <button type="button" className="secondary" disabled={binding.value === '' || staging} onClick={() => stageSecret(binding)}>
              {staging ? 'Storing…' : 'Store secret'}
            </button>
          </div>
          <p className="hint" role={binding.secretRef ? 'status' : undefined}>
            {binding.secretRef
              ? 'Stored. The IDP keeps only a reference; the value is never shown again.'
              : 'The value is sent to the Secret Store and replaced by a reference.'}
          </p>
        </div>
      )}

      {(binding.source === 'RESOURCE_OUTPUT' || binding.source === 'WORKLOAD_OUTPUT') && (
        <>
          <div className="field">
            <label htmlFor={`${id}-ref`}>{referenceLabel}</label>
            <select
              id={`${id}-ref`}
              value={binding.refId}
              onChange={(event) => {
                dispatch({ type: 'reference', kind, key, refId: event.target.value, outputName: '' });
                if (event.target.value) loadOutputs(referenceKind, workload.id, event.target.value);
              }}
            >
              <option value="">
                {referenceOptions.length === 0
                  ? `${workload.name} depends on no ${referenceLabel.toLowerCase()}; declare the dependency in UC-01 first`
                  : `Choose a ${referenceLabel.toLowerCase()} this workload depends on`}
              </option>
              {referenceOptions.map((option) => (
                <option key={option.refId} value={option.refId}>
                  {option.label}
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
                onChange={(event) => dispatch({ type: 'reference', kind, key, refId: binding.refId, outputName: event.target.value })}
              >
                <option value="">{available === null ? 'Loading outputs…' : 'Choose an output'}</option>
                {selectable.map((output) => (
                  <option key={output.name} value={output.name}>
                    {output.name}
                  </option>
                ))}
              </select>
              {binding.outputName && (
                <p className="hint mono">
                  {(referenceOptions.find((option) => option.refId === binding.refId)?.label ?? binding.refId) + '.' + binding.outputName}
                </p>
              )}
            </div>
          )}
        </>
      )}

      {errors.length > 0 && (
        <ul className="field-errors" id={`${id}-errors`}>
          {errors.map((problem, index) => (
            <li key={index}>{problem.message}</li>
          ))}
        </ul>
      )}
    </fieldset>
  );
}
