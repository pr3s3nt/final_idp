# Project instructions for AI agents

## Purpose and scope

`AGENTS.md` defines how AI agents work throughout this repository. It governs
how agents gather context, establish scope, change documentation or code,
preserve existing work, validate results, and hand off changes.

The user's specific request defines the objective and scope of each task. This
file defines how to carry out that task; it does not expand authorization into
areas the user did not request.

This file does not define product requirements, architecture, implementation
behavior, project status, or operating procedures. Use `docs/INDEX.md` to find
the canonical owner of that information.

These rules apply repository-wide. If a directory later contains a more local
`AGENTS.md`, the closest file may add or narrow rules for that subtree, but it
must not silently contradict higher-level rules.

## Starting a task

Before changing any file:

1. Read the user's request and identify:

   - the required outcome;
   - the authorized scope;
   - the expected deliverables;
   - whether a commit is requested or required by the workflow;
   - the conditions that make the task complete.

2. Classify the task:

   - For an explanation, review, or status report, do not modify files unless
     the user also requests a change.
   - For a diagnosis, determine and report the cause. Apply a fix only when the
     request includes remediation.
   - For a change or build task, implement, validate, and hand off the result
     within the authorized scope.

3. Inspect Git state before editing:

   ```bash
   git status --short --branch
   ```

   Treat all existing modifications, untracked files, and local environment
   data as user-owned until proven otherwise.

4. Read `docs/INDEX.md` and select the route matching the task.

5. Load only the smallest sufficient documentation set:

   - For a use-case-specific task, start with
     `docs/use-cases/UC-*/README.md`.
   - Follow it only to the relevant specification, realization, or shared
     artifact.
   - Do not load all of `docs/` merely in case it might be useful.

6. After understanding the intended behavior and design, inspect the related
   source code, tests, configuration, and migrations to determine actual
   implementation state.

7. Use the change-impact map in this file to identify artifacts and components
   that may be affected.

8. If a conflict, missing decision, or ambiguity could materially change the
   outcome, state the issue and request direction. Do not guess in a way that
   changes the user's intent.

9. Begin editing only after establishing enough context and the smallest plan
   that can complete the request.

Read documentation first to understand intent, then read source code to
understand actual state. Do not infer product requirements solely from current
implementation behavior.

## Changing documentation and source code

Classify a change before editing:

| Change type | Required sequence |
|---|---|
| Required product behavior | Update the use-case specification first, then the realization, affected design, source code, tests, and traceability |
| Shared architecture or domain model | Update architecture and any required decision record first, then affected use cases, source code, and tests |
| Database design | Update `docs/architecture/database/schema.md` first, then the ERD, migration, data-access code, and tests |
| Implementation defect against correct canonical documentation | Keep the correct canonical definition; fix source code and add or correct tests |
| Designed but unimplemented behavior | Confirm that canonical documentation is complete, then implement code and tests and update implementation status |
| Behavior-preserving refactor | Do not change specifications to describe code structure; update only affected code maps, paths, runbooks, or implementation guidance |
| Unexplained design/implementation mismatch | Record it in `docs/implementation/deviations.md`, then resolve it through the conflict process in `docs/INDEX.md` |
| An issue that cannot yet be decided | Record it in `docs/backlog/`; do not promote an unaccepted option into current requirements or architecture |

Mandatory rules:

1. Use `docs/INDEX.md` to identify the canonical owner before editing
   documentation.
2. Edit the canonical owner only; other artifacts must link rather than copy a
   definition.
3. Never change canonical documentation merely to make current code appear
   valid.
4. When design intent changes, update the canonical artifact before
   implementing code.
5. When a decision changes or supersedes an accepted decision, update the old
   record, the new record, and the decision index.
6. Preserve stable IDs such as `UC03`, `UC03-BR-12`, `ADR-006`, and `D13` when
   renaming headings or paths.
7. Update traceability when requirements, design, or tests change.
8. Do not use `CURRENT_STATE.md`, verification records, or iteration plans to
   define product requirements.
