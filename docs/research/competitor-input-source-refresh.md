# Doubao and Chatterfly input-source refresh implementations

## Finding and scope

Both inspected products contain active input-source registration and service-refresh workarounds. Chatterfly's shipped helper has an explicitly reachable argument-dispatch path that force-restarts the clipboard daemon as well as the input-menu agent. Doubao's settings selection path restarts input-menu processes after a selection attempt. Neither static implementation establishes that all applications' persistent keyboard caches are repaired or that the Apple NULL-release crash cannot recur.

This follow-up reuses the [WeChat crash evidence](wechat-input-source-crash.md) and [matched-cache reproduction](input-source-cache-prevention.md), then traces additional caller conditions, helper entry points and preference readers. Inspected versions are installed Doubao **1.0.2 (1)** and installed Chatterfly **1.0.8.13459 (13459)**. Chatterfly's separately retained **1.0.6.13398 installer** is historical evidence only. These are artifact versions, not claims about the latest public release.

No competitor executable, installer, service-restart command or script was run. No application cache or input-source preference was changed. Full binary identities, reused and fresh complete REA records, ordinary assembly, literal-search boundaries and hashes are retained in the [evidence ledger](evidence/competitor-input-source-refresh.json). Commercial binary/decompiler material stays in ignored local storage.

## Comparison

| Area | Doubao | Chatterfly |
| --- | --- | --- |
| Source identity | Mode-aware parent and one visible `.pinyin` child | Same parent/child shape; fixed installed bundle path |
| Registration and enablement | Registers installed bundle, resolves sources, enables parent/child before selecting | Helper registers the installed bundle and enables parent; some entry paths first deselect child/disable parent |
| Selection refresh | Settings selection attempt calls menu refresh before testing whether selection returned `noErr` | An older `useSogouAuth` path refreshes only after enable reports success; a newer retry flow refreshes after successful child selection |
| Services targeted | Settings refresh: `TextInputMenuAgent`, `TextInputSwitcher`; update script also restarts `SystemUIServer` | `TextInputMenuAgent` and `pboard`; a macOS >=13 enable branch also restarts `pboard` before enable |
| Waiting and observation | Main-thread selection, return-code/state handling; no cross-client cache verification established | Observers/timers; newer code has a 0.5-second retry interval and 5-second timeout, with reachability limits below |
| Direct per-app cache operation | No matching literal direct-cache cleanup found in the bounded scan | Same; service restart is not direct `.kbdx` deletion |
| Replacement | Moves old `Contents` to backup then promotes new `Contents` | Retained older installer deletes old bundle then moves new bundle; current installer not established |

## Doubao selection and refresh

The byte-matching installed settings binary has this focused chain:

1. Activation entry `0x1001ceca8` resolves the parent and tests its enabled property. The enabled branch calls selection at `0x1001cedc8`.
2. Selection routine `0x1001cf1e4` resolves the Pinyin source and dispatches selection to the main thread. If resolution produces no child reference, the selection routine returns failure without running the refresh; earlier orchestration handles registration/enablement separately.
3. After the selection attempt, it calls refresh routine `0x1001cf4c8` before comparing the selection status with `noErr`. The refresh constructs a task to restart `TextInputMenuAgent` and `TextInputSwitcher`, waits and observes task outcome. Thus this menu refresh is not restricted to successful selection.

The enabled entry does not prove a refresh on every keystroke or a recurring background repair of every client's cache. A helper/entry return value must not be presented as another application's successful switching acceptance.

The installed `Resources/update.sh` stages complete contents, checks permissions and other users' hosts, stops matching current-user input-method processes, then promotes through a backup slot. At the successful replacement tail it conditionally closes System Preferences and restarts SystemUIServer. It retains a missing-Contents interval and does not directly express per-application keyboard-cache cleanup or a source-disable operation in the inspected shell flow. Its outer updater may do more; do not infer a complete uninstall transaction from this update script.

A focused `AppleEnabledThirdPartyInputSources` routine at `0x1001c78e4` synchronizes preferences, reads using `CFPreferencesCopyValue` with host variants and falls back to a disk plist read. This is not evidence of writing that key or globally flushing other clients' keyboard caches. Finding a preference string does not establish direct preference modification.

## Chatterfly entry paths and timing

The current helper's original arm64 slice is byte-bound to the retained analysis target. The following paths must remain distinct.

### Explicit older helper argument paths

