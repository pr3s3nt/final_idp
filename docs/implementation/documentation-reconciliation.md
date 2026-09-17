---
id: DOC-RECONCILIATION
artifact: documentation-semantic-reconciliation
status: current
last_reviewed: 2026-09-17
---

# Documentation semantic reconciliation

This record is the manual semantic gate for consolidated documents moved to `docs/archive/`. Structural checks cannot prove that a split artifact retains every current concept, so each source group is read in full and mapped below before archive.

## Use-case specification and realization source

Source: [`usecase-realization-step-1-3.md`](../archive/consolidated/usecase-realization-step-1-3.md).

The source was read from the first UC-01 specification through the end of Step 3. Its five specification blocks and the UC-specific responsibility, system-operation, and participating-component blocks were compared with the corresponding split files. Headings, stable IDs, front matter, and navigation links are intentional additions; no normative statement was omitted.

| Source content | Current canonical owner | Result |
|---|---|---|
| UC-01 specification; Step 1–3 UC-01 | [`UC-01/specification.md`](../use-cases/UC-01/specification.md), [`UC-01/realization.md`](../use-cases/UC-01/realization.md) | `COVERED` |
| UC-02 specification; Step 1–3 UC-02 | [`UC-02/specification.md`](../use-cases/UC-02/specification.md), [`UC-02/realization.md`](../use-cases/UC-02/realization.md) | `COVERED` |
| UC-03 specification; Step 1–3 UC-03 | [`UC-03/specification.md`](../use-cases/UC-03/specification.md), [`UC-03/realization.md`](../use-cases/UC-03/realization.md) | `COVERED` |
| UC-04 specification; Step 1–3 UC-04 | [`UC-04/specification.md`](../use-cases/UC-04/specification.md), [`UC-04/realization.md`](../use-cases/UC-04/realization.md) | `COVERED` |
| UC-05 specification; Step 1–3 UC-05 | [`UC-05/specification.md`](../use-cases/UC-05/specification.md), [`UC-05/realization.md`](../use-cases/UC-05/realization.md) | `COVERED` |
| Introductory explanation of the three-step realization process | Archived source | `HISTORICAL` — process provenance, not a current product definition |

## Decision log

Source: [`design-decisions-log.md`](../archive/consolidated/design-decisions-log.md).

All 15 numbered decision sections were compared with ADR-001 through ADR-015. Each ADR preserves its source problem, decision, rationale/application notes, and deferred references; the added current-status preface resolves dated statements without deleting provenance. The source overview and editing chronology are historical process notes.

| Source sections | Current canonical owner | Result |
|---|---|---|
| Decisions 1–15 | [`docs/decisions/`](../decisions/README.md), one ADR with the same number per section | `COVERED` |
| Overview table and branch/editing chronology | Archived source | `HISTORICAL` — summarized by the ADR index and retained only as chronology |

## Deferred-issue log

Source: [`deferred-issues-log.md`](../archive/consolidated/deferred-issues-log.md).

All 14 issue sections were compared with D01 through D14. Each split record preserves the context, risk, proposed-but-unaccepted options, questions, and closure conditions of its source section. The tracking table is represented by the authoritative backlog index.

| Source sections | Current canonical owner | Result |
|---|---|---|
| D1–D14 | [`docs/backlog/`](../backlog/README.md), one record with the same number per section | `COVERED` |
| Tracking table | [`docs/backlog/README.md`](../backlog/README.md) | `COVERED` |
| Source-branch provenance | Archived source | `HISTORICAL` — not a current design definition |

## Consolidated verification log

Source: [`uc03-verification-log.md`](../archive/consolidated/uc03-verification-log.md).

The source was read through all seven execution rounds. The snapshots retain the commands, environments, deployment IDs, observed failures, cleanup results, and explicit limitations from their corresponding source sections.

| Source content | Evidence snapshot | Result |
|---|---|---|
| §1–§3: automated, kind-local, and AWS baseline | [`2026-09-15-uc03-kind-aws.md`](../verification/2026-09-15-uc03-kind-aws.md) | `COVERED` |
| §4: Catalog Version, internal cluster, and propagation | [`2026-09-15-catalog-versioning.md`](../verification/2026-09-15-catalog-versioning.md) | `COVERED` |
| §5: per-application Delivery Repository | [`2026-09-16-delivery-repository.md`](../verification/2026-09-16-delivery-repository.md) | `COVERED` |
| §6: Fleet provider | [`2026-09-16-fleet-provider.md`](../verification/2026-09-16-fleet-provider.md) | `COVERED` |
| §7: UC-05 | [`2026-09-16-uc05.md`](../verification/2026-09-16-uc05.md) | `COVERED` |

## Implementation plan

Source: [`uc03-original-implementation-plan.md`](../archive/planning/uc03-original-implementation-plan.md).

The entire plan was reviewed, including §10, §10.1, the complete A1 list, later progress notes, and the §12 decision dialogue. Statements that describe an earlier checkout remain in the archive; current behavior is owned by the artifacts below.

