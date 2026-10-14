# Validation

What was actually run, on which host, and what remains unverified. Host: Apple M4, 32 GB, macOS 26.6.2 (25G83), arm64, Go 1.27.1, sherpa-onnx-go 1.13.8.

## Automated checks (macOS arm64)

| Check | Result |
| --- | --- |
| `pnpm check` (links, JSON, layering, JS syntax) | Pass |
| Core and catalog tests (`GOWORK=off go test -race`) | Pass: session aggregate, correction rules, model selection, coordinator (commit once, cancel during finalization, stale revisions, overflow, silence, secure/untrusted target, missing model, unknown outcome not retried, mode switch, IME composition routing) |
| Desktop tests (`CGO_ENABLED=1 go test ./...`) | Pass: headless view tests for overlay, menu bar panel, all settings sections and the onboarding flow; provisioning (corrupt and tampered artifacts rejected, failed replacement keeps the prior install, resume, size bound, offline refusal); encrypted vocabulary; replay audio; catalog-to-selection mapping |
| Evaluation tooling tests | Pass |
| Native window E2E (`make e2e`) | Pass: overlay explicit failure without taking focus, settings and onboarding window lifecycle |
| Windows amd64 cross-build with cgo (MinGW-w64 14) | `go vet` clean; `Arion.exe` and the native E2E test binary link against `sherpa-onnx-c-api.dll` |

## Overlay in another application's fullscreen Space (DSK-A13)

On this macOS arm64 host, the new native regression test failed before the fix: MyGo's overlay collection behavior was `513` (`canJoinAllSpaces | fullScreenNone`), `IsVisible()` was true but `isOnActiveSpace` and the on-screen window-server lookup were false. With `macos.ConfigureOverlay`, behavior was `262473` (`canJoinAllSpaces | transient | ignoresCycle | fullScreenAuxiliary | canJoinAllApplications`); the production overlay was on screen and ordered above the separate fullscreen AppKit editor. Repeated show and hide/show kept the fixture frontmost with Editor A focused. `pnpm e2e:native`, `pnpm desktop:check` (tests and vet), `pnpm check` and `make desktop-build` passed. Private compositor metadata and content capture are under `.workspace/e2e/darwin/overlay-fullscreen.*`. This verifies native fullscreen Space membership and focus on macOS 26.6.2; older macOS versions, Stage Manager, Split View, arbitrary application window levels and rendered GPU pixels have not been separately tested.

## Edit learning (ADR 014, macOS arm64)

Core tests with `-race` and desktop tests passed: extraction, memory rules, tracker settlement, hung host, coordinator follow gates, encrypted store faults, concurrent writers and the learned notice in both languages. `go vet` was clean. The desktop app cross-built for windows/amd64 with MinGW-w64. [edit-learning-v1](../../benchmarks/results/edit-learning-v1/README.md) records quality, latency and fault results. `TestLearnFromEditInOwnedEditor` compiles but was not run, because the test runner lacks Accessibility on this host; the native follow/read path is unverified until it runs.

## Interface localization (macOS arm64)