`_main` dispatches `useSogouAuth` to `selectSogouAndShowIconWithAuth`. That method deselects/disables, attempts enablement and, only when enable reports success, starts observation, captures a pasteboard string, force-kills TextInputMenuAgent and pboard, starts a one-second process/selection timer, and attempts to restore the captured string. The process helper resolves force-kill to `/usr/bin/killall -9 <name>`, discards output and waits when launch succeeds. The recovered kill method does not verify the termination status. String preservation is not full clipboard preservation.

`enableOS12` and `checkAndEnableOS12` reach `enableSogouInputMethodForOS12` at `0x100010b04`. It performs deselect/disable synchronously on main and, on macOS >=13, force-kills pboard before attempting enable. The method's name does not limit execution to macOS 12.

These are concrete static command-dispatch paths in the current shipped helper. They do not establish which path the current installer invokes in this session, successful preference persistence, or actual remediation of a stale WeChat cache.

### Newer selection/retry code with unresolved entry reachability

`selectSogouInputMethodNow` at `0x100010bb0` initializes 0.5-second retries and a 5-second timeout. It attempts selection immediately, then uses enabled-source notifications and a timer if needed. `trySelectYuanbaoOnceAndReport` at `0x100011620` null-checks the child match, calls `TISSelectInputSource`, and only after `noErr` sets its once-success guard, stops retrying, reports success and calls refresh.

That refresh schedules a 200-millisecond callback on a global queue, then force-kills pboard and TextInputMenuAgent. This is a complete shipped implementation, but its newer entry has no resolved direct callers in the focused analysis. In particular, helper argument `selectNow` calls the older `selectSogouAndSwitchOut:` path, not this newer entry. Dynamic Objective-C dispatch remains possible: report reachability unresolved, not dead code or verified installer behavior.

A focused helper routine reads `com.apple.inputsources.plist` and `AppleEnabledThirdPartyInputSources` through NSDictionary. No write occurs in that routine. Parent/child disable/enable paths are implemented, but their current-OS acceptance cannot be inferred from static calls.

## Direct-cache search and interpretation

A read-only scan covered 49 selected Mach-O/plist/shell files under the installed Doubao bundle and 41 under Chatterfly. It searched ASCII and UTF-16LE literals, including `IntlDataCache`, `.kbdx`, `islcInputSourceCacheInvalidate`, `/var/folders`, `cfprefsd` and `TSMLanguageIndicatorEnabled`. None of those markers appeared. This excludes an obvious literal implementation in the scanned files, not generated paths, compressed/downloaded components or indirect platform effects. `pboard` matches elsewhere can also originate from ordinary clipboard APIs; only traced command paths establish a restart.

The inspected executables do not declare App Sandbox or `get-task-allow`. That signing observation does not establish effective system preference-write permission or a working policy bypass.

The [Apple cache analysis](input-source-cache-prevention.md) shows shared pasteboard modification metadata is consulted by the native cache layer. Chatterfly's pboard restart could therefore affect more than the visible menu. This is a mechanism hypothesis: the static dependency does not prove that restarting pboard invalidates every mapped client cache or persistent cache pair. No such restart was executed here.

The old caches in our successful missing-Arion reproduction already contain Doubao and Chatterfly. Their lookup success in that snapshot does not prove their installers always invalidate caches correctly; source availability at cache creation and installation order are confounders.

## Implications for Arion

Retain the applicable practices: stable parent/child identities, exact source resolution, explicit statuses, observation-driven bounded waiting and separation of discovered/enabled/selected/connected states. Keep the proposed safe removal and continuous-path replacement work from the [prevention report](input-source-cache-prevention.md).

Do not copy service restarts into routine activation as an established crash fix. Chatterfly's use of pboard is the strongest new candidate for a controlled recovery experiment, with potential clipboard effects; it remains a candidate. In an isolated account or VM, compare unchanged baseline, input-menu-only restart and pboard-plus-menu restart against the same accepted old control/component pair. For each treatment, verify existing and newly launched owned clients, public enabled/all-installed lookup, actual HUD switching, source selection, cache/control hashes and clipboard-type survival. Never substitute a newly created fixture or mismatched component-only copy for the existing stale client.

This report identifies competitor implementation paths. It does not establish competitor immunity, a reliable public global-refresh API, or a new Arion product repair.


## Analysis validation

The current Chatterfly helper produced eight focused complete REA records, including its command entry, version-gated enablement, retry selection, preference reader and kill helper. Doubao's byte-matching prior selection/refresh dossiers were reused; the new pass returned complete caller, activation-entry and preference-reader records. Its importer logged a nonfatal internal analyzer exception, but the requested records returned before shutdown and were inspected. An interior-address request mapped to a different canonical function and is excluded from activation-entry claims. Owned analysis sessions/processes were cleaned up. Evidence hashes and Markdown whitespace were checked; no product runtime acceptance or service-restart treatment was performed.


