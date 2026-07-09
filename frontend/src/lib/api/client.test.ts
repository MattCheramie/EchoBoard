import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, api } from './client';

function mockFetch(res: Partial<Response> & { jsonBody?: unknown }) {
  const impl = vi.fn(async () => {
    const { jsonBody, ...rest } = res;
    return {
      ok: rest.ok ?? true,
      status: rest.status ?? 200,
      statusText: rest.statusText ?? '',
      json: async () => {
        if (jsonBody === undefined) throw new Error('no json');
        return jsonBody;
      }
    } as unknown as Response;
  });
  vi.stubGlobal('fetch', impl);
  return impl;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('api client', () => {
  it('returns decoded JSON on success', async () => {
    mockFetch({ ok: true, status: 200, jsonBody: { id: 'u1', email: 'a@b.co' } });
    const user = await api.me();
    expect(user.id).toBe('u1');
  });

  it('decodes the error envelope into ApiError', async () => {
    mockFetch({
      ok: false,
      status: 400,
      jsonBody: { error: { message: 'invalid request body' } }
    });
    await expect(api.login({ email: 'x', password: 'y' })).rejects.toMatchObject({
      status: 400,
      message: 'invalid request body'
    });
  });

  it('flags 401 as unauthorized (used to drop to anonymous)', async () => {
    mockFetch({ ok: false, status: 401, statusText: 'Unauthorized' });
    const err = await api.me().catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.isUnauthorized).toBe(true);
  });

  it('treats 204 as an empty body', async () => {
    const fetchMock = mockFetch({ ok: true, status: 204 });
    await expect(api.logout()).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/logout',
      expect.objectContaining({ method: 'POST' })
    );
  });

  it('wraps network failures as ApiError(0)', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new TypeError('failed to fetch');
      })
    );
    const err = await api.me().catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(0);
  });
});
