# Step 6: Traceability Matrix

Matrix này dùng trạng thái sau khi đóng 11 review issues. “System operation” là danh sách Bước 2; `executeDeploymentJob`, repository enqueue/step writers và provider staging methods là internal operation và được trace riêng, không cộng giả vào 45 system operations.

## A. System-operation coverage

| UC | Operation Bước 2 | Sequence owner / participant | State/table chính | Contract |
|---|---|---|---|---|
| UC-01 | `createApplication`, `updateApplication` | UI → Application API → Application Service; update đọc Application Repository | Client-owned draft; `application_definition` read | — |
| UC-01 | `addResourceRequirement`, `addWorkload`, `defineConfigurationRequirement`, `defineDependency` | UI → API → stateless Service, full draft in/out | Không write cho tới save | — |
| UC-01 | `validateApplicationDefinition` | Application Definition Validator | Read-only transient validation | Contract 1 |
| UC-01 | `saveApplicationDefinition` | Application Service → Application Repository | Application tables; retire referenced items | Contract 1 |
| UC-01 | `generateApplicationSpecification` | Specification Generator/Repository | `application_specification` | Contract 2 |
| UC-02 | `selectEnvironment`, `loadConfigurationRequirements` | UI/API/Service → Application/Configuration repositories | Client-owned configuration draft | — |
| UC-02 | `setDirectConfigurationValue`, `bindResourceOutput`, `bindWorkloadOutput` | UI/API/stateless Service, full draft in/out | Không backend draft state | Contract 3 |
| UC-02 | `stageSecret` | API/Service → Secret Store | Opaque idempotent staged ref, TTL | Contract 3 |
| UC-02 | `validateEnvironmentConfiguration`, `saveEnvironmentConfiguration` | Validator → Config Repository; promote/revoke Secret Store | Environment configuration tables | Contract 3 |
| UC-03 | `createDeployment`, `validateDeploymentInput` | API → Orchestrator | `deployment`, `workload_deployment`, `deployment_context` | Contract 4 |
| UC-03 | `buildDeploymentGraph`, `resolveResourceDefinitions` | Graph Builder / Resource Resolver | Transient graph/resolution; reads definition catalog | Contract 4 |
| UC-03 | `planInfrastructureChanges`, `loadInfrastructureOverrides`, `applyInfrastructureOverrides` | Infrastructure Planner | Typed transient Plan/Item/Override Definition | Contracts 4–5 |
| UC-03 | `confirmDeployment` | Orchestrator → Deployment Repository/job outbox | Atomic CAS + `deployment_record` + `deployment_execution_job`; HTTP 202 | Contract 5 |
| UC-03 | `reconcileInfrastructure` | Worker → Reconciler → Provisioner | `resource_instance`; internal step 1 | Contract 7 |
| UC-03 | `collectResourceOutputs` | Worker → Resource Output Resolver | Transient Resource Output | Contract 8 precondition |
| UC-03 | `resolvePlanTimeWorkloadOutputs` | Worker → Workload Output Resolver | Transient plan-time Workload Output | Contract 8 |
| UC-03 | `resolveEnvironmentConfiguration` | Worker → Configuration Resolver | Transient resolved configuration; internal step 2 | Contract 8 |
| UC-03 | `generateResolvedApplicationSpecification`, `generateKubernetesManifest`, `adaptManifestForTarget`, `materializeEnvironmentConfiguration`, `materializeSecretConfiguration` | Worker and named components | Transient specification/manifest; internal step 3 | Contract 9 |
| UC-03 | `publishDesiredDeploymentState` | Worker → CD abstraction/provider | `delivery_reference`, separate `delivery_status` | Contract 10 |
| UC-03 | `saveDeploymentRecord` | Worker → Deployment Repository | Final lifecycle/delivery metadata; steps/associations were incrementally written | Contract 11 |
| UC-04 | `listDeployments`, `getDeploymentDetail`, `getDeploymentProgress`, `getDeploymentImages`, `getDeploymentFailureDetail` | Query API/Service → Deployment Repository | DB read-only; failed internal step is optional for CD failure | Contract 12 |
| UC-04 | `getInfrastructureStatus` | Query Service → Resource Repository only if refs exist | DB read-only; otherwise view `PENDING/NOT_AVAILABLE` | Contract 12 |
| UC-04 | `getCDStatus` | Query Service → CD provider only if delivery ref exists | Live delivery read; derives `CD_SYNCED` | Contracts 10, 12 |
| UC-04 | `getWorkloadStatus`, `getDeploymentEndpoints` | Query Service → Kubernetes only after publish prerequisite | Live runtime read; derives `APPLICATION_READY` | Contract 12 |

### Counts

| Use case | Bước-2 operations | Messages represented | Result |
|---|---:|---:|---|
| UC-01 | 9 | 9 | PASS |
| UC-02 | 8 | 8 | PASS |
| UC-03 | 19 | 19 | PASS |
| UC-04 | 9 | 9 | PASS |
| **Total** | **45** | **45** | **PASS** |

Hai operation mới so với baseline 43 là `stageSecret()` và `resolvePlanTimeWorkloadOutputs()`. `confirmDeployment()` đổi signature có `idempotencyKey` nhưng vẫn là một system operation.

## B. Internal operation and writer coverage

