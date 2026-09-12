# Step 6: MVP traceability and design acceptance

Source: [scope](../MVP_SCOPE.md), [deployment design](../MVP_DEPLOYMENT_DESIGN.md). This matrix supersedes the pre-MVP numeric PASS: counts alone do not verify failure paths. Full UC-01/02 editor diagrams remain future references; runtime acceptance is deferred until implementation.

## A. Operation ownership and persistence

| Operation / contract | Owner / sequence | Durable read/write |
|---|---|---|
| Fixture import C1 | Internal loader; editor deferred | Source aggregates/spec/configuration atomic |
| createDeployment C2 | Orchestrator → Snapshot Builder / Graph / Catalog / Planner | Immutable snapshot, deployment, image digests, context, reviewed fingerprint |
| confirmDeployment C3 | API → Orchestrator → Repository | Accepted request hash lookup; lock deployment/scope; atomically job/record/steps/guard |
| executeDeploymentJob C4 | Worker → Repository / snapshot / graph/catalog/planner / secret validator | Claim once; worker_run_id; initial RUNNING step |
| reconcileInfrastructure C5 | Worker → Reconciler → Provisioner/Terraform | Reserve identity/state/binding/association before apply; READY checkpoint |
| resolveEnvironmentConfiguration C6 | Worker → Resource/Workload Output and Configuration resolvers | Configuration step starts before collector; source rows unchanged |
| generateKubernetesManifest C7 | Worker → spec/renderer/score/materializers | Internal step + job phase; final manifest transient before publication |
| publishDesiredDeploymentState C8 | Go Argo CD Adapter → Registry / Repository / Argo API | Persist artifact URI/digest/expected Application before upsert |
| completeExecution C9 | Deployment Repository | One transaction acknowledgment + lifecycle/record/job success + scope release |
| failExecution C10 | Deployment Repository | One transaction failed phase/job/lifecycle, future steps skipped, scope gate |
| markInterruptedJobs C11 | Recovery Service under exclusive host lock | Old CLAIMED -> FAILED/EXECUTION_INTERRUPTED, scope RECOVERY_REQUIRED |
| recoverDeployment C12 | Operator Recovery Service → provider/CD inspection | Same resource identity + verified recovery audit; old execution remains failed |
| getDeploymentDetail C13 | Query Service → Repository / providers / aggregator | Read-only snapshot and revision-correlated observations |

There are five Developer HTTP operations: create, confirm, detail, history and failure detail (the latter may be included in detail). Internal provider/worker methods are not counted as public system operations.

## B. Artifact mapping

| Decision | Domain / schema | Contract | Sequence / state |
|---|---|---|---|
| Immutable inputs | Deployment Input Snapshot; image_digest; DB immutability guards | C2–C4 | UC-03 prepare and worker |
| Reviewed client token + request identity | plan_fingerprint, request_fingerprint, unique job | C3 | UC-03 confirm alternatives; awaiting self-loop |
| Scope serialization | Deployment Scope Guard | C3/C9–C12 | Accept / complete / recovery |
| No pipeline replay | Job worker_run_id, phase; no lease reclaim | C4/C10–C12 | Worker and recovery sequence |
| Pre-provider identity/state | Resource Instance version/fingerprints/recovery_verified + Binding + association | C5/C12 | Resource state machine |
| Output errors own configuration phase | Three step rows + phase | C6/C10 | Configuration starts before collect |
| Atomic terminal writes | Record/job/guard transaction | C9/C10 | UC-03 complete/fail |
| Publication intent and revision | artifact_uri/digest/expected_application_name + delivery_reference UID | C8/C13 | UC-03 publish; UC-04 correlation |
| Permanent Secret reference | Snapshot/source reference; no staging in MVP | C1–C4 | Secret Reference Adapter |
| Recovery evidence | Deployment Recovery | C12 | mvp_recover_deployment.puml |

Existing source tables are retained; three new tables (snapshot, guard, recovery) extend the 21-table draft baseline to 24. schema.md and erd.puml must have the same table/column names; constraints in prose define composite/partial uniqueness and immutability enforcement for implementation.

## C. Review findings disposition

| Finding | MVP disposition | Evidence / practical limit |
|---|---|---|
| R1 retry fingerprint | RESOLVED_DESIGN_MVP | No old-job replay; fresh precheck only; explicit interruption/recovery and new deployment |
| R2 unseen plan accepted | RESOLVED_DESIGN_MVP | Client/stored/rebuilt comparison + request fingerprint; lost PLAN_CHANGED response stays rejected |
| R3 secret promotion | DEFERRED_MVP | Staged-secret API removed from MVP; permanent immutable references only; future staging protocol remains open |
| R4 mutable deployment input | RESOLVED_DESIGN_MVP | Source-only immutable snapshot and image digests; worker cannot reread current source |
| R5 wrong-version readiness | RESOLVED_DESIGN_MVP | Argo UID/source/observed digest + template deployment-id/image/generation/rollout |
| R6 failed instance recovery | RESOLVED_DESIGN_MVP | All-status binding lookup, durable reservation, operator verified READY/absence on same identity |
| R7 split completion | RESOLVED_DESIGN_MVP | One terminal transaction for acknowledgment/lifecycle/record/job/guard |
| R8 collector without active step | RESOLVED_DESIGN_MVP | Infrastructure finish starts configuration before collectors; phase-specific failure |
| R9 no-config app | DEFERRED_MVP | Empty fixture source permitted, but full no-config use case is not in demo acceptance |
| R10 VOPC/sequence mismatch | RESOLVED_DESIGN_MVP | Snapshot-aware worker dependencies, provider output path, recovery and Argo/registry ownership synchronized |

RESOLVED_DESIGN_MVP means specification corrected for this profile, not runtime tests passed. Automatic replay/shared/resize/staging require separate design review when enabled. Historical evidence remains in [review_open_issues.md](review_open_issues.md).

## D. Acceptance

[MVP design review](mvp_design_review.md) records scenario walkthroughs and actual documentation checks. Main may receive this MVP design after those checks pass. Next work is Go implementation on a separate branch; this design change itself creates no application code, migration, Kubernetes cluster or AWS resource.
