// Typed client for the EchoBoard REST API.
//
// All requests are same-origin (relative paths): in development Vite proxies
// /api, /health, and /ws to the Go backend (see vite.config.js); in the embedded
// production build the Go binary serves this app and the API from one origin.
// The session lives in an HttpOnly cookie, so every request sends credentials and
// no token handling happens in JS. Errors decode the backend's standard envelope
// `{"error": {"message": "..."}}` into an ApiError carrying the HTTP status.

import type {
  CreateInviteInput,
  Invite,
  LoginInput,
  RedeemInput,
  User,
  Version
} from './types';

/** ApiError wraps a non-2xx response with its status and decoded message. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }

  /** True for 401 Unauthorized — used by the auth store to drop to anonymous. */
  get isUnauthorized(): boolean {
    return this.status === 401;
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  /** Skip JSON parsing (e.g. 204 No Content). */
  expectEmpty?: boolean;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const init: RequestInit = {
    method: opts.method ?? 'GET',
    credentials: 'same-origin',
    headers: {}
  };
  if (opts.body !== undefined) {
    (init.headers as Record<string, string>)['Content-Type'] = 'application/json';
    init.body = JSON.stringify(opts.body);
  }

  let res: Response;
  try {
    res = await fetch(path, init);
  } catch (cause) {
    throw new ApiError(0, 'network error: could not reach the server');
  }

  if (!res.ok) {
    throw new ApiError(res.status, await errorMessage(res));
  }
  if (opts.expectEmpty || res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

/** Decode the API error envelope, falling back to the status text. */
async function errorMessage(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as { error?: { message?: string } };
    if (data?.error?.message) {
      return data.error.message;
    }
  } catch {
    // Non-JSON body (e.g. middleware's plain-text 401/403) — fall through.
  }
  return res.statusText || `request failed (${res.status})`;
}

export const api = {
  version: () => request<Version>('/api/version'),
  me: () => request<User>('/api/auth/me'),
  login: (input: LoginInput) =>
    request<User>('/api/auth/login', { method: 'POST', body: input }),
  logout: () =>
    request<void>('/api/auth/logout', { method: 'POST', expectEmpty: true }),
  redeem: (input: RedeemInput) =>
    request<User>('/api/auth/redeem', { method: 'POST', body: input }),
  listUsers: () => request<User[]>('/api/users'),
  createInvite: (input: CreateInviteInput) =>
    request<Invite>('/api/invites', { method: 'POST', body: input })
};
