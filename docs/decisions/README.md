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
- [`adr-template.md`](adr-template.md) — template for new ADRs.

## Naming

Use a sequential number and a short kebab-case title:

```
decisions/
  001-use-uuid-for-ids.md
  002-choose-postgresql.md
```
