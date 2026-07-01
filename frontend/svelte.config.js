import adapter from '@sveltejs/adapter-static';

/**
 * EchoBoard uses adapter-static so the built frontend is a set of static assets
 * that can be embedded into the Go binary for single-executable deploys
 * (see backend/internal/web). Real routing/SSR decisions land in Tier 1.
 *
 * @type {import('@sveltejs/kit').Config}
 */
const config = {
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html'
    })
  }
};

export default config;
