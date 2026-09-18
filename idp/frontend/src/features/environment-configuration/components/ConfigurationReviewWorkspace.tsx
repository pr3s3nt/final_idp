import type { ConfigurationRequirementsDto } from '../api/types';
import type { BindingDraft, ConfigurationDraft } from '../draft/model';

const sourceLabels: Record<string, string> = {
  '': 'Not configured',
  DIRECT: 'Environment value',
  RESOURCE_OUTPUT: 'Resource output',
  WORKLOAD_OUTPUT: 'Workload output',
  SECRET_REF: 'Secret value',
};

/** Read-only summary of what will be saved. A Secret value is never shown. */
export function ConfigurationReviewWorkspace({
  draft,
  requirements,
}: {
  draft: ConfigurationDraft;
  requirements: ConfigurationRequirementsDto;
}) {
  const nameOf = (id: string) =>
    requirements.resources.find((resource) => resource.id === id)?.name ??
    requirements.workloads.find((workload) => workload.id === id)?.name ??
    id;
  const workloadName = (id: string) => requirements.workloads.find((workload) => workload.id === id)?.name ?? id;

  const valueOf = (binding: BindingDraft) => {
    switch (binding.source) {
      case 'DIRECT':
        return binding.value;
      case 'RESOURCE_OUTPUT':
      case 'WORKLOAD_OUTPUT':
        return binding.refId && binding.outputName ? `${nameOf(binding.refId)}.${binding.outputName}` : '—';
      case 'SECRET_REF':
        return binding.secretRef ? 'Stored reference' : 'Not stored yet';
      default:
        return '—';
    }
  };

  const rows = [
    ...draft.variables.map((binding) => ({ binding, secret: false })),
    ...draft.secrets.map((binding) => ({ binding, secret: true })),
  ];

  return (
    <section className="workspace" aria-labelledby="configuration-review-title">
      <h2 id="configuration-review-title">Review</h2>
      <p className="muted">
        Scope: environment <strong className="mono">{draft.environment.toLowerCase()}</strong>, catalog version{' '}
        <strong className="mono">v{draft.catalogVersion}</strong>, deployment target{' '}
        <strong className="mono">{draft.deploymentTarget}</strong>. The catalog version and the target are not saved with the
        configuration. Saving does not deploy the application.
      </p>

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th scope="col">Workload</th>
              <th scope="col">Requirement</th>
              <th scope="col">Source</th>
              <th scope="col">Value</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(({ binding, secret }) => (
              <tr key={`${secret ? 'secret' : 'variable'}:${binding.workloadId}/${binding.definitionId}`}>
                <td className="mono">{workloadName(binding.workloadId)}</td>
                <td className="mono">
                  {binding.name}
                  {binding.required && <span className="binding-required"> *</span>}
                  {secret && <span className="binding-tag">secret</span>}
                </td>
                <td>{sourceLabels[binding.source] ?? binding.source}</td>
                <td className={binding.source === 'SECRET_REF' ? undefined : 'mono'}>{valueOf(binding)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
