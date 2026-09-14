# Step 4: Operation Contracts — MVP

Normative MVP: [scope](../MVP_SCOPE.md), [deployment design](../MVP_DEPLOYMENT_DESIGN.md). Names/columns map Step 2/3; transitions map Step 5. API/worker/provider are Go + Terraform + Argo CD. No automatic pipeline replay, shared consumer, UPDATE/resize or staged Secret API in this profile.

Current milestone: UC3 first-deploy happy path on **AWS**, not kind/local. Terraform bootstraps the AWS target/private network separately; C5 resolves the AWS Resource Definition and provisions Aurora PostgreSQL, and C8 delivers both workloads there. The same logical requirement resolves to a PostgreSQL StatefulSet on kind/local for development tests only. AWS service topology/cost must be recorded before provisioning.

## C1. Fixture import (UC-01/02 subset, internal)

- Caller: bootstrap fixture loader, not a public editing API.
- Validate active topology, ownership, unique names, dependencies, images and direct/non-sensitive output bindings; reject cycles/runtime-only same-deployment output. Existing referenced identities are retained/retired, not hard-deleted.
- Save application aggregate, generated current specification and environment configuration atomically in metadata DB. Source edits remain allowed; existing Deployment snapshots never change.
- Environment configuration may be empty when the fixture has no requirements; the frontend/backend/database demo has required configuration. Full UC-01/02 editor is deferred.
- Secret source is target-specific metadata only: kind/local uses a validated immutable Kubernetes Secret identity; AWS uses a `RESOURCE_SECRET` intent resolved after Aurora READY to a Secrets Manager ARN/reference and ExternalSecret destination. No plaintext or staged secret is stored or returned.
- API preflight cannot atomically prevent external Secret deletion; bootstrap/operator must not mutate/delete referenced resources during execution. Missing/replaced reference is an explicit failure, not a fallback credential.

## C2. createDeployment(applicationId, environment, target, images, context)

- Preconditions: authenticated developer; allowlisted AWS Kubernetes target with verified account/region/cluster identity; valid fixture/configuration; image tags resolve to immutable digests accessible by AWS nodes and Argo CD; renderer/naming versions known. Local/kind targets cannot satisfy cloud acceptance.
- Read all source rows under one REPEATABLE READ transaction; build immutable Deployment Input Snapshot with source application/configuration, render context, secret metadata references. Snapshot hash also covers persisted workload image digests and Deployment Context.
- Catalog read and Resource Instance Repository lookup create a transient typed plan. Lookup starts from exact active consumer binding and returns ALL statuses. Uninspected PLANNED/PROVISIONING/FAILED yields RESOURCE_RECOVERY_REQUIRED; compatible READY yields REUSE. Absent binding yields CREATE; recovery-approved PLANNED yields CREATE on retained instance identity. Parameter changes require unsupported UPDATE and are rejected.
- One DB transaction persists Deployment AWAITING_CONFIRMATION, snapshot, Workload Deployments, context, fingerprint and algorithm. No provider write, job, record or step exists yet.
- Return plan, input fingerprint, plan fingerprint and algorithm. Source snapshots/images/context cannot be updated; changing source requires a new deployment.
- Validation failure creates no partial aggregate. Plan payload remains transient.

## C3. confirmDeployment(deploymentId, expectedPlanFingerprint, overrides, idempotencyKey)

- Authenticate before lookup. Normalize request, compute request_fingerprint. Existing accepted job with same key/hash returns same trackingId before checking lifecycle/catalog; same key/different payload returns IDEMPOTENCY_KEY_REUSED; another key returns ALREADY_ACCEPTED.
- Lock Deployment and re-read accepted job/key/hash before status checks, so a concurrent winner returns the same tracking result. If still unaccepted, lock deployment_scope_guard. Status must be AWAITING_CONFIRMATION; EXECUTING/RECOVERY_REQUIRED scope cannot be acquired.
- Rebuild from immutable snapshot + current catalog + all-status bindings. Validate kind/local permanent Secret identity or AWS resource-secret intent as appropriate. Compare client expected fingerprint with BOTH stored and rebuilt fingerprint.
- Mismatch: while still awaiting, refresh stored fingerprint and return 409 PLAN_CHANGED + new plan/token; no job/guard acquisition. A lost response/retry with old token cannot accept refreshed plan.
- Overrides must equal {}; unsupported key/value returns 422 without writes.
- Match: in one transaction CAS lifecycle+fingerprint, guard EXECUTING, one QUEUED job with idempotency/request hash/accepted fingerprint, record QUEUED/NOT_PUBLISHED, exactly three PENDING step rows. Return 202 only after commit.
- Unique job per deployment, scoped guard and CAS prevent duplicate confirmation. No Terraform/render/publish occurs in HTTP confirm.

