import type { ConfigurationWorkloadDto } from '../api/types';
import type { ConfigurationDraft } from '../draft/model';
import { BindingField, type BindingFieldProps } from './BindingField';

type Shared = Omit<BindingFieldProps, 'kind' | 'binding' | 'workload' | 'staging'>;

interface Props extends Shared {
  workload: ConfigurationWorkloadDto;
  draft: ConfigurationDraft;
  /** Binding key currently being staged, or ''. */
  stagingKey: string;
}

/** Every Environment Variable and Secret one workload declares. */
export function WorkloadConfigurationWorkspace({ workload, draft, stagingKey, ...shared }: Props) {
  const variables = draft.variables.filter((binding) => binding.workloadId === workload.id);
  const secrets = draft.secrets.filter((binding) => binding.workloadId === workload.id);

  return (
    <section className="workspace" aria-labelledby="configuration-workspace-title">
      <h2 id="configuration-workspace-title" className="mono">
        {workload.name}
      </h2>

      {variables.length > 0 && (
        <>
          <h3>Environment variables</h3>
          {variables.map((binding) => (
            <BindingField
              key={binding.definitionId}
              kind="variable"
              binding={binding}
              workload={workload}
              staging={false}
              {...shared}
            />
          ))}
        </>
      )}

      {secrets.length > 0 && (
        <>
          <h3>Secrets</h3>
          {secrets.map((binding) => (
            <BindingField
              key={binding.definitionId}
              kind="secret"
              binding={binding}
              workload={workload}
              staging={stagingKey === `${binding.workloadId}/${binding.definitionId}`}
              {...shared}
            />
          ))}
        </>
      )}

      {variables.length === 0 && secrets.length === 0 && (
        <p className="muted">This workload declares no environment variable and no secret.</p>
      )}
    </section>
  );
}
