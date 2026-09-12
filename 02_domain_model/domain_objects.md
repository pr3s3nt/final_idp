# Step 2: Domain objects — MVP deployment profile

Source aggregates remain Application Definition and Environment Configuration; fixture loader replaces the full UC-01/02 editor in MVP. Domain diagram retains future source/reference types, but MVP validators reject shared consumers, staged/sensitive-output secrets and UPDATE parameters.

| Object | Persistent owner / key facts |
|---|---|
| Application Definition, Workload, Resource Requirement, configuration definitions, Dependency | Source aggregate; active query filters retired_at. Referenced identities use retire/RESTRICT. |
| Application Specification | Current generated source artifact, saved atomically with fixture source. |
| Environment Configuration / Environment Variable / Configuration Value / Secret | Direct non-secret input and logical bindings; Secret contains only permanent target/namespace/name/UID/key reference. |
| Workload Output Definition | Embedded PLAN_TIME/RUNTIME metadata; MVP binds only deterministic PLAN_TIME. |
| Resource Definition | Platform/fixture catalog; versioned provisioner reference, default parameters and outputs; MVP allowed_overrides empty. |
| Deployment | Lifecycle and reviewed plan fingerprint/algorithm; source IDs retained for audit, never used as mutable execution input. |
| Deployment Input Snapshot | Immutable source application/configuration/render context, schemaVersion and inputFingerprint including image digests/context. Exactly one per deployment. |
| Workload Deployment | Workload identity, selected image tag, actual immutable image digest; never mutated after prepare. |
| Deployment Context | Immutable target inputs. |
| Deployment Graph / Resource Resolution | Transient derivation from snapshot and current catalog. |
| Infrastructure Plan / Item / Override Definition | Transient canonical plan; MVP actions CREATE/REUSE, empty override schema. UPDATE exists only for future profiles. |
| Resource Instance | Durable owner/scope, state path, provider identity, definition/parameter fingerprints, version and recoveryVerified. |
| Resource Instance Binding | Canonical exact active lookup at ALL statuses; MVP OWNER only. Non-retired instance has one active owner binding. |
| Resource/Workload Output | Transient non-sensitive provider output or deterministic Service URL; collected within configuration phase. |
| Resolved Configuration / Specification | Transient values/reference-bearing output; no plaintext credential. |
| Deployment Record | Lifecycle + separate delivery state; publication intent artifact URI/digest/expected Application name, acknowledged Application UID and errors. |
| Deployment Step | Exactly three internal steps after confirm. Future steps become SKIPPED on terminal failure. |
| Deployment Execution Job | One accepted request/hash, one claim, workerRunId, phase/timestamps/failure. No lease reclaim or replay. |
| Deployment Scope Guard | One (application, environment, target) gate: IDLE/EXECUTING/RECOVERY_REQUIRED. |
| Deployment Recovery | Append-only operator/evidence audit releasing a verified blocked scope; old job stays FAILED. |

## Source snapshot structure

applicationDefinition contains complete active workload IDs/names/types/repositories/ports/output definitions, resource requirements, variable/secret definitions and dependencies. environmentConfiguration contains environment and typed direct/output/secret reference bindings. renderContext contains namespace, naming policy and renderer/adapter versions.

inputFingerprint includes these objects plus separately persisted Workload Deployment image digests and Deployment Context. Map keys and source shapes are validated by mvp-source-v1. JSONB stores source facts, never plan payload, resolved resource values or plaintext Secret.

## Invariants

- Source writes do not change accepted/queued deployment input. Snapshot/input tables reject update/delete via DB permission/trigger.
- Plan fingerprint compares reviewed client token, stored token and freshly rebuilt plan before enqueue; same accepted key/hash returns same tracking ID regardless of lifecycle.
- Resource lookup returns all statuses; only compatible READY becomes REUSE. Recovery-approved PLANNED can CREATE on retained identity.
- Reserve resource, state path and record association before provider effects. Whole pipeline never automatically replays.
- Completion/failure atomically updates lifecycle, record, job, steps and scope guard as applicable.
- Permanent immutable Secret reference UID/key is checked at prepare/confirm/precheck. Staging/rotation is deferred.
- Source naming policy is shared by Workload Output Resolver and final manifest; frontend proxy uses the resulting backend Service URL.
- CD_SYNCED/APPLICATION_READY are read-only revision-correlated views, never deployment steps or lifecycle values.
