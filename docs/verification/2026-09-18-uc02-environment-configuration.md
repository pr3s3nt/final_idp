---
id: VER-2026-09-18-UC02-CONFIG
artifact: verification-record
status: evidence
executed_on: 2026-09-18
implementation: UC-02 implementation in the same logical change
---

# UC-02 environment configuration

> This file records observations from a specific execution. It is evidence,
> not a normative requirement.

Environment: local WSL2 workstation, Go 1.27.1, Node.js 24.16.0, npm 11.13.0
and PostgreSQL in the dedicated local `idp_test` database. The checkout was
branch `uc03-impl` at commit `c35d48c`. No cloud account or Kubernetes cluster
was mutated.

## Scope

This execution verified the first UC-02 implementation: loading the
requirements of an environment, choosing a Catalog Version and a deployment
target, listing valid Resource and Workload Outputs, staging a Secret value,
validating a complete draft and saving it with both concurrency bases. It also
covered the Configuration page built on the accepted
[UC-02 UI design](../use-cases/UC-02/ui/README.md).

Deployment behaviour was not exercised: UC-02 stores references only, and
resolving them stays with UC-03.

## Automated checks

| Command | Directory | Result |
|---|---|---|
| `go build ./...` | `idp/backend` | pass |
| `go test ./...` | `idp/backend` | pass: 11 packages with tests, including the new `configvalidator` |
| `go test -tags integration ./internal/service -count=1` | `idp/backend` | pass in 30.7s against PostgreSQL, including the seven UC-02 cases |
| `npm run lint` | `idp/frontend` | pass: ESLint and strict TypeScript |
| `npm test` | `idp/frontend` | pass: 10 files, 84 tests |
| `npm run build` | `idp/frontend` | pass: Vite production bundle, 50 modules transformed |
| `python3 scripts/check_docs.py` | repository root | fail, see below |
| `git diff --check` | repository root | pass |

Documentation validation reported 98 Markdown files and 96 documented artifact
IDs with no broken link or missing frontmatter, but failed on its PlantUML
step. The failure is `docs/use-cases/UC-05/sequence.puml`, which predates this
change: it declares `interface` participants and also uses `==` dividers, a
combination PlantUML rejects in a sequence diagram. Both PlantUML 1.2020.02 and
1.2024.7 report the same error at line 30. Every other diagram, including the
three added for UC-02, passes `plantuml -checkonly`.

## What the PostgreSQL integration tests observed

- `selectEnvironment()` returns the requirements of the latest Application
  Definition version, the stored configuration with its opaque revision, and
  the Catalog Versions and deployment targets to choose from.
- Saving the loaded configuration again advances the opaque revision and keeps
  exactly one `environment_configuration` row for the application and
  environment.
- A second Save from the same `baseConfigurationRevision` is rejected with
  `DRAFT_CONFLICT`, and row counts in `environment_configuration`,
  `environment_variable`, `configuration_value` and `secret` stay unchanged.
- A draft carrying an older `baseApplicationDefinitionVersion` is rejected the
  same way, again without writing.
- A binding to an output the resolved Resource Definition does not expose is
  rejected with `INVALID_OUTPUT_REFERENCE` and writes nothing.
- A staged Secret value reaches the Secret Store, the configuration database
  holds only the opaque reference, and no row contains the plaintext.
- `requestResourceOutputs()` answers with the definition that the selected
  Catalog Version and deployment target resolve, and refuses a resource the
  workload does not depend on.

## What the frontend tests observed

- The page opens on the first workload that has requirements and shows only
  that workload's variables and secrets; navigation switches workload.
- The newest Catalog Version and the first deployment target are preselected.
- A variable is offered the normal outputs of the resolved definition and a
  secret only the sensitive ones; a workload can reference only components it
  depends on.
- A typed Secret is exchanged for an opaque reference, and neither the request
  body of Save nor `sessionStorage` contains the plaintext.
- Save with a missing required value opens Review, reports the count and does
  not call the API; a problem in the summary leads back to the workload that
  owns it.
- A `DRAFT_CONFLICT` keeps the browser draft; a successful Save clears it.
- `sessionStorage` drops a Secret value even when a value was written into
  storage directly, and refuses a stored draft with an unexpected shape.

## Not covered by this execution

- Real-browser use of the Configuration page. The page was served from
  `idp serve` on this workstation for manual inspection, but no browser
  evidence is recorded here.
- `Environment Configuration` for an application deployed to more than one
  target at the same time.
- The staged-Secret lifecycle of [D07](../backlog/D07-orphaned-secret.md):
  a Secret staged but never saved is still left in the Secret Store.
