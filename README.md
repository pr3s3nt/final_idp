# Internal Developer Platform — design and implementation

This repository applies the Unified Process (UP): development is use-case-driven, architecture-centric, iterative, and risk-driven. It contains the use-case model and design for UC-01 through UC-06, working Go and React implementations, and end-to-end evidence for UC-03 and UC-05.

## Start here

- [Documentation index](docs/INDEX.md) — authoritative map for humans and AI agents.
- [Current project state](docs/CURRENT_STATE.md) — implemented scope, lifecycle position, and known limitations.
- [Glossary](docs/GLOSSARY.md) — canonical project terminology.
- [IDP source code](idp/README.md) — backend/frontend boundaries and source entry points.
- [Backend implementation](idp/backend/README.md) — UC-03/UC-05 design-to-code map.
- [Demo applications](demo-apps/README.md) — workloads used by demos and end-to-end verification.
- [Runbook](docs/operations/uc03/RUNBOOK.md) — build, run, test, and troubleshooting instructions.

## Documentation model

The documentation is organized by UP artifact rather than by lifecycle phase. Phase and iteration are metadata about an artifact; they are not separate copies of requirements, design, and implementation.

Each use case has a small context package under `docs/use-cases/`. Shared architectural models remain canonical in their existing design artifacts and are indexed from `docs/architecture/README.md`. Historical plans and execution evidence are explicitly separated from current requirements.

## Current scope

| Use case | Name | Current state |
|---|---|---|
| UC-01 | Create / Configure Application | Implemented as a React web editor with a Go JSON API; verified by automated tests |
| UC-02 | Configure Application Environment | Designed; the UC-03 implementation consumes imported configuration rather than implementing the full configuration experience |
| UC-03 | Deploy Application | Implemented and verified on `kind-local` and AWS |
| UC-04 | View Deployment Result | Designed; a minimal query/UI path exists to support UC-03 execution tracking |
| UC-05 | Remove Application from Environment | Implemented and verified on `kind-local` and AWS |
| UC-06 | Đăng nhập bằng tài khoản nội bộ | Implemented with React login, local-user CLI, server-side sessions and automated tests; not yet verified in a deployed browser environment |

See [current state](docs/CURRENT_STATE.md) for the precise baseline and unresolved issues.
