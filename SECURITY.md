# Security Policy

## Supported versions

This is a `0.x` boilerplate under active development. Security fixes land on the
latest `0.x` release and `main`.

| Version | Supported |
|---|---|
| latest `0.x` / `main` | ✅ |
| older tags | ❌ |

## Reporting a vulnerability

**Please do not open a public issue for security problems.**

Report privately via GitHub's **[Report a vulnerability](https://github.com/prajwalmahajan101/gin_boilerplate/security/advisories/new)**
(Security → Advisories). If that is unavailable, open a minimal issue asking to
be contacted privately — without any exploit detail.

Please include:

- affected version / commit,
- a description and, where possible, a minimal reproduction,
- impact assessment (what an attacker can do).

### What to expect

- Acknowledgement within a few days.
- An initial assessment and, if accepted, a fix on `main` plus a patched release.
- Credit in the release notes if you'd like it.

## Scope notes

This is a starter template, not a hosted service. A few deliberate ceilings are
documented rather than hidden:

- **SSRF guard** (`AssertPublicURL`) rejects loopback/private/link-local hosts but
  does not pin DNS — a rebind between check and dial is possible. Dial-time
  control is a planned v0.2 hardening (see `docs/adr/0004-resilience-scope.md`).
- Secrets come from the environment / AWS Secrets Manager — never commit real
  secrets; `.env` is git-ignored and `.env.example` ships only placeholders.
