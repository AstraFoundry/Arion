# Windows validation

The alpha.13 release preparation found hosted-runner UI Automation traversal
timeouts despite successful owned-editor runs locally. The reader now compiles
its typed managed traversal during initialization, before READY, instead of
binding every node dynamically in PowerShell. Target access still starts only
after GO and ownership revalidation; password/offscreen filtering, node/text
limits and the 1.5-second traversal budget remain. Repeated fresh hosted runners
exceeded the previous five-second initialization limit before authorization.
Managed initialization now has a separate 15-second ceiling, remains cancellable,
and records elapsed time in its real initialization test; the final rewrite's
independent context wait is unchanged.
CI security tests use an ordinary account, while actual OS-input fixtures use
the runner's original interactive session. Neither scope establishes clean-host
or broad third-party editor compatibility.

The Windows x64 development-host validation passes native interaction, installed-application speech delivery, packaging, installation/removal and isolated updater mechanics for `0.1.0-alpha.11`. This is not broad-release acceptance: the package is unsigned, production-channel upgrades were not published or exercised, and mixed-language recognition errors remain.

## Plan and environment

The execution order was: establish pinned tools and portable checks; exercise native windows and owned editors; repair reproduced failures; build and verify the installer; install into an isolated directory; run real recognition through the installed application; uninstall and verify cleanup; test update success, authenticity rejection and failed replacement. Actual microphone and native diagnostic export were checked separately. All application profiles and editors were owned fixtures; existing user application data was not reset.

Host: Windows 11 25H2 build 26200.9445, AMD Ryzen 7 7840U, Radeon 780M, approximately 30.8 GB OS-visible physical memory. Tools: Node 24.9.0, pinned pnpm 10.33.0, Go 1.27.1, w64devkit 2.10.1 / GCC 16.2.0, existing Visual Studio BuildTools CMake. Downloaded Go/compiler archives were SHA-256 verified. Tools and model weights remain ignored under `.workspace/`.

## Executed checks

| Area | Result and evidence boundary |
| --- | --- |
| Frozen install and repository checks | `pnpm install --frozen-lockfile` and `pnpm check` pass. Windows Git symlink checkout exposed two broken documentation links; links now use the tracked canonical skills directory. |
| Portable tests | `pnpm test` passes core, models, sync/update services, i18n and evaluation tooling. Evaluation suite: 89 pass, 2 explicit platform skips. No PostgreSQL integration database was supplied. |
| Desktop tests | `pnpm desktop:check` passes tests, formatting and vet. Full UI rendering package took 181 seconds on the loaded host. A separate diagnostic run with an intentionally shorter 90-second bound timed out; the normal complete suite passed. |
| Concurrent adapter/application checks | `go test -race ./infrastructure/windows ./bootstrap` passes. |
| Native windows and target adapter | 12 tests plus 4 ownership subcases pass: native lifecycle, onboarding completion, bilingual windows, global hold hook, Unicode/emoji exact-once input, duplicate token, stale identity, cancellation before/during input, changed focus/selection, standard password controls and gated UIA context. UIA extraction completed in 0.70 seconds. |
| Real microphone | Two one-second production capture cycles each delivered 49 frames / 15,680 samples at 16 kHz mono, with correct identity, sequence and offsets. Stop produced no late callbacks and reopening succeeded. Only counters were retained. |
| Real CPU and Vulkan recognition | Each backend completed six offline utterances, three realtime 20 ms streaming cases, cancellation and worker reuse. Original outputs, hashes and reproduction source are in the [speech evidence archive](../../benchmarks/reproduction/windows-native-asr-v1/README.md). |
| Installed-product E2E | Five scenarios pass in 27.30 seconds: actual recognition into owned editor exactly once, cancellation after a real final reached the delay gate, focus-change rejection, silence with no commit, and password-field refusal before capture. Microphone audio is replaced by hash-bound corpus audio; recognition, coordinator, worker and SendInput are production code. |
| Package | `node scripts/build.mjs desktop` and resource/profile/license/PE verification pass. Installer archive integrity passes; all 31 actually installed payload files match the build hashes. Both packaged speech workers execute. |
| Actual installation/removal | Silent per-user Arion installation into an isolated workspace directory passes registry, protocol association, Start menu and uninstaller checks. Five speech scenarios run from that installation. Actual uninstall removes the installation, registry entries and shortcuts, preserving external profile sentinel and model data. Preflight confirmed every touched product registration/shortcut was absent. |
| Updates | Full/delta updates, signature/size rejection, channel/version policy, transient locks, permanent-lock rollback, replacement of a running PE executable, relaunch and `.old` cleanup pass with isolated test keys/loopback sources. Six release-manifest tests pass. No production update channel was changed. |
| Diagnostic export | Actual Windows save dialog cancel then export passes; the ZIP contains manifest and JSONL logs. The first Computer Use state read failed and the unanswered test timed out; the retry completed successfully. Export remained local. |

