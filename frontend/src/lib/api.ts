// Typed client for the EchoBoard REST API. Requests are same-origin: in dev
// Vite proxies /api to the Go backend; in production the backend serves both.
import type { Invite, Role, User, VersionInfo } from './types';

/** ApiError carries the HTTP status and the server's error message. */
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

interface ErrorEnvelope {
  error?: { message?: string };
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: 'same-origin'
  };

  const resp = await fetch(path, init);

  if (resp.status === 204) {
    return undefined as T;
  }

  let data: unknown = null;
  const text = await resp.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  if (!resp.ok) {
    const msg = (data as ErrorEnvelope)?.error?.message ?? resp.statusText ?? 'request failed';
    throw new ApiError(resp.status, msg);
  }
  return data as T;
}

export const api = {
  health: () => request<{ status: string }>('GET', '/health'),
  version: () => request<VersionInfo>('GET', '/api/version'),

  // Auth.
  login: (email: string, password: string) =>
    request<User>('POST', '/api/auth/login', { email, password }),
  logout: () => request<void>('POST', '/api/auth/logout'),
  me: () => request<User>('GET', '/api/auth/me'),
  redeem: (token: string, email: string, name: string, password: string) =>
    request<User>('POST', '/api/auth/redeem', { token, email, name, password }),

  // Admin.
  listUsers: () => request<User[]>('GET', '/api/users'),
  createInvite: (input: { email?: string; role: Role; ttlHours?: number }) =>
    request<Invite>('POST', '/api/invites', input)
};
