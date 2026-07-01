# Contributing to EchoBoard

Thanks for helping build EchoBoard! This project is delivered against a tiered
roadmap — please read [`ROADMAP.md`](./ROADMAP.md) before starting work.

## Working agreement: Tiers → PRs → Commits

- **Tiers** are milestone-sized bands of functionality (see the roadmap).
- Each tier is delivered through several **staged PRs**. Keep a PR scoped to a
  single roadmap PR entry.
- Each PR is broken into several **focused commits** — one logical step each.
- `main` must stay green after every merge.

## Branching

- Branch from `main` using a descriptive name, e.g. `feat/config-loader` or
  `claude/<topic>`.
- Open a PR using the template; fill in the Tier/PR references.
- Never push directly to `main`.

## Repository layout

```
backend/    Go backend (single-binary; embeds the built frontend — see Tier 6)
frontend/   SvelteKit + Tailwind PWA
docs/       Architecture and design docs
.github/    CI workflows and PR template
```

## Local setup

The project is currently skeleton-only (Tier 0). Real setup instructions land
with Tier 1. In the meantime:

```bash
# Backend (Go 1.23+)
cd backend && go vet ./... && go build ./...

# Frontend (Node 20+) — deps land in Tier 1
cd frontend && npm install && npm run dev
```

Copy `.env.example` to `.env` for local configuration (consumed starting Tier 1).

## Commit messages

Use short, conventional-style prefixes (`feat`, `fix`, `chore`, `docs`, `ci`,
`test`, `refactor`) scoped where helpful, e.g. `feat(db): ...`.
