---
id: DOC-INDEX
artifact: documentation-index
status: current
last_reviewed: 2026-09-18
---

# Documentation index

This file helps humans and AI agents find the smallest authoritative set of
project documents required for a task. It identifies where to start, which
artifact owns each kind of information, and whether a document is current,
historical, or evidence-only.

This index does not define product behavior, architecture decisions,
implementation status, or operating procedures. Those definitions live in the
canonical artifacts linked below.

## Project entry points

Humans should start with the [repository README](../README.md) to understand the
project's purpose, scope, and development approach. AI agents must start with
the [agent instructions](../AGENTS.md) to learn the mandatory working rules.
Both then use this index to find the smallest authoritative document set needed
for the task at hand.

These three files provide introduction, instructions, and routing only. They do
not own definitions of product behavior, architecture, implementation status,
or operating procedures.

## Route by task

Choose the task that most closely matches the work and read the specified entry
point first. Follow only the links needed from that document. If a task crosses
multiple concerns, combine the corresponding rows.

| Task | Read first |
|---|---|
| Understand the current scope, completion level, or limitations | [Current project state](CURRENT_STATE.md) |
| Resolve project terminology | [Project glossary](GLOSSARY.md) |
| Change UC-01 behavior | [UC-01 context](use-cases/UC-01/README.md) |
| Change UC-02 behavior | [UC-02 context](use-cases/UC-02/README.md) |
| Change deployment or deployment planning | [UC-03 context](use-cases/UC-03/README.md) |
| Change deployment-result or status presentation | [UC-04 context](use-cases/UC-04/README.md) |
| Change application removal from an environment | [UC-05 context](use-cases/UC-05/README.md) |
| Change authentication, login, logout, session, or local user provisioning | [UC-06 context](use-cases/UC-06/README.md) |
| Change the shared domain model or architecture | [Architecture index](architecture/README.md) |
| Change a database table, column, constraint, or literal enumeration | [Database schema](architecture/database/schema.md) |
| Understand why a design choice was made | [Decision index](decisions/README.md) |
| Work on an open or deferred problem | [Backlog index](backlog/README.md) |
| Compare design with code or inspect an implementation deviation | [Implementation index](implementation/README.md) |
| Change backend source code | The relevant use-case context, then the [backend implementation map](../idp/backend/README.md) |
| Work on an independent IDP frontend | The relevant use-case context, then the [frontend boundary](../idp/frontend/README.md) |
| Change demo workloads | [Demo applications](../demo-apps/README.md) |
| Run, test, or troubleshoot the system | [UC-03 runbook](operations/uc03/RUNBOOK.md) |
| Review evidence from actual executions | [Verification index](verification/README.md) |
| Check requirement-to-design-to-test coverage | [Traceability index](traceability/README.md) |
| Change documentation structure or conventions | [Documentation rules](DOCUMENTATION_RULES.md) |
| Plan or record an iteration | [Iteration index](iterations/README.md) |

The table routes readers to a starting point; it does not grant that entry point
ownership of every subject it links to.

## Canonical information ownership

Index files and `README.md` files primarily provide context and navigation. They
do not own a definition merely because they are read first.

Each kind of current information has one canonical owner. Other artifacts may
summarize or link to that information, but they must not create an independent
definition.

