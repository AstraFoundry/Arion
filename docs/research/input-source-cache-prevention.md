# Preventing recurring input-source selection crashes

## Conclusion and evidence boundary

Other applications can reach the same Apple input-source HUD failure. This is established by a public Youdao crash with the same stack, not just inferred from WeChat. On this machine, several other running applications map keyboard caches without Arion. That is a concrete risk indicator, not a prediction that those applications will crash. No personal application was intentionally switched or crashed in this research.

The [WeChat investigation](wechat-input-source-crash.md) establishes the current recovery: four HUD sequences reconstruct the correct Arion Pinyin child as the failed target; WeChat maps a cache without that source; explicitly authorized regeneration adds it; the user then confirms successful switching. A matched old-cache pair now reproduces failed Arion lookup in the owned sandbox fixture and restoring the fresh pair restores lookup. The exact event that originally failed to invalidate the cache remains unresolved. The native missing null guard is in Apple's client-side code, outside an ordinary Arion input-method callback.

## Other-application exposure

A read-only inventory of the exact `com.apple.IntlDataCache.le.kbdx` filename found 69 accessible caches under the current user's OS cache root: 42 without the Arion child bytes, 27 with them. Some entries belong to system helpers rather than user-facing applications. A separate open-file mapping check found running TextEdit, Mail, Calendar, Preview and QuickTime Player among the processes using caches without Arion. Other processes use the shared ordinary-process cache, which includes Arion. Sandbox status alone is not a sufficient trigger: the fresh owned sandbox fixture resolves Arion successfully.

These are byte-membership and open-mapping observations. They do not sample the client lookup result, fully parse source tables, show that a missing source can never be added through another route, or establish a crash rate. No other application cache was changed. The raw app inventory and cache paths remain in ignored local evidence storage.

A bounded search of 420 accessible top-level user/system `.ips` reports found five matching `TUINSCursorUIController` selection failures, all WeChat. This does not cover archived/removed reports or other accounts and does not establish immunity elsewhere.

Primary public evidence confirms broader exposure:

