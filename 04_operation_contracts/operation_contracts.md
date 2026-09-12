# Step 4: Operation Contracts

Các contract dùng domain names Step 2 và physical enum Step 3/5. Draft UC-01/UC-02 là full DTO do Web UI sở hữu; backend không giữ draft qua request. `Infrastructure Plan`, item, override definition, outputs và resolved objects đều transient.

## 1. `saveApplicationDefinition(applicationDraft)`

- **Cross references:** UC-01 main/A1.
- **Preconditions:** caller có quyền; full draft có identity/version phù hợp, ít nhất một workload; names, ports, dependencies và configuration definitions hợp lệ.
- **Postconditions:** Application Definition aggregate được persist atomically; specification được xử lý bởi Contract 2 kế tiếp. New items được insert, active items được update. Application/Workload/requirement/definition bị bỏ nhưng đã từng được `workload_deployment`, environment configuration, Resource Instance hoặc history tham chiếu được set `retired_at`, không hard-delete. Hard delete chỉ tùy chọn cho item chưa từng có reference và FK vẫn kiểm tra `RESTRICT`.
- **Guarantees:** không sửa/xóa history, deployment, configuration hoặc Resource Instance. Validation/persist failure giữ nguyên DB. Application Service không retain `applicationDraft`.

## 2. `generateApplicationSpecification(applicationDefinition)`

- **Cross references:** UC-01 main.
- **Preconditions:** persisted active definition hợp lệ.
- **Postconditions:** create/update đúng một `application_specification` cho application; content chỉ gồm active topology/requirements, không có image version, resolved value hay secret plaintext.
- **Guarantees:** generation/save atomic; artifact cũ còn nguyên khi lỗi.

## 3. `saveEnvironmentConfiguration(configurationDraft)`

- **Cross references:** UC-02 main/A1.
- **Preconditions:** full client-owned draft tham chiếu đúng application/environment và active definitions. Direct non-secret values hợp lệ. Secret trực tiếp chỉ xuất hiện dưới dạng opaque staged reference từ `stageSecret(..., idempotencyKey)`; resource/workload output được catalog công bố, workload output phải plan-time resolvable.
- **Postconditions:** configuration/value/reference rows phản ánh đúng submitted draft trong một DB transaction; không persist resolved output/plaintext. Sau DB success, staged references được idempotent `promote`; client chỉ giữ opaque reference.
- **Exceptions/guarantees:** validation hoặc DB save failure rollback DB và gọi idempotent compensating `revokeStagedSecrets` chỉ cho references mới stage trong edit session (không revoke permanent references cũ). Secret Store nằm ngoài DB transaction: stable idempotency key, staging TTL, retry và orphan reconciliation xử lý crash window; save không báo thành công trước promote acknowledgement, và hệ thống không tuyên bố distributed atomicity. Backend không retain `configurationDraft`.

## 4. `createDeployment(applicationId, environment, target, images, context)`

- **Cross references:** UC-03 prepare/review plan, A1.
- **Preconditions:** active application/configuration hợp lệ; images đầy đủ; graph không có runtime-only/circular Workload Output input; definitions/context tồn tại.
- **Postconditions:** persist Deployment `AWAITING_CONFIRMATION`, workload image snapshots và context. Build typed transient `DeploymentGraph`, `ResourceResolution`, `InfrastructurePlan`, `InfrastructurePlanItem`, `OverrideDefinition`.
- **Safe reuse:** repository query always starts from an exact active Resource Instance Binding tuple `application_id + environment + resource_requirement_id + deployment_target`, then joins a `READY` instance. No binding yields a `CREATE` plan item; Contract 7 later creates its `OWNER` binding during worker reconciliation. Cross-boundary reuse is visible to lookup only after platform sharing administration (outside the four Developer UCs) has pre-authorized a `SHARED_CONSUMER` binding with matching `sharing_key`, definition/target compatibility and authorization. Owner columns are never a fallback query path.
- **Canonicalization:** canonical JSON starts with application/environment/target and uses stable item/key ordering, normalized number/unit and no timestamps/generated IDs. Each item contains requirement/definition identities; `provisioner_reference`, `supported_contexts`, `default_parameters`, `allowed_overrides`; action, owner/scope/sharing key, referenced instance, resolved parameters and sorted override schema. Persist only SHA-256 fingerprint + algorithm; return plan payload for review, never persist it.
- **Guarantees:** A1 rolls back all deployment rows and creates no job/provider side effect.

