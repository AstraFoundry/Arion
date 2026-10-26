# Quality

This section defines testing strategy, code-review expectations, and quality gates.

## Documents

- [`testing.md`](testing.md) — testing strategy and test types.
- [Native and product E2E](e2e.md) — real window lifecycle and future OS scenario driver.
- [Windows validation](windows-validation.md) — executed native, installed speech, package and updater checks with release limits.
- [Evaluation tooling](../../tools/evaluation/README.md) — executable benchmarks and driver contracts.
- [Update validation](update-validation.md) — staged update, reminder and notification checks with native platform limits.
- [Foundation validation](validation.md) — actual local results and remaining coverage.
- [macOS keyboard shortcut incident](keyboard-shortcut-incident.md) — Shift and Right Option root causes, repairs and acceptance evidence.
- [`code-review.md`](code-review.md) — code-review checklist.
- [`agent-skills.md`](agent-skills.md) — installed Agent Skills for UI craft and anti-slop.

## Quality principles

- Tests must be deterministic.
- Tests must run quickly enough to be executed on every change.
- Code review is required before merging to the main branch.
- Static analysis (lint, type check, formatting) must pass before merge.

## Definition of done

- Feature is implemented according to the documented architecture.
- Unit tests cover domain logic and critical paths.
- Integration tests cover repository and external-service boundaries.
- Documentation is updated if behavior or conventions changed.
- No hardcoded strings, redundant copy, duplicated code, or fallback cleverness.

## Arion release evidence

Stable acceptance IDs live in [specs](../../specs/README.md). Current checks cover foundation invariants, links/imports and shell compilation; they do not establish functional speech or IME support. OS host lifecycle, offline/privacy, checkpoint provenance and concurrent ASR/ranking performance are required before a feature is declared releasable. See [testing](testing.md) and [evaluation](../research/evaluation.md).
