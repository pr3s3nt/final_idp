---
id: ADR-017
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
related: ADR-016, UC-01
---

# ADR-017 — React web frontend for UC-01, served from the backend origin

## Context

UC-01 is a multi-part editor: application information, workloads, resources,
configuration requirements, dependencies and validation. [ADR-016](ADR-016-client-owned-drafts.md)
places the whole draft in the browser tab, restores it from `sessionStorage`
and sends it to the backend only on Save. Server-rendered Go templates can show
this data, but every edit would need either a full page round-trip or a large
amount of hand-written browser script.

The UC-03, UC-04 and UC-05 pages are Go templates under
`idp/backend/internal/web/templates/`. They work and are verified. The
`idp/frontend/` directory was reserved until a frontend technology, backend
integration, build and delivery approach existed.

## Decision

1. **Technology.** The UC-01 editor is a React web application written in
   TypeScript (strict mode) and built with Vite. React state and a reducer own
   the draft; no state-management library, router library or UI framework is
   added.
2. **Code boundary.** Frontend source, its `package.json`, lockfile, lint,
   test and build configuration live in `idp/frontend/`. Go code, migrations
   and templates stay in `idp/backend/`. Neither tree imports source from the
   other; they share only the JSON API contract.
3. **Same-origin JSON API.** The browser calls the Go backend under `/api/`
   on the same origin as the page. UC-01 uses these endpoints:

   | Method and path | Purpose | Success | Failure |
   |---|---|---|---|
   | `GET /api/application-definitions` | List Application Definitions with their latest version number | 200 | — |
   | `GET /api/application-definitions/{applicationId}` | `updateApplication()`: load the latest version as an editable DTO carrying `baseVersion` | 200 | 404 `NOT_FOUND` |
   | `POST /api/application-definitions` | `saveApplicationDefinition()` for a new application (no `baseVersion`) | 201 | 422 validation problems |
   | `POST /api/application-definitions/{applicationId}/versions` | `saveApplicationDefinition()` for an existing application with `baseVersion` | 201 | 404, 409 `DRAFT_CONFLICT`, 422 |

   Errors use the existing `{"problems": [{"code", "message"}]}` body. A
   problem may add `field`, a path such as `workloads.<id>.name`, so the
   editor can show the error next to the input. No endpoint accepts a single
   field or component edit.
4. **Browser routes.** React owns paths under `/ui/`:
   `/ui/applications`, `/ui/applications/new` and
   `/ui/applications/{applicationId}`. Existing Go routes (`/`, `/apps/...`,
   `/deployments/...`) are unchanged. The Go application list links to
   `/ui/applications`.
5. **Development.** `npm run dev` starts the Vite development server. It
   serves the app under `/ui/` and proxies `/api` to the Go backend
   (`http://127.0.0.1:8088` by default, overridable with `IDP_API_ORIGIN`).
   The browser therefore still sees one origin.
6. **Production build and delivery.** `npm run build` writes static assets to
   `idp/frontend/dist/`. `idp serve` serves that directory under `/ui/` from
   the path in `IDP_FRONTEND_DIR` (default `../frontend/dist`, relative to the
   `idp/backend` working directory used by the runbook). Unknown `/ui/` paths
   return `index.html` so browser routes survive refresh. If the directory has
   no `index.html`, `/ui/` answers 404 with a build hint and every other route
   keeps working. `dist/` and `node_modules/` are not committed. The backend
   container image does not yet contain the bundle; a deployment mounts the
   built directory and sets `IDP_FRONTEND_DIR`.
7. **Coexistence.** Go templates keep serving UC-03, UC-04 and UC-05. This
   decision does not migrate them. A later page moves to React only through its
   own decision or use-case change.
8. **Draft storage.** The editor follows ADR-016. The `sessionStorage` key is
   `idp:uc01:application-draft:<applicationId>` for an existing application
   and `idp:uc01:application-draft:new` for a new one. The stored value carries
   a `schemaVersion`; a value that does not parse, has another schema version
   or has the wrong shape is removed and ignored. UC-01 has no Secret value
   input, so no Secret value, token or credential is ever stored.
9. **Testing.**
   - Go unit tests cover the validator and the specification generator.
   - Go HTTP handler tests cover status codes and the response contract.
   - Go integration tests (`-tags integration`, local test database) cover
     persistence: create, new version, stable IDs, rejection without writes,
     stale `baseVersion`, and two concurrent saves from one base.
   - Vitest with Testing Library and jsdom covers the draft reducer,
     validation, `sessionStorage` adapter and editor behaviour against a mocked
     API client.
   - No browser end-to-end framework is added; the handler, integration and
     component tests together cover the create, edit and save path.

## Consequences

- The repository now has two toolchains: Go for `idp/backend` and Node.js for
  `idp/frontend`. The frontend must be built before `/ui/` works outside the
  Vite development server.
- The UC-01 editor and the Go pages share an origin, so no CORS configuration
  or cross-origin credential handling is needed.
- The Go server has no dependency on Node.js at build or run time.
- Packaging the bundle into the backend container image is left open; it
  needs a build-context change in `idp/backend/Dockerfile`.
- Authentication does not exist in the current backend. The UC-01
  precondition "Developer đã đăng nhập" is not enforced by this decision.
