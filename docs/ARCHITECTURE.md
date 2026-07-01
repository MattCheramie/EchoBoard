# EchoBoard Architecture

This document expands on the architecture summary in the root
[`README.md`](../README.md) and records how the pieces fit together. It is a
living document — sections are filled in as the corresponding tiers land
(see [`ROADMAP.md`](../ROADMAP.md)).

## Overview

EchoBoard is a self-hosted, local-first application that ships as a single Go
binary with an embedded SvelteKit frontend.

```
┌─────────────────────────────────────────────┐
│                 echoboard (Go)                │
│                                               │
│  cmd/echoboard  ──►  internal/api (REST + WS) │
│                        │                      │
│     ┌──────────────────┼───────────────────┐ │
│     ▼         ▼        ▼        ▼          ▼ │
│  config     db      auth    content  ...     │
│                                               │
│  internal/web  ──►  embedded SvelteKit build  │
└─────────────────────────────────────────────┘
              │ SQLite (dev) / Postgres (prod)
```

## Monorepo layout

- `backend/` — Go module (`github.com/MattCheramie/echoboard`).
  - `cmd/echoboard/` — process entry point / CLI.
  - `internal/` — one package per domain (config, db, auth, api, content,
    interactions, people, integrations, analytics, scheduler).
  - `internal/web/` — embeds the compiled frontend behind the `embed` build tag.
- `frontend/` — SvelteKit + Tailwind app built with `adapter-static`.

## Single-binary strategy

The frontend builds to static assets (`frontend/build`). For release builds,
those assets are copied into `backend/internal/web/build` and compiled into the
binary with `-tags embed` (see `internal/web/embed.go` and the `Makefile`
`embed` target). Default/dev builds omit the tag and serve the frontend from the
Vite dev server, keeping the two sides decoupled during development.

## Data layer

A single database interface backs two adapters — SQLite for local-first/dev and
Postgres for VPS/production — selected via `DB_DRIVER` (see `.env.example`).
Migrations are embedded and run at startup. *(Implemented in Tier 1, PR 1.1.)*

## Security model

Per the README: OAuth2 for third-party connections, integration tokens/keys
encrypted at rest, strict webhook signature verification, and aggressive input
sanitization on inbound social content. See the relevant tiers for details.

## Realtime

A WebSocket hub (`internal/api`) fans out live updates for the unified inbox and
interactions. *(Implemented in Tier 1, PR 1.4 and Tier 3.)*
