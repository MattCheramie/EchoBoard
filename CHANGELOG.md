# Changelog

All notable changes to EchoBoard are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

User-visible changes should add a bullet under `## [Unreleased]` in the pull
request that makes them, grouped under `Added`, `Changed`, `Fixed`, `Removed`,
`Deprecated`, or `Security`.

## [Unreleased]

### Added

- Content data layer (Tier 2, PR 2.1): a content model with a master draft body,
  lifecycle status, optional schedule time, per-platform targets (each able to
  override the body with tailored copy), free-form tags, and a JSON metadata bag,
  all persisted transactionally through a content repository. A new media
  subsystem adds a pluggable blob Store (path-traversal-safe local-filesystem
  backend), a stdlib-only image thumbnailer, a metadata repository, and an upload
  service that stores bytes, thumbnails images, and records metadata. New
  `MEDIA_DIR` config setting.
- Frontend shell (Tier 1, PR 1.5): SvelteKit + Tailwind SPA with an app shell and
  navigation, light/dark theming, setup/login/invite-redeem screens wired to the
  auth API, a typed API client and auth store (session-cookie based, error-envelope
  aware), and a PWA manifest + offline service worker. Committing the frontend
  lockfile activates Frontend CI (`svelte-check` + production build).
- Project infrastructure: Apache-2.0 `LICENSE`, `SECURITY.md`, this changelog,
  `CLAUDE.md` contributor/AI guidance, `.github/FUNDING.yml`, and branch/tag
  protection rulesets.
- CI: `gofmt` gate and race-enabled tests in Backend CI, a `govulncheck`
  dependency scan, `svelte-check` in Frontend CI, a manual branch-cleanup
  workflow, and a tag-triggered release workflow (Linux amd64/arm64).
- Packaging: multi-stage `Dockerfile` and `docker-compose.yml` for self-hosting
  (backend binary; frontend embed lands in Tier 6).

### Security

- Pin the Go toolchain to `go1.25.12` (`backend/go.mod`), clearing the stdlib
  CVEs that `govulncheck` flagged against the 1.25.0 toolchain (crypto/tls,
  crypto/x509, net/url, net/http, os, encoding/asn1, …).
