---
id: PROJECT-CURRENT-STATE
artifact: project-status
status: current
last_reviewed: 2026-09-18
---

# Current project state

## Lifecycle position

The project follows UP. UC-03 and UC-05 are in a late Construction/verification state: executable software exists and has been exercised on local Kubernetes and AWS. UC-01, UC-02 and UC-06 are implemented and verified by automated tests. The whole product is not in Transition because the complete UC-04 experience is not implemented as a full product use case, and UC-01, UC-02 and UC-06 have not been exercised in a deployed browser environment.

This status was reviewed on branch `uc03-impl` on 2026-09-18. The UC-01
implementation and component-focused Application Builder were reviewed through
commit `1417b5e`. UC-06 now has a local-user migration and CLI, Argon2id
credentials, server-side sessions, route middleware, CSRF protection and a
React login/logout flow. Its automated evidence is recorded in
[the UC-06 verification record](verification/2026-09-18-uc06-local-authentication.md).
UC-02 was implemented through commit `c35d48c`: a stateless Environment
Configuration Service, its JSON API, compare-and-write save and a React
Configuration page built on an accepted user interface design. Its automated
evidence is recorded in
[the UC-02 verification record](verification/2026-09-18-uc02-environment-configuration.md).

## Documentation layout

The AI-facing layout is complete. Sequence diagrams and per-use-case VOPCs live in `docs/use-cases/UC-*`; shared design artifacts live in `docs/architecture/`; the active coverage matrix lives in `docs/traceability/matrix.md`; operations live under `docs/operations/uc03/`. Consolidated sources, the original implementation plan, and the verification log passed [semantic reconciliation](implementation/documentation-reconciliation.md) before removal from the working tree. Retired paths are blocked by `scripts/check_docs.py`; the completed work is recorded in [MIGRATION_PLAN.md](MIGRATION_PLAN.md).

## Use-case baseline

| ID | Status | Notes |
|---|---|---|
| UC-01 | Implemented, automated tests | Component-focused React Application Builder under `/ui/applications` with a same-origin JSON API ([ADR-017](decisions/ADR-017-react-web-frontend.md), [latest UI evidence](verification/2026-09-18-uc01-application-builder.md)). Routes are now protected by UC-06. The editor itself has not been exercised in a deployed environment; fixtures still seed demo applications. |
| UC-02 | Implemented, automated tests | Interactive Configuration page under `/ui/applications/{id}/configuration` with a same-origin JSON API, Secret staging and optimistic save ([ADR-020](decisions/ADR-020-uc02-catalog-version-and-target.md), [UI design](use-cases/UC-02/ui/README.md), [latest evidence](verification/2026-09-18-uc02-environment-configuration.md)). Fixtures still seed configurations for demo applications. The page has not been exercised in a deployed environment, and the staged-Secret lifetime of D07 is unresolved. |
| UC-03 | Implemented and E2E verified | Verified with an existing internal kind cluster and with AWS infrastructure. Fleet is the default CD provider for `kind-local`; AWS continues to use Argo CD. |
| UC-04 | Partially implemented | The implementation exposes the execution status/query path required to observe UC-03. Treat the full UC-04 specification as design scope, not as fully delivered scope. |
| UC-05 | Implemented and E2E verified | Verified on `kind-local` and AWS; unresolved teardown edge cases remain in D13 and D14. |
| UC-06 | Implemented, automated tests | Local account, Argon2id credential, server-side session, login/logout, CSRF, protected-route middleware and CLI provisioning are implemented. PostgreSQL integration tests cover account/session lifecycle; React component tests cover login states. Real-browser and deployed-environment verification remain outstanding. |

## Current architectural baseline

- Deployment confirmation persists a job and returns; a background Deployment Worker performs the long-running work.
- Application Definition and Platform Catalog are versioned and immutable.
- A deployment targets either cloud infrastructure managed by the IDP or a pre-existing internal Kubernetes cluster.
- Each application has its own delivery repository and key pair.
- The CD integration is provider-neutral. Fleet is the default on `kind-local`; Argo CD remains supported and is used on AWS.
- UC-05 reuses deployment orchestration primitives to remove an application from one environment and target.
- UC-01 and UC-02 drafts are owned by the browser tab and saved with optimistic concurrency (ADR-016). Both editors are React apps in `idp/frontend`, served by the Go backend under `/ui/`; UC-03 to UC-05 keep their Go templates (ADR-017).
- UC-02 asks the Developer for a Catalog Version and a deployment target to decide which Resource Definition supplies the valid outputs. Neither is persisted, so an Environment Configuration stays unversioned and UC-03 still checks it against the version and catalog chosen for that deployment (ADR-020).
- UC-06 places local username/password authentication behind middleware that exposes a provider-neutral `Principal`; its login page is a React feature at `/ui/login`, while OIDC/SSO and detailed authorization remain out of scope (ADR-019).

The accepted rationale is indexed in [decisions/README.md](decisions/README.md).

## Known limitations

The authoritative list is [backlog/README.md](backlog/README.md). High-impact items include worker recovery after interruption, plan/input concurrency protection, exact teardown verification, cleanup of CD/desired-state objects in resource-only teardown paths, and authenticated browser-draft isolation (D15).

## Historical provenance

The removed implementation plan and consolidated logs are available through the exact Git references in the [documentation reconciliation](implementation/documentation-reconciliation.md). Their current concepts were migrated before removal. Dated records in the [verification index](verification/README.md) remain the reviewable evidence snapshots; neither Git history nor verification evidence defines required system behavior.
