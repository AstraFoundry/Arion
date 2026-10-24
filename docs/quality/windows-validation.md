# Windows validation

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

Private execution logs are under `.workspace/windows-validation/`; in particular `desktop-check-final.log`, `test-after.log`, `native-target-final.log`, `microphone-lifecycle.log`, `race.log`, `diagnostics/retry.log` and `release/RESULTS.md`. Installed-product traces are also hash-bound in the durable speech archive. Native Content screenshots under `.workspace/e2e/win32/` are render evidence, not a complete OS compositor or editor compatibility audit.

## Repairs

- Windows commits now validate session identity and cancellation, reject secure standard EDIT controls, and revalidate captured standard EDIT selection. Input batches stop on cancellation or changed focus. Already-posted input remains an uncertain outcome instead of claiming nothing was inserted.
- Context reads fail closed when field safety is unknown. Explicit reflection-based registration repairs the .NET UI Automation provider loader under PowerShell; visible text now comes from real providers, with password/offscreen exclusion. Cancellation retains its own error code.
- Update rollback removes newly added files/directories when a later locked destination prevents replacement, while restoring originals. A real Windows file-lock regression failed before the repair and passed afterward.
- Portable test cleanup waits for owned child processes to exit before removing their Windows working directories. Signing fixture assertions use platform path handling.
- Native tests use owned Win32 controls instead of relying on modern Notepad process/window behavior. The packaged speech test exposes only a test-control pending-final observation so cancellation really occurs after recognition, rather than merely before it.

## MSVC release runtime validation

The CI-style CPU worker was additionally built with Visual Studio 2022 / MSVC 19.44.35228.0, x64, static `/MT`, and OpenMP disabled. A minimal `/MT` + OpenMP executable still imported `VCOMP140.DLL`; the new PE gate rejects that unbundled dependency. The full pinned CPU worker imports only `ws2_32.dll`, `shell32.dll`, `kernel32.dll` and `advapi32.dll`, and its `--version` succeeds with only Windows system directories on `PATH`. Staging and package verification now check normal/delay-loaded import closure and this CPU launch; four portable dependency-gate tests pass. Vulkan's optional loader remains supplied by the GPU driver.

This run exposed a separate cancellation lifecycle defect: an otherwise healthy MSVC worker acknowledges the long diagnostic cancellation after approximately 5.6–5.9 seconds, so the adapter's existing five-second watchdog retired it and left later requests using a dead process. Recovery now retires/reaps that generation after the same drain budget and loads the same immutable worker/specification only for a later explicit request; the cancelled request is never retried. Arbitrary crashes and real reload failures remain explicit failures, while caller cancellation during reload does not poison later requests. The final source-bound real MSVC run passes six offline utterances, cancellation followed by a new request, and three realtime streaming cases. Cancellation took 5,183.821 ms; the next request succeeded, including replacement loading, in 6,717.508 ms. These diagnostic timings do not establish latency budgets or recognition quality. The [durable archive](../../benchmarks/reproduction/windows-native-asr-v1/README.md#msvc-release-runtime-validation) retains the original failure, raw protocol observations, intermediate attempt, final outputs and exact source/runtime hashes.

## Retained development artifacts

The retained preview installer is `apps/desktop/build/windows-amd64/Arion Setup 0.1.0-alpha.11 amd64.exe` (15,298,202 bytes), SHA-256 `fe2256cf7738d3d83a4c24d879efe767930cadc3b19149a777a5f57eaf4231b1`. Its application is 23,900,672 bytes, SHA-256 `c822189b8acec48b403dc9f655cca6f09a029257f021a2a54065dee1068089af`. A second retained copy is under `.workspace/windows-validation/release/final-collected/`.

## Remaining acceptance gates

- Publisher signing, a production-key signed update artifact, real published old-to-new channel upgrade, and clean/minimum-supported Windows host testing remain open. Test-key update acceptance does not establish production trust configuration.
- The mixed-language development fixtures misrecognize technical terms, including `standup` and `PR / merge / main / CI`; other Mandarin/English errors are preserved. Runtime success does not satisfy speech quality or latency budgets.
- Physical hotkey-to-microphone-to-insertion with a person speaking, unplug/suspend/resume, OS permission denial and a representative third-party editor matrix remain unverified. Microphone capture and fixture-audio speech delivery were tested separately. The packaged test enables the offline guard but does not independently monitor every process socket.
- Selection ownership and secure-field recognition are verified for standard EDIT controls. Browser/Electron/custom controls and all-format clipboard contention/restoration remain outside this run. The default tested delivery is Unicode SendInput.
- TSF, Windows local polishing, scene/site/cursor extraction, edit learning and desktop sync enrollment/outbox remain unimplemented or unsupported as documented; tests do not simulate them as completed capabilities.

Reproduction commands and opt-in environment variables are in [native E2E](e2e.md). For this tool setup, place `C:\Windows\System32` before w64devkit in PATH: its GNU tar cannot extract the pinned Vulkan ZIP, while Windows bsdtar can. The original failed build is retained; no archive pin or native requirement was bypassed.
