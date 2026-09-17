---
id: ITERATION-INDEX
artifact: iteration-index
status: current
last_reviewed: 2026-09-17
---

# Iteration index

The repository historically used commits, dated verification rounds, and sections of the original implementation plan rather than stable iteration IDs. Do not invent retrospective iteration boundaries: doing so would create false project history.

For future work, create one `Ixx-short-name.md` file before an iteration starts and record:

- lifecycle phase and objective;
- use-case scenarios and risks addressed;
- artifacts expected to change;
- exit criteria;
- resulting commits and verification records;
- unresolved work carried forward.

Historical progress can be reconstructed from the Git source recorded in the [documentation reconciliation](../implementation/documentation-reconciliation.md) and from the [verification index](../verification/README.md); neither should be treated as a current iteration plan.
