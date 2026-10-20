# Competitor input-source onboarding

## Scope

This bounded static follow-up inspects installed Doubao **1.0.2 (1)** and Chatterfly **1.0.7.13415**, plus the separately retained **1.0.6.13398** Chatterfly installer. Their installer versions are not interchangeable. No competitor, script or helper was executed; no input-source or preference state was changed. The [ledger](evidence/input-source-onboarding.json) binds current carrier/slice hashes, 15 focused intervals (7,628 bytes / 1,907 instructions), exact literal/metadata bytes and the retained script digest.

REA 6.4.0 `inspect_macho` records are retained locally. One actual REA/Ghidra 12.1.4 function dossier, `ev_b188b1ff58f09de8a3eef2b118be81a06c685a77edcbb97287e439cf9c5253c7`, analyzes the installed Chatterfly selector's byte-matching arm64 slice. Other instruction findings are ordinary LLVM observations checked against original bytes, not REA decompilation. The first attempt on the universal selector was explicitly rejected; thinning resolved that architecture limitation.

## Doubao: registration, notification and selection are separate

Installed `DoubaoImeSettings` has SHA-256 `13fff30333c99fcd706e30f1e8408d49d04cb5748146a8befa72dff381f57762`. Its `InstallerInputSourceHandler` contains this reachable sequence:

| Step | Concrete anchor |
| --- | --- |
| Arrange callbacks | `enableAndSelectInputMode(needCallback:completion:)` callsite `0x1001cda20` registers enabled-source notification/retry support when requested. |
| Register bundle | Call `0x1001cdf74` at `0x1001cda34`; logs distinguish successful registration, a nonzero return with an already-resolvable source, and failure. |
| Switch away first | `changeToSystemInputMode()` at `0x1001ce5d0` resolves/selects ABC. |
| Enable | `enableInputSource()` at `0x1001ce8ec` resolves the source, handles already-enabled/missing cases, and invokes `TISEnableInputSource` at `0x1001ceb10`; off-main callers marshal synchronously to the main queue. |
| Confirm enabled | `completeIfEnabled()` at `0x1001ceca8` checks actual enablement and logs “input source is not enabled yet” rather than treating registration alone as completion. |
| Select mode | `selectInputMode()` at `0x1001cf1e4`; successful selection reaches menu refresh at `0x1001cf474`. Missing source resolution can retry registration. |

The shipped settings URLs include both `x-apple.systempreferences:com.apple.Keyboard-Settings.extension?InputSources` and `x-apple.systempreferences:com.apple.preference.keyboard?InputSources`. Their exact literals are bound; the active OS-version branch is not recovered. A separate script-construction surface at `0x1000e53b8` reveals an anchor in System Preferences' keyboard pane and activates it. This is not evidence of clicking consent controls through System Events.

The preference read at `0x1001c7908` synchronizes the `com.apple.inputsources` domain, then uses `CFPreferencesCopyValue` for `AppleEnabledThirdPartyInputSources`. This is a query, not a demonstrated consent-list write. The bounded trace does not establish that the entire product never writes other preferences.

After successful selection, `reloadInputMenu()` constructs an `NSTask` for `/usr/bin/killall`. Original static argument-array bytes identify **TextInputMenuAgent** and **TextInputSwitcher**; task launch/wait/status handling is reachable. This process refresh is static product behavior, not evidence that it is required or sufficient to repair modern activation. Doubao's outer installer copy/privilege/button chain is not recovered from the installed settings bundle.

## Chatterfly: explicit enable UI and asynchronous completion

The current host SHA-256 is `1b8277d1ac41bba3db6a15cd292fb108c8f38f748a94d29fcea837df25c7bd42`. Its `SGTools` separates enablement, source-list checking and selection. `trySelectYuanbaoOnceAndReport` at `0x10004e510` finds the Pinyin mode and calls TIS selection. Only return `0` marks selection successful, stops retrying, reports success and invokes refresh. `startSelectRetryFlow` at `0x10004e64c` observes `kTISNotifyEnabledKeyboardInputSourcesChanged` and installs a repeating timer; the timer also has timeout/failure handling. Enable operations marshal to the main thread.

`isThirdPartyInputSourceEnabledInInputList:` at `0x10004dfdc` reads `~/Library/Preferences/com.apple.inputsources.plist` and examines `AppleEnabledThirdPartyInputSources`; no consent-list write occurs in that bounded method. A separate legacy `selectShowInputMenuInMenuBarNeedsReload:` method conditionally inserts `TextInput.menu` into **com.apple.systemuiserver menuExtras** and writes that plist (`0x10004f598`), or uses an `open` command. That changes menu presentation, not the third-party authorization list.

