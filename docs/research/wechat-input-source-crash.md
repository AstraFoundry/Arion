# WeChat crash when selecting Arion

## Finding and scope

Four initial local WeChat reports establish a system input-source switcher failure: Apple's `TUINSCursorUIController` resolves an input-source ID to a null reference, passes it to selection, then releases it without a null check. The release terminates WeChat. A later instrumented reproduction and system HUD log reconstruction identify Arion's Pinyin child as the final selection target, as detailed below; a persistent WeChat-specific keyboard-cache discrepancy is now observed, and targeted regeneration was followed by a user-confirmed successful switching retry. The exact refresh failure and long-term prevention remain unresolved. Arion's removal and replacement lifecycle presents relevant risks, but this investigation does not establish a completed product fix.

The comparison covers installed Doubao **1.0.2 (1)**, installed Chatterfly **1.0.8.13459**, and the separately retained Chatterfly **1.0.6.13398 installer**. Older Chatterfly 1.0.7 findings are explicitly historical. No competitor installer, updater or helper was executed. No input source or system preference was changed. The later user-authorized recovery isolated one WeChat keyboard cache and restarted WeChat; its scope and backups are recorded below. The [evidence ledger](evidence/wechat-input-source-crash.json) records artifact identities, local complete REA records and the investigation limits. Commercial decompiled code and binaries remain in ignored local research storage.

REA 6.4.0 and Ghidra 12.1.4 were available through the installed tools. Setting Ghidra and Java paths in the analysis process resolved the missing provider configuration; no new engine installation or persistent client configuration change was needed. Fresh Ghidra dossiers inspect the installed Chatterfly helper's arm64 slice and Doubao settings selection/refresh functions. The initial target-free plist requests failed; subsequent requests with an active target and explicit plist paths succeeded. These failed requests remain retained rather than being counted as evidence.

## Crash evidence

The four reports concern WeChat 3.7.0 (25070) on macOS 26.6.2 (25G83). All contain `CFRelease() called with NULL`, `EXC_BREAKPOINT/SIGTRAP`, and the same main-thread frames:

```
CFRelease
-[TUINSCursorUIController _selectCurrentInputSource] +84
-[TUINSCursorUIController moveTextInputMenuHUD:] +272
TSMMessagePortCallBack
```

Independent inspection of the loaded system helper shows lookup by `TISCopyInputSourceRefForInputSourceID`, selection by `TSMSelectInputSource`, and unconditional release of the lookup result. The return address after release is +84. The latest report has x20 = 0, matching the register holding that lookup result. This is ordinary local instruction inspection, not a REA decompilation result. Apple's [CFRelease contract](https://developer.apple.com/documentation/corefoundation/cfrelease) independently requires a non-null reference.

The latest two crashes followed an Arion removal/reinstallation sequence. The second WeChat process launched after the first crash and also failed, so an old cache confined to a long-running WeChat process cannot explain the complete sequence. Persistent per-application cache state or session registration inconsistency can survive a process restart. Later HUD reconstruction identifies the correct child ID, narrowing the earlier mismatch hypothesis. The crash reports do not retain the ID string. Current Arion metadata was installed after those crashes and cannot establish the exact crash-time bytes.

A live TIS enumeration found normal enabled, nonselectable parents and enabled, selectable Pinyin children for all three input methods. A disk preference snapshot disagreed with the live result; that snapshot is not authoritative proof of current enablement or corruption.

## Competitor comparison