The rows for changed focus/selection and focus-change rejection, and the sentence "Selection ownership and secure-field recognition are verified", record a run that predates current DSK-A03. Their observed passes stand for that run.

Private execution logs are under `.workspace/windows-validation/`; in particular `desktop-check-final.log`, `test-after.log`, `native-target-final.log`, `microphone-lifecycle.log`, `race.log`, `diagnostics/retry.log` and `release/RESULTS.md`. Installed-product traces are also hash-bound in the durable speech archive. Native Content screenshots under `.workspace/e2e/win32/` are render evidence, not a complete OS compositor or editor compatibility audit.

## Repairs

- The alpha.11 repair made Windows commits validate session identity and cancellation, reject secure standard EDIT controls, and revalidate captured standard EDIT selection. In that repair, input batches stop on cancellation or changed focus. Already-posted input remains an uncertain outcome instead of claiming nothing was inserted. Current rule: commit inserts at the live caret; a focus or selection change does not abort that commit; no editable focus leaves the text on the clipboard as `copied`; a secure field is refused and is not copied; input already posted stays `unknown` and is not retried.
- Context reads fail closed when field safety is unknown. Explicit reflection-based registration repairs the .NET UI Automation provider loader under PowerShell; visible text now comes from real providers, with password/offscreen exclusion. Cancellation retains its own error code.
- Update rollback removes newly added files/directories when a later locked destination prevents replacement, while restoring originals. A real Windows file-lock regression failed before the repair and passed afterward.
- Portable test cleanup waits for owned child processes to exit before removing their Windows working directories. Signing fixture assertions use platform path handling.
- Native tests use owned Win32 controls instead of relying on modern Notepad process/window behavior. The packaged speech test exposes only a test-control pending-final observation so cancellation really occurs after recognition, rather than merely before it.

## Hosted Windows context initialization