9. Do not rewrite an old verification record to describe a new execution;
   create a new dated record.
10. At handoff, canonical documentation, source code, tests, and traceability
    must be consistent, or the remaining deviation must be explicit.

If intent changes, update its canonical owner before code. If code violates
already-recorded intent, fix the code rather than rewriting the intent.

## Handling conflicts and missing information

`docs/INDEX.md` defines canonical ownership and conflict classification. This
section defines how an agent acts on those findings.

| Situation | Required action |
|---|---|
| A minor detail is missing, but a safe assumption cannot materially change the result | State the reasonable assumption and continue |
| A missing decision could change behavior, architecture, data, security, or scope | Stop the dependent work and ask the user |
| Code conflicts with canonical documentation within a requested fix | Classify the cause and correct the appropriate side using `docs/INDEX.md` |
| A conflict is outside the authorized scope | Do not fix it automatically; report it separately |
| The user's request appears to conflict with canonical documentation | Explain the conflict and confirm whether this is an intended change before editing the canonical owner |
| Historical material conflicts with current documentation | Do not overwrite current documentation; use history only for provenance |
| Canonical ownership cannot be determined | Recheck `docs/INDEX.md` and relevant indexes, then ask the user if it remains unclear |
| The task is review-only or diagnostic | Report the conflict without creating backlog items, deviations, or file changes unless requested |
| The task authorizes changes but the direction remains unresolved | Record an appropriate backlog item or deviation only when that action is in scope |

Mandatory rules:

1. Do not choose a source merely because it is newer, closer to code, or more
   detailed.
2. Do not treat current implementation behavior as an implicit product
   decision.
3. Do not expand scope to fix every issue discovered during a task.
4. Ask the user only when the answer could materially change the outcome. For a
   low-risk, reversible detail, make and disclose a reasonable assumption.
5. Before concluding that information is missing, inspect relevant canonical
   documentation, code, tests, and Git history within the safe task scope.
6. When blocked by a missing decision, state:

   - what is missing or conflicting;
   - which sources were checked;
   - the viable options;
   - why choosing without input would change the user's intent.

7. Continue independent work that remains safe instead of stopping the entire
   task.

Assume when risk is low, ask when the decision is material, and never turn an
unstated assumption into a requirement or design decision.

## Source-code boundaries

| Area | Role | Boundary rule |
|---|---|---|
| `idp/backend/` | Current IDP backend | Contains the Go service, embedded HTML UI, migrations, fixtures, infrastructure modules, and operational tools |
| `idp/frontend/` | Reserved for a future independent IDP frontend | It does not currently contain a standalone frontend; do not place demo applications here |
| `demo-apps/` | Workloads deployed by the IDP for demos and verification | These applications are not IDP product source code |
| `docs/` | Requirements, design, implementation guidance, operations, and evidence | It does not contain runtime product source code |
| `scripts/` | Repository-level validation and maintenance tools | Keep it distinct from `idp/backend/scripts/`, which contains backend operational tools |

Mandatory rules:

1. The backend Go module is:

   ```text
   sdp
   ```

2. The demo-application Go module is:

   ```text
   github.com/pr3s3nt/final_idp/demo-apps
   ```

3. Do not introduce imports that make the backend depend on `demo-apps/` or
   vice versa. They have different purposes and lifecycles.

4. The current UI templates live in:

   ```text
   idp/backend/internal/web/templates/
   ```

   They are embedded in the Go binary and served by the backend. Do not move
   them to `idp/frontend/` without changing the frontend build, packaging, and
   delivery design.

5. `demo-apps/cmd/frontend/` is a sample workload deployed by the IDP. It is not
   the IDP administrative UI.

6. When moving source paths or renaming a module, update `go.mod`, imports,
   build tools, Dockerfiles, runbooks, design-to-code maps, and related links in
   the same logical change.

