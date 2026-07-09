import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

// In development the SvelteKit dev server (default :5173) proxies API, realtime,
// and health traffic to the Go backend (:8080) so the browser stays same-origin
// and the session cookie is sent without any CORS configuration. In the embedded
// production build the frontend is served by the Go binary itself, so these same
// relative paths resolve on the same origin. Override the target with
// ECHOBOARD_BACKEND (e.g. when the backend runs on another port).
const backend = process.env.ECHOBOARD_BACKEND || 'http://localhost:8080';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api': { target: backend, changeOrigin: true },
      '/health': { target: backend, changeOrigin: true },
      '/ws': { target: backend, changeOrigin: true, ws: true }
    }
  },
  test: {
    environment: 'node',
    include: ['src/**/*.{test,spec}.{js,ts}']
  }
});
