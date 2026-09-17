---
id: IDP-CODE-README
artifact: code-area-index
status: current
last_reviewed: 2026-09-17
---

# IDP source code

This directory contains the source code of the Internal Developer Platform:

- [`backend/`](backend/) contains the Go service, JSON API, embedded HTML UI
  for UC-03 to UC-05, database migrations, infrastructure modules, fixtures,
  and operator scripts.
- [`frontend/`](frontend/) contains the React + TypeScript web frontend with
  the UC-01 editor ([ADR-017](../docs/decisions/ADR-017-react-web-frontend.md)).

Demo workloads are intentionally kept outside the product source tree under
[`demo-apps/`](../demo-apps/).
