---
id: DOC-MIGRATION-PLAN
artifact: documentation-migration-plan
status: current
migration_stage: mapping-approved
last_reviewed: 2026-09-17
---

# Documentation migration map

This document is the execution contract for completing the documentation refactor after commit `6c835d9`. It fixes the destination and authority of every legacy documentation group before files are moved. During migration, [INDEX.md](INDEX.md) remains the reader entry point.

## Scope and invariants

The migration changes documentation paths and removes duplicate active sources. It does not change application behavior, database behavior, infrastructure, or the intended meaning of accepted design artifacts.

The following invariants apply to every migration commit:

1. Use `git mv` for moves so file history remains visible.
2. Never maintain an active compatibility copy at an old path.
3. Update every inbound relative link in the same commit as a move.
4. Do not use an archived document as the source of a current definition.
5. Compare split artifacts with their consolidated predecessor before archiving the predecessor.
6. Keep stable artifact IDs even when paths and titles change.
7. Preserve `codex_review.md` untouched unless the user explicitly brings it into scope.
8. Run `python3 scripts/check_docs.py` and `git diff --check` after every migration commit.

## Action vocabulary

| Action | Meaning |
|---|---|
| `KEEP` | The path is already final and remains canonical for its declared scope. |
| `MOVE` | Move the file to the exact target path and update all inbound links. |
| `ARCHIVE` | Move a non-normative historical file under `docs/archive/`; it must not own current definitions. |
| `VERIFY_THEN_ARCHIVE` | First prove the replacement preserves the relevant content, then archive the predecessor. |
| `REMOVE_EMPTY_DIR` | Remove the directory after all tracked artifacts have moved out. |
| `EXCLUDE` | The item is intentionally outside this migration. |

## Authority during migration

When an old path and a new split artifact coexist, authority remains as declared in [INDEX.md](INDEX.md): current use-case specifications, realizations, ADRs, backlog records, shared models, and verification snapshots are canonical for their scopes. Historical consolidated files provide provenance only.

After migration, authority will be located as follows:

| Concern | Final canonical location |
|---|---|
| Required use-case behavior | `docs/use-cases/UC-*/specification.md` |
| Use Case Realization | `docs/use-cases/UC-*/realization.md` |
| Per-use-case interaction and participating classes | `docs/use-cases/UC-*/sequence.puml`, `vopc.puml` |
| Consolidated design classes | `docs/architecture/design-class-diagram.puml` and `design-classes.md` |
| Domain concepts and persistence classification | `docs/architecture/domain/` |
| Tables, columns, constraints, and literal ENUMs | `docs/architecture/database/schema.md` |
| Operation contracts | `docs/architecture/contracts/operation-contracts.md` |
| Lifecycle transitions | `docs/architecture/state-machines/` |
| Accepted decisions | `docs/decisions/ADR-*.md` |
| Open/deferred risks and issues | `docs/backlog/D*.md` |
| Cross-artifact coverage | `docs/traceability/matrix.md` |
| Design-to-code navigation and current deviations | `docs/implementation/` |
| Execution evidence | `docs/verification/` |
| Operator and demo instructions | `docs/operations/uc03/` |
| Historical consolidated artifacts | `docs/archive/` |

## Mapping A — Use-case context packages

The specification, realization, and context README files created in `6c835d9` remain at their current paths. Sequence and VOPC sources move beside them.

