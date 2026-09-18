---
id: ADR-018
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-18
related: ADR-017, UC-01, UC-01-UI-DESIGN
---

# ADR-018 — Primer design tokens as the visual foundation of the web frontend

## Context

[ADR-017](ADR-017-react-web-frontend.md) chose React with no UI framework, and
the [UC-01 user interface design](../use-cases/UC-01/ui/README.md) requires
native browser controls, readable labels and keyboard navigation. Neither
document defined a visual system. The result is that `idp/frontend/src/styles.css`
invented one while it was written.

Measured on the file as it stands:

- 49 hexadecimal colour values, of which only 17 are declared as tokens. The
  remaining 32 occurrences are written directly into rules and cover 19 colours
  that have no name. `#344054`, the text colour of every form label, is used
  four times and is one of them.
- 24 distinct pixel values for `padding`, `margin` and `gap`, including `3px`,
  `5px`, `7px`, `9px`, `11px`, `13px` and `15px`. No scale produces those
  numbers; they are the result of adjusting by eye.
- 8 font sizes, two of which (`13px` and `14px`) are close enough to be
  indistinguishable, and a heading step from `20px` to `15px` that is too
  abrupt.
- 6 font weights, including `550`, `650` and `750`. The stack names `Inter`
  first but no stylesheet ever loads it, so the browser renders a system font
  and rounds those three weights unpredictably.

The users of UC-01 are Developers who work in common engineering tools every
day. Familiar conventions cost them less to learn than a visual language
invented for this product alone.

Three options were considered.

1. **Define an original token system.** Full control, but every value would
   need to be chosen and contrast-checked here, and the result would carry no
   familiarity for its users.
2. **Adopt a component framework** such as MUI or Chakra. Rejected: it
   contradicts ADR-017 (no UI framework) and the UC-01 requirement to prefer
   native controls, and it adds a large runtime dependency.
3. **Adopt the tokens of an existing open-source design system** without its
   component layer. Keeps ADR-017 intact, adds no runtime dependency, and
   inherits values that are already contrast-checked.

## Decision

1. **Token source.** The visual foundation is
   [Primer](https://primer.style), GitHub's open-source design system,
   through the `@primer/primitives` package (MIT licence). The accepted values
   are recorded in the *Ngôn ngữ thị giác* section of the
   [UC-01 user interface design](../use-cases/UC-01/ui/README.md), read from
   `dist/css/functional/themes/light.css` in version `11.10.0`.

2. **Scope of inheritance.** Colour tokens, the spacing scale
   (`--base-size-*`: 4, 8, 12, 16, 20, 24, 28, 32, 40, 48), border radius
   (`--borderRadius-medium` of 6px as the default), font weights (400, 500 and
   600 only) and the [Octicons](https://primer.style/octicons) icon set, also
   MIT.

3. **What is not inherited.** Layout, page header, tab bars and feed patterns.
   The information architecture of UC-01 stays as its user interface design
   defines it: header, builder navigation and workspace. Primer is a source of
   values, not of screens.

4. **No brand identity.** The GitHub name, logo and trade dress are not used.
   The MIT licence covers source code, not brand identity. For the same reason
   the `Mona Sans VF` brand typeface is removed from the head of Primer's
   `--fontStack-sansSerif`; the frontend uses the system fallback part of that
   stack and loads no web font.

5. **No runtime dependency.** `@primer/primitives` is not added to
   `package.json`. The accepted values are copied into
   `idp/frontend/src/styles/tokens.css`, and Octicon paths are embedded as
   inline SVG. The frontend keeps `react` and `react-dom` as its only runtime
   dependencies, as ADR-017 requires.

6. **Monospace for technical values.** Identifiers that a Developer types or
   reads as data — application name, workload name and type, image repository,
   port, output name, Environment Variable name, Secret name, component names
   in the topology and summary table, and version numbers — are set in the
   monospace stack. Prose written by the system — description, hints, section
   help and error messages — is not.

7. **Light appearance only.** UC-01 has no dark appearance. Adding one is a
   change to the user interface design and needs its own decision.

8. **Scope of this decision.** It governs `idp/frontend/`. The Go templates
   for UC-03 to UC-05 under `idp/backend/internal/web/templates/` are
   unchanged. A later use case that moves to React inherits these tokens; one
   that stays on Go templates does not have to.

## Consequences

- Every colour, spacing, radius, font size and weight in the frontend must
  resolve to a token declared in `src/styles/tokens.css`. A literal value
  written into a rule is a defect.
- The UC-01 user interface design changes from describing the visual language
  in prose to recording exact values. It gains a table that can be checked
  against the code, and it must be updated when a token changes.
- Upgrading tokens is a manual comparison against a newer
  `@primer/primitives`, because the package is not a dependency. The version
  read is recorded in the user interface design so the comparison has a
  starting point.
- The primary action button becomes green (`bgColor-success-emphasis`), which
  replaces the single blue action colour the earlier design described. Blue
  remains the accent for links and selection.
- Users familiar with common engineering tools recognise the conventions, which
  is the benefit sought. The cost is that the product looks less distinct from
  those tools; this is accepted for an internal platform.
- No web font request is made, so first paint does not wait on a font
  download, and the interface uses the font the Developer's operating system
  already renders.
