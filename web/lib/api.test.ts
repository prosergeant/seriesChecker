import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api } from './api';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function okResponse(body: unknown) {
  return Promise.resolve({
    ok: true,
    json: () => Promise.resolve(body),
  } as Response);
}

function errorResponse(status: number, body: unknown) {
  return Promise.resolve({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  } as Response);
}

beforeEach(() => {
  mockFetch.mockReset();
});

describe('api.auth', () => {
  it('login — sends POST to /api/auth/login with credentials', async () => {
    mockFetch.mockReturnValueOnce(okResponse({ message: 'ok', session_id: 'abc' }));

    await api.auth.login('user@test.com', 'pass123');

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/auth/login');
    expect(opts.method).toBe('POST');
    expect(JSON.parse(opts.body)).toEqual({ email: 'user@test.com', password: 'pass123' });
    expect(opts.credentials).toBe('include');
  });

  it('me — sends GET to /api/auth/me with credentials', async () => {
    mockFetch.mockReturnValueOnce(okResponse({ id: '1', email: 'user@test.com' }));

    const result = await api.auth.me();

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/auth/me');
    expect(opts.credentials).toBe('include');
    expect(result.email).toBe('user@test.com');
  });

  it('throws error with server message on non-ok response', async () => {
    mockFetch.mockReturnValueOnce(errorResponse(401, { error: 'Unauthorized' }));

    await expect(api.auth.me()).rejects.toThrow('Unauthorized');
  });

  it('logout — sends POST to /api/auth/logout', async () => {
    mockFetch.mockReturnValueOnce(okResponse({}));

    await api.auth.logout();

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/auth/logout');
    expect(opts.method).toBe('POST');
  });
});

describe('api.progress', () => {
  it('getAll — no status param when not provided', async () => {
    mockFetch.mockReturnValueOnce(okResponse([]));

    await api.progress.getAll();

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/progress');
    expect(url).not.toContain('?status');
  });

  it('getAll — adds ?status=watching when provided', async () => {
    mockFetch.mockReturnValueOnce(okResponse([]));

    await api.progress.getAll('watching');

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('?status=watching');
  });

  it('delete — sends DELETE to /api/progress/:id', async () => {
    mockFetch.mockReturnValueOnce(okResponse({ message: 'deleted' }));

    await api.progress.delete(42);

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/progress/42');
    expect(opts.method).toBe('DELETE');
  });
});

describe('api.series', () => {
  it('search — encodes query param', async () => {
    mockFetch.mockReturnValueOnce(okResponse([]));

    await api.series.search('во все тяжкие');

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/series/search?q=');
    expect(url).toContain(encodeURIComponent('во все тяжкие'));
  });
});
