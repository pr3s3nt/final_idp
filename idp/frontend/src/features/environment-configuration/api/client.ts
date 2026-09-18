import type {
  ConfigurationRequirementsDto,
  EnvironmentConfigurationDto,
  ResourceOutputsDto,
  WorkloadOutputsDto,
} from './types';
import { call, jsonRequest, type ApiResult } from '../../../shared/api/result';

/**
 * The only backend calls UC-02 makes (ADR-016): load the environment, query
 * the catalog, stage a Secret value, and Save. Field and binding edits stay in
 * the browser.
 */
export interface ConfigurationApi {
  selectEnvironment(applicationId: string, environment: string): Promise<ApiResult<ConfigurationRequirementsDto>>;
  resourceOutputs(input: {
    applicationId: string;
    environment: string;
    workloadId: string;
    resourceId: string;
    catalogVersion: string;
    target: string;
  }): Promise<ApiResult<ResourceOutputsDto>>;
  workloadOutputs(input: {
    applicationId: string;
    environment: string;
    workloadId: string;
    targetWorkloadId: string;
  }): Promise<ApiResult<WorkloadOutputsDto>>;
  stageSecret(input: {
    applicationId: string;
    environment: string;
    workloadId: string;
    definitionId: string;
    value: string;
  }): Promise<ApiResult<{ secretRef: string }>>;
  saveConfiguration(draft: EnvironmentConfigurationDto): Promise<ApiResult<EnvironmentConfigurationDto>>;
}

function base(applicationId: string, environment: string): string {
  return `/api/environment-configurations/${encodeURIComponent(applicationId)}/${encodeURIComponent(environment.toLowerCase())}`;
}

export const httpConfigurationApi: ConfigurationApi = {
  selectEnvironment(applicationId, environment) {
    return call<ConfigurationRequirementsDto>(base(applicationId, environment));
  },
  resourceOutputs({ applicationId, environment, workloadId, resourceId, catalogVersion, target }) {
    const query = new URLSearchParams({ workloadId, catalogVersion, target });
    return call<ResourceOutputsDto>(`${base(applicationId, environment)}/resources/${encodeURIComponent(resourceId)}/outputs?${query}`);
  },
  workloadOutputs({ applicationId, environment, workloadId, targetWorkloadId }) {
    const query = new URLSearchParams({ workloadId });
    return call<WorkloadOutputsDto>(`${base(applicationId, environment)}/workloads/${encodeURIComponent(targetWorkloadId)}/outputs?${query}`);
  },
  stageSecret({ applicationId, environment, workloadId, definitionId, value }) {
    return call<{ secretRef: string }>(`${base(applicationId, environment)}/secrets`, jsonRequest('POST', { workloadId, definitionId, value }));
  },
  saveConfiguration(draft) {
    return call<EnvironmentConfigurationDto>(base(draft.applicationId, draft.environment), jsonRequest('PUT', draft));
  },
};