| Current path | Final path | Action |
|---|---|---|
| `docs/use-cases/README.md` | Same | `KEEP` |
| `docs/use-cases/UC-01/{README,specification,realization}.md` | Same | `KEEP` |
| `docs/use-cases/UC-02/{README,specification,realization}.md` | Same | `KEEP` |
| `docs/use-cases/UC-03/{README,specification,realization}.md` | Same | `KEEP` |
| `docs/use-cases/UC-04/{README,specification,realization}.md` | Same | `KEEP` |
| `docs/use-cases/UC-05/{README,specification,realization}.md` | Same | `KEEP` |
| `sequence_digrams/uc_01_create_configure_application.puml` | `docs/use-cases/UC-01/sequence.puml` | `MOVE` |
| `sequence_digrams/uc_02_configure_application_environment.puml` | `docs/use-cases/UC-02/sequence.puml` | `MOVE` |
| `sequence_digrams/uc_03_deploy_application.puml` | `docs/use-cases/UC-03/sequence.puml` | `MOVE` |
| `sequence_digrams/uc_04_view_deployment_result.puml` | `docs/use-cases/UC-04/sequence.puml` | `MOVE` |
| `sequence_digrams/uc_05_remove_application_from_environment.puml` | `docs/use-cases/UC-05/sequence.puml` | `MOVE` |
| `01_vopc_design_class_diagram/vopc_uc01.puml` | `docs/use-cases/UC-01/vopc.puml` | `MOVE` |
| `01_vopc_design_class_diagram/vopc_uc02.puml` | `docs/use-cases/UC-02/vopc.puml` | `MOVE` |
| `01_vopc_design_class_diagram/vopc_uc03.puml` | `docs/use-cases/UC-03/vopc.puml` | `MOVE` |
| `01_vopc_design_class_diagram/vopc_uc04.puml` | `docs/use-cases/UC-04/vopc.puml` | `MOVE` |
| `01_vopc_design_class_diagram/vopc_uc05.puml` | `docs/use-cases/UC-05/vopc.puml` | `MOVE` |
| `sequence_digrams/` | — | `REMOVE_EMPTY_DIR` |

Before moving the consolidated source, compare every `specification.md` and `realization.md` with its source sections in `usecase_realization_step_1_3.md`. Record only genuine omissions; formatting differences and the new stable IDs are expected.

## Mapping B — Shared architecture

| Current path | Final path | Action |
|---|---|---|
| `01_vopc_design_class_diagram/README.md` | `docs/architecture/design-classes.md` | `MOVE` |
| `01_vopc_design_class_diagram/design_class_diagram.puml` | `docs/architecture/design-class-diagram.puml` | `MOVE` |
| `01_vopc_design_class_diagram/` | — | `REMOVE_EMPTY_DIR` |
| `02_domain_model/domain_model.puml` | `docs/architecture/domain/domain-model.puml` | `MOVE` |
| `02_domain_model/domain_objects.md` | `docs/architecture/domain/domain-objects.md` | `MOVE` |
| `02_domain_model/persistence_classification.md` | `docs/architecture/domain/persistence-classification.md` | `MOVE` |
| `02_domain_model/` | — | `REMOVE_EMPTY_DIR` |
| `03_database_erd/erd.puml` | `docs/architecture/database/erd.puml` | `MOVE` |
| `03_database_erd/schema.md` | `docs/architecture/database/schema.md` | `MOVE` |
| `03_database_erd/` | — | `REMOVE_EMPTY_DIR` |
| `04_operation_contracts/operation_contracts.md` | `docs/architecture/contracts/operation-contracts.md` | `MOVE` |
| `04_operation_contracts/` | — | `REMOVE_EMPTY_DIR` |
| `05_state_machines/README.md` | `docs/architecture/state-machines/README.md` | `MOVE` |
| `05_state_machines/deployment_state.puml` | `docs/architecture/state-machines/deployment.puml` | `MOVE` |
| `05_state_machines/resource_instance_state.puml` | `docs/architecture/state-machines/resource-instance.puml` | `MOVE` |
| `05_state_machines/workload_instance_state.puml` | `docs/architecture/state-machines/workload-instance.puml` | `MOVE` |
| `05_state_machines/` | — | `REMOVE_EMPTY_DIR` |
| `docs/architecture/README.md` | Same | `KEEP`, then update all final links |

The database schema continues to be the sole owner of physical schema and literal ENUM definitions after its move.

## Mapping C — Traceability, decisions, and backlog

