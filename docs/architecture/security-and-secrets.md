---
id: ARCH-SECURITY-SECRETS
artifact: security-and-secrets-architecture
status: current
last_reviewed: 2026-09-18
---

# Security, Authentication and Secret architecture

This document owns the shared security architecture: the implemented UC-06
local authentication boundary and the current storage/materialization rules
for secret values in the UC-03 implementation.

## Authentication boundary

[ADR-019](../decisions/ADR-019-local-authentication-boundary.md) selects local
accounts for the first authentication mechanism. Authentication Middleware is
the only component that translates browser session material into a request
identity. It produces a transient `Principal` containing `userId`, normalized
username and display name. UC-01 through UC-05 consume that principal and must
not read Local Credential, Auth Session or cookies directly.

The boundary is provider-neutral: a future OIDC adapter may establish the same
principal without changing application/domain services. OIDC account linking,
MFA and authorization roles are not implied by the current design.

### Local credential

- Username is normalized before lookup and uniquely constrained in storage.
- Password hashes use Argon2id with a random per-password salt. The encoded
  value includes algorithm version, memory, iteration and parallelism
  parameters so successful login can trigger a later rehash when policy moves.
- Initial parameters are 19 MiB memory, 2 iterations, parallelism 1, a 16-byte
  salt and a 32-byte output. Runtime benchmarking may justify stronger values;
  lowering this floor requires a new security decision.
- Verification uses a fixed dummy encoded hash when the username is absent, and
  the response remains the same for absent, disabled and invalid-password
  accounts.
- Plaintext password is accepted only at the login or trusted CLI boundary and
  must not enter logs, metrics, database fields, command arguments or
  environment variables.
- Reset and account disable atomically revoke every server-side session for the
  affected user.

### Session and cookie

Session Manager generates an opaque token with at least 256 bits of randomness.
Only its SHA-256 hash is used for lookup and stored in `auth_session`; the raw
token exists only in the browser cookie and request. Auth Session also owns
created, last-seen, absolute-expiry and revocation timestamps plus a binding to
the CSRF token. A session is rejected after 30 minutes idle, after 8 hours
absolute lifetime, after explicit revocation or when its account is not
`ACTIVE`.

Production sends the raw token only in `__Host-idp_session` with `Secure`,
`HttpOnly`, `SameSite=Lax`, `Path=/` and no `Domain`. A separate explicitly
enabled local-development profile may use `idp_session` without `Secure` over
HTTP and `idp_csrf` for the CSRF cookie; startup configuration must reject that
profile in production.

### Route and request protection

All page and API routes are protected by default. The React route `/ui/login`,
`GET /api/auth/login-context`, `POST /api/auth/login`, `/ui/assets/...` and
minimal health/readiness endpoints are explicit public exceptions. Other
browser navigation with a missing/invalid session redirects to `/ui/login`
with a validated internal return path, while API calls receive JSON `401` and
are never redirected to HTML.

Every cookie-authenticated state-changing request (`POST`, `PUT`, `PATCH`,
`DELETE`) requires a CSRF token bound to the Auth Session. The raw CSRF value
is sent in a separate `__Host-idp_csrf` cookie readable by same-origin page
code and echoed through a request header or hidden form field; only its hash is
persisted. It is not an authentication credential and cannot create a session.
`SameSite` and Origin/Fetch-Metadata checks are defense in depth, not
replacements for token validation. The public login POST additionally requires
a short-lived pre-auth CSRF nonce issued by
`GET /api/auth/login-context`, plus Origin/Fetch Metadata validation, to prevent
login CSRF. The React login feature keeps the password only in form state until
the request finishes and never writes it to browser storage.

The shared React HTTP transport attaches the session CSRF value to unsafe
requests and performs a full-page navigation to `/ui/login` after protected API
`401`. Authentication errors are not mapped into UC-01 validation errors.

Login rate limiting uses expiring buckets for both normalized username and
request source. Bucket keys persist as keyed HMAC-SHA-256 values and contain no
username, password, session token or request payload. The default thresholds
and visible behavior are owned by the
[UC-06 specification](../use-cases/UC-06/specification.md).
Request source comes from the transport peer; forwarded client-address headers
are trusted only when the direct peer is a configured trusted reverse proxy.
The HMAC key is supplied as a protected runtime secret and is never stored in
the application database or logs; rotating it only invalidates old rate-limit
bucket lookups, not user credentials or sessions.

## Storage boundary

The Secret Store implementation keeps each value in an encrypted file using AES-256-GCM. It derives the 256-bit encryption key from `IDP_SECRET_KEY` with SHA-256, uses a fresh random GCM nonce per write, authenticates the opaque reference as associated data, stores directories with mode `0700`, and stores encrypted files with mode `0600`.

Callers persist only opaque references of the form `idpsecret://<id>`. Plaintext secret values, Git hosting tokens, delivery-repository private keys, and cluster credentials must not be written to the IDP database, Delivery Repository, Deployment Record, deployment-step detail, or application logs. `IDP_SECRET_KEY` must be supplied to every process that reads the same store and is not written by the Secret Store itself.

## Browser storage

UC-01 declares Secret names only; it has no input for a Secret value. Browser
`sessionStorage` holds only non-sensitive draft content, as decided in
[ADR-016](../decisions/ADR-016-client-owned-drafts.md) and
[ADR-017](../decisions/ADR-017-react-web-frontend.md). Tokens, credentials and
plaintext Secret values are never written to browser storage.

## Workload materialization

During deployment, Secret Materializer resolves authorized references in memory and creates or updates a Kubernetes `Secret` through the Kubernetes API. Generated workload manifests refer to that object through `secretKeyRef`; plaintext never enters Git desired state. A keyed HMAC of the materialized secret inputs is placed in the pod-template annotation so secret changes trigger rollout without exposing the value.

The target namespace owns the materialized Kubernetes Secret. Cleanup follows workload/namespace lifecycle; orphan compensation before successful save remains tracked in [D07](../backlog/D07-orphaned-secret.md).

## Implementation anchors

- Authentication service and password hashing: `idp/backend/internal/authentication/`
- Authentication persistence: `idp/backend/internal/persistence/authentication_repository.go`
- HTTP middleware and login/logout API: `idp/backend/internal/web/authentication.go`
- React login and shared transport: `idp/frontend/src/features/authentication/`, `idp/frontend/src/shared/api/http.ts`
- Encrypted store: `idp/backend/internal/integration/secretstore/secretstore.go`
- Manifest references and HMAC: `idp/backend/internal/domain/manifest/pipeline.go`
- Kubernetes write path: the selected CD/Kubernetes adapter invoked by `idp/backend/internal/service/worker.go`
