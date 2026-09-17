---
id: VER-2026-09-18-UC01
artifact: verification-record
status: evidence
executed_on: 2026-09-18
---

# UC-01 React editor, API and optimistic save

> This file records observations from a specific execution. It is evidence, not a normative requirement.

Environment: local WSL2 workstation, Go 1.27.1, Node.js 24.16.0, npm 11.13.0,
PostgreSQL 17 test database `idp_test` in the local `idp-uc03-db` container.
No cloud, Kubernetes or delivery repository was used.

## Automated checks

| Command | Directory | Result |
|---|---|---|
| `go test -count=1 ./...` | `idp/backend` | pass |
| `go build ./...`, `go vet ./...` | `idp/backend` | pass |
| `go test -count=1 -tags integration ./internal/service/` | `idp/backend` | pass: 14 tests, including 6 new UC-01 tests and the existing UC-03/UC-05 tests |
| `npm run lint` (ESLint and strict TypeScript) | `idp/frontend` | pass |
| `npm test` (Vitest, jsdom) | `idp/frontend` | pass: 4 files, 37 tests |
| `npm run build` | `idp/frontend` | pass |
| `python3 scripts/check_docs.py` | repository root | pass; PlantUML not installed locally, so diagram syntax was not checked |

The UC-01 integration tests showed:

- a new application saves as version 1 with the submitted component IDs and a `score-yaml` specification;
- an edit saves version 2, a renamed workload keeps its ID, and version 1 rows are unchanged;
- invalid drafts (duplicate name, missing reference, cycle, image tag) change no row in the nine UC-01 tables;
- a draft based on version 1 after version 2 exists returns `DRAFT_CONFLICT` and changes no row;
- eight concurrent saves from the same base produce exactly one new version; the other seven return `DRAFT_CONFLICT`;
- a component ID owned by another application returns `INVALID_COMPONENT_ID`, a taken name returns `APPLICATION_NAME_TAKEN`, and an unknown application returns `NOT_FOUND`, all without writes.

## Manual smoke run

`idp serve` ran from a scratch build against `idp_test` on `127.0.0.1:18088`
with `IDP_FRONTEND_DIR` pointing at the built bundle:

| Request | Result |
|---|---|
| `GET /` (Go template) | 200 |
| `GET /ui`, `GET /ui/applications/new`, bundle asset under `/ui/assets/` | 200 |
| `POST /api/application-definitions` (valid draft) | 201, `baseVersion` 1 |
| `GET /api/application-definitions/{id}` | 200, `baseVersion` 1 |
| `POST .../{id}/versions` with base 1, renamed workload | 201, `baseVersion` 2, workload ID unchanged |
| Same request again with base 1 | 409 `DRAFT_CONFLICT` |
| `POST /api/application-definitions` with name `Bad` | 422 `INVALID_NAME` |
| `GET /api/applications` (existing UC-03 API) | 200 |

The Vite development server (`npm run dev` with `IDP_API_ORIGIN`) served
`/ui/applications` with 200 and proxied `GET /api/application-definitions` to
the backend.

## Not executed

- No browser end-to-end run; editor behaviour was exercised with Testing Library in jsdom.
- No container image build; the image does not include the React bundle (ADR-017).
- PlantUML syntax check, which CI runs.