| Kind of information | Canonical owner | Must not be replaced by |
|---|---|---|
| Required behavior of a use case | `docs/use-cases/UC-*/specification.md` | Source code, tests, sequence diagrams, or verification records |
| How components collaborate to realize a use case | `docs/use-cases/UC-*/realization.md` | Behavioral specifications or source code |
| Shared architecture and models | The corresponding specialized artifact under the [architecture index](architecture/README.md) | Repeated definitions in use cases or code comments |
| Database tables, columns, constraints, and literal enumerations | [Database schema](architecture/database/schema.md) | Migration files or implementation-plan prose |
| An accepted decision and its rationale | The corresponding record under the [decision index](decisions/README.md) | Discussion notes, iteration plans, or source code |
| An unresolved problem, risk, or deferred concern | The corresponding record under the [backlog index](backlog/README.md) | Current architecture or an accepted decision |
| Project-wide implemented scope and current limitations | [Current project state](CURRENT_STATE.md) | Behavioral specifications or historical plans |
| A known difference between design and implementation | [Implementation deviations](implementation/deviations.md) | Silently changing canonical design to match code |
| Design-to-code mapping | [Code map](implementation/code-map.md) | Architecture or product-behavior definitions |
| Actual implemented behavior | The corresponding source code and tests | Required product behavior |
| Requirement-to-design-to-test mapping | [Traceability matrix](traceability/matrix.md) | The detailed requirement, design, or test itself |
| Current operating and troubleshooting procedure | [UC-03 runbook](operations/uc03/RUNBOOK.md) | Product requirements or verification evidence |
| Result of a specific verification execution | The corresponding dated record under the [verification index](verification/README.md) | Specifications, current status, or commitments about future runs |
| Shared terminology | [Project glossary](GLOSSARY.md) | Inconsistent local definitions in individual artifacts |
| Plan and outcome of an iteration | The corresponding record under the [iteration index](iterations/README.md) | Long-lived requirement or architecture definitions |
| Documentation organization and editing rules | [Documentation rules](DOCUMENTATION_RULES.md) | AI working rules or product decisions |
| AI working rules | [Agent instructions](../AGENTS.md) | Product specifications, architecture, or operations guidance |

Source code and tests are evidence of behavior that has been implemented, but
they cannot change product requirements by themselves. If code differs from a
canonical document, determine whether the code contains a defect or implements
an unrecorded requirement change, then update the correct side explicitly.

An accepted decision record owns the decision and its rationale. When that
decision affects behavior, architecture, or data, update the corresponding
canonical artifacts as well; the decision record does not replace them.

A migration executes a database change, while `schema.md` owns the current
schema definition. If they disagree, record and investigate the discrepancy
instead of assuming that either side is always correct.

Superseded or deleted documents do not define current information. Use Git
history only when provenance must be inspected.

## Resolving conflicts

Do not resolve a conflict merely by choosing the newest, longest, or
code-adjacent file. First identify the conflicting concept and the artifact that
owns that kind of information.

When a conflict is found:

1. Identify the exact concept or behavior in conflict.
2. Use the ownership table to identify its canonical owner.
3. Check whether each source is current, superseded, historical, or evidence.
4. Check for an accepted decision that has not yet been propagated to the
   affected canonical artifacts.
5. Classify the conflict as an implementation defect, an unrecorded requirement
   change, a stale summary, or an unresolved design issue.
6. Make any required change in the canonical owner first, then update dependent
   documentation, source code, tests, and traceability where applicable.
7. If there is not enough information to decide, record the issue in the
   [backlog](backlog/README.md), report the conflicting sources, and do not guess.

| Conflict | Correct response |
|---|---|
| An index or `README.md` disagrees with a canonical owner | Keep the canonical definition and correct the index or `README.md` |
| Source code or tests disagree with the current specification | Determine whether this is an implementation defect or an unrecorded requirement change, and record the deviation before resolving it |
| An accepted decision has not been reflected in architecture or specifications | Treat it as incomplete decision propagation and update every affected canonical artifact |
| Two artifacts both claim ownership of the same concept | Select one canonical owner, move the definition there, and replace the other definition with a link |
| A current artifact disagrees with deleted Git history | Keep the historical source as provenance; do not use it to overwrite the current definition |
| A verification record disagrees with current status | Preserve the verification record as an observation of that execution and update `CURRENT_STATE.md` when appropriate |
| `schema.md` disagrees with a migration | Record the discrepancy, compare both with the database and design intent, and correct the appropriate side |
| The correct definition cannot be determined | Create or update a backlog record with the conflicting sources and stop changes that depend on the unresolved decision |

A conflict is resolved only when the canonical owner, dependent artifacts,
source code, tests, and related traceability are consistent—not merely when one
file has been edited.
