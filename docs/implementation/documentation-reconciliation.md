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

## Gate result

The three predecessors above contain no current concept whose only owner is in the archive. Their split artifacts are canonical and the consolidated files are non-normative provenance.
