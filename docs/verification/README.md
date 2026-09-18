---
id: VERIFICATION-INDEX
artifact: verification-index
status: current
last_reviewed: 2026-09-18
---

# Verification evidence

Verification records capture what was executed in a particular environment on a particular date. They support traceability but do not define required behavior. A passing historical run does not prove that the current checkout still passes.

| Date | Scope | Record |
|---|---|---|
| 2026-09-15 | UC-03 automated tests, kind-local and AWS baseline | [UC-03 baseline](2026-09-15-uc03-kind-aws.md) |
| 2026-09-15 | Catalog Version and pre-existing internal cluster | [Catalog versioning](2026-09-15-catalog-versioning.md) |
| 2026-09-16 | Per-application Delivery Repository | [Delivery Repository](2026-09-16-delivery-repository.md) |
| 2026-09-16 | Fleet as an interchangeable CD provider | [Fleet provider](2026-09-16-fleet-provider.md) |
| 2026-09-16 | UC-05 on kind-local and AWS | [UC-05 removal](2026-09-16-uc05.md) |
| 2026-09-18 | UC-01 React editor, API and optimistic save | [UC-01 editor](2026-09-18-uc01-react-editor.md) |
| 2026-09-18 | UC-01 component-focused Application Builder presentation | [UC-01 Application Builder](2026-09-18-uc01-application-builder.md) |
| 2026-09-18 | UC-06 local accounts, sessions, route protection and React login | [UC-06 local authentication](2026-09-18-uc06-local-authentication.md) |

The removed consolidated evidence predecessor and its mapping to these snapshots are recorded in the [documentation reconciliation](../implementation/documentation-reconciliation.md). Add future executions as new immutable files rather than appending unrelated rounds to an existing snapshot.