## 5. `confirmDeployment(deploymentId, overrides, idempotencyKey)`

- **Cross references:** UC-03 confirm/A1/concurrency.
- **Preconditions:** Deployment is `AWAITING_CONFIRMATION`; persisted definition/config/image/context/fingerprint inputs exist. No prior request memory is assumed.
- **Postconditions:** rebuild same typed plan from current inputs and scope-safe instances, canonicalize with stored algorithm, compare fingerprint, then validate override key/type/range/enum. On match, one DB transaction performs CAS `AWAITING_CONFIRMATION → QUEUED`, creates/updates Deployment Record to `QUEUED`, creates exactly three `PENDING` step rows (`INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`), and inserts `deployment_execution_job` with selected override values, accepted fingerprint, `QUEUED`, attempt policy and unique deployment/idempotency keys. Return `202 Accepted(deploymentId, trackingId)`; no reconcile/resolve/manifest/CD call occurs in HTTP request.
- **Exceptions/guarantees:** mismatch atomically refreshes fingerprint and returns `PLAN_CHANGED` + rebuilt plan; no job. Invalid override leaves state unchanged. CAS loser returns existing conflict/tracking result. Retry with same idempotency key returns same tracking id. Atomic DB enqueue prevents “confirm succeeded but job lost”. Plan payload/override-definition schema is not persisted; only selected override values required by worker are stored.

## 6. `executeDeploymentJob(jobId)` (internal worker operation)

- **Cross references:** UC-03 asynchronous execution/A2.
- **Preconditions:** claimable `QUEUED` job or expired `CLAIMED` lease; Deployment `QUEUED`/`RUNNING`. Worker rebuilds plan, checks `accepted_plan_fingerprint`, then applies stored overrides before external side effects.
- **Postconditions:** atomically claim lease/increment attempts and set lifecycle/record `RUNNING`; execute contracts 7–10; on success mark job `SUCCEEDED` and lifecycle/record `SUBMITTED`.
- **Retry/idempotency:** leases permit recovery; exponential/backoff via `available_at`; stable keys per `(deployment, phase, resource item)` are passed to provisioner/CD. Because claim calls `markExecutionRunning` before rebuild/verify, fingerprint mismatch is terminal `PLAN_STALE_AFTER_ACCEPT` on transition `RUNNING → FAILED`, before external side effects, and marks `INFRASTRUCTURE_READY` failed. Other retryable errors requeue until `max_attempts`; terminal/exhausted error atomically marks job/deployment/record `FAILED` and redacted detail. The active internal step is `FAILED` when failure belongs to one of the three phases; CD publish failure instead uses `delivery_status = FAILED` + record error and does not invent a fourth step.

## 7. `reconcileInfrastructure(finalPlan, deploymentId)`

- **Cross references:** UC-03 worker/A2; Resource Instance state machine.
- **Preconditions:** typed plan fingerprint was rechecked; overrides valid. Every `REUSE` points to a `READY` instance selected through the exact active `resource_instance_binding` tuple for its consumer scope; owner columns are not a lookup path.
- **Postconditions:** create/update/reuse preserves owner fields and exact active Resource Instance Bindings; provider references/state are persisted. Create adds an OWNER binding; cross-boundary reuse validates but does not create/expand the pre-authorized SHARED_CONSUMER binding. Upsert `INFRASTRUCTURE_READY` step `RUNNING → SUCCEEDED` and associate the resolved Resource Instances with the existing Deployment Record so UC-04 can read them while execution continues; failure writes `FAILED` and lifecycle failure through worker.
- **Guarantees:** provider calls are idempotent. No instance from another scope is silently matched.