| Area | Doubao | Chatterfly | Consequence for Arion |
| --- | --- | --- | --- |
| Input modes | One visible `.pinyin` child, `smSimpChinese`; default and primary flags enabled | Same structure in installed 1.0.8 | Arion already uses the same structure. Switching to `smUnicodeScript` is not a demonstrated repair. |
| Registration and activation | The byte-matching settings binary registers, requests enablement, checks state, then selects; missing mode resolution can re-enter registration | Retained 1.0.6 installer registers and enables the parent; fresh 1.0.8 helper decompilation confirms the same sequence in that helper | Preserve parent/child distinction and observe each stage; an API return alone is insufficient. |
| Disable | The installed main executable contains disable imports/calls, including wrappers; no successful write or complete uninstall caller chain was observed | Fresh helper `deselectSogouInputMethod` at `0x100010028` enumerates both IDs, calls deselect on the child and disable on the parent; installed host contains an equivalent method | Disabling is a real implementation, not merely an imported symbol. It does not establish that Arion's denied write succeeds from a helper or that this method is an uninstall entry point. |
| Re-enable path | Registration/selection and refresh are distinct | Helper `selectSogouAndShowIconWithAuth` at `0x10000f880` calls the disable method before enabling and arranging observers/timers | Do not infer a necessary authorization bypass or copy a disable/re-enable cycle into Arion without measuring the executing process and resulting system state. |
| Menu refresh | Settings constructs a `killall` task targeting `TextInputMenuAgent` and `TextInputSwitcher` after a selection attempt | Fresh helper delayed callback at `0x100011b78` targets `pboard` and `TextInputMenuAgent` | Both have process refresh workarounds. Static execution paths do not prove they fix this crash or are public Apple-supported recovery APIs. |
| Update placement | Shipped script moves `Contents` to backup before moving the staged replacement into place | Retained 1.0.6 script deletes the old app before moving the new one | Neither establishes gap-free replacement as a prerequisite. Arion's two-rename gap remains a hardening opportunity, not a proven root cause. |
| Installation scope | System-wide input-method directory | System-wide directory with privileged installation | Apple also supports per-user installation. Changing Arion's scope is not justified by this difference alone. |

Fresh Chatterfly helper dossiers bind the disable implementation, its two callers, the registration/enable method, and the delayed refresh callback. In particular, the disable routine is reached from enable/show-icon paths; calling it an observed uninstall flow would overstate the evidence. Its child `TISDeselectInputSource` call should not be copied as a keyboard removal operation: Apple's contract limits that operation to other source types.

Fresh Doubao settings dossiers at `0x1001cf1e4` and `0x1001cf4c8` confirm a missing-source failure path, main-thread selection dispatch, a subsequent task launch/wait/status path, and a return determined by the selection status. The refresh call occurs before the selection result is compared with `noErr`, so refresh is not restricted to successful selection. Swift decompilation does not recover every argument correctly; the task's executable and process-name constants are separately bound by the matching prior instruction/data evidence.

The [onboarding research](input-source-onboarding.md) preserves the earlier version-bound registration and refresh analysis. Current script/plist hashes were compared with those ledgers. The installed 1.0.8 host must not be described as shipping the retained 1.0.6 installer's exact workflow.

## Apple contracts and current guidance

The installed Apple macosx26.4 SDK `TextInputSources.h` is the primary contract for Carbon TIS; its digest and excerpts are bound by the [API-contract ledger](evidence/input-source-api-contracts.json). Context7 returned unrelated libraries for this API, so none were used.

- **Lookup is a snapshot:** lines 725–768 describe `TISCreateInputSourceList` as a snapshot at the time of the call. Refresh queries after state changes; do not treat a retained list as current.
- **Register, enable and select differ:** registration at lines 1199–1254 makes file/bundle sources available for reference lookup. Enablement at lines 1015–1043 makes a source available for selection. Selection at lines 952–985 requires an enabled, select-capable source, with an enabled parent for a mode.
- **Disable affects selection UI:** lines 1047–1071 describe removal from selection availability and system UI. They do not guarantee that deleting an enabled bundle leaves a harmless inert entry. The public header has no unregister/remove operation.
- **Both installation scopes are supported:** registration documents the per-user and system-wide input-method directories. System-wide installation is not a required repair.
- **Threading is explicit:** UI applications call TIS on the main thread; non-UI callers serialize access. Notifications trigger re-query, not a declaration that a particular source changed.
- **Private lookup stays diagnostic-only:** `TISCopyInputSourceRefForInputSourceID` is absent from the public header. Its observed use explains the crash mechanism, not a public repair API for Arion to adopt.

