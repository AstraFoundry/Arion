# Quality

This section defines testing strategy, code-review expectations, and quality gates.

## Documents

- [`testing.md`](testing.md) — testing strategy and test types.
- [Native and product E2E](e2e.md) — real window lifecycle and future OS scenario driver.
- [Windows validation](windows-validation.md) — executed native, installed speech, package and updater checks with release limits.
- [Product driver contract](../../tools/driver/README.md) — public transport, artifacts and owned E2E scenarios.
- [Research ownership](../project/repository-split.md) — private evaluation, historical replay and evidence integrity.
- [Update validation](update-validation.md) — staged update, reminder and notification checks with native platform limits.
- [Foundation validation](validation.md) — actual local results and remaining coverage.
- [macOS keyboard shortcut incident](keyboard-shortcut-incident.md) — Shift and Right Option root causes, repairs and acceptance evidence.
- [Security review](security-review.md) — findings with severity and status, fixes made and open owner decisions.
- [`code-review.md`](code-review.md) — code-review checklist, PR format and screenshot workflow.
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
- User-facing copy is centralized and concise, preserving accessible names and necessary guidance.
- Existing implementations of the same business rule are reused; similar code shape alone does not require abstraction.
- Failures are not silently masked. Spec-approved recovery paths are tested and preserve privacy, cancellation and target-identity guarantees.
- Documentation changes pass `pnpm check` before handoff, even without a commit. Additional checks follow [testing](testing.md); report results and checks not run with reasons. Repository static checks do not establish native functionality.

## Arion release evidence

Stable acceptance IDs live in [specs](../../specs/README.md). Current checks cover foundation invariants, links/imports and shell compilation; they do not establish functional speech or IME support. OS host lifecycle, offline/privacy, checkpoint provenance and concurrent ASR/ranking performance are required before a feature is declared releasable. See [testing](testing.md) and [evaluation](../research/evaluation.md).