The post-selection delayed refresh at `0x10004e9c8` reaches process handling for **pboard** and **TextInputMenuAgent**. The installed `InstallerSelector` helper (SHA-256 `14616dc101c29b3074b1b50c1f7cae48a6cfc8c123297d0ac1eab7d03d3b8ee2`) itself only enumerates existing sources, checks Pinyin availability, selects it and emits a success notification after TIS returns zero. It is not an all-purpose privileged installer or an authorization bypass.

The retained **1.0.6.13398 installer** provides concrete visible UI evidence. UTF-16 CFString bytes decode the button as “去启用”, the instruction as adding Chatterfly to the input-method list, and the fallback as “若未能自动启用，请前往「系统设置 > 键盘 > 输入法」手动添加 Chatterfly，详情可查看”. `enableButtonClicked:` at `0x100059e24` enters an enabling/debounce state and calls its delegate; enabled-source notifications, a manual-tip label and a switch-check timer advance the guide only after observed state changes. Its `enableInputGuideNew`/`enableInputGuide` images are named in the UI code. These facts do not establish the newer installed version's exact visible text.

Its shipped `install.sh` moves a staged app into `/Library/Input Methods`, installs/registers LaunchAgents, clears quarantine attributes and refreshes processes/menu state. The script assumes privileged installation and includes destructive replacement/migration behavior; it was only read. The original pass did not trace the outer installation chain; the follow-up below resolves the privileged helper and post-install registration paths. Commented selection-helper commands are not counted as executed steps.

## Implication for Arion

“One click” covers several stages: file placement, TIS registration, user-visible enablement, mode selection and onboarding completion. Competitor code also waits for state changes and supplies manual instructions; a zero registration/enable status does not establish completion. Static process refresh and legacy plist manipulation do not justify copying those interventions into Arion as proven repairs. Actual first-install consent behavior, login/cache effects and competitor success on this macOS session remain unobserved. Arion's previously deferred system acceptance remains separate from this read-only investigation.

## Arion actual visibility follow-up

The [v10 control](../../benchmarks/runs/imk-keyboard-v10/README.md) checks the user's actual missing-source report. The add dialog's Arion search was empty despite the installed bundle and pending installation diagnostics. Replacing the ad hoc development host with the existing notarized host passed Gatekeeper, but the source remained absent, including in a fresh System Settings process. TIS later exposed a disabled parent/default-enabled child; this does not establish an available or selected input method. Both controls restored the development host byte-for-byte and kept Chatterfly current, without host launch or source selection. Discovery remains unresolved; logout/login was not performed or proven to fix it. The earlier installation repair addresses rollback/progress/guidance, not successful system acceptance.

## Chatterfly installation-to-registration follow-up

The [registration ledger](evidence/chatterfly-registration.json) binds the retained **1.0.6.13398** installer and its privileged helper to 14 LLVM intervals / 1,301 byte-validated instructions and four actual REA/Ghidra 12.1.4 function dossiers. This is static evidence; neither competitor installation nor system input-source state was changed. It does not establish that the installed **1.0.7.13415** has an identical installer.

**Registration is part of the installer's post-file-install flow, and the enable button re-enters that flow.** It is not deferred exclusively to a later preferences visit. The concrete chain is:

| Stage | Static evidence |
| --- | --- |
| Privileged file installation | `YBHomeViewController.inputAdmi` calls `executeWithPrivileges:atPath:arges:error:` at `0x10005f364`, targeting `Contents/ChatterFlyInstallerHelper`. The helper prepares the extracted `install.sh`; its `install` method calls `startShellInstallAction`, which launches `/bin/sh` with the script path through `STPrivilegedTask`. The original script moves the complete bundle into `/Library/Input Methods`. |
| File-install completion | The helper result is compared with `"0"`; its success path dispatches back to the main queue and reaches `homeViewControllerDidFinishInstall:` at `0x10005f4b8`. That callback marks the install script finished and, for the non-upgrade branch, calls `proceedAfterInstallIfNeeded` at `0x10007815c`. Upgrade post-actions have a separate guard and are not claimed equivalent here. |
| Begin system setup | `proceedAfterInstallIfNeeded` enters `p_dispatchNextViewByOSVersion`. REA `ev_ff98f3508b2a6b3c61dc7cdf7d32bff010b1233cf6ee683bdebe1668160cb1fc` recovers a **macOS >=12** branch to `nextViewForOS12`; the method name does not mean macOS 12 only. |
| Observe before enabling | `nextViewForOS12` installs input-source observation/timer support before scheduling its enable queue. Its block sends a new install, or an existing install not enabled, to `selectYuanbao`; an already-enabled existing install takes a menu-check path. REA `ev_567825397b4c2247f6b02013bfc42ddc756dc960b0a88a442ff6b1a6f405f32d` confirms the observer-before-queue setup; the associated block is bound separately by the LLVM interval. |
| Register and request enablement | `selectYuanbao` dispatches its >=12 branch to `enableSogouInputMethodForOS12`; this eventually marshals `enableSogouInputMethodOnMainThread`. That method creates the installed URL, calls **`TISRegisterInputSource` at `0x100088330`**, enumerates the parent and Pinyin child, then calls **`TISEnableInputSource` for the parent at `0x100088464`**. REA `ev_529a0e05cff49ccfe25c549235fc874c398ba09c7c50170439cb046ba542a75e` confirms registration, enumeration and the parent enable request. |
| Explicit retry | `enabledTipsViewControllerDidClickEnableButton:` calls `retryEnableYuanbaoInputSource` at `0x10007843c`; retry re-enters the same OS dispatch. Earlier guide evidence supplies the visible “去启用” label and notification-driven progression. |
| Selection completion | The guide's enabled-state handler calls `selectYuanbaoOnMenu`. The separately inspected installed host has notification/timer retries and only reports selection success after TIS selection returns zero. The installed helper `InstallerSelector` selects an existing source; it is not the registration stage. |

The installer can therefore **request** registration and enablement immediately after copying files, then continue observing user/system acceptance. A return from the file helper or a zero TIS return does not prove that macOS consent has persisted, that the source appears in the add dialog, or that a real editor can type through it. No fresh-install runtime timing was measured. The script's process termination and attribute clearing remain observed competitor implementation, not validated repairs for Arion.

## Arion comparison before onboarding integration

This comparison records the working tree before the subsequent onboarding integration, including existing uncommitted installation fixes. Those fixes must not be confused with the earlier v10/v11 observations. Onboarding now offers optional input-method setup after Accessibility, shares keyboard install/status controls and the System Settings guide shell, and gates Continue on actual selection. It does not resolve the native activation acceptance gap or add a retry-registration action.

| Concern | Current Arion behavior | Consequence / priority |
| --- | --- | --- |
| Entry point | First-run onboarding contains Welcome, Account, Microphone, Accessibility, Model, Shortcut, Try and Finished; no IME installation step. Dictation's IME choice hands off to Keyboard settings. | **P1:** offer a coherent setup flow when the user chooses the system input method, with install, consent, switch and input verification in one place. This changes product flow and requires an IME-01 update before implementation. |
| Registration mechanics | `InstallIME` validates the signed payload, stages/replaces the per-user bundle, registers LaunchServices then TIS on the main thread, and requests parent/child enablement. | These core API steps already exist. `/Library/Input Methods` versus `~/Library/Input Methods` is an observed packaging difference, not an established root cause or reason to require administrator installation. |
| Pending diagnosis | `darwinIME.State` maps both copied-but-undiscovered and registered-but-disabled to `InputMethodPending`. | **P1:** preserve observed stages so missing discovery and incomplete enablement receive different actions. Do not infer an unobservable consent state from a disabled flag. |
| Recovery | With an unchanged payload, Pending exposes Open Keyboard Settings and Remove; it has no explicit registration/enable retry. | **P1:** add an idempotent retry action for incomplete setup, without requiring remove/reinstall. Chatterfly's enable button has a concrete retry path. |
| Completion | `keyboard.Ready` accepts enabled or selected; an enabled source loses the Dictation-to-Keyboard handoff and has no selection/test action on its source card. | **P1:** distinguish enabled availability from current selection and working input. Keep a clear next action through selection and an actual editor probe. Readiness for future manual selection is not proof of present usability. |
| Native acceptance | v10 failed add-dialog visibility even with the notarized control. v11 records user-reported Allow, then a disabled parent and an ad hoc replacement. Later fixes passed build/payload checks only. | **P0:** run the corrected signed release through first-install discovery, Allow persistence, add-list visibility, selection and actual IMK typing. Capture stage transitions and failures. The activation root cause remains unresolved; logout/login is an unverified next experiment. |

Current code already supplies asynchronous installation/progress, payload preflight, main-thread TIS calls, parent-before-child enablement, retained pending bundles, no immediate automatic settings jump, and one-second state refresh. These are not outstanding recommendations. The remaining issue combines an **unaccepted OS activation path** with **fragmented setup and recovery**; copy changes alone cannot close it.
