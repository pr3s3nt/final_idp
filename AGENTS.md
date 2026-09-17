# Project instructions for AI agents

## Documentation entry point

Read `docs/INDEX.md` before changing requirements, design, implementation, tests, or operational documentation.

The documentation layout migration is complete. Do not recreate `docs/archive/`, the retired numbered directories, `sequence_digrams/`, `uc03/docs/`, the former root `uc03/` code directory, or loose historical root documents. The completed [migration record](docs/MIGRATION_PLAN.md) explains provenance; current destinations are indexed from `docs/INDEX.md`.

For a use-case-specific task, read that use case's `docs/use-cases/UC-*/README.md` context file and only then follow its links to shared artifacts.

## Code boundaries

- `idp/backend/` contains the current Go backend, including the server-rendered UI templates embedded in the backend binary.
- `idp/frontend/` is reserved for a future independently built IDP frontend. Do not move the embedded templates there without also changing the build and delivery design.
- `demo-apps/` contains workloads deployed by the IDP for demos and verification. Its `frontend` command is a demo workload, not the IDP user interface.

## Sources of truth

- Required behavior: `docs/use-cases/UC-*/specification.md`
- Use Case Realization: `docs/use-cases/UC-*/realization.md`
- Shared architecture and data model: `docs/architecture/README.md`
- Accepted design decisions: `docs/decisions/README.md`
- Open or deferred problems: `docs/backlog/README.md`
- Current implementation status and design deviations: `docs/implementation/README.md`
- Requirement-to-design-to-test mapping: `docs/traceability/README.md`

Deleted historical sources remain available through Git provenance recorded in `docs/implementation/documentation-reconciliation.md`; they are never normative. Verification records describe observations from a particular execution; they do not define required behavior.

## Editing rules

1. Use relative links for repository files. Never add workstation-specific paths such as `/home/...`.
2. Do not duplicate a canonical definition. Link to its source instead.
3. Database table, column, constraint, and ENUM definitions are canonical in `docs/architecture/database/schema.md`.
4. Keep PlantUML source next to a textual explanation or link it from a realization/architecture document. Do not make an image the only source of design meaning.
5. Preserve stable IDs such as `UC03`, `ADR-006`, and `D13` when renaming headings or files.
6. When a decision supersedes another decision, update both records and the decision index.
7. When an issue is resolved, update the backlog index, the canonical design, traceability, and verification evidence as applicable.
8. Do not silently turn implementation observations into requirements. Record an ADR when implementation changes the intended design.

## Change impact map

| Change | Artifacts to inspect |
|---|---|
| Use-case flow or business rule | Specification, realization, sequence diagram, traceability, tests |
| System operation or participant | Realization, sequence diagram, VOPC, operation contracts |
| Domain object or relationship | Domain model, persistence classification, schema, repository code |
| Table, column, constraint, or ENUM | Schema, ERD, migration, repositories, tests |
| State transition | State machine, operation contract, schema ENUM, tests |
| Integration provider | Architecture, ADR, adapter, runbook, verification |
| Deployment or teardown behavior | UC-03/UC-05, worker design, state machines, verification |

## Repository safety

- `codex_review.md` may be an untracked working file. Do not modify or delete it unless the user explicitly asks.
- Preserve unrelated user changes and generated/local environment files.

## Validation before handoff

Run:

```bash
python3 scripts/check_docs.py
git diff --check
```

PlantUML is mandatory in CI through `REQUIRE_PLANTUML=1`. If it is unavailable locally, report that only the local diagram syntax check was skipped; all other checks must still pass.