## C4. executeDeploymentJob(jobId, workerRunId)

- Preconditions: supervisor stopped old process group; exclusive host worker lock held; job QUEUED, lifecycle QUEUED, scope guard EXECUTING and owned by this deployment.
- Atomic claim: job CLAIMED + worker_run_id, lifecycle/record RUNNING, phase INFRASTRUCTURE and INFRASTRUCTURE_READY RUNNING. Every later writer predicates on CLAIMED + matching worker_run_id.
- Load immutable snapshot, rebuild/check accepted fingerprint once BEFORE resource reservations/provider writes. No current app/configuration read. Invalid/stale input calls failExecution with no side effect and releases scope.
- Execute C5–C9. Never claim old CLAIMED job or requeue failed job. Provider calls use stable resource state/identity and publication intent; idempotency does not justify replay.
- On uncertain side effect, use C10 and RECOVERY_REQUIRED. Host heartbeat loss is diagnostic, not permission to start a second worker.

## C5. reconcileInfrastructure(finalPlan, deploymentId, workerRunId)

- Preconditions: accepted preflight passed, exclusive worker ownership, stable resource scope. MVP CREATE or REUSE only; no shared binding or UPDATE.
- Before CREATE, atomically reserve PLANNED Resource Instance, deterministic durable provider_state_reference, OWNER binding and record association. Recovery-approved reserved identity is reused; do not allocate a new identity. Before apply, write PROVISIONING, recovery_verified=false and resource version increment.
- Provider uses only the pinned module chosen by the accepted Resource Definition. AWS applies Aurora into the allowlisted region/subnet group/security groups; kind/local applies StatefulSet/PVC/Service to the explicit kube context. Database state is separate from bootstrap state. READY persists infrastructure_reference, fingerprints, version and timestamps.
- REUSE inspects provider state and objects, expected identity and parameters; drift/missing object fails and requires recovery instead of implicit recreate.
- Complete infrastructure and start CONFIGURATION_RESOLVED in one transaction. Do not recompute accepted CREATE fingerprint after reservation/provision.
- Unknown apply outcome records resource failure/available state and retains scope RECOVERY_REQUIRED. No rollback/deletion of a created database.

## C6. resolveEnvironmentConfiguration(snapshot, resourceReferences, workerRunId)

- CONFIGURATION_RESOLVED is already RUNNING before any output collection.
- Resource Output Collector reads provider using durable state/identity, never memory from an earlier process. Only READY resources produce non-sensitive endpoint/port/database/username and credential ARN/reference metadata; credential values are never outputs.
- Workload Output Resolver uses snapshot naming policy/namespace/port to generate backend Service URL. No same-deployment runtime endpoint/cycle.
- Resolve direct values and logical output references. On AWS, Secret Materializer maps the provider ARN/reference to an ExternalSecret and the workload's `secretKeyRef`; on kind/local it keeps the validated Kubernetes `secretKeyRef`. No credential value enters IDP DB or artifact.
- On success, atomically finish configuration and start MANIFEST_GENERATED. Collector/resolver failure belongs to this step and calls C10.

## C7. generateKubernetesManifest(snapshot, resolvedConfiguration)

- MANIFEST_GENERATED covers resolved spec generation, score-k8s, target adaptation, env/secret-reference materialization.
- Final manifest must preserve expected Service/Deployment names, namespace, pinned image digests and idp.deployment-id Pod template annotation. Failure belongs to this phase.
- Resolved objects remain transient; desired manifest is delivered as an external OCI artifact containing non-secret values and secret references only.
- Complete step and set job phase PUBLISH atomically; do not introduce a persisted CD step.

## C8. publishDesiredDeploymentState(desiredState, deploymentId, workerRunId)

- Go Argo CD Adapter packages/pushes immutable OCI artifact. Persist artifact URI/digest and expected_application_name on record BEFORE creating/updating Application.
- Upsert only the owned Application for the deployment scope, with namespace/UID/resourceVersion checks, OCI source targetRevision=digest and explicit AWS destination cluster/namespace. An Argo CD instance running on kind must not use its local in-cluster destination for cloud deployment. Ownership mismatch fails without takeover.
- Return Application acknowledgment (namespace/name/UID + digest). Acceptance is not Synced/Healthy.
- A definite rejection maps delivery FAILED; an ambiguous API timeout maps UNKNOWN and requires operator inspection. Do not publish a different digest from the same accepted execution.

