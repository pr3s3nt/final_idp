---
id: UC-03-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-17
---

# UC-03 context — Deploy Application

## Delivery state

Implemented and end-to-end verified on an existing internal kind cluster and on AWS. Fleet is the default CD provider on `kind-local`; AWS uses Argo CD.

## Read in this order

1. [Specification](specification.md)
2. [Use Case Realization](realization.md)
3. [Sequence diagram](sequence.puml)
4. [VOPC](vopc.puml)
5. [UC-03 implementation map](../../../idp/backend/README.md)
6. [Verification index](../../verification/README.md)

## Shared artifacts

- [Architecture index](../../architecture/README.md)
- [Database schema](../../architecture/database/schema.md)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Deployment state machine](../../architecture/state-machines/deployment.puml)
- [Traceability matrix](../../traceability/matrix.md)

## Important decisions

- [ADR-002 — Dependency-ordered output propagation](../../decisions/ADR-002-workload-output-ordering.md)
- [ADR-003 — Resource Instance ownership](../../decisions/ADR-003-resource-instance-ownership.md)
- [ADR-010 — Background deployment worker](../../decisions/ADR-010-background-deployment-worker.md)
- [ADR-012 — Target infrastructure and catalog versioning](../../decisions/ADR-012-target-and-catalog-versioning.md)
- [ADR-013 — Per-application delivery repository](../../decisions/ADR-013-per-application-delivery-repository.md)
- [ADR-014 — Fleet as the default CD provider](../../decisions/ADR-014-fleet-default-cd-provider.md)

## Open issues

The highest-impact UC-03 items are [D3](../../backlog/D03-typed-infrastructure-plan.md), [D6](../../backlog/D06-worker-recovery.md), [D8](../../backlog/D08-workload-output-reader.md), [D9](../../backlog/D09-shared-and-workload-resources.md), [D10](../../backlog/D10-old-catalog-version-policy.md), and [D11](../../backlog/D11-catalog-formula-change.md).

## Implementation entry points

| Concern | Code |
|---|---|
| Request orchestration and confirmation | [`orchestrator.go`](../../../idp/backend/internal/service/orchestrator.go) |
| Background execution | [`worker.go`](../../../idp/backend/internal/service/worker.go) |
| Graph, wave, and infrastructure planning | [`idp/backend/internal/domain`](../../../idp/backend/internal/domain/) |
| Persistence | [`idp/backend/internal/persistence`](../../../idp/backend/internal/persistence/) |
| External integrations | [`idp/backend/internal/integration`](../../../idp/backend/internal/integration/) |