| Plan area | Current owner or classification | Result |
|---|---|---|
| §1–§4 original context, assumptions, and proposed package layout | Archived source; current navigation is the [implementation index](README.md) and [`idp/backend/README.md`](../../idp/backend/README.md) | `HISTORICAL` |
| §5 component/package map | [`code-map.md`](code-map.md) and [`idp/backend/README.md`](../../idp/backend/README.md) | `COVERED` |
| §6 M1–M24 and business rules | [UC-03](../use-cases/UC-03/specification.md), [UC-05](../use-cases/UC-05/specification.md), their realizations, contracts, and schema | `COVERED` |
| Complete A1-1 through A1-12 catalog | [UC-03 A1](../use-cases/UC-03/specification.md#a1--deployment-input-hoặc-dependency-không-hợp-lệ) | `COVERED` |
| §7–§9 staged execution plan and environment setup | Historical sequencing; current commands and prerequisites are in the [runbook](../operations/uc03/RUNBOOK.md) | `HISTORICAL` / `COVERED` |
| §11 progress and execution claims | [Current state](../CURRENT_STATE.md) and immutable [verification snapshots](../verification/README.md) | `COVERED` |
| §12 design-reconciliation dialogue | Current use cases, architecture, [ADRs](../decisions/README.md), and [backlog](../backlog/README.md); dated discussion remains historical | `COVERED` |

### §10 and §10.1 audit

| Item | Classification and canonical owner | Result |
|---|---|---|
| §10 item 1: target infrastructure graph nodes and `requires` | [UC-03 specification](../use-cases/UC-03/specification.md), [realization](../use-cases/UC-03/realization.md), [ADR-012](../decisions/ADR-012-target-and-catalog-versioning.md) | `COVERED` |
| §10 item 2: TEARDOWN and `deployment.kind` | [UC-05 specification](../use-cases/UC-05/specification.md), [realization](../use-cases/UC-05/realization.md), [schema](../architecture/database/schema.md) | `COVERED` |
| §10 item 3: override baseline and input fingerprint | [UC-03 specification](../use-cases/UC-03/specification.md), [contracts](../architecture/contracts/operation-contracts.md), [schema](../architecture/database/schema.md) including `applied_overrides` and `applied_input_fingerprint` | `COVERED` |
| §10 item 4: overlapping-deployment rejection | [UC-03 A1-12](../use-cases/UC-03/specification.md#a1--deployment-input-hoặc-dependency-không-hợp-lệ) | `COVERED` |
| §10 item 5: atomic `FinishDeployment` | [Operation contract 9](../architecture/contracts/operation-contracts.md#9-savedeploymentrecord) and [IMP-006](deviations.md) | `COVERED` |
| §10 item 6: typed-plan fingerprint and worker recheck | [UC-03 specification](../use-cases/UC-03/specification.md), [contracts](../architecture/contracts/operation-contracts.md), and `PLAN_VERIFIED` in the [schema](../architecture/database/schema.md) | `COVERED` |
| §10 item 7: verify workload disappearance before removal/destruction | Required behavior is in [UC-05](../use-cases/UC-05/specification.md); the remaining unavailable-cluster gap is explicitly owned by [D13](../backlog/D13-teardown-removal-verification.md) | `COVERED` |
| §10 item 8: active Workload Instance partial uniqueness | [Database schema](../architecture/database/schema.md) | `COVERED` |
| §10 item 9: implemented portions of D4/D5/D6/D8 | Physical values and current transaction behavior are in [schema](../architecture/database/schema.md), [contracts](../architecture/contracts/operation-contracts.md), and [deviations](deviations.md); unresolved generalizations remain explicitly open in [backlog](../backlog/README.md) | `COVERED` |
| §10 item 10: `deployment_step.detail` | [Database schema](../architecture/database/schema.md) | `COVERED` |
| §10.1 item 11: transitive `requires` closure | [UC-03 specification](../use-cases/UC-03/specification.md) and [realization](../use-cases/UC-03/realization.md) | `COVERED` |
| §10.1 item 12: platform output does not cascade | `HISTORICAL` — superseded later in the same plan by §12 Q4; current [UC-03](../use-cases/UC-03/specification.md) requires propagation to dependent resources and workloads | `HISTORICAL` |
| §10.1 item 13: logical registry mapping | [IMP-009](deviations.md) | `COVERED` |
| §10.1 item 14: encrypted Secret Store and Secret materialization | [Security and Secret architecture](../architecture/security-and-secrets.md) | `COVERED` |
| §10.1 item 15: `PLAN_VERIFIED`, deployment kind, platform requirement, `requires`, fingerprints, detail, and uniqueness | Each physical field, enum, and constraint is in the [database schema](../architecture/database/schema.md); behavior is cross-referenced from UC-03/UC-05 | `COVERED` |
| §10.1 item 16: Terraform/Git CLI and Kubernetes `client-go` | [IMP-008](deviations.md) | `COVERED` |
| §10.1 item 17: separate kind namespaces | [IMP-010](deviations.md) | `COVERED` |
| §10.1 item 18: `RESOURCE_DEFINITION_CHANGED` | [UC-03 A1-7](../use-cases/UC-03/specification.md#a1--deployment-input-hoặc-dependency-không-hợp-lệ) and [IMP-007](deviations.md) | `COVERED` |
| §10.1 item 19: UC-02 importer validation behavior | [IMP-001 and IMP-011](deviations.md) | `COVERED` |
| §10.1 item 20: step creation timing and `SKIPPED` transition | [IMP-012](deviations.md), [operation contract 9](../architecture/contracts/operation-contracts.md#9-savedeploymentrecord), with the generalized ownership question retained in [D04](../backlog/D04-deployment-progress-ownership.md) | `COVERED` |

## Gate result

The audited predecessors contain no current concept whose only owner is in the archive. Current definitions live in use-case, architecture, implementation, decision, backlog, operations, or evidence artifacts; the archived sources are non-normative provenance.
