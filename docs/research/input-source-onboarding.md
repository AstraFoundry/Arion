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

Its shipped `install.sh` moves a staged app into `/Library/Input Methods`, installs/registers LaunchAgents, clears quarantine attributes and refreshes processes/menu state. The script assumes privileged installation and includes destructive replacement/migration behavior; it was only read. The main installer imports Authorization APIs and a privileged-execution wrapper, but the complete visible-install-button → authorization → script call chain remains untraced. Commented selection-helper commands are not counted as executed steps.

## Implication for Arion

“One click” covers several stages: file placement, TIS registration, user-visible enablement, mode selection and onboarding completion. Competitor code also waits for state changes and supplies manual instructions; a zero registration/enable status does not establish completion. Static process refresh and legacy plist manipulation do not justify copying those interventions into Arion as proven repairs. Actual first-install consent behavior, login/cache effects and competitor success on this macOS session remain unobserved. Arion's previously deferred system acceptance remains separate from this read-only investigation.

## Arion actual visibility follow-up

The [v10 control](../../benchmarks/runs/imk-keyboard-v10/README.md) checks the user's actual missing-source report. The add dialog's Arion search was empty despite the installed bundle and pending installation diagnostics. Replacing the ad hoc development host with the existing notarized host passed Gatekeeper, but the source remained absent, including in a fresh System Settings process. TIS later exposed a disabled parent/default-enabled child; this does not establish an available or selected input method. Both controls restored the development host byte-for-byte and kept Chatterfly current, without host launch or source selection. Discovery remains unresolved; logout/login was not performed or proven to fix it. The earlier installation repair addresses rollback/progress/guidance, not successful system acceptance.