7. Run Go commands from the correct module directory:

   ```bash
   cd idp/backend
   go test ./...

   cd ../../demo-apps
   go test ./...
   ```

8. Do not create source code in `idp/frontend/` merely to populate the
   directory. Initialize an independent frontend only after its technology,
   backend integration, build, and deployment approach are defined.

`idp/` contains the IDP product; `demo-apps/` contains workloads used to verify
the product. A sample application's name does not make it part of the IDP UI.

## Change-impact map

The table requires inspection, not automatic editing. Change only artifacts
whose content is actually affected.

| Change | At minimum, inspect |
|---|---|
| Use-case flow or business rule | Specification, realization, sequence diagram, operation contracts, traceability, and tests |
| Add, rename, or remove a use case | `docs/use-cases/README.md`, the use-case package, `docs/INDEX.md`, `CURRENT_STATE.md`, traceability, and related decisions and architecture; inspect `AGENTS.md` and change it only if workflow or scope changes |
| System operation or participant | Realization, sequence diagram, VOPC, design classes, and operation contracts |
| Domain object or relationship | Domain model, design classes, persistence classification, schema, data-access code, and tests |
| Table, column, constraint, or literal enumeration | `schema.md`, ERD, migration, repositories, queries, and tests |
| State or state transition | Use-case specification, state machine, operation contracts, schema enumeration, handling code, and tests |
| Architecture decision | Decision record, architecture, affected use cases, related backlog, and source code |
| External integration provider | Integration architecture, decision record, adapter, configuration, runbook, and verification |
| Application deployment behavior | UC-03, realization, planning, worker, states, traceability, and verification |
| Application removal behavior | UC-05, realization, worker, state machines, resource cleanup, and verification |
| Secrets or sensitive data | Security architecture, affected specifications, Secret Store, manifest generation, configuration, runbook, and tests |
| API or user interface | Use-case specification, realization, request handlers, templates, tests, and usage guidance |
| Source path or module name | `go.mod`, imports, build tools, Dockerfiles, runbooks, code maps, and documentation links |
| Operational command or environment variable | Configuration, operational scripts, Dockerfiles, runbooks, security, and related verification |
| Shared terminology | `GLOSSARY.md` and every canonical artifact that uses the term |
| Accept or supersede a decision | Old and new decision records, decision index, affected canonical artifacts, backlog, and traceability |
| Resolve a backlog item | Backlog record and index, canonical documentation, source code, tests, traceability, and project state where applicable |
| Change implemented scope | `CURRENT_STATE.md`, implementation documentation, source code, tests, and verification evidence |
| Add verification results | A new dated record, verification index, and project state if the result changes the current conclusion |
| Documentation structure | `INDEX.md`, `AGENTS.md`, `DOCUMENTATION_RULES.md`, documentation validation, and continuous integration |

Use the map as follows:

1. “Inspect” does not mean “edit.” Change an artifact only when its content is
   affected.
2. Start with the canonical owner, then follow its links and dependencies.
3. Search the repository for references to any changed path, name, stable ID,
   or concept.
4. Update `CURRENT_STATE.md` only when implemented scope, limitations, or
   current conclusions change.
5. Never edit an old verification record; create a new record for new results.
6. Be able to explain why an inspected artifact did not require a change.
7. If impact reaches outside the authorized scope, report it rather than
   silently expanding the task.

Completing a change means checking consumers of its meaning, paths, and
interfaces—not only editing the directly named file.

## Repository and data safety

Treat the repository as a workspace shared with the user.

### User-owned changes

1. Inspect Git state before editing and before committing.
2. Do not modify, delete, revert, or commit pre-existing user changes outside
   task scope.
3. Treat untracked files as potentially important user work.
4. If user changes overlap a file that must be edited:

   - preserve the user's content;
   - edit only the task-related portion;
   - stage only the intended file or hunk;
   - stop and ask if the changes cannot be separated safely.

### Secrets and local data

1. Never commit `.env` files, encryption keys, access tokens, SSH keys,
   credentials, Terraform state, or local runtime data and logs containing
   sensitive information.