The [first alpha.12 main CI run](https://github.com/AstraFoundry/Arion/actions/runs/40050619822) passed Windows desktop and updater tests but timed out in the owned-window UI Automation test: the original 1.5-second traversal budget also covered PowerShell/provider initialization. The reader now allows up to 15 seconds for initialization without target access, revalidates the target, then grants the unchanged 1.5-second traversal budget through a READY/GO handshake. Caller cancellation applies throughout; final rewrite context waiting remains 1.5 seconds. Real subprocess tests cover both deadlines, cancellation, authorization and rejection of incomplete output. The actual local UIA read passes with a 740.48 ms total capture; hosted-runner success is checked again before tagging.

Two full local E2E reruns after this repair passed UIA (740.48 ms and 672.14 ms) and the remaining target/hold/input cases, but both failed the unchanged onboarding-to-settings focus assertion. Those runs are not full-suite passes; their logs remain in `.workspace/windows-validation/release/native-uia-phases.log` and `native-uia-phases-repeat.log`. That onboarding case passed in the first hosted run above and remains enabled in subsequent CI.

## MSVC release runtime validation

The CI-style CPU worker was additionally built with Visual Studio 2022 / MSVC 19.44.35228.0, x64, static `/MT`, and OpenMP disabled. A minimal `/MT` + OpenMP executable still imported `VCOMP140.DLL`; the new PE gate rejects that unbundled dependency. The full pinned CPU worker imports only `ws2_32.dll`, `shell32.dll`, `kernel32.dll` and `advapi32.dll`, and its `--version` succeeds with only Windows system directories on `PATH`. Staging and package verification now check normal/delay-loaded import closure and this CPU launch; four portable dependency-gate tests pass. Vulkan's optional loader remains supplied by the GPU driver.

This run exposed a separate cancellation lifecycle defect: an otherwise healthy MSVC worker acknowledges the long diagnostic cancellation after approximately 5.6–5.9 seconds, so the adapter's existing five-second watchdog retired it and left later requests using a dead process. Recovery now retires/reaps that generation after the same drain budget and loads the same immutable worker/specification only for a later explicit request; the cancelled request is never retried. Arbitrary crashes and real reload failures remain explicit failures, while caller cancellation during reload does not poison later requests. The final source-bound real MSVC run passes six offline utterances, cancellation followed by a new request, and three realtime streaming cases. Cancellation took 5,183.821 ms; the next request succeeded, including replacement loading, in 6,717.508 ms. These diagnostic timings do not establish latency budgets or recognition quality. The [durable archive](../../benchmarks/reproduction/windows-native-asr-v1/README.md#msvc-release-runtime-validation) retains the original failure, raw protocol observations, intermediate attempt, final outputs and exact source/runtime hashes.

## Retained development artifacts

The retained preview installer is `apps/desktop/build/windows-amd64/Arion Setup 0.1.0-alpha.11 amd64.exe` (15,298,202 bytes), SHA-256 `fe2256cf7738d3d83a4c24d879efe767930cadc3b19149a777a5f57eaf4231b1`. Its application is 23,900,672 bytes, SHA-256 `c822189b8acec48b403dc9f655cca6f09a029257f021a2a54065dee1068089af`. A second retained copy is under `.workspace/windows-validation/release/final-collected/`.

## Remaining acceptance gates

- Publisher signing, a production-key signed update artifact, real published old-to-new channel upgrade, and clean/minimum-supported Windows host testing remain open. Test-key update acceptance does not establish production trust configuration.
- The mixed-language development fixtures misrecognize technical terms, including `standup` and `PR / merge / main / CI`; other Mandarin/English errors are preserved. Runtime success does not satisfy speech quality or latency budgets.
- Physical hotkey-to-microphone-to-insertion with a person speaking, unplug/suspend/resume, OS permission denial and a representative third-party editor matrix remain unverified. Microphone capture and fixture-audio speech delivery were tested separately. The packaged test enables the offline guard but does not independently monitor every process socket.
- Selection ownership and secure-field recognition are verified for standard EDIT controls. This run predates current DSK-A03. Browser/Electron/custom controls and all-format clipboard contention/restoration remain outside this run. The default tested delivery is Unicode SendInput.
- System TSF installation/selection and editor acceptance, scene/site/cursor extraction, edit learning and desktop sync enrollment/outbox remain unaccepted or unsupported as documented; subsequent native TSF primitives and local polishing are recorded below.

Reproduction commands and opt-in environment variables are in [native E2E](e2e.md). For this tool setup, place `C:\Windows\System32` before w64devkit in PATH: its GNU tar cannot extract the pinned Vulkan ZIP, while Windows bsdtar can. The original failed build is retained; no archive pin or native requirement was bypassed.

## Native TSF voice primitives

The unreleased source now ships a separate x64 COM service with TSF edit locks, build/session/revision-bound named pipes, explicit UAC installation controls and Windows voice-only Keyboard settings. This does not change the historical published-package claims above. The [input-method runbook](../operations/windows-input-method.md) covers setup, supported hosts and security boundaries.

Actual native RichEdit (`SES_USECTF`) checks pass Unicode/emoji/combining-mark draft and exactly-once final commit, selected-text cancellation/restoration, selection/focus invalidation and stale write rejection, password refusal and read-only refusal. A separate real `ITextStoreACP` editor repeats those cases and verifies user replacement winning over late writes, owner refusal returning a null composition, selection changes before deferred write-lock acquisition, and cancellation while a native acquisition is deferred. The final ACP matrix passes in 1.19 seconds. Tests use the production DLL, COM factory, native text stores and real current-user pipe; no machine profile or existing document is modified. The original failures exposed missing selected-text restoration and platform termination cleanup; they remain private diagnostic evidence, not passing runs.

The native transport regression separately passes same-user/session/integrity checks, low-integrity rejection, a bounded stalled-server write and cancellation/join of pending I/O. `go test ./infrastructure/imebridge ./infrastructure/tsf ./bootstrap` passes outside the restricted sandbox. The sandbox's low integrity and filesystem policy cause expected access failures and are not ordinary-desktop acceptance.

Global UAC registration/removal, actual Win+Space selection, remembered per-app source behavior, installed-package speech and third-party editor compatibility remain unverified until user-approved execution. Elevated/high-integrity, low-integrity/AppContainer, 32-bit and transitory contexts are unsupported; Windows Pinyin and live DLL replacement are not implemented. Reproduction commands and complete native limits are in the [host README](../../apps/desktop/ime/windows/README.md).

## Overlay lifecycle regression

Installed-package Computer Use validation caught a Windows voice-settings gate
comparing the architecture-qualified runtime platform (`windows-amd64`) with
`windows`. The gate now recognizes the Windows platform prefix, and both locale
regressions use the actual runtime identifier. The initial installed EXE and its
TSF/LLM payloads matched the build hashes. The corrected installer was then
installed through its real NSIS wizard; the installed application hash matches
the rebuilt executable. Computer Use confirms the final Windows page shows
Arion Voice, UAC guidance and no Pinyin candidate controls. The installed app
also downloaded, verified and loaded the local polishing model, reaching its
local-ready UI and diagnostic state. This proves provisioning/loading, not
end-to-end editor delivery. UAC installation subsequently produced an actual
registered profile with the bundled DLL hash. The installed app reported a
failure before its next status poll discovered that profile; this remains an
installation-status defect, not a fully accepted install transaction. A native
read-only probe also found `IsEnabledLanguageProfile` reporting enabled while
the enumerated profile flags remained zero, explaining the disabled-state
misreport. After correcting the query and rebuilding, the final installer was
installed through Computer Use. Installed app/helper/DLL hashes match the build;
the helper reports registered=true, enabled=true, selected=false, and the actual
Keyboard page shows Arion Voice enabled without an error. Actual source
switching and editor delivery remain unverified; the observed active system
language was Japanese (`0x0411`). Post-install discovery now has bounded unit
coverage, but a fresh UAC registration transaction has not been repeated.

The user's subsequent real Win+Space screenshot disproves switcher acceptance:
only Microsoft Pinyin and Microsoft Japanese IME appear. Read-only inspection
of the user language list confirms only `zh-Hans-CN` and `ja`, with no Arion TIP.
Registered/enabled TSF COM state must not be reported as switcher membership.
Microsoft's [IME installation requirements](https://learn.microsoft.com/en-us/windows/apps/develop/input/input-method-editor-requirements#ime-installation)
explicitly distinguish `RegisterProfile` from `InstallLayoutOrTip`, which adds
the input method to the user's enabled list. The initial implementation omitted
the latter step. A corrected helper uses the documented call under the ordinary
user, with only Arion's exact `0x0409:{CLSID}{profile}` and flags zero. The actual
call succeeded and the user language list now includes `en-US` with the exact
Arion TIP. Before/after comparison confirms the complete existing Chinese and
Japanese language objects are unchanged. No category or DLL modification, UAC,
default-source override or language-pack download was needed for this list
repair. Actual switcher visibility and editor delivery remain separate gates.
The canonical package including the user-list fix subsequently passed all
resource/hash/PE/runtime checks and was installed through Computer Use. Its
installed application, helper and unchanged DLL match the built hashes; final
query reports registered=true, enabled=true, selected=false. The corrected
removal path's exact-TIP uninstall flag is contract-tested, not exercised on
the user's system.

An installed-runtime repeat passes `TestNativeWindowsLifecycle` using the
installed worker and app-downloaded model; original evidence and hashes are
archived in [the installed rewrite repeat](../../benchmarks/reproduction/windows-installed-local-rewrite-v1/README.md).
This is runtime acceptance, not microphone-to-editor acceptance. Reopening
Windows Settings still showed only Chinese and Japanese despite the API user
list containing Arion. A native probe then confirmed that the service lacked
`GUID_TFCAT_TIPCAP_SYSTRAYSUPPORT`. The helper now registers that category along
with the keyboard category and can repair an existing identical DLL registration.
The actual UAC repair exited successfully; a fresh probe confirms both categories,
`dwCaps=0x20000`, and the unchanged DLL hash. Chinese and Japanese user-list
entries remain unchanged. This confirms the registration repair only: visible
Win+Space selection and real editor delivery still require independent acceptance.
The canonical installer rebuild with the category repair passes packaged TSF
hash/architecture checks, ASR and GGUF isolated CPU launch, resource validation,
and repository checks. This newer installer has not been installed or published;
the running system received only the targeted category repair above.
The user's post-repair switcher screenshot still omits Arion: it shows Microsoft
Pinyin, Microsoft Japanese IME and English (United States) / US keyboard.
Therefore the category repair did not establish switcher acceptance. A fresh
registered-COM probe successfully creates the installed service through its
registered class, ruling out a basic x64 COM loading failure. Running the helper's
selection command returns zero while a separate query still reports unselected;
command success must not be treated as observed activation.

The status overlay is not user-closable. If its native window is destroyed, the next dictation session creates and positions a new click-through overlay. `TestOverlayRestoredAfterDestroy` exercises actual destruction and verifies a distinct visible, nonfocused, nonclosable replacement on Windows (passed). This lifecycle test is separate from compositor transparency verification.

`TestWindowsOverlayWindowPolicy` also passes against the native app window: it requires nonactivating, tool-window, click-through, layered and no-redirection styles and rejects the app-window style. The overlay therefore has no separate Alt+Tab entry. Framework desktop-pixel tests verify transparent corners and opaque content through software rendering, compositor recreation and GPU rendering; they also verify click-through hit testing and that pointer handling cannot steal keyboard focus from the target editor. These observe the Windows compositor rather than relying on an offscreen screenshot that can retain alpha despite incorrect native presentation.

## Shared Pinyin acceptance in progress

The worktree now shares conversion, candidate mapping, learning and ranking sources with macOS. The following evidence is deliberately separated from installed-input-method acceptance:

| Boundary | Observed result | Remaining requirement |
| --- | --- | --- |
| Real Rime conversion | Fifteen core scenarios passed with an isolated encrypted profile, including direct umlaut input through shared normalization | Actual system key dispatch and representative editors |
| Production keyboard module | Restricted dynamic loading, multiple clients, serialized thread calls, profile mismatch refusal, active ranking-context reset, relearning and encrypted-profile reopening passed | Multi-editor persistence and installed lifecycle |
| Installer filesystem logic | Locked/tampered/unlisted payloads refuse cleanup; verified and legacy closures clean up | Actual UAC install/removal of the final package |
| Settings | English and Chinese interactions passed; both platforms show automatic update state and offer retry after failure | Final installed source identity and live preference effects |
| Shared storage | Cooperative repair passed the unified native gate: DPAPI, 766 Unicode cases, actual Rime ABI and eight multi-process scenarios, including private pending changes, merges, process death and fresh reopen | Final installed multi-editor persistence and macOS provider regression |
| Registration migration | Six callback/read-only checks passed, including the actual legacy `0x0409` query; synthetic verifier rejects source/manifest mismatch without changing files | Actual legacy removal, `0x0804` installation and visible selection |
| Native language bar | Standard Windows input-mode GUID, native menu population, disabled/stale actions, reentrant callback lifetime and GDI handle contracts passed in `run-gGbWoz` | Installed taskbar icon, left/right-click menus and actual editor actions |
| TSF ACP harness | Thirty-one owned-document cases passed in `run-TdaO0z`, including candidate mapping, partial edits, mode selection preservation, save refusal, badge lifecycle and authenticated voice handoff/disconnect | Registered system key/focus dispatch and installed menu visibility remain separate from manually driven callbacks and test-only menu-object access |
| Canonical package | Windows EXE build and packaged TSF/Pinyin/ASR/GGUF closure, hashes, licenses and architecture checks passed; exact payload identities retained in `.workspace/windows-tsf/migration/shared-pinyin-package.json` | Actual installation/migration and editor acceptance; local update archive is not signed |
| macOS adapter | Shared-core extraction and page-width mapping regression added | Fresh native compilation, IMK/editor and Keychain tests on a reachable Mac |

Reproduction entry points are `scripts/test-windows-pinyin.mjs`, `scripts/test-shared-keyboard-learning.mjs`, `scripts/test-windows-keyboard-module.mjs` and `scripts/test-windows-tsf-keyboard.mjs`. Local reports retain exact binaries/source identities and failures under `.workspace/tests/`. None of these partial results establishes a released Windows Pinyin package.

## October 24 installed defect investigation

A read-only comparison of 86 installed product files found 85 matching the canonical package, with only `Arion.exe` still from the older build. The registered TSF DLL matches the new shared-Pinyin build. This is an observed mixed installation, not evidence that the shared keyboard runtime is missing. Evidence is retained in `.workspace/windows-tsf/current-defects/installed-comparison.json`.

The NSIS regression now exercises an actual temporary installation: a running executable refuses an upgrade; a locked later resource rolls back earlier resource replacement while preserving the previous executable; releasing the resource allows retry; an unrelated top-level file survives upgrade and uninstall. The production installer uses staged extraction and rename rollback, and publishes the executable last. This has not yet repaired or revalidated the user's installed Arion payload.

Model-ready regression tests cover an early hold/toggle during loading and a delayed unavailable-model error after readiness. Loading cannot create a guaranteed-failing session; readiness clears only the completed not-loaded failure without retained text. Unrelated failures and active sessions remain intact. ASR, model runtime and provisioning tests pass. The already downloaded user GGUF also loads through the installed Vulkan worker, but that worker-only result does not establish microphone-to-editor acceptance.

The standard Roaming log path retained an older shutdown, but exporting diagnostics from the actual running settings window produced the current run (`416a6e2dd92d2b35`) in `.workspace/windows-tsf/current-defects/installed-diagnostics.zip`. It records model readiness at 22:20, followed by `ARTIFACT_UNVERIFIED` in **target_capture**, before recognizer/audio creation. Read-only process module inspection confirms existing editors still loaded the older `26213df3…` TSF build while Arion expects `55fc16…`. Mapping that refusal to a missing speech model was wrong. The initial correction suggested reopening the target application; the follow-up removes that routine instruction and separates component verification from protocol incompatibility. The separate early-loading regression is not the cause established by these current-run logs.

After installation of package `9133722b…`, the installed application, helper and
protected TSF `f9f948…` all match their built artifacts. An owned WebView2
editor still reproduces Latin-only typing with real system key dispatch.
The browser input thread differs from the foreground host-window thread;
the service incorrectly inferred initial TSF focus from that foreground HWND.
With the exact same installed DLL, a complete focus transition between two
owned windows makes both textarea and contenteditable commit `你好`.
Evidence is `.workspace/windows-tsf/current-defects/webview-installed-f9f948-refocus.json`;
the harness restores the original session input profile. This is a causal
control, not acceptance of the replacement package. The fix uses TSF's
`IsThreadFocus` and focus notifications and refreshes the focused document.
Installed acceptance must repeat without the optional refocus step.

## TSF candidate layout and stable bootstrap repair

The previous installed `0a9b3af4...` core failed per-key candidate visibility in owned WebView2 textarea and contenteditable fields: 16 of 20 active-preedit observations were hidden although final Chinese commit succeeded. The earlier composition-only acceptance did not cover this property.

The repaired `a216e37b253e808338f1149b5f637808039566dd06dc68eef5a91cffba9ff58e` core subscribes to text-layout changes and uses guarded asynchronous read edit sessions to refresh current geometry. Native ACP regression passes 40 cases, including real `TS_E_NOLAYOUT`, read-only layout recovery, clipping/scrolling and stale focus/context/revision refusal. Core COM lifetime leases now outlive member cleanup.

The fixed loader/processor facade releases its versioned delegate on deactivation. Native real-DLL fixtures verify retained factory/facade handover, active and delayed-reference retention, server locks, explicit load failure, production path refusal, stable bytes across output directories and rollback. Routine core publication changes only `CorePath`; profile/category and user enablement callbacks are not invoked. A real mapped-DLL fixture also verifies removal refusal before registration changes, guard prevention of a new mapping, and loading again after guard release. These are owned fake-core lifecycle tests, not actual Rime cross-version execution in ChatGPT.

The full repair installer (`C8354B8E07E3890A22F2A3712777F312BDBFE7455C48F2A37D357B0936551CAC`) passed canonical resource/PE/manifest/ASR/GGUF checks, silent per-user installation, all 87 installed payload hash comparisons, and actual administrator-approved direct-to-loader migration. Read-only query then reported registered/enabled, `loader:true`, no migration required and the expected core. The source was not selected before or after this transaction.

The clean installed-browser run passes all 20 candidate samples from `n` through `nihao` at 160 ms and after another 500 ms in both fields, exact Chinese commit, production bridge Unicode draft/final commit, password refusal, and ordinary-field cancel/recovery. It uses real OS keys, no refocus and no layout-hook activation; actual core, loader, keyboard and Rime module hashes are recorded. The fixture exited zero and observed successful restoration of the prior profile. Foreground restoration was requested but not separately asserted afterward. A preceding guarded-input interruption remains a failed report and is not included as successful acceptance.

Local evidence: `.workspace/tests/windows-tsf-keyboard/run-wEUq75/report.json`, `.workspace/tests/windows-tsf-loader/run-yJmsSp/report.json`, `.workspace/tests/windows-ime-registration/run-XNmcfj/report.json`, `.workspace/tests/windows-ime-client/run-CMfrsl/report.json`, and `.workspace/windows-tsf/current-defects/webview-candidate-layout-installed-final.json`. Go TSF/bridge/bootstrap tests and client race checks pass; Darwin bridge compile passes, with no fresh macOS native run.

After migration the existing ChatGPT PID4748 still mapped the pre-loader `26213df3...` voice-only DLL. Its already instantiated objects cannot obtain loader behavior through registry updates; that process was preserved. The installed-browser result does not establish repaired interaction in that already-open application. Ordinary future core handover, the one-time direct-DLL migration boundary, and bootstrap-byte maintenance are separate claims. See [the runbook](../operations/windows-input-method.md).
