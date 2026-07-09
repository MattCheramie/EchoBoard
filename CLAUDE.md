# EchoBoard — guidance for Claude

EchoBoard is a self-hosted, multi-user social-media management, content-planning,
and analytics platform: a Go backend (single binary that embeds the built Svelte
frontend — see ROADMAP Tier 6) plus a SvelteKit + Tailwind frontend. This file is
standing guidance for AI-assisted work in this repo. Keep it short.

## Build & test

- `make lint test` — `go vet` + backend unit tests; must be green before any commit.
- `cd backend && go test -race -count=1 ./...` — race detector while iterating.
- `make build` — build the binary (frontend build is a Tier 1+ no-op today).
- Single package while iterating, e.g. `go test ./internal/auth/...`.

Formatting is `gofmt` (no goimports). CI fails on unformatted files — run
`gofmt -w .` under `backend/` before committing.

## Change scope (mirrors CONTRIBUTING.md and ROADMAP.md)

- Work is delivered against the tiered roadmap: **Tiers → staged PRs → focused
  commits**. Keep a PR scoped to a single roadmap PR entry. `main` stays green.
- **Bug fix**: one narrow commit plus a regression test that **fails without the
  fix and passes with it**. If you can't write a test that fails first, you have
  not yet reproduced the bug — keep digging or ask for a reproduction, don't
  guess at a fix.
- **Feature / refactor**: design first; keep refactors out of behaviour-change PRs.

## Issue-closing policy

Closing an issue is a claim that the reported problem is gone. Do not make that
claim until it is verified. A PreToolUse hook
(`.claude/hooks/guard-issue-close.py`) asks for human confirmation before any
close-as-completed.

- **Never close an issue as completed until the fix is verified**: a failing-first
  regression test now passes **and** the reporter has confirmed it, or you have
  reproduced the original symptom and shown this change resolves it.
- **When you can't verify, leave it open.** Post a concise status comment saying
  what you found and what's blocking, rather than closing.
- **Address the latest follow-up, not the original report.** Never re-post the
  initial fix description as a close justification — respond to the most recent
  comment specifically.
- **In PRs, prefer `Refs #N` over `Closes #N`** until the fix is verified, so a
  merge doesn't auto-close an unverified issue.
- Closing as `not_planned` or `duplicate` is fine and is not gated.

## Security-sensitive surfaces (so the next change starts aware)

- Integration tokens (OAuth2 for Meta/TikTok/YouTube/X/…) are **encrypted at rest**
  in the secrets vault (`backend/internal/auth`). Never log or return them in API
  responses; keep the vault the only path that decrypts them.
- User provisioning is **invite-only** with an admin-bootstrap first-run flow
  (`echoboard --setup`). Don't add self-service public registration by default.
- Webhook receivers must verify provider signatures; user-supplied content is
  sanitized before render. See `SECURITY.md` for the full threat model.
