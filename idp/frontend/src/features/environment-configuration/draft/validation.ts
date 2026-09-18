import type { Problem } from '../../application-definition/api/types';
import { bindingKey, type BindingDraft, type ConfigurationDraft } from './model';

// Client-side checks for fast feedback. They follow the UC-02 specification;
// the backend validates the complete draft again and is authoritative.
// Field paths are variables.<workloadId>.<definitionId> and
// secrets.<workloadId>.<definitionId>, so a problem can be traced back to the
// workload whose workspace holds it.

export type BindingKind = 'variable' | 'secret';

export function fieldPath(kind: BindingKind, binding: BindingDraft): string {
  return `${kind === 'variable' ? 'variables' : 'secrets'}.${bindingKey(binding.workloadId, binding.definitionId)}`;
}

/** Workload a problem belongs to, or '' when it belongs to no single workload. */
export function workloadOfField(field?: string): string {
  const match = /^(?:variables|secrets)\.([^/]+)\//.exec(field ?? '');
  return match?.[1] ?? '';
}

export function validateDraft(draft: ConfigurationDraft): Problem[] {
  const problems: Problem[] = [];
  const add = (field: string, code: string, message: string) => problems.push({ field, code, message });

  const check = (kind: BindingKind, binding: BindingDraft) => {
    const field = fieldPath(kind, binding);
    const label = binding.name;
    switch (binding.source) {
      case '':
        if (binding.required) add(field, 'MISSING_REQUIRED_CONFIGURATION', `${label} is required but has no value source.`);
        return;
      case 'DIRECT':
        if (binding.value === '') add(field, 'MISSING_REQUIRED_CONFIGURATION', `${label} has an empty value.`);
        return;
      case 'SECRET_REF':
        if (binding.secretRef === '') {
          add(
            field,
            'SECRET_NOT_STORED',
            binding.value === ''
              ? `${label} has no secret value yet.`
              : `${label} was typed but not stored yet. Choose Store secret so the IDP keeps only a reference.`,
          );
        }
        return;
      case 'RESOURCE_OUTPUT':
      case 'WORKLOAD_OUTPUT': {
        const component = binding.source === 'RESOURCE_OUTPUT' ? 'resource' : 'workload';
        if (binding.refId === '') add(field, 'MISSING_REQUIRED_CONFIGURATION', `${label} has no ${component} selected.`);
        else if (binding.outputName === '') add(field, 'MISSING_REQUIRED_CONFIGURATION', `${label} has no output selected.`);
        return;
      }
    }
  };

  for (const binding of draft.variables) check('variable', binding);
  for (const binding of draft.secrets) check('secret', binding);
  if (draft.catalogVersion === '') add('catalogVersion', 'INVALID_INPUT', 'Choose a platform catalog version.');
  if (draft.deploymentTarget === '') add('deploymentTarget', 'UNSUPPORTED_TARGET_OR_CONTEXT', 'Choose a deployment target.');
  return problems;
}
