---
id: BACKLOG-INDEX
artifact: risk-and-issue-index
status: current
last_reviewed: 2026-09-17
---

# Deferred risks and design issues

This index is the authoritative list of known unresolved design work. Detailed analysis is split by issue; the original consolidated record remains at [`06_traceability/deferred_issues.md`](../../06_traceability/deferred_issues.md) for provenance.

| ID | Summary | Impact | Main scope | Status |
|---|---|---|---|---|
| [D01](D01-draft-lifetime.md) | Draft lifetime across requests | Medium | UC-01, UC-02 | Deferred |
| [D02](D02-uc04-provider-preconditions.md) | UC-04 external-provider preconditions | Medium | UC-04 | Deferred |
| [D03](D03-typed-infrastructure-plan.md) | Typed Infrastructure Plan | Medium | UC-03 | Deferred |
| [D04](D04-deployment-progress-ownership.md) | Progress-marker ownership | Medium | UC-03, UC-04 | Deferred |
| [D05](D05-lifecycle-vs-delivery-status.md) | Lifecycle versus delivery status | High | UC-03, UC-04 | Deferred |
| [D06](D06-worker-recovery.md) | Worker interruption and recovery | High | UC-03, UC-05 | Deferred |
| [D07](D07-orphaned-secret.md) | Orphaned Secret compensation | High | UC-01, UC-02 | Deferred |
| [D08](D08-workload-output-reader.md) | Concrete Workload Output reader | Medium | UC-03 | Deferred |
| [D09](D09-shared-and-workload-resources.md) | Shared and per-workload resources | Medium | UC-03 | Deferred |
| [D10](D10-old-catalog-version-policy.md) | Policy for old Catalog Versions | Medium | UC-03 | Deferred |
| [D11](D11-catalog-formula-change.md) | Catalog formula replacement | High | UC-03 | Deferred |
| [D12](D12-uc02-catalog-version.md) | Catalog Version used by UC-02 | Medium | UC-02, UC-03 | Deferred |
| [D13](D13-teardown-removal-verification.md) | Verify removal before terminal status | High | UC-05 | Deferred |
| [D14](D14-teardown-delivery-cleanup.md) | Clean delivery objects in resource-only teardown | High | UC-05 | Deferred |

Impact values are navigation aids, not a formal delivery commitment. Assign a target iteration before starting work rather than silently changing a deferred item.
