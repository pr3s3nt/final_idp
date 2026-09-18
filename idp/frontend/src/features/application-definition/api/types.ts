// JSON contract of the UC-01 Application Definition API (ADR-017). These types
// mirror idp/backend/internal/service/application.go.

export interface Problem {
  code: string;
  message: string;
  /** Input path such as `workloads.<id>.name`, when the problem has one. */
  field?: string;
}

export interface ConfigRequirementDto {
  id: string;
  name: string;
  required: boolean;
}

export interface WorkloadDto {
  id: string;
  name: string;
  type: string;
  imageRepository: string;
  port: number | null;
  outputs: string[];
  variables: ConfigRequirementDto[];
  /** Secret requirements are names only; UC-01 never carries a Secret value. */
  secrets: ConfigRequirementDto[];
}

export interface ResourceDto {
  id: string;
  name: string;
  type: string;
}

export interface DependencyDto {
  id: string;
  sourceId: string;
  targetId: string;
}

/** ApplicationDefinitionDraft as sent on Save and returned for editing. */
export interface ApplicationDefinitionDto {
  applicationId?: string;
  baseVersion: number | null;
  name: string;
  description: string;
  workloads: WorkloadDto[];
  resources: ResourceDto[];
  dependencies: DependencyDto[];
}

export interface ApplicationListItem {
  applicationId: string;
  name: string;
  description: string;
  latestVersion: number;
}
