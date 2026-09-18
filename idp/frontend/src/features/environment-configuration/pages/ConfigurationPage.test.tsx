import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigurationPage } from './ConfigurationPage';
import { draftKey } from '../draft/storage';
import {
  APP_ID,
  BACKEND_URL_ID,
  FRONTEND_ID,
  SECRET_ID,
  fakeConfigurationApi,
  requirementsDto,
  type FakeConfigurationApi,
} from '../../../test/configuration-fixtures';

type User = ReturnType<typeof userEvent.setup>;

async function open(api: FakeConfigurationApi = fakeConfigurationApi()) {
  render(<ConfigurationPage api={api} applicationId={APP_ID} />);
  await screen.findByRole('heading', { level: 1, name: /Configuration · shop-app/ });
  return { api, user: userEvent.setup() };
}

/** The controls of one requirement; each block is a group named after it. */
function binding(name: string) {
  return within(screen.getByRole('group', { name: new RegExp(`^${name}`) }));
}

const goTo = (user: User, label: string) => user.click(screen.getByRole('button', { name: new RegExp(`^${label}`) }));

describe('UC-02 Configuration page', () => {
  test('opens on the first workload and shows only its requirements', async () => {
    await open();
    expect(screen.getByRole('heading', { level: 2, name: 'backend' })).toBeInTheDocument();
    expect(binding('DB_HOST').getByLabelText('Source')).toBeInTheDocument();
    expect(binding('DB_PASSWORD').getByLabelText('Source')).toBeInTheDocument();
    // BACKEND_URL belongs to frontend, so it is not in this workspace.
    expect(screen.queryByRole('group', { name: /^BACKEND_URL/ })).not.toBeInTheDocument();
  });

  test('navigation switches workload', async () => {
    const { user } = await open();
    await goTo(user, 'frontend');

    expect(screen.getByRole('heading', { level: 2, name: 'frontend' })).toBeInTheDocument();
    expect(binding('BACKEND_URL').getByLabelText('Source')).toBeInTheDocument();
  });

  test('preselects the newest catalog version and a deployment target', async () => {
    await open();
    expect(screen.getByLabelText('Platform catalog version')).toHaveValue('2');
    expect(screen.getByLabelText('Deployment target')).toHaveValue('kind-local');
  });

  test('offers only the outputs of the definition the selected target resolves', async () => {
    const { api, user } = await open();
    await user.selectOptions(binding('DB_HOST').getByLabelText('Source'), 'RESOURCE_OUTPUT');
    await user.selectOptions(binding('DB_HOST').getByLabelText('Resource'), 'postgresql');

    await waitFor(() => expect(api.resourceOutputs).toHaveBeenCalled());
    expect(api.resourceOutputs.mock.calls[0]?.[0]).toMatchObject({ catalogVersion: '2', target: 'kind-local' });
    const outputs = await binding('DB_HOST').findByLabelText('Output');
    // A variable may use the normal outputs, never the sensitive one.
    expect([...outputs.querySelectorAll('option')].map((option) => option.textContent)).toEqual(['Choose an output', 'host', 'port']);
  });

  test('a secret may use only a sensitive output', async () => {
    const { user } = await open();
    await user.selectOptions(binding('DB_PASSWORD').getByLabelText('Source'), 'RESOURCE_OUTPUT');
    await user.selectOptions(binding('DB_PASSWORD').getByLabelText('Resource'), 'postgresql');

    const outputs = await binding('DB_PASSWORD').findByLabelText('Output');
    expect([...outputs.querySelectorAll('option')].map((option) => option.textContent)).toEqual(['Choose an output', 'password']);
  });

  test('a workload may only reference a component it depends on', async () => {
    const { user } = await open();
    await goTo(user, 'frontend');
    await user.selectOptions(binding('BACKEND_URL').getByLabelText('Source'), 'WORKLOAD_OUTPUT');

    const workloads = binding('BACKEND_URL').getByLabelText('Workload');
    expect([...workloads.querySelectorAll('option')].map((option) => option.textContent)).toEqual([
      'Choose a workload this workload depends on',
      'backend',
    ]);
  });

  test('a typed Secret is exchanged for an opaque reference and never stored in the browser', async () => {
    const { api, user } = await open();
    await user.selectOptions(binding('DB_PASSWORD').getByLabelText('Source'), 'SECRET_REF');
    await user.type(binding('DB_PASSWORD').getByLabelText('Secret value'), 'hunter2');
    await user.click(binding('DB_PASSWORD').getByRole('button', { name: 'Store secret' }));

    await waitFor(() => expect(api.stageSecret).toHaveBeenCalled());
    expect(api.stageSecret.mock.calls[0]?.[0]).toMatchObject({ definitionId: SECRET_ID, value: 'hunter2' });
    expect(await screen.findByText(/Stored\. The IDP keeps only a reference/)).toBeInTheDocument();
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).not.toContain('hunter2');
  });

  test('the header reports unsaved changes kept in this tab', async () => {
    const { user } = await open();
    expect(screen.getByText('Saved')).toBeInTheDocument();

    await user.selectOptions(binding('DB_HOST').getByLabelText('Source'), 'DIRECT');
    await user.type(binding('DB_HOST').getByLabelText('Value'), 'db.internal');

    expect(await screen.findByText('Unsaved changes · kept in this tab')).toBeInTheDocument();
  });

  test('Save with a missing required value opens Review instead of calling the API', async () => {
    const { api, user } = await open();
    await user.click(screen.getByRole('button', { name: 'Save configuration' }));

    expect(await screen.findByRole('heading', { level: 2, name: 'Review' })).toBeInTheDocument();
    expect(screen.getByText(/3 problems prevent saving/)).toBeInTheDocument();
    expect(api.saveConfiguration).not.toHaveBeenCalled();
  });

  test('a problem in the summary leads back to the workload that owns it', async () => {
    const { user } = await open();
    await user.click(screen.getByRole('button', { name: 'Save configuration' }));
    await screen.findByRole('heading', { level: 2, name: 'Review' });

    await user.click(screen.getByRole('button', { name: /BACKEND_URL is required/ }));

    expect(screen.getByRole('heading', { level: 2, name: 'frontend' })).toBeInTheDocument();
  });

  test('a typed Secret that was not stored blocks Save', async () => {
    const { api, user } = await open();
    await fillVariables(user);
    await user.selectOptions(binding('DB_PASSWORD').getByLabelText('Source'), 'SECRET_REF');
    await user.type(binding('DB_PASSWORD').getByLabelText('Secret value'), 'hunter2');

    await user.click(screen.getByRole('button', { name: 'Save configuration' }));

    expect(await screen.findByText(/was typed but not stored yet/)).toBeInTheDocument();
    expect(api.saveConfiguration).not.toHaveBeenCalled();
  });

  test('Save sends the complete draft and clears the browser draft', async () => {
    const { api, user } = await open();
    await fillEverything(user);

    await user.click(screen.getByRole('button', { name: 'Save configuration' }));

    await waitFor(() => expect(api.saveConfiguration).toHaveBeenCalled());
    const sent = api.saveConfiguration.mock.calls[0]?.[0];
    expect(sent).toMatchObject({
      baseApplicationDefinitionVersion: 2,
      baseConfigurationRevision: 'rev-1',
      catalogVersion: '2',
      deploymentTarget: 'kind-local',
    });
    expect(sent?.variables).toHaveLength(2);
    expect(sent?.secrets).toEqual([
      { workloadId: expect.any(String), definitionId: SECRET_ID, name: 'DB_PASSWORD', source: 'SECRET_REF', secretRef: 'idpsecret://shop/staging/db' },
    ]);
    expect(JSON.stringify(sent)).not.toContain('hunter2');
    expect(await screen.findByText(/Environment configuration saved/)).toBeInTheDocument();
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).toBeNull();
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

    expect(await screen.findByRole('heading', { name: /The configuration changed while you were editing/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Reload configuration' })).toBeInTheDocument();
    // The browser keeps the draft so the Developer can reapply the changes.
    expect(sessionStorage.getItem(draftKey(APP_ID, 'STAGING'))).not.toBeNull();
  });

  test('a rejected configuration is reported per problem in Review', async () => {
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
            variables: [
              { workloadId: FRONTEND_ID, definitionId: BACKEND_URL_ID, name: 'BACKEND_URL', source: 'DIRECT', value: 'http://backend' },
            ],
            secrets: [],
          },
        }),
      })),
    });
    const { user } = await open(api);
    await goTo(user, 'frontend');

    expect(binding('BACKEND_URL').getByLabelText('Source')).toHaveValue('DIRECT');
    expect(binding('BACKEND_URL').getByLabelText('Value')).toHaveValue('http://backend');
  });

  test('Review lists what will be saved without showing a secret value', async () => {
    const { user } = await open();
    await fillEverything(user);
    await goTo(user, 'Review');

    const rows = within(screen.getByRole('table')).getAllByRole('row');
    expect(rows).toHaveLength(4); // header + two variables + one secret
    expect(within(screen.getByRole('table')).getByText('Stored reference')).toBeInTheDocument();
    expect(screen.queryByText('hunter2')).not.toBeInTheDocument();
  });
});

/** Gives both environment variables a direct value. */
async function fillVariables(user: User) {
  await user.selectOptions(binding('DB_HOST').getByLabelText('Source'), 'DIRECT');
  await user.type(binding('DB_HOST').getByLabelText('Value'), 'db.internal');
  await goTo(user, 'frontend');
  await user.selectOptions(binding('BACKEND_URL').getByLabelText('Source'), 'DIRECT');
  await user.type(binding('BACKEND_URL').getByLabelText('Value'), 'http://backend');
  await goTo(user, 'backend');
}

/** Gives every required requirement a value, staging the secret. */
async function fillEverything(user: User) {
  await fillVariables(user);
  await user.selectOptions(binding('DB_PASSWORD').getByLabelText('Source'), 'SECRET_REF');
  await user.type(binding('DB_PASSWORD').getByLabelText('Secret value'), 'hunter2');
  await user.click(binding('DB_PASSWORD').getByRole('button', { name: 'Store secret' }));
  await screen.findByText(/Stored\./);
}
