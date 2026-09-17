---
id: UC-02-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-17
---

# UC-02 context — Configure Application Environment

## Delivery state

Designed. The current Go implementation imports Environment Configuration data as fixtures; it does not implement the complete UC-02 interaction flow.

## Read in this order

1. [Specification](specification.md)
2. [Use Case Realization](realization.md)
3. [Sequence diagram](sequence.puml)
4. [VOPC](vopc.puml)

## Shared artifacts

- [Domain objects](../../architecture/domain/domain-objects.md)
- [Database schema](../../architecture/database/schema.md)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)

## Open issues

- [D7 — Orphaned secret after save failure](../../backlog/D07-orphaned-secret.md)
- [D12 — Catalog version used to list outputs](../../backlog/D12-uc02-catalog-version.md)

## Resolved decisions

- [ADR-016 — Browser-owned drafts](../../decisions/ADR-016-client-owned-drafts.md)

## Implementation entry points

- Fixture/import support: [`idp/backend/internal/fixtures`](../../../idp/backend/internal/fixtures/)
- Configuration persistence used by deployment: [`environment_configuration_repository.go`](../../../idp/backend/internal/persistence/environment_configuration_repository.go)
