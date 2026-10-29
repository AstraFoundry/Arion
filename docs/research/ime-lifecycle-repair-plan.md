# macOS input-method removal and update repair

## Decision and scope

Separate source removal from update host shutdown. Removal should first attempt public programmatic disablement, retaining valid installed bytes until the source is independently observed unavailable for selection. Manual System Settings removal is recovery for a denied or unverified automatic attempt, not the required normal flow. Updating an existing bundle should exchange complete directories atomically, with reverse exchange for rollback. Registration, host readiness and other applications' cache coherence remain separate checks.

The working tree now implements the removal gate, atomic exchange/reverse rollback, cross-process exclusion, transaction journal and automatic update scheduling described below. Signed local acceptance verified update, rollback, removal after system confirmation, reinstallation and physical Pinyin input. This is not yet a published release and does not establish permanent prevention of the [stale-client failure](input-source-cache-prevention.md). The comparison tables below describe the original baseline and repair rationale; see [acceptance report](ime-lifecycle-acceptance.md) for actual execution limits.

## Current defects and preserved behavior

| Current source | Finding | Proposed change |
| --- | --- | --- |
| `macos/ime.go`, `releaseIME` | Switches away and stops the host, intentionally preserving source enablement | Retain this update preparation behavior; rename it to express host shutdown rather than source removal |
| `macos/ime_install_darwin.go`, `removeInputBundle` | Calls that preparation then deletes the bundle without verified system removal | Require a separate removal transaction and unavailable-for-selection gate before deletion |
| `macos/ime.go`, `QueryIME` | Returns absent when bundle files are missing, before checking native state | Keep product file status, but add independent native source observations for removal; absence of files is not absence of an OS entry |
| `macos/ime_install_darwin.go`, promotion/rollback | Two renames temporarily leave the registered destination absent | Use supported atomic exchange for existing complete bundles in both directions |
| `macos/ime.go`, `InstallIME` | Validates source before copying; existing readiness binds executable identity | Validate the completed staged copy too; preserve cancellation, signature checks and expected-host verification |
| `bootstrap/actions.go`, removal | Synchronous removal with no explicit pending-removal result | Make potentially waiting work asynchronous and represent removal pending separately from source state |

Keep the stable parent/child IDs and final installed path. Do not introduce per-release IDs, blanket disable/re-enable loops, cache deletion or service termination as part of these repairs.

## P0: removal transaction

The public `TISDisableInputSource` contract makes the source unavailable for selection and removes it from system UI. It is intended for input methods/source-supplying applications. This supports a programmatic removal path; the reviewed contract does not require a manual System Settings visit, and also does not promise that modern policy will always permit silent completion. The earlier denied write is evidence about Arion's actual desktop caller, not proof that every legitimate process context is forbidden.

The preferred product flow is automatic first, independently verified, with manual recovery only when necessary. Automatic means continuation of the user's explicit Remove action, not unsolicited removal on ordinary startup. Use a short-lived signed maintenance process so a rejected request's misleading TIS cache does not poison the long-running desktop state. Test the actual signed helper/IME-host maintenance context before selecting the production implementation; changing processes is not proof of additional permission and does not justify bypassing platform policy.

