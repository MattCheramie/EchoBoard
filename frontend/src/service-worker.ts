/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

// Offline-capable app shell. SvelteKit registers this worker automatically in
// the production build. On install we precache the built assets and static
// files; API, realtime, and health traffic always go to the network so data is
// never stale. `build`/`files`/`version` are injected by SvelteKit.
import { build, files, version } from '$service-worker';

const sw = self as unknown as ServiceWorkerGlobalScope;

const CACHE = `echoboard-cache-${version}`;
const ASSETS = [...build, ...files];

sw.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      .then((cache) => cache.addAll(ASSETS))
      .then(() => sw.skipWaiting())
  );
});

sw.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      for (const key of await caches.keys()) {
        if (key !== CACHE) await caches.delete(key);
      }
      await sw.clients.claim();
    })()
  );
});

sw.addEventListener('fetch', (event) => {
  if (event.request.method !== 'GET') return;

  const url = new URL(event.request.url);
  if (url.origin !== sw.location.origin) return;
  // Dynamic backend traffic must never be served from cache.
  if (
    url.pathname.startsWith('/api') ||
    url.pathname.startsWith('/ws') ||
    url.pathname === '/health'
  ) {
    return;
  }

  event.respondWith(
    (async () => {
      const cache = await caches.open(CACHE);

      // Cache-first for precached build/static assets.
      if (ASSETS.includes(url.pathname)) {
        const cached = await cache.match(url.pathname);
        if (cached) return cached;
      }

      // Network-first for everything else, falling back to cache when offline.
      try {
        return await fetch(event.request);
      } catch (err) {
        const cached = await cache.match(event.request);
        if (cached) return cached;
        throw err;
      }
    })()
  );
});
