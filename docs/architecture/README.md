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
| Participating classes and layers | [VOPC guide](design-classes.md) | [Consolidated design class diagram](design-class-diagram.puml) |
| Domain objects and invariants | [Domain objects](domain/domain-objects.md) | [Domain model](domain/domain-model.puml) |
| Persistence classification | [Persistence classification](domain/persistence-classification.md) | — |
| Database tables, constraints, and ENUMs | [Database schema](database/schema.md) | [ERD](database/erd.puml) |
| Operation pre/postconditions | [Operation contracts](contracts/operation-contracts.md) | — |
| Lifecycle behavior | [State-machine guide](state-machines/README.md) | [`state-machines/`](state-machines/) |
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
