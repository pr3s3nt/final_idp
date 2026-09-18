import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigurationPage } from './ConfigurationPage';
import { draftKey } from '../draft/storage';
import {
  APP_ID,
  BACKEND_URL_ID,
  FRONTEND_ID,
  DB_HOST_ID,
  SECRET_ID,
  fakeConfigurationApi,
  requirementsDto,
  type FakeConfigurationApi,
} from '../../../test/configuration-fixtures';

async function open(api: FakeConfigurationApi = fakeConfigurationApi()) {
  render(<ConfigurationPage api={api} applicationId={APP_ID} />);
  await screen.findByRole('heading', { name: /Configuration · shop-app/ });
  return { api, user: userEvent.setup() };
}

const sourceOf = (name: string) => screen.getByLabelText(new RegExp(`^${name}`));

describe('UC-02 Configuration page', () => {
  test('shows what UC-01 declared, per workload', async () => {
    await open();
    expect(screen.getByRole('heading', { name: 'backend' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'frontend' })).toBeInTheDocument();
    expect(sourceOf('DB_HOST')).toBeInTheDocument();
    expect(sourceOf('DB_PASSWORD')).toBeInTheDocument();
    expect(sourceOf('BACKEND_URL')).toBeInTheDocument();
  });

  test('preselects the newest catalog version and asks for a deployment target', async () => {
    await open();
    expect(screen.getByLabelText('Platform catalog version')).toHaveValue('2');
    expect(screen.getByLabelText('Deployment target')).toHaveValue('kind-local');
  });

  test('offers only the outputs of the definition the selected target resolves', async () => {
    const { api, user } = await open();
    await user.selectOptions(sourceOf('DB_HOST'), 'RESOURCE_OUTPUT');
    await user.selectOptions(screen.getByLabelText('Resource'), 'postgresql');

    await waitFor(() => expect(api.resourceOutputs).toHaveBeenCalled());
    expect(api.resourceOutputs.mock.calls[0]?.[0]).toMatchObject({ catalogVersion: '2', target: 'kind-local' });
    const outputs = await screen.findByLabelText('Output');
    // A variable may use the normal outputs, never the sensitive one.
    expect([...outputs.querySelectorAll('option')].map((o) => o.textContent)).toEqual(['Choose an output', 'host', 'port']);
  });

  test('a secret may use only a sensitive output', async () => {
    const { user } = await open();
    await user.selectOptions(sourceOf('DB_PASSWORD'), 'RESOURCE_OUTPUT');
    await user.selectOptions(screen.getByLabelText('Resource'), 'postgresql');

    const outputs = await screen.findByLabelText('Output');
    expect([...outputs.querySelectorAll('option')].map((o) => o.textContent)).toEqual(['Choose an output', 'password']);
  });

  test('a workload may only reference a component it depends on', async () => {
    const { user } = await open();
    await user.selectOptions(sourceOf('BACKEND_URL'), 'WORKLOAD_OUTPUT');

    const workloads = screen.getByLabelText('Workload');
    expect([...workloads.querySelectorAll('option')].map((o) => o.textContent)).toEqual([
      'Choose a component this workload depends on',
      'backend',
    ]);
  });

  test('a typed Secret is exchanged for an opaque reference and never stored in the browser', async () => {
    const { api, user } = await open();
    await user.selectOptions(sourceOf('DB_PASSWORD'), 'SECRET_REF');
    await user.type(screen.getByLabelText('Secret value'), 'hunter2');
    await user.click(screen.getByRole('button', { name: 'Store secret' }));

    await waitFor(() => expect(api.stageSecret).toHaveBeenCalled());
    expect(api.stageSecret.mock.calls[0]?.[0]).toMatchObject({ definitionId: SECRET_ID, value: 'hunter2' });
    expect(await screen.findByText(/Stored\. The IDP keeps only a reference/)).toBeInTheDocument();
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).not.toContain('hunter2');
  });

  test('Save sends the complete draft and clears the browser draft', async () => {
    const { api, user } = await open();
    await user.selectOptions(sourceOf('DB_HOST'), 'DIRECT');
    await user.type(screen.getByLabelText('Value'), 'db.internal');
    await user.selectOptions(sourceOf('BACKEND_URL'), 'DIRECT');
    await user.type(screen.getAllByLabelText('Value')[1]!, 'http://backend');
    await user.selectOptions(sourceOf('DB_PASSWORD'), 'SECRET_REF');
    await user.type(screen.getByLabelText('Secret value'), 'hunter2');
    await user.click(screen.getByRole('button', { name: 'Store secret' }));
    await screen.findByText(/Stored\./);

    await user.click(screen.getByRole('button', { name: 'Save configuration' }));

    await waitFor(() => expect(api.saveConfiguration).toHaveBeenCalled());
    const sent = api.saveConfiguration.mock.calls[0]?.[0];
    expect(sent).toMatchObject({ baseApplicationDefinitionVersion: 2, baseConfigurationRevision: 'rev-1', catalogVersion: '2', deploymentTarget: 'kind-local' });
    expect(sent?.variables).toHaveLength(2);
    expect(JSON.stringify(sent)).not.toContain('hunter2');
    expect(await screen.findByText(/Environment configuration saved/)).toBeInTheDocument();
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).toBeNull();
  });

  test('Save is blocked while a required value has no source', async () => {
    await open();
    expect(screen.getByRole('button', { name: 'Save configuration' })).toBeDisabled();
    expect(screen.getByText(/3 required value\(s\) still have no source/)).toBeInTheDocument();
  });

  test('a stale draft is reported as a concurrent edit, not merged', async () => {
    const api = fakeConfigurationApi({
      saveConfiguration: vi.fn(async () => ({
        kind: 'conflict' as const,
        problems: [{ code: 'DRAFT_CONFLICT', message: 'the configuration changed while this draft was saved' }],
      })),
    });
    const { user } = await open(api);
    await fillEverything(user);

    await user.click(screen.getByRole('button', { name: 'Save configuration' }));

    expect(await screen.findByText(/The configuration changed while you were editing/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Reload the configuration' })).toBeInTheDocument();
    // The browser keeps the draft so the Developer can reapply the changes.
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).not.toBeNull();
  });

  test('an invalid configuration is reported per problem', async () => {
    const api = fakeConfigurationApi({
      saveConfiguration: vi.fn(async () => ({
        kind: 'validation' as const,
        problems: [{ code: 'INVALID_OUTPUT_REFERENCE', message: 'backend.DB_HOST references output "reader_host"' }],
      })),
    });
    const { user } = await open(api);
    await fillEverything(user);

    await user.click(screen.getByRole('button', { name: 'Save configuration' }));

    expect(await screen.findByText(/references output "reader_host"/)).toBeInTheDocument();
  });

  test('switching environment loads that environment', async () => {
    const { api, user } = await open();
    await user.selectOptions(screen.getByLabelText('Environment'), 'PRODUCTION');

    await waitFor(() => expect(api.selectEnvironment).toHaveBeenCalledWith(APP_ID, 'PRODUCTION'));
  });

  test('a saved configuration comes back prefilled', async () => {
    const api = fakeConfigurationApi({
      selectEnvironment: vi.fn(async () => ({
        kind: 'ok' as const,
        data: requirementsDto({
          configuration: {
            applicationId: APP_ID,
            environment: 'STAGING',
            baseApplicationDefinitionVersion: 2,
            baseConfigurationRevision: 'rev-1',
            catalogVersion: '',
            deploymentTarget: '',
            variables: [{ workloadId: FRONTEND_ID, definitionId: BACKEND_URL_ID, name: 'BACKEND_URL', source: 'DIRECT', value: 'http://backend' }],
            secrets: [],
          },
        }),
      })),
    });
    await open(api);

    expect(sourceOf('BACKEND_URL')).toHaveValue('DIRECT');
  });
});

/** Gives every required requirement a value so Save is enabled. */
async function fillEverything(user: ReturnType<typeof userEvent.setup>) {
  await user.selectOptions(sourceOf('DB_HOST'), 'DIRECT');
  await user.type(screen.getByLabelText('Value'), 'db.internal');
  await user.selectOptions(sourceOf('BACKEND_URL'), 'DIRECT');
  await user.type(screen.getAllByLabelText('Value')[1]!, 'http://backend');
  await user.selectOptions(sourceOf('DB_PASSWORD'), 'SECRET_REF');
  await user.type(screen.getByLabelText('Secret value'), 'hunter2');
  await user.click(screen.getByRole('button', { name: 'Store secret' }));
  await screen.findByText(/Stored\./);
  void DB_HOST_ID;
}