| Internal operation | Owner | Evidence / write |
|---|---|---|
| `executeDeploymentJob(jobId)` | Deployment Worker | UC-03 sequence; Contract 6; claims job and drives Contracts 7–11 |
| `acceptConfirmationAndEnqueue(...)` | Deployment Repository | One DB transaction: Deployment/Record `QUEUED` + exactly three `PENDING` step rows + job row |
| `claimNextJob`, `completeJob`, `retryOrFail` | Deployment Execution Job / Outbox | Lease/attempt/idempotency path in UC-03 sequence |
| `recordDeploymentStep` | Deployment Repository | Writes only three internal `deployment_step.step_name` literals |
| `markExecutionRunning` | Deployment Repository | Deployment/Record `RUNNING` writer |
| `stageSecret`, `promoteStagedSecrets`, `revokeStagedSecrets` | Secret Store adapter | UC-02 sequence/Contract 3; opaque refs and compensation |
| `findReusableInstances(...)` | Resource Instance Repository | Always starts at exact active `resource_instance_binding` tuple; owner columns are never an alternative read path |
| `aggregate(...)` | Deployment Result Aggregator | Derives live `CD_SYNCED`/`APPLICATION_READY`; no write |

## C. VOPC class coverage

Consolidated VOPC có **48 class/interface**. Tất cả có operation hoặc participant justification:

- 5 boundary/controller classes and 5 application services (including Deployment Worker) receive sequence messages.
- 21 domain components/typed plan classes: validators, builders/resolvers/planner/reconciler, three typed plan objects, render/materialize components and aggregator. Typed plan classes are data carriers used by canonicalization, so do not need behavior beyond fields/association.
- 6 integration abstractions/implementations own secret/provisioner/CD/status operations.
- 4 external systems receive provider/renderer/runtime messages.
- 7 persistence classes own read/write/job operations.

**Result: 48/48 PASS.** Per-UC VOPCs match the consolidated ownership: UI owns drafts, Orchestrator accepts/enqueues, Worker executes, and Query Service remains read-only.

## D. Persistent domain/table coverage

| Persistent object/table group | Writer | Reader | Result |
|---|---|---|---|
| Six Application Definition tables | `saveApplicationDefinition` (update/retire; restricted delete) | UC-01/02/03 repositories | PASS |
| `application_specification` | `generateApplicationSpecification` | artifact consumer | PASS |
| Four Environment Configuration tables | `saveEnvironmentConfiguration` | UC-02/03 | PASS |
| `resource_definition` | Platform administration outside four UC | catalogs/resolver | Accepted scope boundary |
| `resource_instance`, OWNER `resource_instance_binding` | `reconcileInfrastructure`; SHARED_CONSUMER writer thuộc platform sharing administration ngoài scope | exact-active-binding planner/output/UC-04 | PASS / accepted admin boundary |
| `deployment`, `workload_deployment`, `deployment_context` | create/confirm/worker | confirm/worker/UC-04 | PASS |
| `deployment_record`, `deployment_step`, join table | confirm creates record/steps; worker updates steps and associates infrastructure as soon as ready | UC-04 | PASS |
| `deployment_execution_job` | atomic confirm enqueue; worker retry/complete | worker | PASS |

ERD có **21 tables** và cả **21/21** có read hoặc write path. `Resource Definition` vẫn là persistent platform-managed catalog với writer ngoài scope, được ghi rõ thay vì tạo operation giả.

Transient exclusions đầy đủ: Draft DTOs ở client; Deployment Graph, Resource Resolution, Infrastructure Plan/Item/Override Definition, Resource/Workload Output, Resolved Configuration/Specification không có table. Chỉ fingerprint + algorithm persist trên Deployment; selected override values cần cho worker nằm trong job, không phải plan payload.

## E. Contract and state-machine coverage

- **12/12 contract groups PASS:** 1–3 save/generation/configuration, 4–5 prepare/confirm, 6 async worker, 7–11 execution phases/record, 12 UC-04 query guarantees.
- Deployment transitions are driven by create, atomic confirm enqueue, worker claim, CD accept or terminal failure: `AWAITING_CONFIRMATION → QUEUED → RUNNING → SUBMITTED|FAILED`.
- Resource Instance transitions are driven by plan/reconcile/collect/retire and obey scoped reuse.
- Deployment/Record lifecycle share the same five-value enum. Delivery has its own eight-value enum. Step and Resource Instance literals are physically enumerated in ERD/schema/state README.

## F. Review-issue closure

| # | Closure evidence |
|---:|---|
| 1 | UC-01/02 sequences and VOPCs put full draft in Web UI; service has no draft attribute. |
| 2 | Workload Output Resolver and plan-time/cycle rule appear in UC-03, VOPC, domain and Contract 8. |
| 3 | Canonical lookup always starts from exact active consumer binding; owner columns are original ownership only, and explicit shared bindings are platform-pre-authorized. |
| 4 | UC-04 sequence guards every provider call and returns `PENDING/NOT_AVAILABLE`. |
| 5 | Contract 1 + schema use retire and `ON DELETE RESTRICT`; history FKs remain. |
| 6 | Infrastructure Plan/Item/Override Definition are typed transient objects with canonical schema. |
| 7 | Exactly three internal rows have a worker writer; two live markers are derived. |
| 8 | Lifecycle and external `delivery_status` are separate fields/enums. |
| 9 | Physical enum registry matches domain, ERD and schema. |
| 10 | Confirm atomically enqueues; durable worker has lease/retry/idempotency. |
| 11 | Secret staging TTL/idempotency plus promote/revoke compensation and transaction limit are explicit. |

## Overall acceptance

**PASS with intentional platform-administration boundaries:** 45/45 Step-2 operations, 48/48 consolidated VOPC classes/interfaces, 21/21 ERD tables with read/write coverage, 12/12 contract groups and 2/2 lifecycle state machines. Resource Definition catalog administration and pre-authorization of SHARED_CONSUMER bindings remain outside the four Developer use cases.