1. **Begin explicit removal.** Exclude activation/install/update, cancel owned voice composition and stale continuations, record an operation generation, and inspect the installed identity plus native parent/child/current-source state. If selected, switch to an already enabled/selectable alternative and verify the result. Do not treat keyboard `TISDeselectInputSource` as that switch.
2. **Attempt programmatic disable with bytes retained.** Resolve the exact own child/parent using public TIS APIs, check enable capability, and request disable of the visible child and parent with bounded, serialized/main-thread execution. Record native statuses and caller/build identity; never treat `noErr` or caller-local flags as persistence proof. Do not write preference files or restart global services. A short-lived maintenance process exits after returning its observations. The separate observer in the next steps decides whether deletion can proceed.
3. **Observe actual disablement/removal.** An enabled-source notification wakes a fresh query; it is not proof of removal. Confirm that Arion's selectable child is no longer available in a successful enabled-keyboard-source enumeration and is not the current source. Inspect parent/child flags through a successful all-installed enumeration as context. Do not require the IDs to disappear from all-installed lists: registered disabled sources can remain there, and an enabled nonselectable parent alone is not equivalent to an available child.
4. **Require independent agreement.** Run a second fresh, read-only, signed process, separate from the mutation process, using public TIS enumeration and current-source APIs. Compare its own parent/child observations with Arion's main-thread snapshot. Prefer successful unfiltered enumeration over interpreting a null filtered list as removal; null/failure, conflicting views and unavailable helper results remain unknown and retain files. Querying native state must bypass the installed-file gate.
5. **Keep manual recovery conditional.** If policy rejects the request, independent observations disagree, authorization remains incomplete, or the deadline expires, keep the bundle and report `pending_system_removal` with the existing Keyboard settings action. Manual completion can resume the same independent verification. Do not present a successful uninstall, silently delete first, or continuously retry mutations. Source state and operation state remain separate.
6. **Finalize under exclusive ownership.** Reacquire the cross-process maintenance lock and recheck the operation generation, actual bundle identity, selection availability and process state. If a different installer changed the bundle or the source was reselected, retain bytes and report the conflict. Stop the matching host and verify exit; if the OS relaunches it or it cannot exit, retain files and report incomplete removal. Only then delete the exact verified installation and check the filesystem result.
7. **Cancel safely.** Cancel/window close/app quit revokes pending continuation. Keep files and the actual source state; do not silently re-enable a source the user already removed. Restart reads pending transaction metadata and exposes recovery, rather than deleting files in the background.

A fresh helper is a comparator, not a global authority over every client's cache. Neither TIS and filesystem operations nor user preference changes form one atomic transaction. This gate reduces the specific enabled-entry/deleted-bundle risk but cannot prove that every already-open HUD has discarded an old ID. Conflicting clients require separate native recovery acceptance; do not advertise universal cache invalidation.

Proposed removal outcomes: `completed`, `pending_system_removal`, `state_conflict`, `host_exit_failed`, `delete_failed`. Keep these orthogonal to copied/discovered/enabled/selected source states. A failed deletion is not success; a disabled-but-still-installed bundle remains visible as retained files.

## P1: update transaction

Do not implement update as remove then install, and do not disable the source just to replace its files.

1. **Preflight and stage.** Acquire a per-user cross-process maintenance lock shared by app/probe/helper. Capture installed build identity and selection intent. Copy complete new bytes to private transaction storage outside the input-source scan directories, on the same filesystem as the installed path. Verify the completed staged copy's metadata, Developer ID signature/profile consistency and executable/resource identities before interrupting the old host. Check actual volume swap support. Do not register staging or backup URLs.
2. **Prepare handover.** Cancel owned composition, switch away when needed, and verify old host exit. Preserve source authorization. An in-process mutex does not exclude a separate probe or the OS relaunching the IME; monitor actual host identity and treat conflicts explicitly.
3. **Promote atomically.** For an existing bundle, exchange staged and installed directory names with `renameatx_np(..., RENAME_SWAP | RENAME_NOFOLLOW_ANY)`, using anchored directory descriptors and verified paths. The installed path then contains the new build; the transaction path contains the complete previous build. Both complete builds remain available. First install has no old name to exchange: use a separately checked exclusive promotion, not the existing-destination swap path.
4. **Activate from the final path.** Register only the stable installed URL, observe public source state, and retain the current parent-first enable rules. Resume prior Arion selection only while the maintenance operation still owns that selection intent; do not overwrite an intervening user choice of another source. Launch the expected host and verify the existing build-bound connection. When only resources change, identical executable hashes alone do not prove a new process; retain actual old-process-exit and new-connection evidence, plus transaction resource identity.
5. **Commit after observed readiness.** Do not treat successful rename, registration or enablement alone as completion. First-install discovery/consent can remain pending with valid bytes retained, preserving current behavior. A replacement of an already operational source must meet expected readiness or enter rollback/recovery. Retain a bounded known-good copy outside source scan roots until commit and recovery policy permit cleanup.
6. **Rollback symmetrically.** If new activation fails, stop and verify exit of the failed new host, then reverse the same exchange. Re-register only the restored final path and verify the old build/selection according to current user intent. Report original and restoration errors separately. If a host cannot stop or reverse exchange fails, preserve both builds, final-path bytes and transaction metadata; do not delete recovery copies or claim rollback completed.

