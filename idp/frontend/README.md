---
id: IDP-FRONTEND-README
artifact: code-area-readme
status: current
last_reviewed: 2026-09-18
---

# IDP frontend

React + TypeScript web frontend built with Vite. The technology, API
integration, build and delivery approach are decided in
[ADR-017](../../docs/decisions/ADR-017-react-web-frontend.md). It currently
implements the [UC-01](../../docs/use-cases/UC-01/README.md) editor.
Its accepted information architecture, states and interaction rules are in the
[UC-01 Application Builder UI design](../../docs/use-cases/UC-01/ui-design.md).

The server-rendered pages for UC-03 to UC-05 stay in
[`../backend/internal/web/templates/`](../backend/internal/web/templates/).
Do not move the demo workload in `demo-apps/cmd/frontend` here; that program is
an application deployed by the IDP, not the IDP user interface.

## Commands

Run every command in this directory. Node.js 24 and npm 11 were used.

```bash
npm ci              # install exact dependencies from package-lock.json
npm run dev         # Vite dev server on http://127.0.0.1:5173/ui/, proxies /api to the Go backend
npm run lint        # ESLint and TypeScript type check
npm test            # Vitest component and unit tests (jsdom)
npm run build       # production bundle in dist/
```

`npm run dev` proxies `/api` to `http://127.0.0.1:8088`. Set
`IDP_API_ORIGIN` to use another backend address. After `npm run build`, the Go
server (`idp serve`) serves `dist/` under `/ui/`; see the
[runbook](../../docs/operations/uc03/RUNBOOK.md).

## Layout

| Path | Role |
|---|---|
| `src/api/` | JSON DTO types and the API client (list, load for edit, Save) |
| `src/draft/model.ts`, `reducer.ts` | `ApplicationDefinitionDraft` in the browser and its local edit actions |
| `src/draft/storage.ts` | `sessionStorage` adapter with schema version and shape checks |
| `src/draft/validation.ts` | Client-side UC-01 rules for fast feedback; the backend re-validates |
| `src/pages/` | Application list and editor pages |
| `src/components/` | Editor sections, fields and validation summary |
| `src/router.ts` | Minimal History API router for `/ui/` paths |
