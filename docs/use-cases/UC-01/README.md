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
3. [Sequence diagram](../../../sequence_digrams/uc_01_create_configure_application.puml)
4. [VOPC](../../../01_vopc_design_class_diagram/vopc_uc01.puml)

## Shared artifacts

- [Domain objects](../../../02_domain_model/domain_objects.md)
- [Database schema](../../../03_database_erd/schema.md)
- [Operation contracts](../../../04_operation_contracts/operation_contracts.md)

## Open issues

- [D1 — Draft lifetime across requests](../../backlog/D01-draft-lifetime.md)
- [D7 — Orphaned secret after save failure](../../backlog/D07-orphaned-secret.md)

## Implementation entry point

Fixture/import support is under [`uc03/internal/fixtures`](../../../uc03/internal/fixtures/); it is not a substitute for the complete use case.
