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
| IMP-001 | UC-02 is designed but the current executable imports its data through fixtures rather than implementing its complete interaction. The fixture importer also still writes UC-01 application versions without the UC-01 validator. | Keep the UC-02 specification as required future scope; do not describe fixture import as UC-02 delivery or as a UC-01 Save. |
| IMP-002 | UC-04 has a minimal query/UI path needed for deployment tracking, not a proven implementation of every specified query scenario. | Treat UC-04 as partially implemented until its traceability and test coverage are completed. |
| IMP-003 | Worker crash recovery is not fully implemented or exercised. | [D06](../backlog/D06-worker-recovery.md) remains authoritative. |
| IMP-004 | Teardown can mark workload removal without definitive cluster verification in some unavailable-cluster paths. | [D13](../backlog/D13-teardown-removal-verification.md) remains authoritative. |
| IMP-005 | A resource-only teardown path may leave CD objects, desired state, or namespace cleanup incomplete. | [D14](../backlog/D14-teardown-delivery-cleanup.md) remains authoritative. |
| IMP-006 | `FinishDeployment` atomically updates the Deployment Record, final Deployment status, execution-job status, unfinished steps, and Resource Instance associations. | This is accepted current behavior and is reflected in operation contract 9; preserve the single transaction. |
| IMP-007 | A running Resource Instance whose resolved Resource Definition identity changes is rejected with `RESOURCE_DEFINITION_CHANGED`; the implementation does not replace it automatically. | Treat this as UC-03 A1-7. Catalog-version policy beyond this safety rule remains in [D11](../backlog/D11-catalog-formula-change.md). |
| IMP-008 | Terraform and Git are invoked through their CLIs; the implementation intentionally does not use `terraform-exec` or `go-git`. Kubernetes integration uses `client-go`. | This is an implementation choice, not a domain requirement; tests and runtime images must provide the selected binaries. |
| IMP-009 | Logical image hosts such as `registry.company.local` are mapped per target from `k8s-cluster.default_parameters.image_registry_mirror`. | Keep logical repositories in application definitions and perform mapping in the Target Manifest Adapter. |
| IMP-010 | On kind, data resources use module-owned `res-<name>` namespaces while workloads use `<application>-<environment>`. | Do not infer that all resources share the workload namespace; namespace cleanup follows the owner that created it. |
| IMP-011 | The fixture importer resolves component IDs across all known application versions and does not validate UC-02 configuration against only the latest version. UC-03 validates bindings again against the version being deployed. | Fixture import is not the complete UC-02 interaction (IMP-001); deployment-time validation is authoritative for execution. |
| IMP-013 | The backend has no authentication or authorization. The UC-01 preconditions "Developer đã đăng nhập" and "có quyền tạo hoặc chỉnh sửa application" are not enforced; this applies to every current API and page. | Do not treat the UC-01 implementation as enforcing access control; authentication needs its own design decision. |
| IMP-014 | UC-01 Save commits the new version, then generates and stores its Application Specification in a separate write. If that second write fails, the request returns an error although the version exists without a specification, and nothing retries it. | Contract 2 keeps previous specifications intact; the missing-specification repair path is not implemented. |
| IMP-012 | Deployment steps are inserted when execution begins rather than pre-created as `PENDING`; on finish, any remaining `PENDING`/`RUNNING` step becomes `SKIPPED`. | This is current progress behavior. [D04](../backlog/D04-deployment-progress-ownership.md) remains open for the final generalized model. |

When a deviation is resolved, update the canonical specification/design, tests, verification record, and this table in the same change.
