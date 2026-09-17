---
id: UC-05-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-17
---

# UC-05 context — Remove Application from Environment

## Delivery state

Implemented and end-to-end verified on `kind-local` and AWS. Two teardown edge cases remain explicitly deferred.

## Read in this order

1. [Specification](specification.md)
2. [Use Case Realization](realization.md)
3. [Sequence diagram](sequence.puml)
4. [VOPC](vopc.puml)
5. [Verification index](../../verification/README.md)

## Important decision

- [ADR-015 — Teardown is a separate use case](../../decisions/ADR-015-separate-remove-use-case.md)

## Open issues

- [D13 — Removed status without cluster verification](../../backlog/D13-teardown-removal-verification.md)
- [D14 — Resource-only teardown leaves delivery objects](../../backlog/D14-teardown-delivery-cleanup.md)

## Implementation entry points

UC-05 deliberately reuses the UC-03 orchestrator, worker, plan, persistence, and integration layers. Start with [`orchestrator.go`](../../../idp/backend/internal/service/orchestrator.go), [`worker.go`](../../../idp/backend/internal/service/worker.go), and [`plan.go`](../../../idp/backend/internal/domain/plan.go).
