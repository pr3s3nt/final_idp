---
id: VER-2026-09-18-UC01-FRONTEND
artifact: verification-record
status: evidence
executed_on: 2026-09-18
related: UC-01, UC-01-UI-DESIGN, ADR-017, ADR-018
---

# UC-01 frontend restructure and Primer token adoption

> This file records observations from a specific execution. It is evidence, not a normative requirement.

Environment: local WSL2 workstation, Node.js 24.16.0, npm 11.13.0, Go 1.27.1.
The Vite development server served the editor and proxied `/api` to an already
running `idp serve` on `127.0.0.1:8088`, backed by the local `idp-uc03-db`
PostgreSQL 17 container. No backend code changed, and no cloud, Kubernetes or
delivery repository was used.

Scope of the change under verification: eleven commits from
`docs: adopt Primer design tokens as the UC-01 visual foundation` to
`fix(frontend): correct the layout and affordances found in visual review`,
touching 46 files with 1030 insertions and 493 deletions, all inside
`idp/frontend/` and `docs/`.

## Automated checks

| Command | Directory | Result |
|---|---|---|
| `npm run lint` (ESLint and strict TypeScript, two projects) | `idp/frontend` | pass, no error and no warning |
| `npm test` (Vitest, jsdom) | `idp/frontend` | pass: 5 files, 40 tests |
| `npm run build` | `idp/frontend` | pass |
| `python3 scripts/check_docs.py` | repository root | pass: 88 Markdown files, 86 documented artifact IDs; PlantUML not installed locally, so diagram syntax was not checked |
| `git diff --check` | repository root | clean |

The 40 tests passed unmodified at every step. No test file was edited during
the restructure, so they held the behaviour fixed while the structure moved.

Two observations support the claim that the refactor changed no behaviour:

- After `refactor(frontend): split styles.css into files that own one concern
  each`, the built stylesheet kept the content hash it had before the split
  (`index-DuacidAb.css`), so the ten files produce byte-identical output.
- The structural commits changed no `.test.ts` or `.test.tsx` file. The only
  production change outside file moves and import rewrites was dropping the
  unused `draft` prop from `ResourceWorkspace`.

## Stale-reference searches

`AGENTS.md` requires a repository search after a renamed path. Three searches
were run after the moves and all came back clean:

| Search | Result |
|---|---|
| `src/components`, `src/pages/`, `src/draft/`, `src/api/`, `Sections.tsx`, `src/styles.css` under `docs/` and `idp/frontend/README.md` | one hit, in `ADR-018`, describing the state at the time of the decision; a deliberate provenance record |
| hexadecimal colour values outside `src/styles/tokens.css` | none |
| the glyphs used as icons before the change, in any `.tsx` | none |

## Manual review in the browser

The editor was opened against real data: two Application Definitions,
`shop-app` at version 3 with four workloads, PostgreSQL and Redis, and
`reporting-app` at version 2.

Confirmed working:

- the list page, the Overview, Workload, Resource and Review workspaces, and
  navigation between them;
- validation errors shown next to the field that owns them, with the problem
  count on the owning navigation item;
- the restored-draft notice and the `Discard` confirmation.

Five defects were found that no automated check covers. All were fixed and are
recorded here because they show what this kind of change can break silently.

| Defect | Cause | Fix |
|---|---|---|
| The whole page carried a grey that reads as beige on a warm display | `--canvas` was mapped to Primer's `bgColor-muted`, which Primer reserves for recessed areas | Map `--canvas` to `bgColor-default` and leave `--surface-subtle` on `bgColor-muted` |
| Large empty margins beside the builder | `.page` capped at 88rem and `.workspace-panel` at 58rem | Drop both caps |
| A port field stretched across the window | Removing the caps left the two-column grid unbounded | Cap the field tracks at 34rem and centre the workspace on 70rem, the width of the widest form row |
| Input and secondary-button borders were barely visible | `--line` at `#d1d9e0` gives about 1.4:1 on white, below the 3:1 that WCAG 1.4.11 requires for the boundary of a control | Add `--control-border` from `borderColor-emphasis` at about 3.5:1 and apply it to interactive outlines only |
| Sections ran together, and row `Remove` actions did not read as buttons | `.form-section` had only a top rule; `.icon-button` had a transparent border and muted text | Give each section a bordered box with tinted panels inside, and give row actions a boundary at rest plus the trash Octicon |

The contrast defect is the one worth keeping in mind: GitHub itself uses the
lighter border on form controls, so copying Primer faithfully reproduced a
failure. `ADR-018` already states that Primer is a source of values, not of
decisions; this is the first case where that boundary mattered.

## Not covered

- No automated check enforces that every value resolves to a token. The rule in
  `ADR-018` is held by review and by the searches above.
- No automated check enforces the WCAG 1.4.11 contrast ratio. The two ratios
  quoted here were computed by hand from the sRGB values and should be
  re-measured with a tool before the rule is relied on elsewhere.
- The backend was not rebuilt or retested, because no backend file changed.
- Only the light appearance exists, as `ADR-018` decided, so no dark appearance
  was checked.
- The editor was reviewed on a desktop browser only. UC-01 targets desktop web,
  so the narrow layouts in `responsive.css` were not exercised.
