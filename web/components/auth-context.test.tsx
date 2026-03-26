import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider, useAuth } from './auth-context';

// Mock api module
vi.mock('@/lib/api', () => ({
  api: {
    auth: {
      me: vi.fn(),
      login: vi.fn(),
      logout: vi.fn(),
      register: vi.fn(),
    },
  },
}));

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { api } from '@/lib/api';
const mockApi = api.auth as Record<string, ReturnType<typeof vi.fn>>;

beforeEach(() => {
  vi.clearAllMocks();
});

// Helper component that exposes auth context state
function AuthConsumer() {
  const { user, isLoading, isAuthenticated } = useAuth();
  return (
    <div>
      <span data-testid="loading">{String(isLoading)}</span>
      <span data-testid="authenticated">{String(isAuthenticated)}</span>
      <span data-testid="email">{user?.email ?? 'none'}</span>
    </div>
  );
}

function AuthWithActions() {
  const { login, logout } = useAuth();
  return (
    <>
      <AuthConsumer />
      <button onClick={() => login('a@b.com', 'pw')}>login</button>
      <button onClick={() => logout()}>logout</button>
    </>
  );
}

describe('AuthProvider', () => {
  it('calls api.auth.me on mount to check session', async () => {
    mockApi.me.mockResolvedValue({ id: '1', email: 'x@x.com' });

    render(<AuthProvider><AuthConsumer /></AuthProvider>);

    await waitFor(() => expect(mockApi.me).toHaveBeenCalledTimes(1));
  });

  it('isAuthenticated = true when me() returns a user', async () => {
    mockApi.me.mockResolvedValue({ id: '1', email: 'x@x.com' });

    render(<AuthProvider><AuthConsumer /></AuthProvider>);

    await waitFor(() =>
      expect(screen.getByTestId('authenticated').textContent).toBe('true')
    );
    expect(screen.getByTestId('email').textContent).toBe('x@x.com');
  });

  it('isAuthenticated = false when me() throws (not logged in)', async () => {
    mockApi.me.mockRejectedValue(new Error('401'));

    render(<AuthProvider><AuthConsumer /></AuthProvider>);

    await waitFor(() =>
      expect(screen.getByTestId('loading').textContent).toBe('false')
    );
    expect(screen.getByTestId('authenticated').textContent).toBe('false');
  });

  it('login() calls api.auth.login and then re-checks session', async () => {
    mockApi.me
      .mockRejectedValueOnce(new Error('401'))   // initial check
      .mockResolvedValueOnce({ id: '1', email: 'a@b.com' }); // after login
    mockApi.login.mockResolvedValue({ message: 'ok', session_id: 'abc' });

    render(<AuthProvider><AuthWithActions /></AuthProvider>);

    await waitFor(() =>
      expect(screen.getByTestId('loading').textContent).toBe('false')
    );

    await act(async () => {
      await userEvent.click(screen.getByText('login'));
    });

    expect(mockApi.login).toHaveBeenCalledWith('a@b.com', 'pw');
    await waitFor(() =>
      expect(screen.getByTestId('authenticated').textContent).toBe('true')
    );
  });

  it('logout() calls api.auth.logout and sets user to null', async () => {
    mockApi.me.mockResolvedValue({ id: '1', email: 'a@b.com' });
    mockApi.logout.mockResolvedValue({});

    render(<AuthProvider><AuthWithActions /></AuthProvider>);

    await waitFor(() =>
      expect(screen.getByTestId('authenticated').textContent).toBe('true')
    );

    await act(async () => {
      await userEvent.click(screen.getByText('logout'));
    });

    expect(mockApi.logout).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('authenticated').textContent).toBe('false');
  });
});
