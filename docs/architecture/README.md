---
id: ARCHITECTURE-INDEX
artifact: architecture-index
status: current
last_reviewed: 2026-09-17
---

# Architecture index

Shared models are canonical here; use-case packages link to them rather than duplicating their definitions.

## Views and artifacts

| Concern | Textual source | Diagram/source |
|---|---|---|
| Participating classes and layers | [VOPC guide](../../01_vopc_design_class_diagram/README.md) | [Consolidated design class diagram](../../01_vopc_design_class_diagram/design_class_diagram.puml) |
| Domain objects and invariants | [Domain objects](../../02_domain_model/domain_objects.md) | [Domain model](../../02_domain_model/domain_model.puml) |
| Persistence classification | [Persistence classification](../../02_domain_model/persistence_classification.md) | — |
| Database tables, constraints, and ENUMs | [Database schema](../../03_database_erd/schema.md) | [ERD](../../03_database_erd/erd.puml) |
| Operation pre/postconditions | [Operation contracts](../../04_operation_contracts/operation_contracts.md) | — |
| Lifecycle behavior | [State-machine guide](../../05_state_machines/README.md) | [`05_state_machines/`](../../05_state_machines/) |
| Cross-artifact coverage | [Traceability matrix](../../06_traceability/traceability_matrix.md) | — |

## Canonical ownership

- Use-case specifications own externally required behavior.
- Use Case Realizations own operation/participant mappings.
- The domain model owns business concepts and relationships.
- The database schema owns physical table, column, constraint, and literal ENUM definitions.
- State machines own permitted lifecycle transitions.
- ADRs own the rationale and consequences of accepted architectural choices.

## Main architectural flow

The Deployment API asks the Deployment Orchestrator to validate input and build a plan. Confirmation atomically changes status and creates a job. A background Deployment Worker executes the job by dependency wave, reconciles infrastructure, resolves configuration, publishes desired state through a CD provider, verifies workload health, collects outputs, and persists progress/results. UC-05 reuses the same execution framework with a removal plan ordered in reverse dependency order.

Detailed interactions remain in the per-use-case sequence diagrams and realizations.
