import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/**
 * EchoBoard uses adapter-static so the built frontend is a set of static assets
 * that can be embedded into the Go binary for single-executable deploys
 * (see backend/internal/web). `fallback: index.html` makes it a client-side SPA:
 * the Go server serves the same shell for every non-API path and the app routes
 * on the client (see the root +layout.ts, which disables SSR).
 *
 * @type {import('@sveltejs/kit').Config}
 */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html'
    })
  }
};

export default config;
