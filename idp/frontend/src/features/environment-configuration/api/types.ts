// JSON contract of the UC-02 Environment Configuration API. These types mirror
// idp/backend/internal/service/configuration.go.

import type { ConfigRequirementDto } from '../../application-definition/api/types';

export type ValueSource = 'DIRECT' | 'RESOURCE_OUTPUT' | 'WORKLOAD_OUTPUT' | 'SECRET_REF';

/** One configured Environment Variable or Secret with exactly one source. */
export interface BindingDto {
  workloadId: string;
  definitionId: string;
  name: string;
  source: ValueSource;
  /** Direct value of a non-secret variable. */
  value?: string;
  /** Opaque reference the Secret Store returned; never a plaintext Secret. */
  secretRef?: string;
  /** Referenced Resource Requirement ID or Workload ID. */
  refId?: string;
  outputName?: string;
}

/** EnvironmentConfigurationDraft as sent on Save and returned after it. */
export interface EnvironmentConfigurationDto {
  applicationId: string;
  environment: string;
  baseApplicationDefinitionVersion: number;
  baseConfigurationRevision: string;
  /** Selects which Resource Definition applies; not persisted (ADR-020). */
  catalogVersion: string;
  deploymentTarget: string;
  variables: BindingDto[];
  secrets: BindingDto[];
}

export interface ConfigurationWorkloadDto {
  id: string;
  name: string;
  variables: ConfigRequirementDto[];
  secrets: ConfigRequirementDto[];
  /** Only these components may supply an output for this workload. */
  dependsOnResources: string[];
  dependsOnWorkloads: string[];
}

export interface ConfigurationResourceDto {
  id: string;
  name: string;
  type: string;
}

export interface CatalogVersionDto {
  id: string;
  number: number;
  createdAt: string;
}

export interface TargetOptionDto {
  target: string;
  cloudProvider: string;
  region?: string;
}

/** What selectEnvironment() returns. */
export interface ConfigurationRequirementsDto {
  applicationId: string;
  applicationName: string;
  environment: string;
  baseApplicationDefinitionVersion: number;
  baseConfigurationRevision: string;
  workloads: ConfigurationWorkloadDto[];
  resources: ConfigurationResourceDto[];
  catalogVersions: CatalogVersionDto[];
  targets: TargetOptionDto[];
  configuration: EnvironmentConfigurationDto | null;
}

export interface OutputOptionDto {
  name: string;
  sensitive: boolean;
}

export interface ResourceOutputsDto {
  resourceId: string;
  resourceName: string;
  definitionName: string;
  outputs: OutputOptionDto[];
}

export interface WorkloadOutputsDto {
  workloadId: string;
  workloadName: string;
  outputs: OutputOptionDto[];
}
