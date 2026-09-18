---
id: VER-2026-09-18-UC01-BUILDER
artifact: verification-record
status: evidence
executed_on: 2026-09-18
implementation: 1417b5e
---

# UC-01 component-focused Application Builder

> This file records observations from a specific execution. It is evidence,
> not a normative requirement.

Environment: local WSL2 workstation, Node.js 24.16.0 and npm 11.13.0. The
checkout was branch `uc03-impl` at implementation commit `1417b5e`. No
database, backend process, cloud environment or Kubernetes cluster was used.

## Scope

This execution followed the presentation refactor described by
[UC-01 Application Builder UI design](../use-cases/UC-01/ui-design.md). The
backend API, persistence and domain implementation did not change.

## Automated checks

| Command | Directory | Result |
|---|---|---|
| `npm run lint` | `idp/frontend` | pass: ESLint and strict TypeScript |
| `npm test -- --run --reporter=dot` | `idp/frontend` | pass: 5 files, 40 tests |
| `npm run build` | `idp/frontend` | pass: Vite production bundle, 29 modules transformed |
| `python3 scripts/check_docs.py` | repository root | pass: 86 Markdown files and 84 artifact IDs before this record was added; PlantUML was unavailable locally |
| `git diff --check` | repository root | pass before the implementation commit |

The component tests observed that:

- adding a Workload or Resource updates the browser-owned draft and opens that
  component's focused workspace;
- Workload runtime, outputs, configuration requirements and dependencies are
  edited together without an API request;
- Review presents a read-only topology and component summary;
- a failed Save selects Review, exposes total and per-workspace problem counts,
  and can navigate from the summary back to the affected field;
- Save still submits one complete draft with stable component IDs;
- a `DRAFT_CONFLICT` preserves the draft, allows its JSON to be copied, and
  requires confirmation before loading the latest durable version;
- draft restore, Discard, successful Save, failed Save and the absence of any
  Secret value input continue to pass their existing tests.

The generated bundle contained 11.43 kB of CSS and 259.37 kB of JavaScript
before gzip. `dist/` remained ignored and was not committed.

## Not executed

- No real-browser or visual-regression test; responsive behavior and final
  rendering still need human review in a browser.
- No backend Go test or database integration test because this change did not
  modify backend code, API contracts, persistence or migrations. The earlier
  backend/API execution remains recorded in
  [UC-01 React editor, API and optimistic save](2026-09-18-uc01-react-editor.md).
- No container image build or deployed-environment run.
- No PlantUML syntax check; no PlantUML file changed.