[Apple DTS guidance](https://developer.apple.com/forums/thread/775526) acknowledges that System Settings can fail to discover a newly installed input source until logout, and reports no known way to avoid that logout. The reporter supplied Feedback FB23026482. This is staff guidance about discovery; it does not prove logout fixes this WeChat crash, require logout for every activation, or endorse terminating input-menu processes.

[Apple's user guide](https://support.apple.com/guide/mac-help/write-in-another-language-on-mac-mchlp1406/mac) supplies the supported manual Input Sources management path. [QA1810](https://developer.apple.com/library/archive/qa/qa1810/_index.html) concerns older palette behavior and is not a modern keyboard authorization bypass.

## Repair recommendations

### P0 Diagnose the failed identity and remove the unsupported removal assumption

Two tracks should proceed together:

1. In a controlled reproduction, capture the exact failing HUD ID and compare it with parent/mode IDs and fresh-process resolution before and after registration changes. Record the installed build identity and source lifecycle stages. Do not collect editor contents. If the failing ID is the parent, investigate mode advertisement/selection; if it is the correct child, investigate registration and session discovery rather than changing IDs speculatively.
2. Separate update host shutdown from destructive removal. Current `releaseIME` intentionally leaves enablement in place, and `removeInputBundle` then deletes the bundle. Retain valid bytes while system removal is unresolved. If programmatic disable is denied, expose System Settings removal and defer deletion until fresh observation establishes removal from selection availability. This reduces an identified risk; it is not yet a verified cure for the recorded crashes.

Do not simply restore `TISDisableInputSource` in the existing Arion process. Retained native logs show policy-denied `com.apple.inputsources` writes while that process still reported a disabled source and a fresh process reported enabled. Chatterfly's helper implementation is a reason to test process-specific behavior in isolation, not proof that moving the same call to an unsigned or signed helper bypasses the policy. Preserve parent/child and cross-process state evidence.

### P1 Verify supported recovery and reconcile lifecycle specifications

With installation stable, compare existing/new client processes, menu versus Control+Space selection, and a fresh login session. The fresh WeChat restart has already failed in the observed sequence; do not describe restart alone as a verified fix. If discovery is incomplete, present logout/login as Apple-documented discovery recovery, with user-controlled timing, rather than silently restarting system processes.

Keep installed/registered, enabled, selected and expected-build connected states distinct. Existing controller and isolated-editor acceptance do not cover this system HUD failure. Update IME-01, IME-UPDATE-01, IME-A01/A06 and ADR 006 together when adopting a revised lifecycle. ADR 006 currently says disable before deletion while the spec/code retain enablement; a report alone does not resolve that normative conflict.

Prepare a minimal Apple Feedback report with the repeated stack, runtime versions, exact failed ID when captured, and a synthetic reproduction. Apple's missing null guard cannot be repaired through an ordinary Arion IMK client callback. This investigation did not submit a report externally.

### P2 Evaluate path-preserving updates and optional refresh experiments

Evaluate a supported atomic directory exchange to eliminate the two-rename gap, preserving prior selection, process exit, signature validation, expected-build verification and rollback. Treat it as hardening unless controlled tests connect the gap to failed resolution.

Competitor menu-process termination may be tested separately in an isolated account after recording the failure, if needed to distinguish a stale switching service from registration. Do not ship `killall`, clipboard-service restarts, direct preference edits, or disabling SIP as a default fix. None is established by the consulted Apple contracts or this comparative evidence.

## Acceptance and remaining work

A repair must pass menu and Control+Space switching in an already-running and newly launched WeChat, after both selected/unselected updates, after controlled removal/reinstallation, after rollback, and after a new login. Verify that denied removal preserves the bundle and that successful manual removal precedes deletion. Include another standard AppKit editor and a clean-account discovery/authorization case. Preserve failures rather than reporting a successful API request as successful editor acceptance.

Completed: matching crash analysis, fresh Chatterfly helper and Doubao settings decompilation, competitor identity/script/metadata comparison, and Apple contract review. The later instrumented run and HUD reconstruction resolve the target-identity question and show recurrence without an immediate installation operation. Unresolved: the process-specific lookup trigger, cross-process permission behavior for competitor helpers, fresh-login outcome, and whether a lifecycle change prevents recurrence. No removal/update repair or passing crash remediation is claimed.

## Implemented diagnostic follow-up

Arion now records bounded `ime_native` lifecycle snapshots through its normal local diagnostic store: its own requested parent/mode identity, native source flags, current-source category, operation result, process ID and installed executable digest. Queries log only changes, while explicit mutations record before/after state. The [logging contract](../backend/logging.md#macos-input-source-diagnostics) describes the schema and privacy limits. This instruments Arion's process; it neither captures the failed ID within WeChat nor implements the removal/update repairs proposed above. Native crash-remediation acceptance remains open.

## Failed ID reconstructed from the latest system HUD log

The subsequent WeChat report captures the same failure at local time **11:52:33.7789**, process **61101**. The instrumented Arion run started at **11:52:17**: it found enabled parent/mode identities, observed Arion selected at **11:52:27**, then another source at **11:52:31**. No registration, enablement, install or removal operation was recorded in that run before the crash. A fresh query independently found the enabled, selectable Arion mode. This does not rule out inherited session state, but immediate replacement is not needed to explain the observed sequence.

Standard LLDB attachment to the reopened WeChat was denied by `task_for_pid`. The target has hardened runtime and no `get-task-allow`; its signature and SIP were not changed. This attempt did not capture a live argument. Instead, retained `com.apple.TextInputUI:CursorUI` messages from the crashed WeChat process supplied the actual HUD inputs:

| Zero-based index | Logged source ID |
| --- | --- |
| 0 | `com.bytedance.inputmethod.doubaoime.pinyin` |
| 1 | `com.apple.keylayout.ABC` |
| 2 | `com.tencent.inputmethod.chatterfly.pinyin` |
| 3 | `com.apple.inputmethod.Kotoeri.RomajiTyping.Japanese` |
| 4 | `com.apple.inputmethod.SCIM.ITABC` |
| 5 | `si.astrafoundry.inputmethod.arion.pinyin` |

At **11:52:32.544**, `showTextInputMenuHUD:` logged that list with `tsmSelection = 1`. It then logged `moveTextInputMenuHUD:` with `tsmDelta = 1` at **11:52:33.139**, **.351**, **.550**, and **.756**. The last event immediately precedes the crash.

Ordinary inspection of the same installed system methods binds those dictionary keys to the switcher's list and initial index. `move:` adds the delta modulo the list size; `inputSourceID` returns `inputSourceIDs[selectedIndex]`. The cursor controller calls `_selectCurrentInputSource` immediately after each move, matching its +272 return address in the crash stack. A controlled replay used the real `TUINSCursorUISwitcher` with a nil assertion controller, the logged list/index and four moves; it produced ABC initially, then Chatterfly, Japanese, system Pinyin and finally **`si.astrafoundry.inputmethod.arion.pinyin`**. No cursor-controller selection operation was invoked by the replay.

The final failed target is therefore reconstructed as Arion's existing Pinyin child, not the parent or a differently spelled ID. This is a derived conclusion from historical runtime logs plus controlled system-code replay, not a sampled live `x0` argument. Raw records, method instructions, decoded key bindings and replay output are bound in the [ledger](evidence/wechat-input-source-crash.json). The unresolved question is why this correct child ID returns no reference inside WeChat while other process queries find it. Capturing that lookup's internal state, or isolating a session/process discovery discrepancy, remains necessary before claiming a root-trigger repair.


## Historical corroboration and process-specific keyboard cache

Three earlier HUD sequences also end at `si.astrafoundry.inputmethod.arion.pinyin`: WeChat processes 57229, 30163 and 719. Each starts at index 1 and records four +1 moves, ending at index 5, despite differences in the other sources' order. Together with the latest sequence, this gives four independently logged failures targeting the same child. This remains reconstruction, not a directly sampled lookup argument.

Inspection of the installed private lookup shows an ID-filtered `TISCreateInputSourceList` query with `includeAllInstalled = false`. An empty array produces a null result; its all-installed retry is reached for a null array, not an empty one. Fresh ordinary and hardened App Sandbox fixture processes resolve both Arion identities using public enumeration and that same private lookup. App Sandbox alone therefore does not reproduce this failure in the tested fixture.

The per-application keyboard cache provides a concrete difference:

| Scope | Cache bytes | Arion ID occurrences | Observation |
| --- | ---: | ---: | --- |
| WeChat | 140020 | 0 | Cache modification time precedes the latest reproductions; Doubao and Chatterfly IDs are present |
| Ordinary fresh process | 140436 | 4 | Arion ID and installed bundle path present |
| Owned sandbox fixture | 140436 | 4 | Arion ID and installed bundle path present |

The file is `com.apple.IntlDataCache.le.kbdx` in each application's OS cache directory. A reopened WeChat process (65098) maps its existing WeChat-specific file, confirming this is a live cache dependency rather than an unrelated disk artifact. Raw byte searches establish the table above; a complete cache-format parser was not used. These files and personal directory paths remain in ignored local evidence storage.

A controlled fresh/old-copy/restored comparison modified only the owned fixture's cache. The original WeChat cache remained byte-identical. When given the old WeChat bytes, the fixture logged invalid previous cache data, regenerated its own cache, and still resolved Arion. Thus copying old bytes into another sandbox does **not** reproduce the failure, and the missing-ID observation alone is not causal proof. The subsequent matched-cache investigation below explains the fixture rebuild as a control/component size mismatch; why WeChat originally missed invalidation remains unresolved.

The next discriminating recovery test is limited to this one WeChat keyboard cache: preserve the original bytes and digest, quit WeChat normally, isolate the file, reopen WeChat, and inspect its rebuilt cache before retrying the same switching sequence. This is an explicitly authorized diagnostic experiment, not a product-level cache-clearing feature or a verified repair. It must not be generalized into deleting other WeChat caches, restarting clipboard services, or mutating source preferences.


With explicit user authorization, WeChat was quit normally and that single cache was backed up, hash-checked and renamed aside. Reopening WeChat generated a 140436-byte replacement containing four Arion ID occurrences and two installed bundle-path occurrences. The original 140020-byte cache remains preserved both beside its previous location and in the ignored investigation directory; both copies have SHA-256 `3eb9582ffc35bd15d33daa15e431b4df0cc6376e8aafc8ede328fb9b13fc4151`. The replacement initially has SHA-256 `94f4217a79a83913f7ab1b341a2564d0f7e2488e782a4ea1bd503b6b5be88fa2`. No chat data, other WeChat cache or input-source preference was modified. This verifies cache regeneration and restored Arion cache membership; the user subsequently confirmed that repeating the previously failing shortcut switch succeeded without crashing, as recorded below.


## Recovery verification and revised conclusion

After the single-cache regeneration, the user repeated the previously failing shortcut switch to Arion and reported successful switching without a crash. An independent check found WeChat running as process 71120 and the rebuilt cache unchanged, still containing Arion. The newest WeChat report in the inspected diagnostic directory remained the earlier 11:52 report. The active process differs from the immediate recovery reopen process 66569; this pass did not establish why that earlier process ended, and absence of a newer report is not proof that every intervening run succeeded. Retained CursorUI messages do not contain a complete successful show/move source-ID sequence, so the switching result is explicitly user-observed.

The strongest current explanation is a persistent discrepancy between the HUD's enabled-ID list and WeChat's keyboard-source cache: the HUD targets the correct Arion child, the lookup cannot obtain a reference, and Apple's unconditional `CFRelease(NULL)` traps. The old mapped cache lacked Arion, and regenerating only that cache restored Arion membership followed by a successful retry. This supports stale per-application cache as the leading trigger and validates this local recovery. It does not establish why invalidation failed, prove that the old bytes alone are sufficient, or show that uninstall/update lifecycle changes prevent recurrence.

Do not ship deletion of third-party application caches as an Arion feature. Preserve stable identities and valid bundle bytes during lifecycle transitions, add native regression coverage for already-running and newly launched clients, and validate removal/update behavior before claiming a permanent prevention fix. The original cache backups remain available; no rollback was performed after the successful user test.


## Prevention and broader application follow-up

The [cache-prevention report](input-source-cache-prevention.md) adds a read-only cross-application inventory, public non-WeChat crash evidence, and a matched cache/control A/B/A reproduction in the owned fixture. Matching old TextEdit cache files cause the same lookup used by the HUD to fail for Arion; restoring the fixture's fresh files restores lookup. This resolves the earlier single-component replay's size-mismatch limitation and strengthens the stale-cache mechanism. No other personal application was switched or intentionally crashed. Original invalidation failure and permanent product prevention remain open; the report maps proposed lifecycle changes and native acceptance separately from the verified local recovery.
