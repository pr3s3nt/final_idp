---
id: ADR-020
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-18
related: D12
---

# ADR-020 — UC-02 resolves valid outputs from a Developer-selected Catalog Version and deployment target

## Context

UC-02 lets a Developer bind an Environment Variable or Secret to a Resource
Output, for example `DB_HOST ← postgresql.host`. The IDP may only offer outputs
that a `Resource Definition` actually exposes.

ADR-012 made the Platform Catalog versioned and let the Developer choose a
Catalog Version when deploying. It left two questions open, recorded as D12:

1. Which Catalog Version supplies the output list shown by UC-02, given that an
   Environment Configuration is not versioned?
2. A single `resource_type` may be served by several Resource Definitions whose
   exposed outputs differ. `postgresql` can be Aurora on AWS and an in-cluster
   Postgres on an internal Kubernetes cluster, and only one of them exposes
   `reader_host`. Without knowing where the application will be deployed,
   UC-02 cannot tell which output list is correct.

The resolution rule already exists in the design: a Resource Definition is
selected from `resource_type`, the deployment context derived from the
deployment target, and the definition's applicability conditions. UC-03 applies
that rule at deployment time. UC-02 previously had no equivalent input.

Two candidate answers were rejected. Showing the union of every definition of a
`resource_type` accepts bindings that fail later at deployment time. Showing the
intersection hides outputs that are legitimately available on the target the
Developer intends to use.

## Decision

1. The UC-02 Configuration page asks the Developer to select a Catalog Version
   and a deployment target before value sources are chosen. The Catalog Version
   defaults to the newest one.
2. Both values belong to the client-owned `EnvironmentConfigurationDraft` of
   ADR-016. They are sent with every catalog query and with Save.
3. The IDP resolves each Resource Requirement to exactly one Resource
   Definition using the selected Catalog Version, the deployment context of the
   selected target, and the definition's applicability conditions — the same
   selection rule UC-03 uses. The valid output list offered to the Developer,
   and the output list enforced by `validateEnvironmentConfiguration()`, are the
   `exposedOutputs` and `sensitiveOutputs` of that single resolved definition.
4. Neither value is persisted. No column is added to
   `environment_configuration`, and an Environment Configuration remains
   unversioned and not bound to a Catalog Version or a target. Reopening the
   page starts from the defaults again.
5. UC-03 keeps its own check. Contract 6 still validates the stored
   configuration against the Application Definition version and Catalog Version
   chosen for that deployment, and still reports A1 when a referenced output
   does not exist there.

## Consequences

- UC-02 offers exactly the outputs that the intended target can provide, so
  `reader_host` is selectable when the Developer selects a target whose
  definition exposes it, and is rejected otherwise.
- A Developer who configures for one target and then deploys to another can
  still hit UC-03 A1. That path is unchanged and remains the authoritative
  check; UC-02 reduces how often it is reached rather than replacing it.
- The database schema, the "configuration has no version" rule, and UC-03
  contract 6 are unchanged.
- The Configuration page depends on the Catalog Version list and the deployment
  target list, so the Environment Configuration Service reads the
  platform-managed catalog in addition to the Application Repository.
- D12 is resolved. D07 remains open and still owns the staged Secret lifecycle.
