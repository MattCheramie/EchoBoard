// Auth store: the single source of truth for who is signed in.
//
// On first load `refresh()` probes GET /api/auth/me. A 401 is the normal
// "not signed in" answer and resolves to the `anonymous` state; any other error
// surfaces so the UI can show it. Components read `authStore` reactively and call
// the action helpers (login / redeem / logout) which keep the store in sync.

import { writable } from 'svelte/store';
import { ApiError, api } from '$lib/api/client';
import type { LoginInput, RedeemInput, User } from '$lib/api/types';

export type AuthStatus = 'loading' | 'authenticated' | 'anonymous';

export interface AuthState {
  status: AuthStatus;
  user: User | null;
}

const initial: AuthState = { status: 'loading', user: null };

const store = writable<AuthState>(initial);

function setUser(user: User) {
  store.set({ status: 'authenticated', user });
}

function setAnonymous() {
  store.set({ status: 'anonymous', user: null });
}

/** Load the current session, if any. Safe to call more than once. */
async function refresh(): Promise<void> {
  try {
    setUser(await api.me());
  } catch (err) {
    if (err instanceof ApiError && err.isUnauthorized) {
      setAnonymous();
      return;
    }
    throw err;
  }
}

async function login(input: LoginInput): Promise<User> {
  const user = await api.login(input);
  setUser(user);
  return user;
}

async function redeem(input: RedeemInput): Promise<User> {
  const user = await api.redeem(input);
  setUser(user);
  return user;
}

async function logout(): Promise<void> {
  try {
    await api.logout();
  } finally {
    // Whatever the server said, the client is now signed out.
    setAnonymous();
  }
}

export const authStore = {
  subscribe: store.subscribe,
  refresh,
  login,
  redeem,
  logout
};
