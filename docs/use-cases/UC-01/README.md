---
id: UC-01-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-17
---

# UC-01 context — Create / Configure Application

## Delivery state

Designed. The current Go implementation imports Application Definitions as fixtures; it does not implement the complete UC-01 authoring workflow.

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

- [D1 — Draft lifetime across requests](../../backlog/D01-draft-lifetime.md)
- [D7 — Orphaned secret after save failure](../../backlog/D07-orphaned-secret.md)

## Implementation entry point

Fixture/import support is under [`idp/backend/internal/fixtures`](../../../idp/backend/internal/fixtures/); it is not a substitute for the complete use case.
