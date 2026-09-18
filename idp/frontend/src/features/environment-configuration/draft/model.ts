import type {
  BindingDto,
  ConfigurationRequirementsDto,
  EnvironmentConfigurationDto,
  ValueSource,
} from '../api/types';

// EnvironmentConfigurationDraft as held by the browser tab (ADR-016). One entry
// exists for every Environment Variable and Secret the latest Application
// Definition version declares, so the draft always mirrors what UC-01 asked
// for.

export const ENVIRONMENTS = ['STAGING', 'PRODUCTION'] as const;
export type Environment = (typeof ENVIRONMENTS)[number];

export interface BindingDraft {
  workloadId: string;
  definitionId: string;
  name: string;
  required: boolean;
  /** Empty until the Developer picks a value source. */
  source: ValueSource | '';
  /** Direct value of a variable, or the typed Secret value awaiting staging. */
  value: string;
  /** Opaque reference returned by staging; the only Secret data that is kept. */
  secretRef: string;
  /** Referenced Resource Requirement ID or Workload ID. */
  refId: string;
  outputName: string;
}

export interface ConfigurationDraft {
  applicationId: string;
  environment: Environment;
  baseApplicationDefinitionVersion: number;
  baseConfigurationRevision: string;
  /** Chosen by the Developer; decides which Resource Definition applies. */
  catalogVersion: string;
  deploymentTarget: string;
  variables: BindingDraft[];
  secrets: BindingDraft[];
}

/** Placeholder draft used before the environment has been loaded. */
export function emptyDraft(environment: Environment): ConfigurationDraft {
  return {
    applicationId: '',
    environment,
    baseApplicationDefinitionVersion: 0,
    baseConfigurationRevision: '',
    catalogVersion: '',
    deploymentTarget: '',
    variables: [],
    secrets: [],
  };
}

export function isEnvironment(value: string): value is Environment {
  return (ENVIRONMENTS as readonly string[]).includes(value.toUpperCase());
}

export function bindingKey(workloadId: string, definitionId: string): string {
  return `${workloadId}/${definitionId}`;
}

function emptyBinding(workloadId: string, definitionId: string, name: string, required: boolean): BindingDraft {
  return { workloadId, definitionId, name, required, source: '', value: '', secretRef: '', refId: '', outputName: '' };
}

function fromDto(binding: BindingDto, required: boolean): Partial<BindingDraft> {
  return {
    source: binding.source,
    // A stored Secret has a reference, never a value.
    value: binding.value ?? '',
    secretRef: binding.secretRef ?? '',
    refId: binding.refId ?? '',
    outputName: binding.outputName ?? '',
    required,
  };
}

/**
 * Builds the draft for one environment: every declared requirement, prefilled
 * with the stored binding when the environment already has one.
 */
export function draftFromRequirements(dto: ConfigurationRequirementsDto, catalogVersion: string, target: string): ConfigurationDraft {
  const stored = new Map<string, BindingDto>();
  for (const b of dto.configuration?.variables ?? []) stored.set('variable:' + bindingKey(b.workloadId, b.definitionId), b);
  for (const b of dto.configuration?.secrets ?? []) stored.set('secret:' + bindingKey(b.workloadId, b.definitionId), b);

  const variables: BindingDraft[] = [];
  const secrets: BindingDraft[] = [];
  for (const w of dto.workloads) {
    for (const def of w.variables) {
      const base = emptyBinding(w.id, def.id, def.name, def.required);
      const saved = stored.get('variable:' + bindingKey(w.id, def.id));
      variables.push(saved ? { ...base, ...fromDto(saved, def.required) } : base);
    }
    for (const def of w.secrets) {
      const base = emptyBinding(w.id, def.id, def.name, def.required);
      const saved = stored.get('secret:' + bindingKey(w.id, def.id));
      secrets.push(saved ? { ...base, ...fromDto(saved, def.required) } : base);
    }
  }
  return {
    applicationId: dto.applicationId,
    environment: (isEnvironment(dto.environment) ? dto.environment.toUpperCase() : 'STAGING') as Environment,
    baseApplicationDefinitionVersion: dto.baseApplicationDefinitionVersion,
    baseConfigurationRevision: dto.baseConfigurationRevision,
    catalogVersion,
    deploymentTarget: target,
    variables,
    secrets,
  };
}

function bindingToDto(b: BindingDraft): BindingDto {
  const dto: BindingDto = { workloadId: b.workloadId, definitionId: b.definitionId, name: b.name, source: b.source as ValueSource };
  if (b.source === 'DIRECT') dto.value = b.value;
  if (b.source === 'SECRET_REF') dto.secretRef = b.secretRef;
  if (b.source === 'RESOURCE_OUTPUT' || b.source === 'WORKLOAD_OUTPUT') {
    dto.refId = b.refId;
    dto.outputName = b.outputName;
  }
  return dto;
}

/** Builds the complete Save request; bindings without a source are left out. */
export function draftToDto(draft: ConfigurationDraft): EnvironmentConfigurationDto {
  return {
    applicationId: draft.applicationId,
    environment: draft.environment,
    baseApplicationDefinitionVersion: draft.baseApplicationDefinitionVersion,
    baseConfigurationRevision: draft.baseConfigurationRevision,
    catalogVersion: draft.catalogVersion,
    deploymentTarget: draft.deploymentTarget,
    variables: draft.variables.filter((b) => b.source !== '').map(bindingToDto),
    secrets: draft.secrets.filter((b) => b.source !== '').map(bindingToDto),
  };
}
