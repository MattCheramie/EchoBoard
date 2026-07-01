# EchoBoard Frontend

SvelteKit + Tailwind CSS single-page app for EchoBoard.

> **Tier 0 status:** scaffold only. Dependencies are declared in `package.json`
> but not installed, and there is no real app yet beyond a stub landing page.
> The app shell, auth screens, and PWA setup arrive in Tier 1 (PR 1.5) — see the
> root [`ROADMAP.md`](../ROADMAP.md).

## Development (Tier 1+)

```bash
npm install
npm run dev      # start the Vite dev server
npm run build    # produce static assets in build/
```

The static build output is embedded into the Go binary for single-executable
deploys — see `backend/internal/web` and the root `Makefile` `embed` target.
