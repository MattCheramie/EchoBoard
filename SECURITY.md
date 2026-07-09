# Security policy

EchoBoard's threat model centres on operators self-hosting the server for a
small team — a local machine or a single VPS they own. The binary is a single
process that stores content, contacts, and integration credentials to a local
database (SQLite for single-user, PostgreSQL for VPS/production). It is a
multi-user web application: several authenticated humans share one instance, and
it holds OAuth2 tokens for third-party platforms (Meta, TikTok, YouTube,
Snapchat, X/Twitter, Shopify, Spotify) on their behalf. Those tokens are
encrypted at rest in a secrets vault, and user provisioning is invite-only.

## Supported versions

Security fixes land on the default branch (`main`). The most recent tagged
release receives back-ported fixes; older tags are not maintained.

| Version    | Status              |
| ---------- | ------------------- |
| `main`     | Supported           |
| Latest tag | Supported           |
| Older tags | Best-effort, no SLA |

## Reporting a vulnerability

**Do not open a public issue** for a suspected vulnerability.

Use **GitHub's private security advisory** workflow:

1. Visit the repository's **Security → Advisories** tab.
2. Click **Report a vulnerability** (or hit
   `https://github.com/mattcheramie/echoboard/security/advisories/new`).
3. Provide a description, a reproducer (the minimal API request, OAuth flow, or
   webhook payload that triggers the issue), and the affected version / commit
   SHA.

GitHub's advisory workflow keeps the report private until a fix ships. The
maintainer aims for an initial response within 7 days and a fix or mitigation
within 30 days for critical issues; less severe issues land on the regular
development cadence.

If GitHub advisories aren't available to you, open a public issue marked
`security: contact requested` asking the maintainer to reach out — do **not**
include exploit details in the public issue.

## What counts as a vulnerability

In scope:

- Authentication / authorisation bypass: session/cookie handling, the
  invite-redemption flow, privilege escalation between users, or an
  unauthenticated caller reaching an authenticated endpoint
  (`backend/internal/auth`, `backend/internal/api`).
- Disclosure or exfiltration of **integration tokens** or other secrets held in
  the vault — anything that returns a decrypted third-party token to a user who
  should not have it, or that leaks the vault key.
- **Webhook forgery**: accepting a provider webhook without verifying its
  signature, letting an attacker inject events or content.
- Stored / reflected **XSS** in user-authored content (posts, inbox messages,
  CRM notes) that renders in another user's browser, and CSRF against
  state-changing endpoints.
- **SQL injection** through any query built by string concatenation rather than
  parameterised statements (the `internal/db` layer uses parameterised queries;
  report any path that doesn't).
- **SSRF** via user-supplied URLs (custom links, media fetches, OAuth redirect
  handling) that lets a caller reach internal network resources.
- Path traversal / arbitrary file read or write through any operator- or
  user-supplied path (media uploads, config).
- Memory-safety bugs (out-of-bounds slice access, nil-deref DoS) in any non-test
  Go code, even where Go turns them into runtime panics.

Out of scope:

- "The server does what its config tells it to" — operator misconfiguration
  isn't a vulnerability.
- Local-only deployments where the operator already has root on the host (you
  can't escalate above the privilege you started with).
- Missing rate limiting / brute-force hardening on a deployment the operator
  chose to expose publicly without a reverse proxy — hardening guidance, not a
  vulnerability class, unless a specific endpoint is trivially amplifiable.
- Vulnerabilities that require an already-authenticated admin acting against
  their own instance.

## Cryptography

- Integration tokens and other secrets are encrypted at rest via the vault; the
  encryption key is supplied by the operator (`SECRET_KEY`, a 32-byte base64
  value) and is never written to the database.
- Passwords are hashed with `bcrypt` (`golang.org/x/crypto/bcrypt`, never stored
  or logged in plaintext); session and invite tokens are compared in constant
  time.
- Transport security (HTTPS) is expected to be terminated by a reverse proxy in
  front of the server for public deployments; run behind TLS.

## Disclosure timeline expectations

| Severity | Initial response | Fix / mitigation |
| -------- | ---------------- | ---------------- |
| Critical (RCE, auth bypass, token/secret disclosure, data loss) | ≤ 7 days  | ≤ 30 days |
| High (stored XSS, webhook forgery, SSRF, DoS)                    | ≤ 14 days | ≤ 60 days |
| Medium / low                                                    | ≤ 30 days | Next release cycle |

Fixes are coordinated via the GitHub advisory; once a fix is merged on `main`
and (where applicable) back-ported to the latest tag, the advisory is published
and a CVE requested for issues at the high or critical level.