## Documentation-based suitability review

The competitor workarounds are understandable recovery experiments, but the reviewed Apple documentation does not justify routine automatic service termination as an input-source cache refresh protocol. This is an engineering suitability assessment, not a claim that Apple explicitly prohibits every operator-initiated restart or that use of `killall` necessarily causes distribution rejection.

| Mechanism | Documentation and evidence assessment | Arion recommendation |
| --- | --- | --- |
| Register, enable, select, observe and re-query | Public TIS operations with distinct preconditions and main-thread requirements | Use and verify each state; do not equate the caller's success with every client refreshing |
| Restart input-menu agents after a diagnosed discovery problem | Targets relevant UI components, but no reviewed contract guarantees rebuilding other clients' persistent caches | Consider a bounded, user-initiated recovery only after isolated validation; do not run on every activation |
| Force-restart pboard | Shared cross-application service; no reviewed pasteboard/input-source contract makes this a global keyboard-cache reset | Keep as an isolated experiment candidate; do not add to normal activation or update by default |
| Save/restore a single clipboard string around restart | Does not preserve the documented multi-item, multi-representation model | Do not describe it as lossless clipboard protection |
| Disable/re-enable cycle | Enable/disable are public state changes, not documented cache-reset operations; keyboard deselect is ineffective by contract | Use only for a justified lifecycle transition with observed results, not a blind repair loop |

### Pasteboard scope and preservation

Apple's current [NSPasteboard documentation](https://developer.apple.com/documentation/appkit/nspasteboard) states that its server is shared across running applications and handles multiple items and data types. The general board also participates in Universal Clipboard. The archived [Pasteboard Programming Guide](https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/PasteboardGuide106/Articles/pbConcepts.html) additionally describes dragging, Services, named boards, multiple representations and promised data. Thus restarting the server has a wider scope than refreshing the input menu. These contracts do not promise restart-safe clipboard or transfer state.

The current [NSPasteboardItem contract](https://developer.apple.com/documentation/appkit/nspasteboarditem) says items become stale after ownership changes and should be used within a single interaction. The [changeCount contract](https://developer.apple.com/documentation/appkit/nspasteboard/changecount) supports checking whether ownership has changed. It does not promise a global restart-spanning transaction or an atomic backup/restore operation. Saving only a string omits other items/types; an unconditional later restore can also overwrite newer user content. This consequence follows from the ownership model; such loss was not intentionally induced here. Even an eager multi-type snapshot must not be advertised as preserving all lazy providers, named pasteboards and in-flight transfers through daemon termination.

The installed `pboard(8)` manual describes the pasteboard server, exposes no configuration options, and says users should not start it manually. This is not an explicit prohibition of terminating it, and supplies no input-source recovery contract. The installed `killall(1)` manual documents sending process signals, with TERM by default; executing a command successfully is not evidence of successful keyboard-cache invalidation. Chatterfly's traced force-kill flag requests SIGKILL, so it is a materially stronger action than the default termination request.

### Input-source contract limits

The installed SDK `TextInputSources.h` remains the direct TIS contract. Registration describes necessary cache rebuilds so the installer can obtain source references immediately, not an assurance about every application's existing mapping. Source-change notifications report state changes; their declarations do not prescribe forging notifications to reset caches. `TISDeselectInputSource` is intended for palette/ink sources and has no effect on other source types, so the keyboard-child deselect observed in Chatterfly cannot be treated as a supported keyboard reset operation. Disable/enable require actual state verification, particularly after the policy-denied write observed for Arion.

The inspected shared pasteboard dependency in Apple's native cache code makes Chatterfly's restart hypothesis plausible, but that implementation observation is not a public API guarantee. The narrow prior WeChat component-cache recovery has better local efficacy evidence than a service restart: the former was actually followed by a user-confirmed successful retry; the latter has not been tested against the accepted old cache pair.

For Arion, prioritize valid bundle continuity and safe removal, public lifecycle APIs, bounded state observation and native switching acceptance. If adding a repair action, first prove the least invasive effective treatment in a disposable account, including rich/multi-item clipboard, promised data and concurrent copy behavior. Document supported OS scope and failure results. Do not claim a general global-cache reset or permanent prevention on the strength of competitor implementation alone.

This review used Context7's Apple AppKit documentation plus directly retrieved official documentation JSON, the archived guide and installed SDK/manuals. The Apple DTS forum thread used in earlier research could not be freshly revalidated: the thread fetch timed out or returned a Security Verification page. No new staff quotation or recommendation is inferred from that response. No service was restarted and no clipboard content was read for this documentation review.
