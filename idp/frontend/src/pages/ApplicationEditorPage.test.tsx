import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { App } from '../App';
import { draftKey } from '../draft/storage';
import { APP_ID, BACKEND_ID, fakeApi, shopDto } from '../test/fixtures';

function open(path: string, api = fakeApi()) {
  window.history.replaceState(null, '', path);
  const user = userEvent.setup();
  const view = render(<App api={api} />);
  return { api, user, ...view };
}

const stored = (id: string | null) => sessionStorage.getItem(draftKey(id));

async function fillNewApplication(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/^Application name/), 'orders-app');
  await user.click(screen.getByRole('button', { name: 'Add workload' }));
  const workload = screen.getByRole('listitem', { name: 'unnamed workload' });
  await user.type(within(workload).getByLabelText(/^Workload name/), 'api');
  await user.type(within(workload).getByLabelText(/^Workload type/), 'Backend Service');
  await user.type(within(workload).getByLabelText(/^Image repository/), 'registry.company.local/orders-api');
  await user.type(within(workload).getByLabelText(/^Port/), '8080');
  await user.click(screen.getByRole('button', { name: 'Add output to workload api' }));
  await user.type(screen.getByLabelText('Output 1 name'), 'endpoint');

  await user.click(screen.getByRole('button', { name: 'Add resource' }));
  const resource = screen.getByRole('listitem', { name: 'unnamed resource' });
  await user.type(within(resource).getByLabelText(/^Resource name/), 'ordersdb');
  await user.type(within(resource).getByLabelText(/^Resource type/), 'PostgreSQL');

  await user.click(screen.getByRole('button', { name: 'Add Environment Variable to workload api' }));
  await user.type(screen.getByLabelText('Environment Variable name'), 'DB_HOST');
  await user.click(screen.getByRole('button', { name: 'Add Secret to workload api' }));
  await user.type(screen.getByLabelText('Secret name'), 'DB_PASSWORD');

  const newDependency = screen.getByRole('group', { name: 'New dependency' });
  await user.selectOptions(within(newDependency).getByLabelText('Workload'), 'Workload api');
  await user.selectOptions(within(newDependency).getByLabelText('Depends on'), 'Resource ordersdb');
  await user.click(screen.getByRole('button', { name: 'Add dependency' }));
}

