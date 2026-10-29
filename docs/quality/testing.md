# Testing

## Current checks

- `pnpm check`: local docs links, configuration JSON, placeholders, Go package layering, JavaScript syntax, and documentation consistency (documented commands exist, ADR numbers are unique and indexed, no unlinked docs, translated READMEs link what `README.md` links). The rules are tested in `scripts/lib/doc-consistency.test.mjs`.
- `make core-check`: portable core and catalog tests, vet and formatting. Coordinator tests run under `-race`: exactly-once commit, cancellation during finalization, stale revisions, queue overflow, silence, secure/untrusted targets, missing model, unknown-outcome no-retry, mode switching and IME composition routing.
- `make desktop-check`: desktop tests and vet with cgo: headless view tests for every window, encrypted vocabulary (ciphertext on disk, wrong key and tampering fail closed), replay audio.
- `make e2e`: production windows on the real OS loop (overlay explicit failure without focus, settings and onboarding lifecycle).
- `make e2e-macos`: eight owned product scenarios against the packaged app and an AppKit editor fixture; requires Accessibility access for `Arion.app`.
- `make bench-run CONFIG=benchmarks/configs/desktop-models.json …`: real-time model comparison through the production pipeline; `scripts/catalog-evaluation.mjs` records completed reports in the catalog.
- Windows: `CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go vet ./...` and `go build` compile the app and native E2E test binary; execution needs Windows.

## Required implementation evidence

Core reducers need cancellation, stale generation/revision, audio discontinuity and uncertain insertion tests. Adapter integration tests use synthetic content and actual OS permission/host ranges. IMK/TSF registration/deactivation and UTF-16/Unicode ownership tests require the target OS, not mocks alone.

ASR/rank/correction selection follows [evaluation](../research/evaluation.md), immutable artifacts and [model gates](../../specs/006-models.md). Model benchmarks are explicit opt-in tasks, not an unbounded dependency download in CI. Do not fake production inference for a passing test.

Sync tests cover atomic outbox, operation deduplication, concurrent sibling revisions, tombstone retention, cursor expiry, revoked devices and tampered encryption metadata. Storage/log privacy tests inject synthetic canaries. Hardware, fixture consent/license and observed limits accompany results.

Every new feature cites the stable acceptance IDs in [specs](../../specs/README.md). Write meaningful invariant/boundary tests, not tests that repeat the implementation or chase coverage percentages.
