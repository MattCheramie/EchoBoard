# EchoBoard Roadmap 🗺️

This roadmap turns the vision in [`README.md`](./README.md) into an executable delivery
plan. Work is organized into **tiers**. Each tier is a coherent slice of the product and
is delivered through several **staged pull requests (PRs)**. Each PR is broken into
several **commits** so history stays readable and reviewable.

> **Working agreement**
> - **Tier** → a milestone-sized band of functionality.
> - **PR** → a reviewable, independently mergeable unit inside a tier.
> - **Commit** → a single logical step inside a PR.
> - `main` must stay green after every PR. Feature work happens on
>   `claude/<topic>` branches and merges via PR.
> - Later tiers assume earlier tiers are merged; PRs inside a tier are ordered but
>   may overlap where dependencies allow.

**Legend:** ✅ done · 🚧 in progress · ⬜ planned

---

## Tier 0 — Foundation & Scaffolding 🚧

**Goal:** Establish the repository skeleton, tooling, and this roadmap so every later
tier has a place to plug in. Skeleton-only — structure and stubs, no runnable features
yet. Monorepo (`backend/` + `frontend/`) with the single-binary embed path pre-wired.

### PR 0.1 — Roadmap + monorepo scaffolding 🚧
Goal: Lay down docs, directory structure, module/config/CI stubs.
Deliverables: `ROADMAP.md`, monorepo skeleton, single-binary embed scaffold, CI stubs.
Commits:
- `docs: add ROADMAP.md tiered delivery plan`
- `chore: add root project scaffolding (.gitignore, .editorconfig, .env.example, Makefile)`
- `chore(backend): initialize Go module and internal package skeleton`
- `chore(backend): scaffold single-binary frontend embed`
- `chore(frontend): scaffold SvelteKit + Tailwind config and stub landing page`
- `ci: add backend/frontend CI stubs and PR template`
- `docs: add CONTRIBUTING and ARCHITECTURE, refresh README getting-started`

---

## Tier 1 — Core Platform ✅

**Goal:** A running, secured, single-user-then-multi-user backbone: configuration,
database, admin bootstrap, invite-only user management, authentication, the REST + WebSocket
skeleton, and the frontend shell (PWA-ready).

> **Status:** complete. The backend backbone (PRs 1.1–1.4) is implemented, tested, and
> runnable (`--setup` bootstraps an admin; the server serves auth + admin APIs and `/ws`),
> and the SvelteKit frontend shell (PR 1.5) provides the app shell, auth screens wired to
> the auth API, a typed API client + auth store, and PWA manifest/service worker.

### PR 1.1 — Configuration & database layer ✅
Goal: Load config from env/flags; connect to SQLite (dev) or Postgres (prod) behind one
interface; run migrations.
Commits:
- `feat(config): typed config loader (env + flags) with defaults`
- `feat(db): database interface + SQLite and Postgres adapters`
- `feat(db): embedded migration runner and initial schema`
- `test(db): adapter + migration tests`

### PR 1.2 — Admin bootstrap & user model ✅
Goal: First-run `--setup` creates the master admin; invite-only provisioning thereafter.
Commits:
- `feat(user): user model, roles, and repository`
- `feat(cli): --setup admin-bootstrap prompt`
- `feat(invite): time-limited invite links + redemption`
- `test(user): bootstrap and invite flow tests`

### PR 1.3 — Authentication & secrets vault ✅
Goal: Sessions/JWT, auth middleware, encrypted-at-rest storage for integration tokens.
Commits:
- `feat(auth): password hashing + login/logout sessions`
- `feat(auth): auth + role middleware`
- `feat(vault): encrypt integration tokens/keys at rest`
- `test(auth): auth + vault round-trip tests`

### PR 1.4 — API & realtime skeleton ✅
Goal: REST router, health/version endpoints, WebSocket hub for realtime fan-out.
Commits:
- `feat(api): router, middleware chain, error envelope`
- `feat(api): /health and /version endpoints`
- `feat(ws): WebSocket hub with subscribe/broadcast`
- `test(api): router and websocket hub tests`

### PR 1.5 — Frontend shell (PWA-ready) ✅
Goal: App layout, auth screens, API client, PWA manifest + service worker.
Commits:
- `feat(fe): app shell, navigation, and theming`
- `feat(fe): login/setup screens wired to auth API`
- `feat(fe): typed API client + auth store`
- `feat(fe): PWA manifest and service worker`

---

## Tier 2 — Content Engine 🚧

**Goal:** Create, review, approve, schedule, and visualize content — the heart of the
product (Content Calendar + Approval Workflow from the README).

### PR 2.1 — Content model & media storage ✅
Commits:
- `feat(content): content + draft + platform-target models`
- `feat(media): media upload, storage abstraction, thumbnails`
- `feat(content): tags and metadata`
- `test(content): model + media tests`

### PR 2.2 — Approval workflow & search ⬜
Commits:
- `feat(workflow): creator → reviewer → approved state machine`
- `feat(workflow): comments, edits, and approval actions`
- `feat(search): content search by tag/keyword/platform`
- `test(workflow): state transition + search tests`

### PR 2.3 — Calendar API & UI ⬜
Commits:
- `feat(calendar): schedule query API (day/week/month/custom)`
- `feat(fe): calendar views with drag-and-drop rescheduling`
- `feat(fe): content composer and review panels`
- `test(calendar): scheduling API tests`

### PR 2.4 — Scheduler service ⬜
Commits:
- `feat(scheduler): concurrency-safe due-post queue`
- `feat(scheduler): dispatch hooks to integration adapters`
- `feat(scheduler): retry + failure handling`
- `test(scheduler): scheduling and retry tests`

---

