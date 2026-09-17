---
id: TRACEABILITY-INDEX
artifact: traceability-index
status: current
last_reviewed: 2026-09-17
---

# Traceability index

The detailed cross-artifact coverage matrix is [matrix.md](matrix.md).

## Required chain

For behavior changes, maintain the following chain:

```text
Use-case flow or business rule
→ system operation
→ sequence interaction
→ participating class/component
→ operation contract and persistent data
→ implementation entry point
→ automated scenario or verification record
```

## Stable ID convention

Use IDs when new traceability rows are introduced:

- `UC03-MF-07`: main-flow step 7.
- `UC03-A1-04`: alternative/exception-flow item 4.
- `UC03-BR-12`: business rule 12.
- `UC03-SO-03`: system operation 3.
- `TS-UC03-08`: test scenario 8.

Existing prose without IDs remains valid; introduce IDs incrementally when touching the relevant section rather than renumbering everything in one mechanical change.