If swap capability is absent/unknown or the syscall fails, report an explicit unsupported/failure outcome and keep the old installation. Do not silently fall back to the current two-rename gap. Atomic name exchange neither pins multiple separate resource reads to one generation nor replaces the need for host shutdown and signature validation.

## Interruption and crash recovery

Use a small private transaction record containing an operation token, old/new executable and resource identities, verified paths and intent. Proposed phases are `prepared`, `exchanged`, `activated`, `committed`, `rollback_pending`. Atomic metadata-file publication and a defined synchronization policy need their own crash tests; filesystem namespace atomicity alone is not a power-loss durability claim.

A crash can occur between exchange and phase-record update. On restart, inspect actual signed bytes at both paths and their recorded identities rather than blindly replaying the last phase. Preserve ambiguous or externally modified files. Pending transactions must not cause automatic repeated registration/selection during ordinary startup. Committed cleanup must be idempotent and limited to positively identified Arion transaction files.

## Apple contract and owned validation

The installed SDK's `rename(2)` manual defines `RENAME_SWAP` as an atomic source/target exchange on supporting filesystems, and identifies `VOL_CAP_INT_RENAME_SWAP`. Apple's [volumeSupportsSwapRenaming](https://developer.apple.com/documentation/foundation/urlresourcevalues/volumesupportsswaprenaming) exposes the corresponding capability. `RENAME_EXCL` and `RENAME_SWAP` are mutually exclusive. Do not infer an equivalent continuous-directory-path contract merely from a high-level replacement method offering a backup option.

Context7 resolution for Darwin rename APIs returned unrelated libraries, which were excluded. Its Apple Foundation documentation supplied the volume-capability contract; installed SDK headers/manuals supplied exact flags and syscall behavior.

The native arm64 owned-fixture experiment on macOS 26.6.2 verified:

- APFS and positive swap capability; fixture and installed input-method parent are on the same device.
- Complete nonempty directory promotion and reverse exchange restore the expected content hashes.
- A missing-target failure preserves both complete directory contents.
- 2000 repeated exchanges with 160955 concurrent destination-stat observations produced no observed missing path or syscall failure.

The initial x86_64-process run is retained separately and is not Intel hardware coverage. Both runs used unsigned non-executable markers, not real app bundles. They do not cover different-parent-directory permissions, unsupported/cross-volume filesystems, signed-resource validation, crash/power loss, TIS state or real editor switching. Finite stat observations corroborate the documented primitive and do not prove all race properties.

Exact sources, observations and hashes are archived under [IME bundle-swap reproduction](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/reproduction/ime-bundle-swap-v1/README.md`) and [raw runs](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/runs/ime-bundle-swap-v1/README.md`).

## Implementation sequence and acceptance

| Priority | Work | Required acceptance |
| --- | --- | --- |
| P0 | Replace the inert-enabled-entry claim; define automatic-first removal and conditional pending recovery in IME-01 and ADR 006 | Spec/code agree; no claim that deleting an enabled bundle is harmless |
| P0 | Separate shutdown, source observation and final deletion; add helper result and async pending UI | Denied/cancelled/unknown/conflicting removal retains bytes; independently verified API/manual removal precedes deletion; reselected or relaunched host blocks cleanup; verified automatic disable needs no manual visit |
| P1 | Add a volume-checked native exchange adapter, staged validation and reverse exchange | Existing paths remain complete; first install is separate; unsupported syscall/capability, signature failure and rollback failure are explicit |
| P1 | Add cross-process maintenance locking and transaction recovery | Competing app/probe cannot interleave; interrupted/ambiguous transactions preserve both builds and user selection intent |
| P1 | Extend IME-A01/IME-A06 and IME-UPDATE-01 native acceptance | Selected/unselected update, resource-only update, denied/failed stop, failed handshake, rollback and process restart |
| P1 | Validate actual clients with persistent caches | Existing and newly launched owned sandbox/AppKit clients, then affected WeChat; menu plus Control+Space, remove/reinstall and new login |

Likely implementation areas are `apps/desktop/infrastructure/macos/ime.go`, `ime_install_darwin.go`, native bridge and probe; bootstrap removal orchestration and existing keyboard feature state/copy; `specs/007-ime-ranking.md` and ADR 006. The proposal does not yet alter those normative contracts or product behavior. Existing lifecycle tests must be revised to assert refusal to delete enabled/unknown sources, rather than a generic release callback standing in for verified removal.


## Silent-removal validation gate

The ability to call the public API is established; successful silent removal under this macOS policy is not yet established. This statement described the original proposal review; subsequent signed execution and its policy denial are recorded in the acceptance report.

Compare the desktop caller, a genuinely signed short-lived maintenance helper, and an IME-host maintenance entry using the actual shipped signing/launch context in a disposable account or explicit reversible native acceptance. The helper's runtime bundle identity and effective policy must be observed, not inferred from its filename or directory. Avoid running a second IMK server for a maintenance-only operation.

For each attempt, retain the original enabled/selected state and valid bundle, request only the exact own IDs, record OSStatus plus focused policy-denial logs, then verify from a different fresh process that the selectable child is absent and current selection is different. Repeat verification after the mutation caller exits and another client starts. Test restoration/enablement and actual switching separately, since the historical failure involved rejected cache state on re-enable. If a supported context passes, use it as the normal one-click path; if none passes, preserve files and expose the already-designed conditional recovery. Root elevation, ad hoc entitlement changes, direct preferences mutation and pretending to be an Apple component are not established solutions.

The two independent facts remain explicit: Apple provides a programmatic disable operation; this account's prior Arion desktop attempt failed to persist it. Competitor disable calls justify testing their legitimate architectural pattern, not declaring the permission issue solved.


## Comparison with the published route

GitHub release metadata identifies `v0.1.0-alpha.11` as the newest published release in the inspected set. Its tag resolves locally to commit `253d86130d9611d45069e623ad0291d786cf7f77`. The relevant `ime.go`, `ime_install_darwin.go`, `ime_activation_darwin.go` and `native.m` files are identical between alpha.10 and alpha.11. This comparison is against published tag source, not an assertion that the current dirty working tree is shipped.

| Area | Published alpha.11 route | Proposed route |
| --- | --- | --- |
| Installation | Validate source payload, stage, register/enable, observe state, verify expected host when operational; consent/discovery can remain pending | Preserve this route; add completed-stage validation and stronger transaction ownership |
| Update preparation | Cancel composition, switch away, verify old host exit, retain enablement | Preserve this; updating does not require disabling/removing the OS source |
| Existing-bundle promotion | Rename installed directory to backup, then staged directory to installed path | One capability-checked atomic exchange of the complete directories |
| Update readiness | Registration/state and expected executable connection already required | Preserve these checks; add transaction/resource identity and intervening user-selection handling |
| Rollback | Stop failed new host, move failed bundle aside, move old bundle back, restore registration/selection; failures can preserve copies | Stop failed host then reverse exchange, retaining explicit transaction recovery state |
| Explicit removal | Switch away, verify host exit, `RemoveAll`; deliberately does not disable | Automatic-first public disable in validated short-lived context, independent observation, then verified host exit and delete |
| Removal failure | Switch-away/host-stop failure preserves files; successful release permits deletion without an OS-disable gate | Denied/unknown/conflicting disable preserves files; manual recovery only when automatic completion cannot be verified |
| Concurrency and restart | In-process mutex and UI busy checks; no persistent install transaction journal in the inspected replacement implementation | Cross-process maintenance lock, operation generation and identity-based interruption recovery |
| Source-state observation | Primarily Arion's process; `QueryIME` first gates on installed files | Separate file presence from native availability; fresh independent observer for removal completion |

The installed main executable embeds the alpha.11 commit but also `vcs.modified=true`, and its plist uses a generic `0.1.0`; these do not establish byte identity with the public release asset. The investigated diagnostic packaging added visibility without implementing the proposed removal/exchange transactions. The implemented local repair is not represented as a published release. The [published source](https://github.com/AstraFoundry/Arion/blob/v0.1.0-alpha.11/apps/desktop/infrastructure/macos/ime.go) and [replacement implementation](https://github.com/AstraFoundry/Arion/blob/v0.1.0-alpha.11/apps/desktop/infrastructure/macos/ime_install_darwin.go) bind the comparison.