| Current path | Final path | Action |
|---|---|---|
| `06_traceability/traceability_matrix.md` | `docs/traceability/matrix.md` | `MOVE` |
| `docs/traceability/README.md` | Same | `KEEP` |
| `docs/decisions/ADR-*.md` | Same | `KEEP` |
| `docs/decisions/README.md` | Same | `KEEP` |
| `docs/backlog/D*.md` | Same | `KEEP` |
| `docs/backlog/README.md` | Same | `KEEP` |
| `06_traceability/design_decisions.md` | `docs/archive/consolidated/design-decisions-log.md` | `VERIFY_THEN_ARCHIVE` |
| `06_traceability/deferred_issues.md` | `docs/archive/consolidated/deferred-issues-log.md` | `VERIFY_THEN_ARCHIVE` |
| `06_traceability/` | — | `REMOVE_EMPTY_DIR` |

When the consolidated logs move, update `source_record` in every ADR and backlog file to the new archive path. Current indexes must link to the split records first and the archived logs only as provenance.

## Mapping D — Implementation, operations, and evidence

| Current path | Final path | Action |
|---|---|---|
| `docs/implementation/` | Same | `KEEP` |
| `uc03/README.md` | Same | `KEEP` as the package-level code entry point |
| `uc03/docs/RUNBOOK.md` | `docs/operations/uc03/RUNBOOK.md` | `MOVE` |
| `uc03/docs/DEMO.md` | `docs/operations/uc03/DEMO.md` | `MOVE` |
| `docs/verification/*.md` | Same | `KEEP` |
| `uc03/docs/VERIFICATION.md` | `docs/archive/consolidated/uc03-verification-log.md` | `VERIFY_THEN_ARCHIVE` |
| `uc03/docs/` | — | `REMOVE_EMPTY_DIR` |
| `implementation_plan.md` | `docs/archive/planning/uc03-original-implementation-plan.md` | `ARCHIVE` |

Before archiving the consolidated verification log, verify that all five dated snapshots preserve the corresponding sections and update each snapshot's `source_record` to the archive path.

## Mapping E — Root, process, and tooling

| Current path | Final path | Action |
|---|---|---|
| `README.md` | Same | `KEEP` |
| `AGENTS.md` | Same | `KEEP`; remove the temporary migration notice when migration closes |
| `docs/INDEX.md` | Same | `KEEP` |
| `docs/CURRENT_STATE.md` | Same | `KEEP`; update after each completed stage |
| `docs/GLOSSARY.md` | Same | `KEEP` |
| `docs/DOCUMENTATION_RULES.md` | Same | `KEEP` |
| `docs/iterations/README.md` | Same | `KEEP` |
| `docs/archive/README.md` | Same | `KEEP` |
| `scripts/check_docs.py` | Same | `KEEP`, then strengthen final-layout checks |
| `.github/workflows/documentation.yml` | Same | `KEEP`, then make PlantUML validation mandatory |
| `usecase_realization_step_1_3.md` | `docs/archive/consolidated/usecase-realization-step-1-3.md` | `VERIFY_THEN_ARCHIVE` |
| `codex_review.md` | — | `EXCLUDE`; untracked user-owned file |

## Execution order

1. Move sequence and VOPC sources into the five use-case context packages.
2. Move shared architecture artifacts under `docs/architecture/`.
3. Move the traceability matrix and update all active links.
4. Compare split artifacts with consolidated predecessors and archive the predecessors.
5. Move operational documents and archive the consolidated verification and implementation plan.
6. Remove empty legacy directories, normalize remaining links, and enforce final-layout checks.

Each numbered item should be a separate reviewable commit unless a move and its required link updates cannot safely be separated.

## Definition of done

The migration is complete only when:

- no tracked documentation remains in `01_*` through `06_*`, `sequence_digrams/`, or loose historical Markdown files at repository root;
- every use case owns its specification, realization, sequence source, VOPC source, and context map;
- every current concept has exactly one canonical owner;
- archived files are clearly historical and no current definition depends on them;
- all internal links resolve and no workstation-specific paths exist;
- PlantUML validation is installed and required in CI rather than skipped;
- the final-layout rules are enforced by `scripts/check_docs.py`;
- `docs/INDEX.md`, `CURRENT_STATE.md`, and `AGENTS.md` no longer describe migration as in progress.
