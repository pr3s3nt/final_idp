# Step 5: State machines — MVP

Contracts C2–C13 and [deployment design](../MVP_DEPLOYMENT_DESIGN.md) define transitions. This profile has one worker, no automatic retry/replay, permanent secret references and Argo CD delivery.

## Physical enum registry

| Field | Literals |
|---|---|
| deployment.status / deployment_record.status | AWAITING_CONFIRMATION, QUEUED, RUNNING, SUBMITTED, FAILED |
| deployment_step.step_name | INFRASTRUCTURE_READY, CONFIGURATION_RESOLVED, MANIFEST_GENERATED |
| deployment_step.status | PENDING, RUNNING, SUCCEEDED, FAILED, SKIPPED |
| resource_instance.status | PLANNED, PROVISIONING, READY, FAILED, RETIRED |
| deployment_execution_job.status | QUEUED, CLAIMED, SUCCEEDED, FAILED |
| deployment_execution_job.phase | INFRASTRUCTURE, CONFIGURATION, MANIFEST, PUBLISH, COMPLETE (NULL before claim) |
| deployment_scope_guard.status | IDLE, EXECUTING, RECOVERY_REQUIRED |
| deployment_record.delivery_status | NOT_PUBLISHED, ACCEPTED, SYNCING, SYNCED, OUT_OF_SYNC, DEGRADED, FAILED, UNKNOWN |
| dependency.target_type | WORKLOAD, RESOURCE |
| configuration_value.value_source | DIRECT, RESOURCE_OUTPUT, WORKLOAD_OUTPUT |
| secret.value_source | SECRET_REF, RESOURCE_OUTPUT (MVP accepts SECRET_REF only) |
| resource_instance.sharing_scope | APPLICATION_ENVIRONMENT, EXPLICIT_SHARED (MVP accepts first only) |
| resource_instance_binding.binding_role | OWNER, SHARED_CONSUMER (MVP accepts OWNER only) |

## Deployment/job/guard transactions

| Event | Deployment/Record | Job | Guard | Steps |
|---|---|---|---|---|
| Prepare | AWAITING_CONFIRMATION; no record yet | absent | unchanged | absent |
| Confirm | QUEUED | QUEUED | EXECUTING | 3 PENDING |
| Claim once | RUNNING | CLAIMED, phase INFRASTRUCTURE, workerRunId | EXECUTING | infrastructure RUNNING |
| Infrastructure complete | RUNNING | phase CONFIGURATION | EXECUTING | infra SUCCEEDED, config RUNNING |
| Configuration complete | RUNNING | phase MANIFEST | EXECUTING | config SUCCEEDED, manifest RUNNING |
| Manifest complete | RUNNING | phase PUBLISH | EXECUTING | all 3 SUCCEEDED |
| Publish acknowledged + complete | SUBMITTED | SUCCEEDED, COMPLETE | IDLE | unchanged |
| Known-safe terminal error | FAILED | FAILED | IDLE | active FAILED, future SKIPPED |
| Uncertain effect / interruption | FAILED | FAILED | RECOVERY_REQUIRED | active FAILED, future SKIPPED |
| Verified manual recovery | stays FAILED | stays FAILED | IDLE with audit | preserved |

Each row that writes multiple objects is ONE DB transaction. Claim/normal worker writes require matching worker_run_id; interruption recovery requires exclusive host lock after old process group is confirmed stopped. No CLAIMED → QUEUED, FAILED → QUEUED or FAILED → RUNNING transition exists for a job/deployment. Retry from UI with same accepted key returns tracking data, not a fresh execution.

Source snapshot, image digests and context are immutable. Fingerprint preflight happens once before provider effects. Scope guard serializes accept and blocks new deployment while external outcome is unresolved.

## Resource recovery and teardown

C5 reserves durable identity/state/binding before apply. READY permits compatible REUSE only; lookup itself does not filter status. Interrupted/failed instances require operator inspection. Recovery can restore READY or set PLANNED + recovery_verified for proven absence, always keeping the same identity/binding. Fresh deployment reviews a new plan. No automated database destruction, replacement or resize.

Explicit final teardown can retire verified cleaned/absent resources and active bindings; preserve rows used by history. Do not delete another cluster's objects or lose Terraform state while provider objects still exist.

## View-only delivery markers

UC-04 compares Argo Application UID, desired source and observed digest before Kubernetes health. Wrong source/UID -> NOT_AVAILABLE with SUPERSEDED/REPLACED; not yet observed/synced or provider unavailable -> PENDING. APPLICATION_READY also requires expected template deployment-id, image digests, observedGeneration and complete replica rollout.

Argo terminal error of the expected revision can yield FAILED in the view; this never rewrites deployment lifecycle or internal steps. Deployment SUBMITTED means acknowledged publication, not application health.
