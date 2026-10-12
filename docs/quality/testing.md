# Testing

## Current checks

- `pnpm check`: local docs links, configuration JSON, unresolved active-template placeholders Go package layering and JavaScript syntax.
- `make core-check`: core tests, vet and Go formatting. Existing tests cover terminal session immutability/review-before-commit and candidate ID permutation/finite-score invariants.
- `pnpm test` / `make test`: portable core checks plus benchmark metric, corpus-isolation, stream scoring and driver-conformance tests. Synthetic data is not inference evidence.
- `make e2e`: actual production native window lifecycle/render/resize/close; requires an interactive macOS/Windows session.
- CI native jobs: compile/test/vet, explicitly run native E2E and package the shell on macOS/Windows. They do not establish microphone/IME behavior.
- `make e2e-scenarios`: 16 owned product scenarios through a reviewed live OS driver; current product adapters are not implemented. See [E2E](e2e.md).
- `make ablation-plan` / `make stream-plan`: reproducible configurations without inference; live runs require corpus/model hashes and drivers. See [tooling](../../tools/evaluation/README.md).

## Required implementation evidence

Core reducers need cancellation, stale generation/revision, audio discontinuity and uncertain insertion tests. Adapter integration tests use synthetic content and actual OS permission/host ranges. IMK/TSF registration/deactivation and UTF-16/Unicode ownership tests require the target OS, not mocks alone.

ASR/rank/correction selection follows [evaluation](../research/evaluation.md), immutable artifacts and [model gates](../../specs/006-models.md). Model benchmarks are explicit opt-in tasks, not an unbounded dependency download in CI. Do not fake production inference for a passing test.

Sync tests cover atomic outbox, operation deduplication, concurrent sibling revisions, tombstone retention, cursor expiry, revoked devices and tampered encryption metadata. Storage/log privacy tests inject synthetic canaries. Hardware, fixture consent/license and observed limits accompany results.

Every new feature cites the stable acceptance IDs in [specs](../../specs/README.md). Write meaningful invariant/boundary tests, not tests that repeat the implementation or chase coverage percentages.
