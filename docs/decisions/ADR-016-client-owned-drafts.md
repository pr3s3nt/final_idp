---
id: ADR-016
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
supersedes: ADR-001
related: D01
---

# ADR-016 — Browser-owned drafts for UC-01 and UC-02

## Context

UC-01 and UC-02 are multi-step editing flows. The previous design placed one
draft object on each backend application service and sent every edit through
the API. It did not define how a draft survived requests, distinguished users,
worked across multiple backend instances, or handled concurrent edits. D01
tracked this unresolved design after ADR-001 deferred it.

The current UI is rendered by the Go backend, but rendering ownership does not
require the backend service to own mutable draft state between requests.

## Decision

1. `ApplicationDefinitionDraft` and `EnvironmentConfigurationDraft` are
   client-owned DTOs. The browser tab owns the active draft; they are not
   persistent domain objects and no backend draft repository or table exists.
2. Creating a UC-01 application initializes an empty draft in the Web UI.
   Editing an application loads the latest immutable Application Definition
   once, including its version as `baseVersion`, then all field/component edits
   mutate the client draft locally.
3. Opening UC-02 loads the latest Application Definition requirements and the
   current Environment Configuration once. Its draft carries
   `baseApplicationDefinitionVersion` and an opaque
   `baseConfigurationRevision` when configuration already exists.
4. The Web UI stores the non-sensitive draft in browser `sessionStorage`,
   scoped by use case, application and environment where applicable. This
   restores a draft after refresh in the same browser tab. Save or explicit
   Discard clears it; closing the tab/session may discard it.
5. Plaintext Secret values are never written to `sessionStorage`. After a
   direct Secret value is sent through the secure staging flow, the client may
   retain only the opaque reference. The staging, compensation and cleanup
   lifecycle remains owned by D07 and is not decided here.
6. Field edits do not call the backend. Backend calls are reserved for loading
   initial durable state, querying server-owned catalogs, securely staging a
   Secret when required, and Save.
7. Save submits the complete draft. The backend remains stateless between edit
   requests, validates the complete DTO again, and performs the durable write
   atomically.
8. Save uses optimistic concurrency, and comparison plus durable write execute
   as one atomic transaction/CAS. UC-01 rejects an edit when the current
   latest Application Definition version differs from `baseVersion`. UC-02
   rejects when either the latest Application Definition version or current
   Environment Configuration revision differs from the draft bases. Rejection
   performs no write and requires the Developer to reload and review/reapply
   the draft; automatic merge is outside this decision.

## Consequences

- Drafts do not depend on backend process memory, sticky sessions, or a draft
  cleanup job.
- Refresh in the same tab is recoverable without adding draft tables.
- Closing the tab may lose unsaved work; this is an explicit boundary of the
  MVP behavior.
- Browser code must serialize, restore and clear drafts deterministically and
  must exclude plaintext Secrets.
- Save request DTOs carry base concurrency metadata and complete draft content.
- Server-side validation remains authoritative even when the UI validates
  fields locally.
- ADR-001 is superseded and D01 is resolved. D07 remains open.