## C9. completeExecution(deploymentId, workerRunId, acknowledgment)

- Requires job CLAIMED owned by run, phase PUBLISH, all three internal steps SUCCEEDED and acknowledgment matching persisted publication intent.
- ONE DB transaction sets delivery_reference/status=ACCEPTED, lifecycle/record SUBMITTED, job SUCCEEDED/COMPLETE, completed_at and scope guard IDLE.
- Repeated completion with identical committed result is a no-op. No independent saveDeploymentRecord + completeJob commits; rollback leaves CLAIMED/RUNNING for interruption handling.

## C10. failExecution(deploymentId, workerRunId, code, outcome)

- ONE transaction sets job/deployment/record FAILED, sanitized failure_code/error; active internal step FAILED, future PENDING steps SKIPPED. Prior successful steps remain SUCCEEDED.
- Publish errors use delivery_status/record error only. UNKNOWN is used if acknowledgment/outcome cannot be established.
- Confirmed no side effects, or phase error after all infrastructure is checkpointed READY and before Argo write: guard IDLE. Terraform/CD ambiguity, interrupted execution or provider drift: guard RECOVERY_REQUIRED.
- No row deletion, no job requeue, no automatic resource destroy.

## C11. markInterruptedJobs(newWorkerRunId)

- Only after exclusive worker lock and proof that old worker/Terraform processes stopped. No lease-stealing/replay.
- Old CLAIMED jobs become FAILED/EXECUTION_INTERRUPTED with active phase mapping per C10 and guard RECOVERY_REQUIRED. QUEUED jobs stay queued but cannot bypass blocked scope.
- Old writer predicate no longer matches; no second active job is started for that scope.

## C12. recoverDeployment(deploymentId, operator, evidence)

- Internal operator tool under exclusive worker lock, not Developer API. Scope RECOVERY_REQUIRED and old process/state lock safely quiescent.
- Inspect deterministic state and actual Kubernetes objects. Import exact owned object if provider created it before state write; never discard state/apply blindly. Matching object -> READY; proven no object -> PLANNED + recovery_verified=true with SAME binding/identity. Unknown/partial incompatible result stays blocked.
- If publication intent exists, inspect expected Argo Application/digest; record observed acknowledgment if established, otherwise retain UNKNOWN/blocked until operator resolves it. Never republish as part of recovery.
- One DB transaction updates verified resource metadata/version, inserts Deployment Recovery audit/evidence, fills observed record metadata where known, releases guard IDLE. Old job/deployment remain FAILED; next attempt is a NEW deployment/review, not replay of old fingerprint.
- No database cleanup; teardown is separate and only targets owned demo resources.

## C13. getDeploymentDetail(deploymentId) / UC-04

- Read immutable snapshot/images, optional record and zero pre-confirm or exactly three post-confirm steps. No write or reconcile operation.
- Infrastructure query only with known resource identities. Argo query only with delivery_reference; absence is PENDING while running or NOT_AVAILABLE when terminal.
- Argo Application UID/source mismatch -> NOT_AVAILABLE/SUPERSEDED or REPLACED. Current source correct but observed revision not expected digest -> PENDING.
- CD_SYNCED succeeds only for Synced plus expected artifact digest/revision. Runtime query is allowed only after revision correlation. Argo Healthy alone cannot establish application readiness.
- APPLICATION_READY additionally requires all expected Kubernetes names/namespace, matching template deployment-id and image digests, observedGeneration >= generation, and updated/ready/available replica counts at desired with no old replicas. Observed terminal error for expected revision -> FAILED; missing/converging -> PENDING; unavailable provider -> PENDING with reason.
- External markers are view-only. SUBMITTED lifecycle never becomes CD SYNCED/DEGRADED. History for a replaced deployment cannot borrow current health.
- listDeployments reads DB only; getDeploymentFailureDetail reads failed step/record/job failure and recovery state.

## Acceptance boundary

Design review scenarios and R1–R10 disposition: [traceability](../06_traceability/traceability_matrix.md). UC-01/02 full editor, staged secrets, shared resources, automatic replay and in-place database update are not promised by these MVP contracts.
