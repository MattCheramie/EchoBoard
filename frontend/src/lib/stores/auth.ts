// Global authentication state: the current user plus actions that call the API
// and keep the store in sync. Session state lives in an HttpOnly cookie managed
// by the backend; this store only mirrors "who is logged in".
import { derived, writable } from 'svelte/store';
import { api, ApiError } from '$lib/api';
import type { User } from '$lib/types';

interface AuthState {
  user: User | null;
  /** false until the first `refresh()` resolves, so guards can wait. */
  ready: boolean;
}

const store = writable<AuthState>({ user: null, ready: false });

/** Load the current user from the session cookie, if any. */
async function refresh(): Promise<void> {
  try {
    const user = await api.me();
    store.set({ user, ready: true });
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      store.set({ user: null, ready: true });
      return;
    }
    // Network or server error: still mark ready so the UI can proceed.
    store.set({ user: null, ready: true });
  }
}

async function login(email: string, password: string): Promise<User> {
  const user = await api.login(email, password);
  store.set({ user, ready: true });
  return user;
}

async function redeem(token: string, email: string, name: string, password: string): Promise<User> {
  const user = await api.redeem(token, email, name, password);
  store.set({ user, ready: true });
  return user;
}

async function logout(): Promise<void> {
  try {
    await api.logout();
  } finally {
    store.set({ user: null, ready: true });
  }
}

export const auth = {
  subscribe: store.subscribe,
  refresh,
  login,
  redeem,
  logout
};

export const currentUser = derived(store, ($s) => $s.user);
export const isAuthenticated = derived(store, ($s) => $s.user !== null);
export const isAdmin = derived(store, ($s) => $s.user?.role === 'admin');
export const authReady = derived(store, ($s) => $s.ready);
