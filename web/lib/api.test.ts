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
  it('login — отправляет POST на /api/auth/login с credentials', async () => {
    mockFetch.mockReturnValueOnce(okResponse({ message: 'ok', session_id: 'abc' }));

    await api.auth.login('user@test.com', 'pass123');

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/auth/login');
    expect(opts.method).toBe('POST');
    expect(JSON.parse(opts.body)).toEqual({ email: 'user@test.com', password: 'pass123' });
    expect(opts.credentials).toBe('include');
  });

  it('me — отправляет GET на /api/auth/me с credentials', async () => {
    mockFetch.mockReturnValueOnce(okResponse({ id: '1', email: 'user@test.com' }));

    const result = await api.auth.me();

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/auth/me');
    expect(opts.credentials).toBe('include');
    expect(result.email).toBe('user@test.com');
  });

  it('бросает ошибку с сообщением сервера при !ok ответе', async () => {
    mockFetch.mockReturnValueOnce(errorResponse(401, { error: 'Unauthorized' }));

    await expect(api.auth.me()).rejects.toThrow('Unauthorized');
  });

  it('logout — отправляет POST на /api/auth/logout', async () => {
    mockFetch.mockReturnValueOnce(okResponse({}));

    await api.auth.logout();

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/auth/logout');
    expect(opts.method).toBe('POST');
  });
});

describe('api.progress', () => {
  it('getAll — без параметра status если не передан', async () => {
    mockFetch.mockReturnValueOnce(okResponse([]));

    await api.progress.getAll();

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/progress');
    expect(url).not.toContain('?status');
  });

  it('getAll — добавляет ?status=watching когда передан', async () => {
    mockFetch.mockReturnValueOnce(okResponse([]));

    await api.progress.getAll('watching');

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('?status=watching');
  });

  it('delete — отправляет DELETE на /api/progress/:id', async () => {
    mockFetch.mockReturnValueOnce(okResponse({ message: 'deleted' }));

    await api.progress.delete(42);

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/progress/42');
    expect(opts.method).toBe('DELETE');
  });
});

describe('api.series', () => {
  it('search — кодирует кириллицу в параметре запроса', async () => {
    mockFetch.mockReturnValueOnce(okResponse([]));

    await api.series.search('во все тяжкие');

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/series/search?q=');
    expect(url).toContain(encodeURIComponent('во все тяжкие'));
  });
});
