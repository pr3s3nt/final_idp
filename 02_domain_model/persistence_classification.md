# Persistence classification — MVP

| Objects | Classification | Storage / reason |
|---|---|---|
| Application Definition and children | PERSISTENT | Existing source tables; fixture writes atomic; retired identities remain queryable by history. |
| Application Specification | PERSISTENT | Current source artifact in application_specification. |
| Environment Configuration and values/references | PERSISTENT | Existing source tables; no plaintext Secret. |
| Resource Definition | PERSISTENT, platform-managed | Catalog populated by bootstrap; shared/update features not enabled. |
| Deployment, Workload Deployment, Deployment Context | PERSISTENT | Immutable image digests/context and lifecycle metadata. |
| Deployment Input Snapshot | PERSISTENT | deployment_input_snapshot; immutable typed source copy, not a transient plan. |
| Resource Instance and Binding | PERSISTENT | Provider identity/state path reserved before apply, source fingerprints/version and recovery flag; exact consumer scope. |
| Deployment Record, Steps, resource associations | PERSISTENT | Publication intent/ack and three internal phases; associations exist before provider writes. |
| Deployment Execution Job | PERSISTENT | Transactional outbox, one attempt, request fingerprint and worker run ownership. |
| Deployment Scope Guard | PERSISTENT | Serialized scope gate retained during uncertain outcome. |
| Deployment Recovery | PERSISTENT | Append-only operator/evidence audit. |
| Source editor draft DTOs | CLIENT-OWNED, future editor | Not needed by fixture-driven MVP. |
| Deployment Graph / Resource Resolution / Plan / Item / Override Definition | TRANSIENT | Rebuilt from immutable input + catalog + all-status resource bindings. Plan payload never stored. |
| Resource Output / Workload Output | TRANSIENT | Non-sensitive outputs from provider state or snapshot naming policy. |
| Resolved Configuration / Specification | TRANSIENT | Generated per fresh execution, secretKeyRef metadata only. |
| Desired deployment manifest | EXTERNAL ARTIFACT | OCI registry; immutable digest persists in record before Argo Application write. May contain resolved non-secret config, never credential values. |
| Terraform state | EXTERNAL DURABLE STATE | Stable scope-derived path on durable storage, state lock; state/reference survives worker crash. |
| CD_SYNCED / APPLICATION_READY | DERIVED VIEW | UID/revision/template/generation-aware query, no DB writes. |

MVP Secret bootstrap uses immutable Kubernetes Secret; source snapshot stores only target/namespace/name/UID/key. Secret staging/promote/revoke is deferred (R3). Credentials are not Terraform variables/data sources and are not encoded in OCI manifests.

See [schema](../03_database_erd/schema.md) for constraints and [contracts](../04_operation_contracts/operation_contracts.md) for atomic writers.
