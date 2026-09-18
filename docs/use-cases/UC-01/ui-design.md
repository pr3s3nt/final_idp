---
id: UC-01-UI-DESIGN
artifact: user-interface-design
status: current
last_reviewed: 2026-09-18
related: UC-01, ADR-016, ADR-017
---

# UC-01 — Application Builder UI design

## Purpose and boundary

This document defines the accepted interaction and screen design for the
UC-01 React editor. The [UC-01 specification](specification.md) remains the
owner of required product behaviour; this document turns that behaviour into
a reviewable interface contract for humans and coding agents.

The editor is a **structured application builder**. Developers edit real
components in focused forms and use a read-only review topology to verify the
result. The topology is not a drag-and-drop authoring surface.

## Design principles

1. Organize the editor around the application components a Developer thinks
   about: application, workloads and resources.
2. Keep all properties of one workload together: identity, runtime, outputs,
   configuration requirements and dependencies.
3. Make the current editing location, validation state and unsaved state
   visible without requiring the Developer to scan one long form.
4. Use Review as a deliberate final check. It summarizes the draft and shows
   topology without introducing another editable representation.
5. Preserve the browser-owned draft, complete-draft Save and optimistic
   concurrency behavior defined by ADR-016.
6. Prefer native controls, readable labels and keyboard navigation over
   visually novel controls. Do not require a UI framework or icon library.

## Information architecture

The editor shell contains three persistent regions on desktop:

- **Header** — back navigation, application identity, version/draft status,
  Discard and Save.
- **Builder navigation** — Overview, one item for each Workload, one item for
  each Resource, add-component actions and Review. Items show a problem count
  when their fields have validation problems.
- **Workspace** — the focused form or the read-only Review view.

Overview edits application name and description and explains the boundary of
UC-01. A Workload workspace owns its general fields, outputs, Environment
Variable definitions, Secret definitions and dependencies. A Resource
workspace owns resource name and type. Review owns the topology, validation
summary and component/configuration summary.

Changing a component name must not change navigation identity: React keys and
selection use stable component IDs, while visible labels update immediately.

## Desktop wireframes

### Component editor

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ ← Applications   shop-app   Version 2 · Unsaved       Discard   Save app   │
├──────────────────────┬───────────────────────────────────────────────────────┤
│ APPLICATION          │ Workload / backend                    Remove workload │
│   Overview           │                                                       │
│                      │ General                                               │
│ WORKLOADS            │ Name       [backend                ]                  │
│ ● backend        2 ! │ Type       [Backend Service        ]                  │
│ ○ frontend           │ Repository [registry/.../backend   ]  Port [8080]     │
│ + Add workload       │                                                       │
│                      │ Outputs                                               │
│ RESOURCES            │ [endpoint                              ] [Remove]     │
│ ◇ postgresql         │ + Add output                                          │
│ + Add resource       │                                                       │
│                      │ Configuration requirements                            │
│                      │ Environment variables        Secrets                  │
│ REVIEW               │ [DB_HOST] [Required]         [DB_PASSWORD] [Required]│
│   Review         2 ! │                                                       │
│                      │ Dependencies                                          │
│                      │ backend depends on [postgresql ▼]                     │
└──────────────────────┴───────────────────────────────────────────────────────┘
```

### Review

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ ← Applications   shop-app   Version 2 · Unsaved       Discard   Save app   │
├──────────────────────┬───────────────────────────────────────────────────────┤
│ Overview             │ Review application                                    │
│ Workloads            │ 2 problems must be fixed before saving                │
│ ● backend        1 ! │                                                       │
│ ○ frontend       1 ! │ Topology (read only)                                  │
│ Resources            │ [frontend] ──depends on──▶ [backend]                  │
│ ◇ postgresql         │                              │                        │
│                      │                              └──▶ [postgresql]        │
│ Review           2 ! │                                                       │
│                      │ Components                                            │
│                      │ 2 workloads · 1 resource · 2 dependencies             │
│                      │ backend: 1 output · 1 variable · 1 secret             │
└──────────────────────┴───────────────────────────────────────────────────────┘
```

The topology may use a deterministic CSS/HTML layout. It does not need to be a
general graph canvas. Every relationship must also be available as text so the
view remains understandable to assistive technology and for large graphs.

## Responsive behavior

At narrow widths the header wraps, actions remain reachable, and builder
navigation becomes a horizontally scrollable component selector above the
workspace. The selected item stays visually and programmatically identified.
Forms become one column. Tables or topology content may scroll inside their
own region, but the whole page must not require horizontal scrolling.

The responsive layout keeps the same sections and actions; it does not hide
fields or create a separate mobile workflow.

## Interaction rules

### Navigation and editing

- New application opens Overview. An existing application also opens Overview
  after its latest version and any tab-local draft are restored.
