---
id: PROJECT-CURRENT-STATE
artifact: project-status
status: current
last_reviewed: 2026-09-17
---

# Current project state

## Lifecycle position

The project follows UP. UC-03 and UC-05 are in a late Construction/verification state: executable software exists and has been exercised on local Kubernetes and AWS. The whole product is not in Transition because UC-01, UC-02, and the complete UC-04 experience are not implemented as full product use cases.

This status was reviewed on branch `uc03-impl` against commit `ab74aa2`; later changes must update this document when they change the baseline.

## Documentation layout

The AI-facing layout is complete. Sequence diagrams and per-use-case VOPCs live in `docs/use-cases/UC-*`; shared design artifacts live in `docs/architecture/`; the active coverage matrix lives in `docs/traceability/matrix.md`; operations live under `docs/operations/uc03/`. Consolidated sources, the original implementation plan, and the verification log passed [semantic reconciliation](implementation/documentation-reconciliation.md) before removal from the working tree. Retired paths are blocked by `scripts/check_docs.py`; the completed work is recorded in [MIGRATION_PLAN.md](MIGRATION_PLAN.md).

## Use-case baseline

| ID | Status | Notes |
|---|---|---|
| UC-01 | Designed | Application data is currently supplied to the implementation through fixtures/import rather than the complete authoring workflow. |
| UC-02 | Designed | Environment configuration is currently supplied through fixtures/import rather than the complete interactive workflow. |
| UC-03 | Implemented and E2E verified | Verified with an existing internal kind cluster and with AWS infrastructure. Fleet is the default CD provider for `kind-local`; AWS continues to use Argo CD. |
| UC-04 | Partially implemented | The implementation exposes the execution status/query path required to observe UC-03. Treat the full UC-04 specification as design scope, not as fully delivered scope. |
| UC-05 | Implemented and E2E verified | Verified on `kind-local` and AWS; unresolved teardown edge cases remain in D13 and D14. |

## Current architectural baseline

- Deployment confirmation persists a job and returns; a background Deployment Worker performs the long-running work.
- Application Definition and Platform Catalog are versioned and immutable.
- A deployment targets either cloud infrastructure managed by the IDP or a pre-existing internal Kubernetes cluster.
- Each application has its own delivery repository and key pair.
- The CD integration is provider-neutral. Fleet is the default on `kind-local`; Argo CD remains supported and is used on AWS.
- UC-05 reuses deployment orchestration primitives to remove an application from one environment and target.

The accepted rationale is indexed in [decisions/README.md](decisions/README.md).

## Known limitations

The authoritative list is [backlog/README.md](backlog/README.md). High-impact items include worker recovery after interruption, plan/input concurrency protection, exact teardown verification, and cleanup of CD/desired-state objects in resource-only teardown paths.

## Historical provenance

The removed implementation plan and consolidated logs are available through the exact Git references in the [documentation reconciliation](implementation/documentation-reconciliation.md). Their current concepts were migrated before removal. Dated records in the [verification index](verification/README.md) remain the reviewable evidence snapshots; neither Git history nor verification evidence defines required system behavior.
