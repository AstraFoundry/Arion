# Testing

Research commands qualified with `/path/to/Arion-evaluation` run in the private [Arion evaluation repository](https://github.com/AstraFoundry/Arion-evaluation). Product checks run in this public checkout.

## Current checks

- `pnpm check`: local docs links, configuration JSON, placeholders, Go package layering, JavaScript syntax, and documentation consistency (documented commands exist, ADR numbers are unique and indexed, no unlinked docs, translated READMEs link what `README.md` links). The rules are tested in `scripts/lib/doc-consistency.test.mjs`.
- `make core-check`: portable core and catalog tests, vet and formatting. Coordinator tests run under `-race`: exactly-once commit, cancellation during finalization, stale revisions, queue overflow, silence, secure/untrusted targets, missing model, unknown-outcome no-retry, mode switching and IME composition routing.
- `make desktop-check`: desktop tests and vet with cgo. Tests execute with `-count=1` even with warm build caches. On macOS this command also runs the complete keyboard checks once. CI splits this into `node scripts/check-go.mjs desktop --skip-keyboard` and `node scripts/check-keyboard.mjs` on separate runners; the local default still runs both: headless view tests for every window, encrypted vocabulary (ciphertext on disk, wrong key and tampering fail closed), replay audio.
- `make e2e`: production windows on the real OS loop (overlay explicit failure without focus, settings and onboarding lifecycle).
- `make e2e-macos`: eight owned product scenarios against the packaged app and an AppKit editor fixture; requires Accessibility access for `Arion.app`.
- `make -C /path/to/Arion-evaluation bench-run CONFIG=benchmarks/configs/desktop-models.json …`: real-time model comparison through the production pipeline. The private `scripts/export-catalog.mjs` verifies a completed raw run and hashes, then exports content-free aggregates with exact source identities. After review, public `node scripts/catalog-evaluation.mjs <reviewed-summary.json> --reviewed` imports that JSON into `models/evaluations/<runId>.json` and updates catalog report links; it does not read a private raw-run directory.
- Windows: `CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go vet ./...` and `go build` compile the app and native E2E test binary; execution needs Windows.

Go formatting commands use paths relative to each module so longer Windows worktree roots do not exhaust the OS command-line length limit.

Command references use Arion's Makefile and package scripts only for project-authored documentation. Frozen benchmark evidence retains its historical command context. Imported upstream Markdown listed as local source files in `scripts/keyboard-runtime.json` retains its upstream commands only while its bytes match the pinned SHA-256; changed notices fail the check. Link, JSON and project-command checks remain enabled.

## Microphone device reuse

From `apps/desktop`, run `go test -race -count=1 ./infrastructure/audio ./infrastructure/diagnostics` and `go test github.com/gen2brain/malgo -run TestResetCaptureIsolation -count=1`. The lifecycle tests exercise the production ownership and reuse orchestration with injected hardware: cold versus warm initialization counts, fresh frame identities/offsets, no idle delivery, cancellation during open/resume, endpoint changes, active/idle invalidation, native failures without retry and close with late lease cleanup. The dependency's C harness verifies actual miniaudio input-buffer and resampler isolation across sample rates without opening hardware.

These checks do not establish macOS microphone readiness or indicator behavior. On a Mac, compare cold and repeated warm starts with the built-in mic and an external device. Verify the system microphone indicator turns off after stop, permission denial prevents capture, changing the system default or configured microphone uses the new endpoint, unplug/sleep/lock ends capture, a subsequent session has no prior PCM or filter tail, and quit releases all devices. Compare `audio_device_init`, `audio_device_reuse`, `audio_device_start` and total `audio_start`; cancelled attempts must be reported separately from successful readiness. A warm improvement cannot be reported as eliminating cold CoreAudio stalls. See [logging](../backend/logging.md).

## Windows candidate-page regression

The native candidate-window fixture exercises the default page bound in [KBD-03](../../specs/007-ime-ranking.md) and an explicitly sized nine-candidate row. The TSF owned-editor fixture exposes `ITfCandidateListUIElement.GetPageIndex` results; selection and expanded-row tests use those actual page starts instead of assuming a fixed page size. It verifies a new profile and an explicit saved nine-candidate preference separately. These checks run under `pnpm keyboard:windows:check` on Windows.

## Windows pipe identity fixture

The pipe peer-proof test runs directly under an ordinary medium-integrity token. From an elevated development shell, it first verifies that production rejects the elevated token, then launches the same test executable with a restricted, non-elevated medium-integrity token. The child performs the real named-pipe connection, kernel PID/SID/session checks and held-process lifetime checks. Token setup or child execution failure fails the test. No account, UAC setting or desktop ACL is changed. Low-integrity refusal remains a separate test. This fixture does not replace the standard-user profile and DPAPI acceptance in the Windows keyboard CI launcher.

## Context-aware editing checks

Run the prompt, shared pipeline, GGUF transport and cloud adapter packages after changes to reference delivery. Cover all cursor/window/history switch combinations through both gateway formats, separate current source from escaped references, omit unavailable sources, retain selector protocol parity and reject echoed context wrappers before punctuation conversion. These synthetic checks establish payload and orchestration behavior; actual model fidelity and latency need supported-host inference review under [ADR 043](../decisions/043-context-aware-polishing.md).

## Required implementation evidence

Core reducers need cancellation, stale generation/revision, audio discontinuity and uncertain insertion tests. Adapter integration tests use synthetic content and actual OS permission/host ranges. IMK/TSF registration/deactivation and UTF-16/Unicode ownership tests require the target OS, not mocks alone.

ASR/rank/correction selection follows [evaluation](../research/evaluation.md), immutable artifacts and [model gates](../../specs/006-models.md). Model benchmarks are explicit opt-in tasks, not an unbounded dependency download in CI. Do not fake production inference for a passing test.

Sync tests cover atomic outbox, operation deduplication, concurrent sibling revisions, tombstone retention, cursor expiry, revoked devices and tampered encryption metadata. Storage/log privacy tests inject synthetic canaries. Hardware, fixture consent/license and observed limits accompany results.

Every new feature cites the stable acceptance IDs in [specs](../../specs/README.md). Write meaningful invariant/boundary tests, not tests that repeat the implementation or chase coverage percentages.

Keyboard selector regression checks cover RNK-A06 at the domain, scheduler and completion adapter boundaries. Public tests retain product span, identity, cancellation and label-isolation invariants. Complete study scoring and historical bundle integrity/cleanup tests run in the private evaluation repository. The [v2 replay](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/reproduction/llm-ime-alignment-v2/README.md`) reconstructs original requests and scores without loading a model; it does not establish current native dispatch or model quality.

Opt-in keyboard neural ranking (ADR 031, RNK-A07) requires bridge revision/activation and lifecycle regressions, local worker speech-priority checks, persisted-default-off settings and both UI catalogs. Run `go test -race ./infrastructure/imebridge ./infrastructure/rewriting ./infrastructure/settings ./bootstrap` in `apps/desktop`, `go test ./ui/app -run TestKeyboardNeuralSettings`, `node scripts/check-shared-keyboard.mjs`, and `node scripts/test-keyboard-input.mjs`. The optional real-worker command and artifact identities are in [runtime smoke evidence](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/runs/keyboard-neural-runtime-v1/README.md`). Native protocol fixtures use supplied replies and cannot establish model quality. Windows still requires `pnpm keyboard:windows:check` on its actual OS; installed editor acceptance remains separate.
