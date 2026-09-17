---
id: UC-04-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-17
---

# UC-04 context — View Deployment Result

## Delivery state

Designed and partially implemented. The current code provides the query/status path required to observe UC-03 and UC-05, but the complete use-case scope must not be assumed delivered.

## Read in this order

1. [Specification](specification.md)
2. [Use Case Realization](realization.md)
3. [Sequence diagram](sequence.puml)
4. [VOPC](vopc.puml)

## Open issues

- [D2 — Provider call without required conditions](../../backlog/D02-uc04-provider-preconditions.md)
- [D5 — Deployment lifecycle mixed with delivery status](../../backlog/D05-lifecycle-vs-delivery-status.md)

## Implementation entry points

- Query service: [`query.go`](../../../uc03/internal/service/query.go)
- Persistence queries: [`deployment_queries.go`](../../../uc03/internal/persistence/deployment_queries.go)
- Web/API layer: [`uc03/internal/web`](../../../uc03/internal/web/)