## Tier 3 — Engagement Hub ⬜

**Goal:** Unified inbox for comments/DMs/emails and an integrated CRM (Interactions &
People from the README), with realtime updates and strict sanitization.

### PR 3.1 — Interactions model & unified inbox API ⬜
Commits:
- `feat(interactions): normalized message/thread model`
- `feat(security): XSS sanitization pipeline for inbound content`
- `feat(inbox): unified inbox + reply API`
- `test(interactions): model + sanitization tests`

### PR 3.2 — Unified inbox UI ⬜
Commits:
- `feat(fe): chat-style thread list and conversation view`
- `feat(fe): realtime updates via WebSocket`
- `feat(fe): reply composer with platform routing`
- `test(fe): inbox interaction tests`

### PR 3.3 — People / CRM ⬜
Commits:
- `feat(people): unified cross-platform contact profiles`
- `feat(people): global people search`
- `feat(people): relationship + interaction-history tracking`
- `test(people): profile merge + search tests`

---

## Tier 4 — Platform Integrations 🚧

**Goal:** Connect the external platforms named in the README via official APIs, on top of
a reusable OAuth2 + webhook framework.

> **Note:** PR 4.1 (the framework) is pulled forward ahead of Tiers 2–3 per the delivery
> plan, because the Tier 2 scheduler dispatches through its adapter registry. A built-in
> **sandbox** adapter exercises the full publish/webhook pipeline without live credentials;
> the real platform adapters (4.2–4.5) each require operator-supplied OAuth app credentials.

### PR 4.1 — Integration framework ✅
Commits:
- `feat(integrations): adapter interface + registry`
- `feat(oauth): OAuth2 connection + token refresh flow`
- `feat(webhooks): signature-verification middleware`
- `test(integrations): framework + webhook verification tests`

### PR 4.2 — Meta (Facebook & Instagram) ⬜
Commits:
- `feat(meta): OAuth connect + page/account selection`
- `feat(meta): publish + schedule adapter`
- `feat(meta): comment/DM webhook ingestion`
- `test(meta): adapter tests (mocked API)`

### PR 4.3 — TikTok & YouTube ⬜
Commits:
- `feat(tiktok): connect + publish adapter`
- `feat(youtube): connect + publish adapter`
- `feat(integrations): shared media-format normalization`
- `test(integrations): tiktok + youtube adapter tests`

### PR 4.4 — X/Twitter & Snapchat ⬜
Commits:
- `feat(x): connect + publish + mentions adapter`
- `feat(snapchat): connect + publish adapter`
- `feat(integrations): rate-limit-aware request client`
- `test(integrations): x + snapchat adapter tests`

### PR 4.5 — Shopify & Spotify ⬜
Commits:
- `feat(shopify): connect + social-to-sales pipeline events`
- `feat(spotify): connect + posting/sales/reporting hooks`
- `feat(integrations): commerce/audio metric normalization`
- `test(integrations): shopify + spotify adapter tests`

---

## Tier 5 — Analytics & Tracking ⬜

**Goal:** Own the metrics — custom tracking links/pixels, aggregation, custom reports, and
dashboards.

### PR 5.1 — Tracking infrastructure ⬜
Commits:
- `feat(tracking): custom short-link generation + redirect`
- `feat(tracking): pixel-style page/usage tracking endpoint`
- `feat(tracking): click/conversion event pipeline`
- `test(tracking): link + pixel tracking tests`

### PR 5.2 — Metrics aggregation & report builder ⬜
Commits:
- `feat(analytics): metrics ingestion + rollups`
- `feat(analytics): custom report definition API`
- `feat(analytics): engagement/reach/conversion queries`
- `test(analytics): aggregation + report tests`

### PR 5.3 — Analytics dashboards ⬜
Commits:
- `feat(fe): dashboard layout + chart components`
- `feat(fe): custom report builder UI`
- `feat(fe): export (CSV/PDF) of reports`
- `test(fe): dashboard rendering tests`

---

## Tier 6 — Hardening, Packaging & Beyond ⬜

**Goal:** Ship it. Single-binary release, security hardening, deployment docs, and
groundwork for Phase 2 native apps.

### PR 6.1 — Single-binary release build ⬜
Commits:
- `feat(build): embed compiled frontend into Go binary`
- `feat(build): production build pipeline (make build)`
- `feat(cli): polished --setup and first-run UX`
- `test(build): smoke test of embedded binary`

### PR 6.2 — Security hardening ⬜
Commits:
- `feat(security): rate limiting + brute-force protection`
- `feat(security): audit logging`
- `feat(security): CSRF + security headers`
- `test(security): hardening tests`

### PR 6.3 — Deployment & operations ⬜
Commits:
- `docs(deploy): VPS deployment guide`
- `feat(deploy): Dockerfile + docker-compose + systemd unit`
- `ci(release): tagged release build + artifacts`
- `docs(ops): backup/restore and upgrade guide`

### PR 6.4 — Native-app groundwork (Phase 2) ⬜
Commits:
- `docs(api): stabilize + version the public API`
- `feat(api): API tokens for native clients`
- `feat(push): push-notification service abstraction`
- `docs(mobile): native client integration guide`

---

## Progress

| Tier | Title | Status |
|------|-------|--------|
| 0 | Foundation & Scaffolding | ✅ |
| 1 | Core Platform | ✅ |
| 2 | Content Engine | 🚧 (content model + media/PR 2.1 done) |
| 3 | Engagement Hub | ⬜ |
| 4 | Platform Integrations | 🚧 (framework/PR 4.1 done; adapters pending) |
| 5 | Analytics & Tracking | ⬜ |
| 6 | Hardening, Packaging & Beyond | ⬜ |
