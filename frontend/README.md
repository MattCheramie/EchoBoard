# EchoBoard Frontend

SvelteKit + Tailwind CSS single-page app for EchoBoard.

The app is a client-side SPA (SSR disabled) built with `adapter-static`, so its
output is a set of static assets the Go binary can embed for single-executable
deploys. It talks to the backend over the same-origin REST API; in development
the Vite dev server proxies `/api`, `/ws`, and `/health` to the Go backend
(default `http://localhost:8080`, override with `ECHOBOARD_BACKEND`).

## Development

```bash
npm install      # install deps (generates package-lock.json)
npm run dev      # start the Vite dev server (proxies the API to :8080)
npm run check    # svelte-check type-checking
npm test         # Vitest unit tests
npm run build    # produce static assets in build/
```

Run the backend alongside it (`cd ../backend && go run ./cmd/echoboard`) so the
proxied API calls resolve. The static build output is embedded into the Go
binary for single-executable deploys — see `backend/internal/web` and the root
`Makefile` `embed` target (wired in Tier 6, PR 6.1).
