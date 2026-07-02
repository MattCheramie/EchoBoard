import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/**
 * EchoBoard uses adapter-static so the built frontend is a set of static assets
 * that can be embedded into the Go binary for single-executable deploys
 * (see backend/internal/web). SSR/prerender are disabled in src/routes/+layout.ts,
 * making this a client-rendered SPA with an index.html fallback.
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
