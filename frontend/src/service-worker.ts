/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

// App-shell service worker: precache built assets and static files, serve them
// cache-first, and fall back to the cached shell for navigations when offline.
// API and WebSocket traffic is never intercepted.
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
  const req = event.request;
  if (req.method !== 'GET') return;

  const url = new URL(req.url);
  if (url.origin !== location.origin) return;
  // Let API, health, and websocket requests hit the network directly.
  if (url.pathname.startsWith('/api') || url.pathname.startsWith('/ws') || url.pathname === '/health') {
    return;
  }

  event.respondWith(
    (async () => {
      const cache = await caches.open(CACHE);

      // Cache-first for immutable built assets and static files.
      if (ASSETS.includes(url.pathname)) {
        const cached = await cache.match(url.pathname);
        if (cached) return cached;
      }

      // Network-first for everything else, falling back to cache / app shell.
      try {
        const response = await fetch(req);
        if (response.ok && response.type === 'basic') {
          cache.put(req, response.clone());
        }
        return response;
      } catch {
        const cached = await cache.match(req);
        if (cached) return cached;
        if (req.mode === 'navigate') {
          const shell = await cache.match('/');
          if (shell) return shell;
        }
        return Response.error();
      }
    })()
  );
});