describe('create application', () => {
  test('builds a complete draft locally and saves it in one request', async () => {
    const { api, user } = open('/ui/applications/new');
    await fillNewApplication(user);

    expect(api.saveApplication).not.toHaveBeenCalled();
    expect(api.loadApplication).not.toHaveBeenCalled();
    expect(screen.getByText('Unsaved changes (kept in this browser tab)')).toBeInTheDocument();
    expect(JSON.parse(stored(null) ?? '{}').draft.name).toBe('orders-app');

    await user.click(screen.getByRole('button', { name: 'Save application' }));

    expect(api.saveApplication).toHaveBeenCalledTimes(1);
    const sent = api.saveApplication.mock.calls[0]![0];
    expect(sent).toMatchObject({
      baseVersion: null,
      name: 'orders-app',
      workloads: [{ name: 'api', type: 'Backend Service', imageRepository: 'registry.company.local/orders-api', port: 8080, outputs: ['endpoint'] }],
      resources: [{ name: 'ordersdb', type: 'PostgreSQL' }],
    });
    expect(sent.applicationId).toBeUndefined();
    expect(sent.workloads[0]!.variables).toEqual([{ id: expect.any(String), name: 'DB_HOST', required: true }]);
    expect(sent.workloads[0]!.secrets).toEqual([{ id: expect.any(String), name: 'DB_PASSWORD', required: true }]);
    expect(sent.dependencies).toEqual([{ id: expect.any(String), sourceId: sent.workloads[0]!.id, targetId: sent.resources[0]!.id }]);

    await screen.findByText(/Saved version 1/);
    expect(window.location.pathname).toBe(`/ui/applications/${APP_ID}`);
    expect(stored(null)).toBeNull();
    expect(stored(APP_ID)).toBeNull();
  });

  test('shows client validation at the field and in the summary without calling the backend', async () => {
    const { api, user } = open('/ui/applications/new');
    await user.type(screen.getByLabelText(/^Application name/), 'Orders App');
    await user.click(screen.getByRole('button', { name: 'Save application' }));

    const summary = await screen.findByRole('alert');
    expect(within(summary).getByRole('heading')).toHaveTextContent('2 problems prevent saving');
    expect(within(summary).getByText(/An application needs at least one workload/)).toBeInTheDocument();
    const name = screen.getByLabelText(/^Application name/);
    expect(name).toHaveAttribute('aria-invalid', 'true');
    expect(name).toHaveAccessibleDescription(/must use lowercase letters/);
    expect(api.saveApplication).not.toHaveBeenCalled();

    await user.click(within(summary).getByRole('button', { name: /Application name "Orders App"/ }));
    expect(name).toHaveFocus();

    await user.clear(name);
    await user.type(name, 'orders-app');
    expect(name).not.toHaveAttribute('aria-invalid');
  });

  test('restores the draft after a refresh in the same tab and discards it on request', async () => {
    const first = open('/ui/applications/new');
    await first.user.type(screen.getByLabelText(/^Application name/), 'orders-app');
    first.unmount();

    const { user } = open('/ui/applications/new');
    expect(screen.getByLabelText(/^Application name/)).toHaveValue('orders-app');
    expect(screen.getByText(/Restored unsaved changes kept in this browser tab/)).toBeInTheDocument();

    vi.spyOn(window, 'confirm').mockReturnValue(true);
    await user.click(screen.getByRole('button', { name: 'Discard changes' }));
    await waitFor(() => expect(screen.getByLabelText(/^Application name/)).toHaveValue(''));
    expect(stored(null)).toBeNull();
    expect(screen.getByText('No unsaved changes')).toBeInTheDocument();
  });

  test('offers no way to enter a Secret value', async () => {
    const { user, container } = open('/ui/applications/new');
    await user.click(screen.getByRole('button', { name: 'Add workload' }));
    await user.click(screen.getByRole('button', { name: 'Add Secret to unnamed workload' }));
    expect(container.querySelector('input[type="password"]')).toBeNull();
    expect(screen.queryByLabelText(/secret value/i)).toBeNull();
    expect(JSON.stringify(JSON.parse(stored(null) ?? '{}'))).not.toMatch(/"value"/);
  });
});

