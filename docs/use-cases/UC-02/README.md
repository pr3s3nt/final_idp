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
3. [Sequence diagram](../../../sequence_digrams/uc_02_configure_application_environment.puml)
4. [VOPC](../../../01_vopc_design_class_diagram/vopc_uc02.puml)

## Shared artifacts

- [Domain objects](../../../02_domain_model/domain_objects.md)
- [Database schema](../../../03_database_erd/schema.md)
- [Operation contracts](../../../04_operation_contracts/operation_contracts.md)

## Open issues

- [D1 — Draft lifetime across requests](../../backlog/D01-draft-lifetime.md)
- [D7 — Orphaned secret after save failure](../../backlog/D07-orphaned-secret.md)
- [D12 — Catalog version used to list outputs](../../backlog/D12-uc02-catalog-version.md)

## Implementation entry points

- Fixture/import support: [`uc03/internal/fixtures`](../../../uc03/internal/fixtures/)
- Configuration persistence used by deployment: [`environment_configuration_repository.go`](../../../uc03/internal/persistence/environment_configuration_repository.go)