2. `idp/backend/.env` is ignored local state. Do not read or display it unless
   the task genuinely requires it.
3. Before committing, inspect every staged path and verify that no sensitive
   data is included.
4. Do not place secret values in commands, messages, documentation, tests, or
   persisted output.

### Generated files and environment state

Do not commit these areas unless explicitly requested:

```text
idp/backend/bin/
idp/backend/var/
.terraform/
*.tfstate
*.tfstate.*
```

If validation creates files, identify the exact files created by the agent and
remove them without disturbing pre-existing user data.

### Destructive actions

Before deleting, overwriting, or moving data:

1. Resolve the exact target.
2. Use read-only checks when needed.
3. Do not rely on broad paths, unverified variables, or unsafe globs.
4. Prefer recoverable operations.
5. Ask the user when scope is unclear.
6. After deleting material data, report what was removed and how it can be
   recovered.

### External systems

1. Do not create, modify, or delete cloud resources, Kubernetes clusters,
   databases, remote repositories, or external secrets unless the user placed
   them in task scope.
2. Authorization to change code does not authorize deployment or mutation of a
   real environment.
3. Do not push commits when the user requested only a local commit.
4. Identify tests that may mutate external state before running them.

Protect user work, prevent secret exposure, and do not turn a file-edit request
into authorization to change external data or systems.

## Validation, commits, and handoff

Validation must be proportional to the change and its risk. Do not claim
completion without running relevant checks or explaining why a check could not
run.

### General validation

Before every commit, run:

```bash
git diff --check
git status --short
```

Inspect both the content and the complete path list being committed; do not rely
only on test results.

### Validation by change type

| Changed scope | Minimum validation |
|---|---|
| Documentation | `python3 scripts/check_docs.py` and `git diff --check` |
| Go backend | Run `go test ./...` and `go build ./...` in `idp/backend/` |
| Demo applications | Run `go test ./...` and `go build ./...` in `demo-apps/` |
| Shell scripts | Run `bash -n` on changed scripts |
| Terraform modules | Run formatting checks and `terraform validate` for affected modules when the environment permits |
| Migration or data access | Check schema, migration, repositories, and relevant database tests |
| PlantUML diagrams | Validate syntax when PlantUML is available; otherwise report that continuous integration still requires it |
| Renamed path or identifier | Search the repository for unintended stale references, excluding deliberate provenance records |
| Added, renamed, or removed use case | Run documentation validation to confirm a complete package and required index registration |

Run tests requiring a database, cloud account, or Kubernetes cluster only when
the environment is available and external state changes are authorized.

### Commit rules

1. Keep each complete logical change in a separate reviewable commit.
2. Unless the user requests different grouping, create a local commit after
   completing and validating each logical change.
3. Commit only task-related files or hunks.
4. Before committing, inspect:

   ```bash
   git diff --cached --check
   git diff --cached
   git status --short
   ```

5. Write a commit message that describes the achieved outcome, not merely the
   editing action.
6. Do not amend old commits or rewrite history unless requested.
7. Do not push unless requested.
8. After committing, inspect Git state and confirm that every remaining change
   is intentional.
9. Report any user-owned changes that remain outside the commit.

### Definition of done

A task is complete only when:

1. The requested outcome exists.
2. The change stayed within authorized scope.
3. Canonical documentation, source code, tests, and traceability are consistent,
   or any remaining deviation is explicit.
4. No broken links, unintended stale references, or generated files were
   committed.
5. Relevant checks passed or their limitations were reported.
6. A commit was created when required by the workflow.
7. Remaining uncommitted changes are identified as user-owned or otherwise
   explained.

### Handoff

Report concisely:

- the main outcome;
- important areas changed;
- commits created;
- checks run and their results;
- skipped checks and why;
- remaining changes or issues the user should know about.

Validate the affected surface, commit only task-owned work, and provide enough
information for review without requiring the user to guess.
