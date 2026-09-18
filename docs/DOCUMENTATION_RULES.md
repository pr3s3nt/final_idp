---
id: DOC-RULES
artifact: documentation-standard
status: current
last_reviewed: 2026-09-17
---

# Documentation rules

## Artifact states

Use exactly one of these values in document metadata:

| Status | Meaning |
|---|---|
| `draft` | Proposed content that has not been accepted. |
| `current` | Current canonical content for its declared scope. |
| `superseded` | Replaced by another named artifact. |
| `deferred` | Accepted as a known problem but intentionally postponed. |
| `historical` | Retained to explain project history; not normative. |
| `evidence` | Observation from a particular test or execution; not normative. |

## Required metadata

New canonical Markdown artifacts should start with YAML metadata containing at least:

```yaml
---
id: UC-03
artifact: use-case-specification
status: current
last_reviewed: 2026-09-17
---
```

Use `related`, `implementation`, `supersedes`, or `superseded_by` when they materially improve navigation. Dates describe review/evidence time; they do not determine authority by themselves.

## Content boundaries

- Specification states required externally observable behavior.
- Realization maps behavior to operations and collaborating design elements.
- Architecture defines shared structures and cross-cutting constraints.
- ADRs explain accepted decisions and consequences.
- Backlog items describe unresolved risks or issues.
- Implementation documents map design to code and disclose deviations.
- Verification records capture what was actually executed in a specific environment.

Do not combine these roles in one new document.

## Links and stable references

- Use repository-relative Markdown links.
- Prefer stable artifact IDs and headings over line-number references.
- Use IDs such as `UC03-MF-07`, `UC03-A1-04`, `UC03-BR-12`, `ADR-006`, and `D13` when cross-document traceability is needed.
- When a file is moved, update inbound links in the same change.

## Diagrams

PlantUML is the editable source. Every important diagram must be reachable from a textual context document that explains its purpose, scope, key participants, and invariants. A rendered image is never canonical without its source.

Each `.puml` file under `docs/` is committed together with a rendered `<name>.png` beside it, so a diagram can be read without installing PlantUML. Regenerate the image in the same change that edits the source:

```bash
plantuml -tpng -o . docs/use-cases/UC-02/ui/*.puml
```

Rendered output outside `docs/` stays out of the repository.

Two PlantUML pitfalls have already cost a broken diagram here. `interface` is a class-diagram keyword: in a sequence diagram it parses until the first `==` divider and then fails, so declare `participant "..." as X <<interface>>` instead. `Header` is the page-header directive, so a relation line starting with an element aliased `Header` is read as header text; use another alias such as `HeaderBar`.

## Semantic removal gate

Link, metadata, and ID validation cannot prove that a split artifact preserves every current concept from a consolidated predecessor. Before removing a superseded or consolidated predecessor from the working tree:

1. Audit the complete predecessor manually, not only a sample of known identifiers.
2. Record a canonical destination or a historical rationale for every current-looking concept.
3. Verify schema claims against executable migrations and behavior claims against implementation/tests when applicable.
4. Migrate every current gap before removing the predecessor.
5. Record the last commit and original path so the predecessor remains retrievable with `git show <commit>:<path>`.
6. Leave no current concept owned only by deleted Git history.

The completed migration gates and their reconciliation checklists are defined in [MIGRATION_PLAN.md](MIGRATION_PLAN.md). Do not recreate `docs/archive/` as a compatibility location.

## Change completion checklist

1. Update the canonical artifact.
2. Follow the change-impact map in the root `AGENTS.md`.
3. Update related ADR/backlog status when applicable.
4. Update traceability from requirement through test.
5. Add verification evidence when the change was executed.
6. Run documentation link, metadata, and PlantUML checks.

Run the repository check with:

```bash
python3 scripts/check_docs.py
```

The script validates front matter, unique artifact IDs, internal links, portable paths, retired-layout absence, and PlantUML syntax when `plantuml` is installed. GitHub Actions installs PlantUML and runs with `REQUIRE_PLANTUML=1`, so diagram validation cannot be skipped in CI.