Pinned pnpm 10.33.0 repository checks, portable tests and desktop tests/vet passed with the English/Simplified Chinese catalogs. Headless tests exercised the picker in both directions, every Chinese settings section, panel/overlay states, download formatting and existing error retranslation. Native E2E passed the existing lifecycle checks and DSK-A09: open settings/onboarding windows updated their native titles without changing recognition preferences. Rendered frames are private under `.workspace/i18n-snapshots/` and `.workspace/e2e/darwin/`. `make desktop-build` also passed; the packaged app contains both generated permission localizations, with valid `.strings` syntax and matching catalog contents. No real permission state was reset. This change has not been run on Windows. Framework editing menu labels remain English as documented in [i18n boundaries](../frontend/i18n.md#native-platform-boundaries).

## Windows (CI, GitHub-hosted windows-2022)

CI run [38343240111](https://github.com/stvlynn/Arion/actions/runs/38343240111) executed the native E2E suite on an interactive Windows Server 2022 desktop and packaged `Arion.exe`:

| Test | Result |
| --- | --- |
| Overlay explicit failure without taking focus | Pass |
| Settings and onboarding window lifecycle (open, draw nonblank, resize, close once, release) | Pass |
| Production `SendInput` route types "Arion 中文 test" into an owned Notepad exactly once | Pass |
| Focus moved to a second Notepad before commit: `not_applied/TARGET_CHANGED`, no text in either | Pass |

## Model selection run (`models-dev-2`)

Real-time replay of 118 dev fixtures per bundle through the production pipeline (coordinator, sherpa-onnx streaming recognizer, finalizer; correction off), warm, 4 threads, AC power. Report: [`benchmarks/results/models-dev-2`](../../benchmarks/results/models-dev-2/summary.json). Rates are corpus micro-averages; latency is stop (end of audio) to review-ready.

| Bundle | Mandarin CER | English WER | Mixed MER | Stop→review p95 | RTF |
| --- | ---: | ---: | ---: | ---: | ---: |
| **X-ASR 480 ms + X-ASR final pass (default)** | 0.071 | 0.037 | 0.253 | 157 ms | 0.10 |
| X-ASR 160 ms + X-ASR final pass | 0.071 | 0.037 | 0.253 | 190 ms | 0.19 |
| X-ASR 160 ms + SenseVoice final pass | 0.072 | 0.131 | 0.380 | 248 ms | 0.20 |
| X-ASR streaming 480 ms | 0.126 | 0.068 | 0.337 | 44 ms | 0.08 |
| X-ASR streaming 160 ms | 0.096 | 0.061 | 0.359 | 1,019 ms | 0.31 |
| Zipformer bilingual (2023) | 0.375 | 0.107 | 1.283 | 51 ms | 0.09 |

Limits: one seed and the warm stratum only; 48 Mandarin/40 English/30 synthetic mixed fixtures (the mixed set is macOS-voice TTS, and its references write numbers as words, which penalizes digit output); speaker-cluster CIs are exploratory. The X-ASR streaming 160 ms block overlapped developer test builds on the same machine, which inflated its tail latency; the default's block ran on an idle machine. SenseVoice emitted text on one of three noise fixtures; the default emitted none. The two X-ASR final-pass bundles produce identical final text, so the latency tie-break chose the 480 ms preview. First-partial latency (≈1.6 s p95 from audio start, including leading silence) is not a speech-onset measurement.

## Product scenarios (`make e2e-macos`)

All eight scenarios in [`macos-shortcut-scenarios.json`](../../tests/e2e/macos-shortcut-scenarios.json) passed against the packaged `Arion.app` rebuilt from commit `585488d` (binary SHA-256 `fc74ffac…`, run `run-2026-10-13T06-45-28-709Z`), with Accessibility re-granted by a person for that build and microphone consent from onboarding, an isolated profile, `ARION_OFFLINE=1` and the AppKit editor fixture:

| Scenario | Observed |
| --- | --- |
| shortcut-offline-once | Editor A = "Hello world.", one commit, `applied`, 0 sockets |
| mixed-language-dictation | Editor B = "请把这个 PR merge 到 main 分支。", one commit |
| cancel-stale-callback | Late final result after cancel: no text, no commit |
| focus-change-blocks-commit | Focus moved A→B before commit: no text anywhere, `not_applied/TARGET_CHANGED` |
| secure-field-blocked | `SECURE_FIELD_BLOCKED`, microphone never started |
| uncertain-commit-no-retry | Lost acknowledgement: outcome `unknown`, one insertion attempt, text once |
| input-mode-switch-cancels | Switch to `system_ime` during finalization cancels; capture inactive, no commit |
| silence-no-commit | `NO_SPEECH`, no text |

The first-run path was also exercised on the real profile: onboarding completed, and the app downloaded, verified and installed the two archives of the default bundle (only listed members, archives removed). An earlier run (binary `e28ce58a…`) predated the hold-to-talk and bilingual-interface changes and passed the same eight scenarios.

## Hold-to-talk (DSK-02, DSK-A10)

| Check | Where | Result |
| --- | --- | --- |
| Gesture rules: hold stops on release, tap latches hands-free, chord cancels, busy ignores presses, preference normalization | `packages/core/application/hold_test.go` | Pass |
| Raw event interpretation: fn/right Option/right Command device bits, auto-repeat, AltGr's synthetic left Ctrl | `infrastructure/holdkey` (any OS) | Pass |
| Settings picker, missing Input Monitoring explanation and grant action, fn/🌐 hint, pill "release to finish" | `ui/app` headless view test | Pass |
| Low-level keyboard hook on a real desktop; hold key through the production app starts a session and release ends it | `e2e/windows_hold_test.go` (Windows CI run [38358017734](https://github.com/stvlynn/Arion/actions/runs/38358017734)) | Pass |

The macOS event tap needs Input Monitoring consent, which CI cannot grant to a test binary; the tap was compiled and vetted but not exercised on a desktop.

## Consent guide (DSK-11)

| Check | Where | Result |
| --- | --- | --- |
| Guide card in English, Simplified Chinese and dark appearance; nothing drawn without a pane; close button calls `CloseGuide` | `ui/app` headless view test | Pass |
| Placement under the pane beside the sidebar, minimum width, clamping at display edges and on a secondary display | `bootstrap/guide_test.go` | Pass |

DSK-A11 needs a packaged `Arion.app` and a person: the drop of Arion's file URL into the System Settings list, hiding and reappearing on app switches, and closing on grant or quit have not been exercised.

## Live rewriting (LRW-A01–A05, macOS arm64, Apple M4)

- Core: `go test -race ./packages/core/...` covers previews during capture, latest-wins and stale-preview suppression, cancel during the final rewrite, explicit `unavailable`/`timeout`/rejection outcomes, whole-utterance versus per-segment finalization and the pause cut; `TestValidateRewrite` covers each validation verdict.
- Worker: client tests run the real protocol against a fake worker (cancel, crash); `TestWorkerDependencies` proves no speech-runtime import; `TestReuseParity` passed against the provisioned Qwen3 1.7B model (KV reuse output equals a cold run).
- Packaging: `make desktop-build` staged the pinned GenAI 0.17.0 / ONNX Runtime 1.30.0 libraries and `arion-llm` into `Contents/Resources/llm`; `codesign --verify --deep --strict` passed (ad-hoc signature); the packaged worker loaded the model and rewrote held-out cases.
- UI: `TestPolishRowStates` and `TestCopyOriginal` render the polish row (download size, progress, failure retry, ready, unsupported in Chinese) and the "Copy original" action; `TestNoTechnicalModelNames` renders Settings › Model and the onboarding model step in both languages and rejects engine or model-family names.
- Downloads: `TestMirrorFallbackIsSilentAndRemembered` and `TestMirrorBytesStillVerified` cover the silent Hugging Face mirror; a proxy-only network was verified to download through the macOS system proxy. The mirror itself redirected this Japan-exit host back to huggingface.co, so mirror-served bytes were not observed from a mainland network.
- Benchmark: [live-rewrite-v1](../../benchmarks/results/live-rewrite-v1/README.md) meets the LRW-A05 development gate. The running app was not exercised end to end with a microphone; rendered-preview timing and IME composition with previews remain unmeasured.

## Not verified

- The live microphone path with real speech (capture → recognition → insertion) was not exercised by automation; scenarios replay fixture audio into the production pipeline.
- On Windows, microphone capture, recognition, clipboard delivery, DPAPI, tray and the global shortcut have not run; only the tests above did. No Windows scenario replays audio through the packaged app.
- IME-A01 (install/select/remove on a real input-source list) and host-edit scenarios IME-A04/A05 need a person or a dedicated test profile; IMK composition was not exercised in a host application.
- Microphone denial (DSK-A01 with a denied consent database) needs a test profile; it is covered by unit tests and the native E2E's untrusted-Accessibility path only.
- Streaming latency claims with annotated speech onsets (STR-A01–A04) need reviewed timestamp labels; first-partial values above are measured from audio start, not speech onset.
- The macOS hold-key event tap with real key presses (needs Input Monitoring granted to the app by a person).
- The consent guide against real System Settings (DSK-A11), including on macOS 27, where the Accessibility pane is renamed but keeps its URL.
- No Developer ID signing or notarization. The updater is covered by state and schedule unit tests and a locally signed archive; installing a published release has not run, since none is published.