- [Qingjian issue 31](https://github.com/qingjian-team/qingjian/issues/31) contains a complete Youdao Dictionary 9.1.7 (924) report on macOS 15.7.7 with `CFRelease(NULL)`, `_selectCurrentInputSource +84` and `moveTextInputMenuHUD +272`. The reporter additionally names WeCom and WeChat without their complete reports. A different input method was involved; the same stack does not prove the same cache trigger.
- [Squirrel issue 951](https://github.com/rime/squirrel/issues/951) contains the same signature for WeChat 3.8.8.18 on macOS 15.0 beta. Reboot was reported to restore behavior; the report does not inspect keyboard caches or demonstrate permanent prevention.

Qingjian's mode-declaration repair is not directly transferable: Arion already declares a visible Pinyin child. One reporter's browser exception does not establish that browsers are immune.

## Cache invalidation and supported API limits

The installed SDK describes `TISRegisterInputSource` locating installed sources and rebuilding necessary caches so the installer can immediately enumerate them. It does not explicitly guarantee that every existing client or its persistent sandbox cache has refreshed. Enabled/selected notifications describe state changes; they are not documented global cache-reset commands.

Context7 resolution for Apple Carbon returned unrelated libraries, which were excluded. The reviewed SDK and [Apple DTS discussion](https://developer.apple.com/forums/thread/775526) supply the relevant contracts. DTS acknowledges that System Settings can need logout to discover a new source and reports no known way to avoid it. This is discovery guidance, not a guarantee that logout prevents every selection crash. No public per-application keyboard-cache flush API was found in these reviewed sources. Loaded system-code inspection also shows process-local mappings, explicit invalidation callbacks, directory modification checks and shared pasteboard modification metadata. Those paths explain why one process view is insufficient, but do not identify the missed event or establish a supported command for Arion to force every client to refresh.

The old-copy experiment must preserve the cache's control-file relationship. The native cache layer compares the component file length with the length in its sibling `com.apple.IntlDataCache.le` control file before mapping it. Copying only a 140020-byte old keyboard component under a fixture control file expecting 140436 bytes causes rejection and regeneration. The earlier single-file replay therefore did not test acceptance of a complete old cache; it cannot establish that valid stale caches automatically refresh. The original WeChat recovery still stands: removing the bad component causes regeneration and successful user-observed switching.

A subsequent controlled A/B/A comparison used a matching old TextEdit control/component pair copied only into the owned fixture's cache directory. Fresh baseline resolves both Arion IDs. With the matched old pair, enabled-only and all-installed enumeration return null and the same private lookup used by the HUD returns no reference for either Arion ID. The old pair remains byte-identical after the query, with no rebuild diagnostic. Restoring the original fresh pair restores successful lookup. Both TextEdit originals remain unchanged, and the fixture files are restored exactly. No selection operation or null release was invoked.

A second run includes positive controls: ABC and Doubao remain queryable through both public views and the private lookup under all three treatments; only Arion fails under the old pair. This excludes a wholesale sandbox/query failure caused by relocating the files.

| Cache treatment in owned sandbox | ABC / Doubao lookup | Arion parent / child lookup |
| --- | --- | --- |
| Fresh baseline | Found | Found |
| Matching old pair | Found | Null |
| Restored fresh pair | Found | Found |

This reproduces the lookup failure with a cache from an application other than WeChat and corrects the earlier single-component experiment's limitation. It establishes that a complete stale cache can survive fresh process startup and defeat lookup; it does not intentionally reproduce a TextEdit crash or identify which original lifecycle event missed invalidation. Raw pairs are private local evidence, not redistributable fixtures.

## Recommended prevention changes

These are proposed product changes, not implemented or verified crash-prevention fixes.

| Priority | Change | Reason and acceptance |
| --- | --- | --- |
| P0 | Separate destructive uninstall from update host shutdown | `releaseIME` switches away and stops the host but preserves enablement. `removeInputBundle` then deletes valid bundle bytes. Retain the bundle while removal is pending; require observed removal from enabled selection availability before deletion. |
| P0 | Treat denied disable as an incomplete removal | Prior logs show a policy-denied preference write despite the caller reporting disabled. Use the supported System Settings flow when needed. A successful status or one caller's cached flag is insufficient. |
| P1 | Preserve final path continuously during updates and rollback | Current promotion uses two renames with a missing-path interval. Evaluate supported same-filesystem atomic directory exchange, staged signature verification, old-host exit, build-bound connection verification and symmetrical rollback. This reduces a known risk; its causal role in these crashes remains unproved. |
| P1 | Preserve release parent/child IDs and mode metadata | Do not generate new IDs per release or change script/install scope speculatively. Standard user/system installation-directory inspection found one installed Arion bundle; this does not enumerate the full LaunchServices database. |
| P1 | Compare public enabled-only and all-installed lookups from fresh processes | Current native product enumeration uses all-installed queries in Arion; the crashed system helper first requests enabled-only sources inside its client. Record both views without claiming a fresh helper certifies every application's cache. |
| P1 | Add actual system HUD switching acceptance | Host IPC and text-insertion tests do not exercise another application's source lookup. Preserve native process survival and selected-source observations separately from API request success. |
| P2 | Prepare an Apple Feedback reproducer | Include correct HUD ID, null lookup path, cache/control identities, native crash stack and owned reproduction. The missing null guard requires an Apple fix. No report was submitted externally. |

Removal validation must bypass `QueryIME`'s installed-file gate: that method returns absent after file deletion even if an OS entry remains. Query native parent and child availability directly while valid bytes are retained, including fresh-process comparison. Registered-but-disabled sources can remain in all-installed lists; public unregister is not a prerequisite that can be demanded from this API.

Update `specs/007-ime-ranking.md` IME-01/IME-A01, IME-UPDATE-01/IME-A06 and ADR 006 together when implementing the revised lifecycle. The current 'inert enabled entry' assertion and the ADR's disable-before-delete requirement conflict. This research does not silently resolve their product contract.

Do not ship third-party cache deletion, forged notifications, `killall`, clipboard/preference-daemon restarts, preference-file imports or private TIS functions as prevention. Keep narrowly scoped cache isolation as a backed-up support recovery for a diagnosed affected application. Rebuilding WeChat's cache does not refresh other applications' separate caches.

## Native acceptance matrix

Run destructive lifecycle cases in an owned disposable account/VM, preserving application identity and caches across stages. Launching a brand-new fixture after each stage misses the actual failure class.

| Scenario | Required observations | Current evidence |
| --- | --- | --- |
| Stable install, existing affected client | Shortcut HUD cycles to Arion, correct selected ID, surviving client | WeChat recovery retry passed by user observation |
| Existing old-cache sandbox client | Public lookup views, cache/control hashes, menu and shortcut outcomes | Matched-pair owned A/B/A reproduces failed lookup; personal clients not switched |
| Fresh sandbox and ordinary clients | Enabled-only/all-installed lookups and real switching | Fresh lookup probes pass; full HUD matrix not run |
| First install after client cache creation | Existing and newly launched owned editors discover/select the child | Not run in an isolated account |
| Selected and unselected update | Complete destination throughout promotion; correct expected host; existing/new client switching | Not validated for crash prevention |
| Failed update and rollback | Complete old or new signed bundle, stable IDs, native switching after rollback | Not validated for crash prevention |
| Denied/manual removal | Pending state preserves bytes; observed source unavailability precedes deletion | Proposed contract; not implemented |
| Removal/reinstallation | Old/new clients and persistent caches, menu and shortcut switching | Historical failure exists; repaired lifecycle not tested |
| New login and clean account | Discovery/consent distinct from actual client switching | Not run |
| Framework diversity | Owned AppKit and sandbox editors; separately verified available third-party clients | No framework-wide immunity claim |

Use both menu selection and the previously failing Control+Space HUD path, including repeated cycling through other sources. Collect content-free source identities, native status, process/build identity and cache digests in research. A byte inventory is not a product health verdict. No new production diagnostic collection or package was introduced by this report.


## Competitor implementation follow-up

The [focused Doubao/Chatterfly comparison](competitor-input-source-refresh.md) traces service-refresh conditions and distinguishes Chatterfly's explicit helper argument paths from newer retry code with unresolved entry reachability. Both ship process-restart workarounds; neither establishes direct per-application keyboard-cache repair. Chatterfly's pboard restart is a candidate for isolated causal testing, not a verified Arion prevention measure.


## Concrete lifecycle repair design

The [removal/update proposal](ime-lifecycle-repair-plan.md) specifies retained-files removal pending, independent native availability checks, same-filesystem directory exchange with reverse rollback, and interruption recovery. Its owned native arm64 APFS fixture validates the exchange primitive; production lifecycle changes and client cache prevention remain unimplemented and unverified.
