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

PlantUML is the editable source. Every important diagram must be reachable from a textual context document that explains its purpose, scope, key participants, and invariants. Generated images are optional and are not canonical without their source.

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

The script validates front matter, unique artifact IDs, internal links, portable paths, and PlantUML syntax when `plantuml` is installed. GitHub Actions runs the same documentation contract on pushes and pull requests.
