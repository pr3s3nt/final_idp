---
id: UC-02-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-18
---

# UC-02 context — Configure Application Environment

## Delivery state

Implemented and covered by automated tests. The Configuration page lives at
`/ui/applications/{applicationId}/configuration` and saves through the
Environment Configuration API; fixtures still import Environment Configuration
data for the demo applications. The page has not been exercised in a deployed
browser environment, and the staged-Secret lifetime of
[D07](../../backlog/D07-orphaned-secret.md) is still unresolved. The latest
evidence is the
[UC-02 verification record](../../verification/2026-09-18-uc02-environment-configuration.md).

## Read in this order

1. [Specification](specification.md)
2. [Use Case Realization](realization.md)
3. [Configuration UI design](ui/README.md)
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
- [ADR-020 — Catalog Version and deployment target selected in UC-02](../../decisions/ADR-020-uc02-catalog-version-and-target.md)

## Implementation entry points

- Fixture/import support: [`idp/backend/internal/fixtures`](../../../idp/backend/internal/fixtures/)
- Configuration persistence used by deployment: [`environment_configuration_repository.go`](../../../idp/backend/internal/persistence/environment_configuration_repository.go)
