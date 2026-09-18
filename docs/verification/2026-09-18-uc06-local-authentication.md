---
id: VER-2026-09-18-UC06-AUTH
artifact: verification-record
status: evidence
executed_on: 2026-09-18
implementation: UC-06 local implementation in the same logical change
---

# UC-06 local authentication

> This file records observations from a specific execution. It is evidence,
> not a normative requirement.

Environment: local WSL2 workstation, Go 1.27.1, Node.js 24.16.0, npm 11.13.0
and PostgreSQL in the dedicated local `idp_test` database. The checkout was
branch `uc03-impl`. No cloud account or Kubernetes cluster was mutated.

## Scope

This execution verified the first UC-06 implementation: local-user lifecycle,
Argon2id credentials, PostgreSQL-backed login attempts and sessions, protected
route middleware, CSRF/cookie handling, React login states, logout and the
shared frontend API transport.

## Automated checks

| Command | Directory | Result |
|---|---|---|
| `go test ./...` | `idp/backend` | pass: authentication, configuration, web and all existing backend packages |
| `go build ./...` | `idp/backend` | pass |
| `go test -tags=integration ./internal/service -run TestLocalAuthenticationPersistence -count=1 -v` | `idp/backend` | pass against PostgreSQL after reset/migration of dedicated `idp_test` |
| `npm run lint` | `idp/frontend` | pass: ESLint and strict TypeScript |
| `npm test -- --run` | `idp/frontend` | pass: 7 files, 46 tests |
| `npm run build` | `idp/frontend` | pass: Vite production bundle, 38 modules transformed |
| `python3 scripts/check_docs.py` | repository root | pass: 96 Markdown files and 94 artifact IDs; PlantUML unavailable locally |
| `git diff --check` | repository root | pass |

The PostgreSQL integration test observed that:

- migration `0004_local_authentication.sql` creates the account, credential,
  session and login-attempt persistence required by UC-06;
- a local user can be created and authenticated with the normalized username;
- a duplicate username is rejected;
- successful login persists only token/CSRF hashes and produces a valid
  request principal;
- password reset invalidates the old password and revokes the existing
  session; the new password can establish another session;
- disabling the account revokes that session and prevents authentication.

Unit and component tests additionally observed generic invalid-credential
handling, account/source rate-limit semantics, safe internal return paths,
browser redirect versus API `401`, session and pre-auth CSRF rejection,
development/secure cookie profiles, password clearing after a failed submit,
rate-limit and expired-form presentation, and CSRF injection by the shared
React transport.

## Not executed

- No real-browser/manual visual review of `/ui/login` or the authenticated
  shell was performed.
- No HTTPS reverse-proxy or deployed-environment verification was performed;
  production `Secure` cookie behavior still requires that environment.
- No full UC-03/UC-05 Kubernetes or cloud E2E run was repeated because UC-06
  changes only their HTTP access boundary, not deployment execution.
- No PlantUML syntax check was run because PlantUML was unavailable locally;
  this implementation round did not change PlantUML sources.
