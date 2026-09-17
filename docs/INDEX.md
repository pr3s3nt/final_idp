---
id: DOC-INDEX
artifact: documentation-index
status: current
last_reviewed: 2026-09-17
---

# Documentation index

This is the authoritative entry point for project documentation. It tells readers and AI agents which artifact is current and which context to load for a task.

## Current baseline

- Development method: Unified Process (UP).
- Current emphasis: Construction, with Transition-style deployment and end-to-end verification activities for UC-03 and UC-05.
- Implementation branch at the time of this review: `uc03-impl`.
- Detailed status: [CURRENT_STATE.md](CURRENT_STATE.md).
- Documentation conventions: [DOCUMENTATION_RULES.md](DOCUMENTATION_RULES.md).
- Active physical-layout migration: [MIGRATION_PLAN.md](MIGRATION_PLAN.md).

## Read by task

| Task | Read first |
|---|---|
| Understand overall scope or terminology | [Current state](CURRENT_STATE.md), then [glossary](GLOSSARY.md) |
| Change UC-01 behavior | [UC-01 context](use-cases/UC-01/README.md) |
| Change UC-02 behavior | [UC-02 context](use-cases/UC-02/README.md) |
| Change deployment or planning | [UC-03 context](use-cases/UC-03/README.md) |
| Change result queries/status presentation | [UC-04 context](use-cases/UC-04/README.md) |
| Change teardown/removal | [UC-05 context](use-cases/UC-05/README.md) |
| Change domain or persistence model | [Architecture index](architecture/README.md) |
| Change a database table, constraint, or ENUM | [Database schema](architecture/database/schema.md) |
| Understand why a design choice was made | [Decision index](decisions/README.md) |
| Pick up an unresolved design problem | [Backlog index](backlog/README.md) |
| Compare design with code | [Implementation index](implementation/README.md) |
| Run or troubleshoot the system | [Runbook](../uc03/docs/RUNBOOK.md) |
| Review actual test executions | [Verification index](verification/README.md) |

## Artifact map

| UP artifact or concern | Status | Canonical location |
|---|---|---|
| Use-Case Model | Current | [`use-cases/`](use-cases/) |
| Use Case Realizations | Current | Each use-case package's `realization.md` |
| Software architecture and shared models | Current | [Architecture index](architecture/README.md) |
| Operation contracts | Current | [Operation contracts](architecture/contracts/operation-contracts.md) |
| State machines | Current | [State-machine index](architecture/state-machines/README.md) |
| Design decisions | Current split index plus original record | [Decision index](decisions/README.md) |
| Deferred risks/issues | Current split index plus original details | [Backlog index](backlog/README.md) |
| Traceability | Current | [Traceability index](traceability/README.md) |
| UC-03/UC-05 source code | Current | [`uc03/`](../uc03/) |
| Original implementation plan | Historical/mixed | [`implementation_plan.md`](../implementation_plan.md) |
| End-to-end execution records | Evidence, not specification | [Verification index](verification/README.md) |
| Iteration records | Current policy; historical work was not formally numbered | [Iteration index](iterations/README.md) |
| Archived/superseded material | Historical, never normative | [Archive policy](archive/README.md) |
| Documentation migration map | Current until the legacy layout is removed | [Migration plan](MIGRATION_PLAN.md) |

## Authority rules

If two documents disagree, use this order:

1. An accepted, non-superseded ADR for the decision it owns.
2. The current use-case specification for required behavior.
3. Shared canonical architecture/schema artifacts for their model elements.
4. The use-case realization and traceability mapping.
5. Implementation documentation and code observations.
6. Iteration plans, verification records, reviews, and archived material.

Do not resolve a genuine contradiction merely by choosing the newest file date. Record the contradiction in `docs/implementation/deviations.md` or the backlog and reconcile the canonical artifacts.
