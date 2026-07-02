// Light/dark theme, persisted to localStorage and applied as a `dark` class on
// <html> (Tailwind's class strategy).
import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'echoboard-theme';

export const theme = writable<Theme>('light');

function apply(next: Theme) {
  document.documentElement.classList.toggle('dark', next === 'dark');
}

/** Read the saved (or system) preference and apply it. Call once on mount. */
export function initTheme(): void {
  const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
  const prefersDark = window.matchMedia?.('(prefers-color-scheme: dark)').matches;
  const initial: Theme = saved ?? (prefersDark ? 'dark' : 'light');
  theme.set(initial);
  apply(initial);
}

export function toggleTheme(): void {
  theme.update((cur) => {
    const next: Theme = cur === 'dark' ? 'light' : 'dark';
    localStorage.setItem(STORAGE_KEY, next);
    apply(next);
    return next;
  });
}
