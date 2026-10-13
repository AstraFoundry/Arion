# Decisions

This section records architecture decision records (ADRs). Each ADR explains why a significant decision was made and what alternatives were considered.

## When to write an ADR

Write an ADR when the decision:

- Is hard to reverse.
- Affects multiple layers or teams.
- Has non-obvious trade-offs.

## Existing decisions

- [001: Desktop native boundaries](001-desktop-native-boundaries.md).
- [002: Local pipeline and self-hosted memory](002-local-pipeline-and-memory.md).
- [003: Shortcut-mode insertion routes](003-insertion-routes.md).
- [004: The default model is chosen by measured evaluation](004-automatic-model-selection.md).
- [005: Local encryption of approved vocabulary](005-local-encryption.md).
- [006: macOS input-method host and local IPC](006-macos-input-method-host.md).
- [007: Sunbeam design system and coss anatomy in Go](007-sunbeam-design-system.md).
- [008: Contextual rewrite evaluation](008-contextual-rewrite-evaluation.md) — proposed capability and runtime evaluation boundary.
- [009: AGPL-3.0 license](009-agpl-license.md) — open source with a commercial hosted edition kept possible.
- [010: In-app updates from GitHub releases](010-in-app-updates.md) — signed MyGo updates, automatic checks on by default.
- [`adr-template.md`](adr-template.md) — template for new ADRs.

## Naming

Use a sequential number and a short kebab-case title:

```
decisions/
  001-use-uuid-for-ids.md
  002-choose-postgresql.md
```
