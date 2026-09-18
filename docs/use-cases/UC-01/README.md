---
id: UC-01-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-18
---

# UC-01 context — Create / Configure Application

## Delivery state

Implemented and verified by automated tests on 2026-09-18. The
[API/editor record](../../verification/2026-09-18-uc01-react-editor.md) covers
the original end-to-end implementation checks; the
[Application Builder record](../../verification/2026-09-18-uc01-application-builder.md)
covers the accepted component-focused presentation. The React editor under
`/ui/applications` covers the main flow, A1 and A2. Known limits are listed as
IMP-013 and IMP-014 in
[implementation deviations](../../implementation/deviations.md).

## Read in this order

1. [Specification](specification.md)
2. [Use Case Realization](realization.md)
3. [Application Builder UI design](ui/README.md)
4. [Sequence diagram](sequence.puml)
5. [VOPC](vopc.puml)

## Shared artifacts

- [Domain objects](../../architecture/domain/domain-objects.md)
- [Database schema](../../architecture/database/schema.md)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)

## Open issues

- [D7 — Orphaned secret after save failure](../../backlog/D07-orphaned-secret.md)

## Resolved decisions

- [ADR-016 — Browser-owned drafts](../../decisions/ADR-016-client-owned-drafts.md)
- [ADR-017 — React web frontend](../../decisions/ADR-017-react-web-frontend.md)

## Implementation entry point

- Web UI: [`idp/frontend`](../../../idp/frontend/README.md)
- API, service, validator, repository: see the UC-01 rows in the [backend implementation map](../../../idp/backend/README.md)
- Fixture import under [`idp/backend/internal/fixtures`](../../../idp/backend/internal/fixtures/) still seeds demo data; it does not apply the UC-01 validator.