describe('edit application', () => {
  test('loads the latest version, edits locally and saves a new version with stable IDs', async () => {
    const { api, user } = open(`/ui/applications/${APP_ID}`);
    const workload = await screen.findByRole('listitem', { name: 'Workload backend' });
    expect(screen.getByText('Editing from version 2. Saving creates version 3.')).toBeInTheDocument();

    const name = within(workload).getByLabelText(/^Workload name/);
    await user.clear(name);
    await user.type(name, 'api');
    await user.click(screen.getByRole('button', { name: 'Remove resource postgresql' }));
    expect(screen.getByText(/No dependencies/)).toBeInTheDocument();
    expect(api.loadApplication).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'Save application' }));
    const sent = api.saveApplication.mock.calls[0]![0];
    expect(sent).toMatchObject({ applicationId: APP_ID, baseVersion: 2, resources: [], dependencies: [] });
    expect(sent.workloads[0]).toMatchObject({ id: BACKEND_ID, name: 'api', secrets: [{ id: '55555555-5555-4555-8555-555555555555', name: 'DB_PASSWORD', required: true }] });
    await screen.findByText(/Saved version 3/);
    expect(stored(APP_ID)).toBeNull();
  });

  test('keeps the draft and content when Save fails', async () => {
    const api = fakeApi({ saveApplication: vi.fn(async () => ({ kind: 'error' as const, problems: [{ code: 'NETWORK_ERROR', message: 'The IDP backend could not be reached.' }] })) });
    const { user } = open(`/ui/applications/${APP_ID}`, api);
    const description = await screen.findByLabelText('Description');
    await user.type(description, ' v2');
    await user.click(screen.getByRole('button', { name: 'Save application' }));

    expect(await screen.findByRole('heading', { name: 'Save failed' })).toBeInTheDocument();
    expect(description).toHaveValue('Demo shop v2');
    expect(JSON.parse(stored(APP_ID) ?? '{}').draft.description).toBe('Demo shop v2');
  });

  test('shows backend validation problems next to the field', async () => {
    const api = fakeApi({
      saveApplication: vi.fn(async () => ({
        kind: 'validation' as const,
        problems: [{ code: 'APPLICATION_NAME_TAKEN', field: 'name', message: 'application name "shop-app" is already used by another application' }],
      })),
    });
    const { user } = open(`/ui/applications/${APP_ID}`, api);
    await screen.findByRole('listitem', { name: 'Workload backend' });
    await user.type(screen.getByLabelText('Description'), '!');
    await user.click(screen.getByRole('button', { name: 'Save application' }));
    await screen.findByText('1 problem prevents saving');
    expect(screen.getByLabelText(/^Application name/)).toHaveAccessibleDescription(/already used by another application/);
  });

  test('on DRAFT_CONFLICT keeps the user draft and asks to reload before reapplying', async () => {
    const api = fakeApi({
      saveApplication: vi.fn(async () => ({
        kind: 'conflict' as const,
        problems: [{ code: 'DRAFT_CONFLICT', message: 'this draft is based on version 2 but version 3 is the latest' }],
      })),
    });
    const { user } = open(`/ui/applications/${APP_ID}`, api);
    const description = await screen.findByLabelText('Description');
    await user.clear(description);
    await user.type(description, 'my change');
    await user.click(screen.getByRole('button', { name: 'Save application' }));

    expect(await screen.findByRole('heading', { name: 'Someone saved a newer version first' })).toBeInTheDocument();
    expect(description).toHaveValue('my change');
    expect(JSON.parse(stored(APP_ID) ?? '{}').draft.description).toBe('my change');
    expect(api.saveApplication).toHaveBeenCalledTimes(1);

    api.loadApplication.mockResolvedValueOnce({ kind: 'ok', data: { ...shopDto(3), description: 'their change' } });
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true);
    await user.click(screen.getByRole('button', { name: 'Discard my draft and load the latest version' }));
    expect(confirm).toHaveBeenCalled();
    await waitFor(() => expect(screen.getByLabelText('Description')).toHaveValue('their change'));
    expect(screen.getByText('Editing from version 3. Saving creates version 4.')).toBeInTheDocument();
    expect(stored(APP_ID)).toBeNull();
  });

  test('warns when a restored draft started from an older version', async () => {
    const first = open(`/ui/applications/${APP_ID}`);
    await first.user.type(await screen.findByLabelText('Description'), '!');
    first.unmount();

    const api = fakeApi({ loadApplication: vi.fn(async () => ({ kind: 'ok' as const, data: shopDto(5) })) });
    open(`/ui/applications/${APP_ID}`, api);
    expect(await screen.findByText(/started from version 2, but version 5 is now the latest/)).toBeInTheDocument();
    expect(screen.getByLabelText('Description')).toHaveValue('Demo shop!');
  });

  test('shows a load failure with retry', async () => {
    const api = fakeApi({ loadApplication: vi.fn(async () => ({ kind: 'error' as const, problems: [{ code: 'NETWORK_ERROR', message: 'The IDP backend could not be reached.' }] })) });
    const { user } = open(`/ui/applications/${APP_ID}`, api);
    expect(screen.getByRole('status')).toHaveTextContent('Loading the latest application version');
    expect(await screen.findByRole('heading', { name: 'The application could not be loaded' })).toBeInTheDocument();
    api.loadApplication.mockResolvedValueOnce({ kind: 'ok', data: shopDto() });
    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByRole('listitem', { name: 'Workload backend' })).toBeInTheDocument();
  });
});

describe('application list', () => {
  test('lists applications and opens the editor', async () => {
    const { user } = open('/ui/applications');
    expect(screen.getByRole('status')).toHaveTextContent('Loading applications');
    await user.click(await screen.findByRole('link', { name: 'Edit shop-app' }));
    expect(window.location.pathname).toBe(`/ui/applications/${APP_ID}`);
    expect(await screen.findByRole('listitem', { name: 'Workload backend' })).toBeInTheDocument();
  });

  test('shows the empty state', async () => {
    open('/ui/applications', fakeApi({ listApplications: vi.fn(async () => ({ kind: 'ok' as const, data: [] })) }));
    expect(await screen.findByText(/No application definitions yet/)).toBeInTheDocument();
    await act(async () => {});
  });
});