- Adding a Workload or Resource creates it in the client draft and selects its
  workspace immediately.
- Removing the selected component first asks for confirmation. Confirmation
  removes the component and its dependencies through the draft reducer, then
  selects the nearest remaining item or Overview.
- Dependency source is implicit in a Workload workspace. The Developer chooses
  only the component that the current workload depends on. Existing relations
  are displayed and removable there.
- Review is always reachable. It does not mutate the draft.

### Validation

- Local validation runs from the current draft. Field messages appear next to
  the related control after the first Save attempt or when server validation
  returns that field.
- Navigation badges show problem counts for Overview, each component and
  Review. Selecting an item exposes its problems; Review provides links back
  to the affected workspace or field.
- Save validates the complete draft. If local problems exist, the editor opens
  Review, focuses its validation summary and performs no API request.
- Backend validation is authoritative and is merged into the same navigation,
  field and Review presentation.

### Draft status and actions

- Header status distinguishes `Saved`, `Unsaved changes`, `Restored draft` and
  `Saving…`. For an existing definition it also shows the base version and the
  version that a successful Save will create.
- Save and Discard remain in the header on every workspace. Save sends the
  complete draft once; field and navigation changes make no API call.
- Discard asks for confirmation only when changes exist. It clears the local
  draft and reloads the durable definition (or an empty new definition).
- Success confirms the saved version and clears the local draft as defined by
  ADR-016.

### Save conflict

A `DRAFT_CONFLICT` must not replace or clear the Developer's draft. The editor
shows a blocking conflict panel with:

1. an explanation that another version was saved first;
2. **Copy draft JSON**, so work can be kept outside the tab;
3. **Download draft JSON**, when browser download support is available;
4. **Load latest version**, which explicitly confirms that the local draft will
   be discarded.

Automatic merge and a side-by-side diff are outside UC-01. The exported JSON
is a recovery aid, not an import or public API contract.

## Screen states

| State | Presentation and available recovery |
|---|---|
| Initial load | Workspace skeleton/status; no editable empty flash |
| Load failed | Error explanation, Try again and Back to applications |
| Application not found | Not-found explanation and Back to applications |
| Clean draft | `Saved` status; Discard disabled |
| Dirty draft | `Unsaved changes · kept in this tab`; Save and Discard enabled |
| Restored draft | Informational message and `Restored draft` header status |
| Saving | Save disabled with `Saving…`; current content remains visible |
| Validation failed | Review selected, summary focused, badges and field errors visible |
| Save failed | Non-destructive error; Retry uses the normal Save action |
| Conflict | Draft preserved; copy/download recovery and confirmed Load latest |
| Save succeeded | Saved-version message, clean draft, durable version reloaded |

## Action-to-state/API mapping

| User action | Client draft/session state | HTTP request |
|---|---|---|
| Open new application | Restore `new` draft or create empty draft | None |
| Open existing application | Load latest, then restore matching tab-local draft when present | One `GET /api/application-definitions/{applicationId}` |
| Select workspace | Selection only | None |
| Add, edit, rename or remove component/field | Reducer update and `sessionStorage` update | None |
| Open Review | Selection only; validate current draft for display | None |
| Discard | Clear `sessionStorage`; reinitialize/reload | Existing application performs the normal load GET |
| Save valid new application | Keep draft until success, then clear it | One `POST /api/application-definitions` |
| Save valid existing application | Keep draft until success, then clear it | One `POST /api/application-definitions/{applicationId}/versions` |
| Copy/download conflicted draft | No state change | None |
| Load latest after conflict | Confirm, clear local draft, load latest | One load GET |

## Accessibility contract

- The shell has one `h1`; workspace sections use an ordered heading hierarchy.
- Builder navigation is a labelled navigation region. The selected workspace
  uses `aria-current`; problem badges include screen-reader text.
- Every control has a persistent label. Meaning is never communicated by
  color or an icon alone.
- Validation summary and save/load failures use appropriate live-region roles;
  focus moves only after an explicit Save attempt or recovery action.
- Add/remove controls include the affected component or requirement name in
  their accessible name.
- All editing, navigation, confirmation and conflict-recovery actions work by
  keyboard.
- The interface respects reduced-motion preferences and maintains visible
  focus and sufficient contrast.

## Visual language

Use a quiet neutral canvas, white work surfaces, dark navy text and one blue
action color. Component types may use restrained, redundant shapes/labels
(circle for workload, diamond for resource), but shape and color never replace
text. Borders and spacing establish hierarchy; avoid nested decorative cards,
large hero areas and dashboard metrics that do not help complete UC-01.

The result should feel like an engineering tool: compact enough to compare
configuration, but not so dense that labels, errors or recovery guidance are
hidden.
