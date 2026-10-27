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

## Guarded shortcut delivery (macOS arm64)

`pnpm e2e:paste` passed against the production capture/commit adapter and real Command-V dispatch: AppKit selection replacement with CJK, emoji, combining marks and multiline text (one edit), WebKit textarea, and Chrome 155.0.8059.80 textarea/contenteditable (one input event each, isolated temporary browser profile). That run rejected insertion after a focus or selection change, and also for pre-post cancellation, stale identity, a duplicate target and a secure field. Clipboard restoration was checked after successful delivery. The adapter correctly keeps all posted results `unknown`; assertions use the independently observed editor text/input events. Current DSK-A03 is different: a caret move or a switch to another editable field inserts at the caret focused at commit, and no editable focus leaves the text on the system clipboard.

Race-enabled core application, macOS adapter, IME bridge, bootstrap and diagnostic tests passed. Injected transaction tests cover serialized concurrent owners, cancellable waiters, cancellation before prepare/after prepare/after stage/after post, unavailable clipboard, restoration failure and unknown-result non-retry. Native named-pasteboard tests passed for complete multi-item/binary snapshots, empty originals, another owner before stage/before post/after post and final native focus/selection/secure rejection; delivery is intercepted in those isolated tests. The actual AppKit/IMK composition regression also passed. These layers are distinguished from host E2E.

