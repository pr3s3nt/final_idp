import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { App } from '../../../app/App';
import type { AuthenticationApi } from '../api/client';
import { LoginPage } from './LoginPage';

function fakeAuthenticationApi(overrides: Partial<AuthenticationApi> = {}): AuthenticationApi {
  return {
    loadLoginContext: vi.fn(async () => ({ kind: 'ok' as const, data: { csrfToken: 'login-csrf', returnTo: '/ui/applications' } })),
    signIn: vi.fn(async () => ({ kind: 'invalid' as const, problems: [{ code: 'INVALID_CREDENTIALS', message: 'The username or password is incorrect.' }] })),
    signOut: vi.fn(async () => ({ kind: 'ok' as const, data: null })),
    ...overrides,
  };
}

describe('login page', () => {
  test('loads pre-auth context and submits credentials without retaining the password after failure', async () => {
    window.history.replaceState(null, '', '/ui/login?return_to=%2Fui%2Fapplications%2Fnew');
    const api = fakeAuthenticationApi();
    const user = userEvent.setup();
    render(<LoginPage api={api} />);

    expect(screen.getByRole('status')).toHaveTextContent('Loading sign-in form');
    const username = await screen.findByLabelText('Username');
    const password = screen.getByLabelText('Password');
    await user.type(username, 'developer');
    await user.type(password, 'a sufficiently long password');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    await waitFor(() => expect(api.signIn).toHaveBeenCalledWith(
      'developer',
      'a sufficiently long password',
      '/ui/applications',
      'login-csrf',
    ));
    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('The username or password is incorrect.');
    expect(alert).toHaveFocus();
    expect(username).toHaveValue('developer');
    expect(password).toHaveValue('');
    expect(api.loadLoginContext).toHaveBeenCalledWith('/ui/applications/new');
  });

  test('shows retry timing after rate limiting', async () => {
    window.history.replaceState(null, '', '/ui/login');
    const api = fakeAuthenticationApi({
      signIn: vi.fn(async () => ({
        kind: 'rate-limited' as const,
        problems: [{ code: 'RATE_LIMITED', message: 'Too many sign-in attempts. Try again later.' }],
        retryAfter: 42,
      })),
    });
    const user = userEvent.setup();
    render(<LoginPage api={api} />);

    await user.type(await screen.findByLabelText('Username'), 'developer');
    await user.type(screen.getByLabelText('Password'), 'a sufficiently long password');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Too many sign-in attempts. Try again later.');
    expect(alert).toHaveTextContent('Try again in about 42 seconds.');
  });

  test('requires reloading an expired sign-in form', async () => {
    window.history.replaceState(null, '', '/ui/login');
    const api = fakeAuthenticationApi({
      signIn: vi.fn(async () => ({
        kind: 'form-expired' as const,
        problems: [{ code: 'INVALID_CSRF', message: 'The sign-in form has expired. Reload and try again.' }],
      })),
    });
    const user = userEvent.setup();
    render(<LoginPage api={api} />);

    await user.type(await screen.findByLabelText('Username'), 'developer');
    await user.type(screen.getByLabelText('Password'), 'a sufficiently long password');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(await screen.findByRole('button', { name: 'Reload sign-in form' })).toBeInTheDocument();
    expect(screen.getByLabelText('Username')).toBeDisabled();
    expect(screen.getByLabelText('Password')).toBeDisabled();
  });

  test('shows the signed-out explanation without the authenticated shell', async () => {
    window.history.replaceState(null, '', '/ui/login?reason=logged-out');
    render(<App authenticationApi={fakeAuthenticationApi()} />);

    expect(await screen.findByText('You have signed out.')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Sign in to IDP' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Sign out' })).not.toBeInTheDocument();
  });
});
