---
id: IDP-FRONTEND-README
artifact: code-area-readme
status: current
last_reviewed: 2026-09-17
---

# IDP frontend

This directory is reserved for a future independently built IDP frontend.

The current user interface is server-rendered by the Go backend. Its templates
remain under [`../backend/internal/web/templates/`](../backend/internal/web/templates/)
so that Go can embed them in the backend binary. Do not move the demo workload
in `demo-apps/cmd/frontend` here; that program represents an application
deployed by the IDP, not the IDP user interface.