An initial owned-editor E2E failed to receive paste because its minimal AppKit application had no Edit/Paste key equivalent. Adding the standard menu fixed the fixture; production delivery did not acquire a fallback or retry. The existing fullscreen overlay, native-window lifecycle and real owned-editor edit-learning checks also passed with the updated fixture. Lark/Codex, office/terminal paste policies, older OS versions and Windows were not exercised by this change; the paste E2E does not run microphone or recognition inference. See [ADR 003](../decisions/003-insertion-routes.md) and [E2E commands](e2e.md#guarded-macos-system-paste).

## Local diagnostics and export (macOS arm64)

Core and catalog checks, desktop tests/vet, i18n and repository checks passed. Race tests cover core application plus the diagnostic recorder/export controller. Sensitive canary, corruption, rotation, destination preservation and concurrent export/close cases passed; About renders both languages and disables export while busy. `TestDiagnosticExportNativeDialog` passed in an isolated app with actual macOS save sheets: cancel made no ZIP; save produced `manifest.json` and JSONL logs with 0600 permissions; Computer Use observed Finder selecting it. Private output and UI frames are under `.workspace/e2e/diagnostics-native/` and `.workspace/diagnostics-ui/`. This verifies local export only; it does not establish Windows save/reveal behavior or competitor runtime logging policies.

## Overlay in another application's fullscreen Space (DSK-A13)

On this macOS arm64 host, the new native regression test failed before the fix: MyGo's overlay collection behavior was `513` (`canJoinAllSpaces | fullScreenNone`), `IsVisible()` was true but `isOnActiveSpace` and the on-screen window-server lookup were false. With `macos.ConfigureOverlay`, behavior was `262473` (`canJoinAllSpaces | transient | ignoresCycle | fullScreenAuxiliary | canJoinAllApplications`); the production overlay was on screen and ordered above the separate fullscreen AppKit editor. Repeated show and hide/show kept the fixture frontmost with Editor A focused. `pnpm e2e:native`, `pnpm desktop:check` (tests and vet), `pnpm check` and `make desktop-build` passed. Private compositor metadata and content capture are under `.workspace/e2e/darwin/overlay-fullscreen.*`. This verifies native fullscreen Space membership and focus on macOS 26.6.2; older macOS versions, Stage Manager, Split View, arbitrary application window levels and rendered GPU pixels have not been separately tested.

## Edit learning (ADR 014, macOS arm64)

Core tests with `-race` and desktop tests passed: extraction, memory rules, tracker settlement, hung host, coordinator follow gates, encrypted store faults, concurrent writers and the learned notice in both languages. `go vet` was clean. The desktop app cross-built for windows/amd64 with MinGW-w64. [edit-learning-v1](../../benchmarks/results/edit-learning-v1/README.md) records quality, latency and fault results. `TestLearnFromEditInOwnedEditor` now passes with the guarded system-paste adapter and existing Accessibility access: the real owned AppKit editor is followed, the user correction is learned after 15 reads with zero failures, and the secure field is not followed.

## Interface localization (macOS arm64)

Pinned pnpm 10.33.0 repository checks, portable tests and desktop tests/vet passed with the English/Simplified Chinese catalogs. Headless tests exercised the picker in both directions, every Chinese settings section, panel/overlay states, download formatting and existing error retranslation. Native E2E passed the existing lifecycle checks and DSK-A09: open settings/onboarding windows updated their native titles without changing recognition preferences. Rendered frames are private under `.workspace/i18n-snapshots/` and `.workspace/e2e/darwin/`. `make desktop-build` also passed; the packaged app contains both generated permission localizations, with valid `.strings` syntax and matching catalog contents. No real permission state was reset. This change has not been run on Windows. Framework editing menu labels remain English as documented in [i18n boundaries](../frontend/i18n.md#native-platform-boundaries).

## Windows (CI, GitHub-hosted windows-2022)

CI run [38343240111](https://github.com/stvlynn/Arion/actions/runs/38343240111) executed the native E2E suite on an interactive Windows Server 2022 desktop and packaged `Arion.exe`:

| Test | Result |
| --- | --- |
| Overlay explicit failure without taking focus | Pass |
| Settings and onboarding window lifecycle (open, draw nonblank, resize, close once, release) | Pass |
| Production `SendInput` route types "Arion 中文 test" into an owned Notepad exactly once | Pass |
| Focus moved to a second Notepad before commit: `not_applied/TARGET_CHANGED`, no text in either. This run predates current DSK-A03, which inserts into the focused editor | Pass for that build |

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
| focus-change-blocks-commit | That run: focus moved A→B before commit and neither editor changed (`not_applied/TARGET_CHANGED`). The scenario is now `focus-change-inserts-at-caret`: B receives the text once |
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

The current product runtime is [ADR 018](../decisions/018-staged-gguf-rewriting.md): Qwen3.5-2B Q4_K_M, llama.cpp, and delivery-only validation. The measurements below are the historical Qwen3 1.7B / `live-v2` ONNX gate. They are not a measurement of the staged GGUF chain. 16GB, older macOS, concurrent speech and held-out meaning review remain open.

- Core: `go test -race ./packages/core/...` covers recognized text during capture, one rewrite after release, cancel during the final rewrite, explicit `unavailable`/`timeout`/rejection outcomes, whole-utterance versus per-segment finalization and the pause cut; `TestValidateRewrite` covers each validation verdict.
- Worker: client tests run the real protocol against a fake worker (cancel, crash); `TestWorkerDependencies` proves no speech-runtime import; `TestReuseParity` passed against the provisioned Qwen3 1.7B model (KV reuse output equals a cold run). That parity check does not apply to the llama.cpp slot, which preparation does not prefill.
- Packaging: `make desktop-build` staged the pinned GenAI 0.17.0 / ONNX Runtime 1.30.0 libraries and `arion-llm` into `Contents/Resources/llm`; `codesign --verify --deep --strict` passed (ad-hoc signature); the packaged worker loaded the model and rewrote held-out cases. Current staging no longer puts GenAI libraries in the rewrite runtime.
- UI: `TestPolishRowStates` and `TestCopyOriginal` render the polish row (download size, progress, failure retry, ready, unsupported in Chinese) and the "Copy original" action; `TestNoTechnicalModelNames` renders Settings › Speech & Polish and the onboarding model step in both languages and rejects engine or model-family names.
- Downloads: `TestMirrorFallbackIsSilentAndRemembered` and `TestMirrorBytesStillVerified` cover the silent Hugging Face mirror; a proxy-only network was verified to download through the macOS system proxy. The mirror itself redirected this Japan-exit host back to huggingface.co, so mirror-served bytes were not observed from a mainland network.
- Benchmark: [live-rewrite-v1](../../benchmarks/results/live-rewrite-v1/README.md) meets the LRW-A05 development gate. The running app was not exercised end to end with a microphone; rendered-preview timing and IME composition with previews remain unmeasured.

## Not verified

- The live microphone path with real speech (capture → recognition → insertion) was not exercised by automation; scenarios replay fixture audio into the production pipeline.
- On Windows, microphone capture, recognition, clipboard delivery, DPAPI, tray and the global shortcut have not run; only the tests above did. No Windows scenario replays audio through the packaged app.
- IME-A01 and host-edit scenarios IME-A04/A05 remain open: both ad hoc and signed/notarized macOS 26.6.2 source probes failed selection. The signed probe returned enable `0` with its parent still disabled, then select `-50`; installed IMK event dispatch and external-editor composition were not reached. Isolated native engine/AppKit results are recorded below.
- Microphone denial (DSK-A01 with a denied consent database) needs a test profile; it is covered by unit tests and the native E2E's untrusted-Accessibility path only.
- Streaming latency claims with annotated speech onsets (STR-A01–A04) need reviewed timestamp labels; first-partial values above are measured from audio start, not speech onset.
- The macOS hold-key event tap with real key presses (needs Input Monitoring granted to the app by a person).
- The consent guide against real System Settings (DSK-A11), including on macOS 27, where the Accessibility pane is renamed but keeps its URL.
- A canonical macOS arm64 build completed Developer ID signing/notarization and passed Gatekeeper, but clean-machine acceptance and a published old-to-new update have not run. The updater also has state/schedule tests and a locally signed archive; the retained build did not sign an update.

## Release pipeline checks

- `pnpm check`, pinned actionlint (`pnpm workflows:check`), portable core/catalog/relay checks and all 50 evaluation/release tooling tests passed on macOS arm64. The same 50 Node tests passed in an actual Linux arm64 container with a read-only repository and no network. The previously failing diagnostic-default test validates malformed input before the macOS-only inference gate; the native inference restriction remains.
- `pnpm desktop:check` and `pnpm e2e:native` passed on macOS arm64. Permission-guide view tests use an owned temporary absolute path on each OS; the changed UI test binary also cross-compiled for Windows x64 with MinGW. The changed test has not been executed on Windows in this change set.
- `pnpm site:check` passed: complete dictionaries, types, lint, nine tests and the static export advertising macOS 14.
- `make desktop-build` passed with `si.astrafoundry.arion`, the matching IMK identifier, and an explicit macOS 14 cgo deployment target. The app/DMG contains the LLM worker, inference libraries, localized permission resources, module/runtime licenses and model provenance. The verifier checked required files, Mach-O architectures and minimum OS versions, package versions/identifiers, relocatable rpaths and the ad hoc signature. The new minimum-OS gate initially caught the host SDK producing a main executable with a 26.0 minimum; compiler and linker flags corrected that root cause.
- Release tests reject missing/mismatched signing keys, incomplete platform sets, same-size archive tampering, substituted versions/origins/platforms and size changes. Unsigned installer collection passed for the packaged macOS app. Test keys are ephemeral fixtures; no production key was generated or configured.

The updated hosted runner matrix has not run from these local changes. Intel/Windows runtime acceptance, clean-machine signed-package acceptance, a complete signed release and a published old-to-new update remain open. No GitHub release was created or changed by this validation.

## Full OCR delivery and correction word boundaries

On macOS arm64, regression tests cover full OCR beyond the former 1,200-character cutoff in both preview and final requests, independent screen/cursor gates, unsafe context exclusion, same-place history and source-separated prompt rendering. The production worker retains its explicit token-budget rejection; screen text is no longer cut to a character count.

Native Core Foundation segmentation and code-point offset conversion (including emoji before a correction) are tested. The real extraction/encrypted-store/reload path learns 公示 → 公式 and recalls it in different wording, and persists both 爆吗 → 报吗 and 播吗 → 报吗. Domain tests cover changes merged within a word, unchanged numeric/grammar exclusions and invalid/missing segmentation. Unknown single-character boundaries are deliberately skipped, including 全免 → 全棉 when the OS splits both compounds into single characters. Older stored fragments cannot be repaired without their original sentences and are left for user review.

`pnpm core:check`, `pnpm desktop:check` (tests and vet), `pnpm docs:check`, focused race tests, native segmentation/store tests and CGO-disabled research helper tests pass. These are request/extraction/persistence checks with authored text, not a measured increase in 1.7B model quality. No new screen-capture, native field-edit or Windows runtime result is claimed; prior benchmark numbers remain historical.

## Build-time center profiles

On macOS arm64, `make build desktop` produced and verified both the standard AstraFoundry App/DMG and a custom device-only profile at `https://sync.example.org/prefix`, using ad hoc signing for these checks. Verification executed the packaged app's public profile diagnostic with no center environment and matched its compiled fingerprint. Separate Go overlay tests verified both custom device and OIDC profiles and proved runtime center ENV cannot replace compiled values.

Desktop tests/vet, repository checks, workflow validation and tooling tests passed. Headless UI tests cover hidden device-mode account settings, skipped account onboarding and suppressed custom OIDC waitlist controls. Account tests reject device-mode store/browser access; keystore tests check profile-separated item names and vocabulary-key separation. Windows credential code cross-compiles, but Windows host behavior, live custom-provider sign-in, signed/notarized custom packages and real sync enrollment/transport remain unverified. Legacy unscoped account tokens are retained and ignored; users sign in again.

## Offline Pinyin keyboard and system-source boundary

The current keyboard-interaction regression passes **34/34** cases on native arm64 and x86_64 through Rosetta 2 on the same arm64 Mac. Added cases exercise standalone left/right Shift release, chord/hold/dual-Shift/focus invalidation, Shift+Space without a second toggle, direct symbols including actual Shift punctuation, composition followed by slash, English symbol pass-through, and the actual nonactivating badge window's screen clamping, zero-width caret support, replacement timer and dismissal. Commands: `node scripts/test-keyboard-input.mjs` and `node scripts/test-keyboard-input.mjs --arch x86_64 --report .workspace/tests/keyboard-input-intel.json`; reports remain local in `.workspace/tests/`. The universal unsigned IMK host builds successfully. These are engine/AppKit checks, not installed IMK dispatch, physical Intel or external-editor acceptance. Existing Chatterfly static Shift evidence informed the trigger; the badge design and lifetime are Arion choices.

The preceding native arm64 actual-engine/AppKit run passed **28/28** cases (added: the candidate bar's next-page chevron, context ranking, short pages and the expanded grid) using pinned librime 1.17.0, production keyboard/typing/encrypted-storage code, an actual `NSTextView` protocol client and isolated synthetic learning keys. It covers Pinyin forms, candidate paging/partial choice/rollback, editing, cancellation, switching, stale callbacks, Unicode neighbors, restart learning, corrupt-data rejection and a real storage-write failure that preserves an already-produced commit. Added cases verify failed mouse-candidate callbacks cannot keep ownership and owner deallocation destroys an actual engine session without client callbacks. No user Keychain or input-source access occurs in that regression runner. The [alignment report and ledger](../research/imk-typing-alignment.md) retain exact report/source hashes and failed attempts.

A completed canonical macOS arm64 build passed Developer ID signing, Accepted notarization and package Gatekeeper checks; run 05's exact copied installed nested host was also accepted as Notarized Developer ID. Explicit LaunchServices registration resolved that installed native bundle. Nevertheless, the macOS 26.6.2 parent source `si.astrafoundry.inputmethod.arion` remained disabled while child `si.astrafoundry.inputmethod.arion.pinyin` was enabled: enable returned `0`, select returned `-50`, and production installation rejected the failed post-enable state and rolled back. No actual typing was reached. The probe restored ABC, the original installed bundle hash inventory and absent keyboard key, leaving disabled cached TIS entries. No input preferences, login session or security settings were changed. Missing signing is not an established root cause or sufficient fix; source activation, external-editor/Keychain lifecycle, Intel runtime, quality and latency remain open. See the [v2 functional/system evidence archive](../../benchmarks/runs/imk-keyboard-v2/README.md).

The [v3 archive](../../benchmarks/runs/imk-keyboard-v3/README.md) retains the completed final native desktop/keyboard checks, Go IME race checks and i18n checks, plus a separate **22/22** x86_64 engine/AppKit run through Rosetta 2 on an arm64 Mac. Physical Intel was not tested. The original attempt-13 runner/source identities remain distinct from a later runner metadata patch, whose metadata-only checks did not rerun conversion cases.

The final signed build completed Developer ID signing, hardened runtime, notarization-ticket and Gatekeeper/package verification. A subsequent offline isolated launch of that signed main app observed real onboarding and menus, but automation lost access after onboarding closed into the windowless menu-bar app. Run 07 never reached the install button and did not attempt installation. It restored ABC, no own-source entries, the original bundle inventory and absent keyboard key, with no errors; all owned test processes stopped. This is an incomplete UI attempt, not evidence of an installation-button failure. System activation/external-editor typing, physical Intel, OS-kept-key lifecycle and quality/latency remain unaccepted.

The historical strict Data Protection continuation passed native storage/policy checks and 23 engine/AppKit cases on each of arm64 and Rosetta x86_64. Real owned legacy-key rejection passed and restored both stores; its signed capability attempt failed with `errSecMissingEntitlement` (`-34018`) before a matching provisioning profile was available. The provider rejects legacy keys and requires verified device-only/nonsynchronizing attributes. The [v5 archive](../../benchmarks/runs/imk-keyboard-v5/README.md) preserves that original missing-profile failure, exact sources and restored GUI-source diagnostics. The later profile-authorized lifecycle and signed build below supersede that blocker.

The [frozen 100-case development evaluation](../../benchmarks/results/imk-quality-v1/README.md) retains 600 real engine runs and evaluator-only references: exact intended commit 71%, character error rate 13.43%, first-page oracle 81%. Process-key-to-snapshot p95 is about 0.30 ms on the recorded M4 host. This excludes native rendering, OS dispatch, real Keychain access and speech contention; it is not actual key-to-preedit latency or competitor parity.

A literal-umlaut regression first failed with only the original 22 cases passing: direct `ü` was rejected by the ASCII-only key mapping and cancelled the existing composition. Mapping `ü`/`Ü` to the declared schema's `v`/`V` fixes that input boundary. That literal-umlaut native arm64 and Rosetta x86_64 continuation passed 23/23, including actual `lüse` → `绿色`, uppercase alias mapping, Option-key relinquishment and English-mode pass-through. The current universal unsigned host rebuild and structural verification pass. These tests still do not establish installed IMK dispatch or candidate-window callback attribute preservation.

The [v7 evidence archive](../../benchmarks/runs/imk-keyboard-v7/README.md) preserves the literal-umlaut failure and corrected 23-case native/translated runs, rebuilt unsigned bundle checks, the inconclusive inactive-client candidate probe and bounded Doubao UI-refresh follow-up.

The profile-authorized real-key continuation passes all six phases on macOS arm64: absent-namespace guard, production writer, fresh-process learned ranking, exact-reference deletion, missing-key rejection with unchanged ciphertext and cleanup. OS attributes establish `AfterFirstUnlockThisDeviceOnly` (`cku`), nonsynchronizable policy and the same owned reference in the Data Protection store, with absence in the legacy store. Cleanup restores both stores absent and removes the owned profile/receipt. A first successful lifecycle retained a numeric-boolean reporter mismatch; the final normalized reporter preserves its raw value separately and reports the policy verified. This is actual OS key-policy/lifecycle evidence, not physical migration/backup or installed IMK dispatch. Native plist scalar extraction and certificate-extraction argument defects found during profile preparation were corrected; 87 tooling tests pass, including actual macOS plist type checks. The [v8 archive](../../benchmarks/runs/imk-keyboard-v8/README.md) preserves the original failures, final lifecycle, tooling checks and completed Developer ID signed/notarized package build.

The [input-source onboarding investigation](../research/input-source-onboarding.md) distinguishes competitor static enable/retry/selection and manual guidance from unobserved system execution. The input-source setup repair retains a successfully registered bundle when enable returns success but macOS still reports it disabled. Installer callback/temporary-filesystem tests cover that pending result, genuine registration/enable/discovery rollback and strict restoration of an already-enabled source. Native TIS entry points dispatch GUI calls to the main thread and serialize non-UI CLI calls according to the installed Apple SDK contract; bundle copying remains on the worker. Controller race tests cover immediate progress, duplicate/removal suppression, main-queue completion, the earlier pending-only automatic settings navigation (superseded by the consent-flow correction below), failure projection and preservation of the chosen application mode. English/Chinese view tests cover pending instructions/settings action, disabled progress controls and retry after failure; these are rendered synthetic states, not actual OS activation. The current Arion Dev settings window was observed showing the original generic installation failure. The Mac locked before updated live-window inspection; no new input-source installation or selection was attempted. Native installer/controller/diagnostic race tests, view tests/vet, catalog and repository checks pass. A compile-only run of every desktop Go package also passes; it does not execute their acceptance scenarios. Historical signed packages and system experiments above predate this setup repair.

The user's subsequent [actual add-dialog visibility control](../../benchmarks/runs/imk-keyboard-v10/README.md) found no Arion result. Both the development host and the existing notarized host remained unavailable in that UI; a fresh Settings process with the notarized host also failed discovery. The temporary controls restored original development bytes and current Chatterfly without source selection or host launch. System source acceptance remains open; the earlier rollback/progress repair is not a discovery fix.

The actual Allow-prompt follow-up includes a user-provided Arion Dev authorization screenshot and a reported successful Allow click, followed by an absent added-source entry. Subsequent native queries still showed no enabled parent; grant persistence and the OS-level cause are unproven. The installed host had become ad hoc again after repeated Dev installs, replacing the previously staged Developer ID host. The corrected installer validates Developer ID signature/identity consistency before installed-path access or mutation, keeps valid copied-but-undiscovered hosts pending, and leaves native consent in place rather than immediately navigating to settings. Native payload/copy-state/controller race tests and bilingual view/i18n checks pass. A read-only positive test passes against the existing profile-bearing notarized host. This preflight does not pin the publisher or independently validate provisioning CMS authorization; release packaging retains those gates. Actual signed-app authorization, source addition and typing remain separate acceptance steps.

The current consent-flow package completes `make build desktop`: Developer ID signing, notarization, ticket/Gatekeeper checks and package verification all pass. The new nested IMK host also passes the actual runtime payload preflight through the read-only configured-artifact test. The current DMG is an installation-acceptance candidate; the user must run the signed release app rather than Arion Dev, remove the old ad hoc host if present, then install the release payload. No successful post-Allow source addition or typing is claimed.
