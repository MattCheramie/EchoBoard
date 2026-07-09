// Theme store: light/dark toggle persisted in localStorage and reflected as the
// `dark` class on <html> (Tailwind's class strategy). Defaults to the OS setting
// on first visit. All access is browser-only; the app runs with SSR disabled.

import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'echoboard:theme';

function initialTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'light' || stored === 'dark') {
    return stored;
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function apply(theme: Theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark');
}

function createTheme() {
  const store = writable<Theme>('light');

  return {
    subscribe: store.subscribe,
    /** Read the persisted/OS preference and apply it. Call once on mount. */
    init() {
      const theme = initialTheme();
      apply(theme);
      store.set(theme);
    },
    toggle() {
      store.update((current) => {
        const next: Theme = current === 'dark' ? 'light' : 'dark';
        localStorage.setItem(STORAGE_KEY, next);
        apply(next);
        return next;
      });
    }
  };
}

export const theme = createTheme();