## 8. `resolveEnvironmentConfiguration(configuration, resourceOutputs, workloadOutputs)`

- **Cross references:** UC-03 worker/A2.
- **Preconditions:** Resource Outputs were collected only from READY instances. `WorkloadOutputResolver.resolvePlanTimeWorkloadOutputs(graph, context)` already produced deterministic values (for example Kubernetes Service DNS/endpoint) from graph/workload metadata/target context. Runtime-only output of the same deployment and circular bindings are invalid.
- **Postconditions:** transient Resolved Configuration contains direct values, resource outputs and plan-time workload outputs; references in DB remain unchanged. Upsert `CONFIGURATION_RESOLVED` step `RUNNING → SUCCEEDED`.
- **Guarantees:** missing/unresolvable output fails the active step and never persists resolved secret/value.

## 9. `generateKubernetesManifest(resolvedSpecification)` and materialization chain

- **Cross references:** UC-03 worker/A2.
- **Preconditions:** configuration resolved.
- **Postconditions:** score renderer, target adapter and configuration/secret materializers produce desired state; worker upserts `MANIFEST_GENERATED` `RUNNING → SUCCEEDED` only after all materialization succeeds.
- **Guarantees:** failure records only this internal step and redacted detail. No `CD_SYNCED`/`APPLICATION_READY` row is created.

## 10. `publishDesiredDeploymentState(desiredState, deploymentId, idempotencyKey)`

- **Cross references:** UC-03 worker/A2.
- **Preconditions:** Manifest Generated succeeded; stable publish idempotency key exists.
- **Postconditions:** CD returns `deliveryReference` and external `deliveryStatus`; worker stores them separately from lifecycle. CD acceptance permits platform lifecycle `SUBMITTED`, even while delivery status is `ACCEPTED` or `SYNCING`.
- **Guarantees:** `SYNCING`, `SYNCED`, `OUT_OF_SYNC`, `DEGRADED`, etc. are never written to `deployment.status` or `deployment_record.status`.

## 11. `saveDeploymentRecord(deploymentId, infrastructureReferences, deliveryReference, deliveryStatus, lifecycleStatus)`

- **Cross references:** UC-03 asynchronous success/A2 and UC-04 reads.
- **Preconditions:** Deployment exists; references belong to scope-safe Resource Instances; lifecycle and delivery values belong to their distinct physical enums.
- **Postconditions:** upsert one Deployment Record and association rows; record lifecycle mirrors Deployment lifecycle. Three internal Deployment Step rows carry their own status/timing/error. Delivery reference may be null and delivery status is `NOT_PUBLISHED` before publish.
- **Guarantees:** actual images remain in `workload_deployment`; history FK targets are retained. UC-04 is the only live aggregation path and does not write `CD_SYNCED`/`APPLICATION_READY`.

## 12. UC-04 query guarantees

`getDeploymentDetail()` loads lifecycle, optional record/reference, target, workload snapshots and zero steps before confirmation or the three persisted steps afterward. `getInfrastructureStatus()` is called only with non-empty infrastructure references. `getCDStatus()` is called only when `delivery_reference` exists. Kubernetes health/endpoints are queried only when delivery reference and publish prerequisite exist. Missing future prerequisite maps to `PENDING`; a terminal failure or non-applicable source maps to `NOT_AVAILABLE`. Live `SYNCED` maps `CD_SYNCED` to `SUCCEEDED`, `ACCEPTED/SYNCING/UNKNOWN` to `PENDING`, and terminal adverse delivery values to `FAILED`; `APPLICATION_READY` is `SUCCEEDED` only when every expected workload is Healthy, `PENDING` while converging and `FAILED` on observed terminal unhealthy state. These operations are read-only and never update lifecycle/steps.
