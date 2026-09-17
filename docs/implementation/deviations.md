---
id: IMPLEMENTATION-DEVIATIONS
artifact: design-implementation-deviations
status: current
last_reviewed: 2026-09-17
---

# Known design and implementation deviations

This file lists current differences that an AI agent must not infer away. Historical deviations already reconciled into design or ADRs do not belong here.

| ID | Difference | Authority/action |
|---|---|---|
| IMP-001 | UC-01 and UC-02 are designed but the current executable imports their data through fixtures rather than implementing their complete interactions. | Keep the use-case specifications as required future scope; do not describe fixture import as full UC-01/UC-02 delivery. |
| IMP-002 | UC-04 has a minimal query/UI path needed for deployment tracking, not a proven implementation of every specified query scenario. | Treat UC-04 as partially implemented until its traceability and test coverage are completed. |
| IMP-003 | Worker crash recovery is not fully implemented or exercised. | [D06](../backlog/D06-worker-recovery.md) remains authoritative. |
| IMP-004 | Teardown can mark workload removal without definitive cluster verification in some unavailable-cluster paths. | [D13](../backlog/D13-teardown-removal-verification.md) remains authoritative. |
| IMP-005 | A resource-only teardown path may leave CD objects, desired state, or namespace cleanup incomplete. | [D14](../backlog/D14-teardown-delivery-cleanup.md) remains authoritative. |

When a deviation is resolved, update the canonical specification/design, tests, verification record, and this table in the same change.
