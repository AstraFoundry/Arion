# Update notifications and optional automatic installation

Status: source research and original proposal. The subsequent authorized implementation is described in [update lifecycle](../project/update-lifecycle.md), with current verification in [update validation](../quality/update-validation.md). The observations and unexecuted proposal details below are retained as research; the implementation documents and specs take precedence.

## Scope and evidence

The open issue search returned two issues, both without comments:

- [#12](https://github.com/AstraFoundry/Arion/issues/12): make updates discoverable during ordinary shortcut use; defer reminders until input completes, offer update/later, and suppress repeated prompts.
- [#13](https://github.com/AstraFoundry/Arion/issues/13): persist an optional automatic download/install setting; keep progress and failures visible and never interrupt input for a restart. Default and platform rollout are intentionally unspecified.

Arion evidence is the inspected working tree based on `e0983ab6a1aa7746f13d5022d9beb168de2dd838`, which already contained unrelated local changes. Magpie evidence is source inspection at `cec54c1d01f64cc0dd846d8561ab6dd92877b300`; its tests were read, not executed. OS documentation was retrieved through Context7 and official Apple/Microsoft pages. No native notification, update swap or cold-launch experiment was performed. Windows support must remain unverified until tested on Windows.

## Existing Arion implementation

| Area | Observed implementation | Consequence |
| --- | --- | --- |
| Scheduling | `apps/desktop/bootstrap/updates.go`: launch check, six-hour interval, fifteen-minute poll, thirty-second check timeout | Reuse it; shortcuts should trigger reminder evaluation, not a fresh network request |
| Trust | ADRs [010](../decisions/010-in-app-updates.md), [019](../decisions/019-center-update-channels.md); center stable/beta feed and MyGo archive/delta verification | Preserve the feed and built-in Ed25519 key |
| Installation | `third_party/mygo/updater.go`: `Install` downloads, verifies, extracts and immediately swaps the bundle/files | It is not a staging-only API; background calling it can replace resources while the old process runs |
| Restart | `apps/desktop/bootstrap/actions.go`: `RestartToUpdate` directly calls `mygo.App.Relaunch()` | Both manual and future automatic restart need a shared safety gate |
| Presentation | `ui/features/update/update.go`, tray/menu update entries | Keep these as persistent update access, including when OS notifications are disabled |
| macOS notifications | `third_party/mygo/internal/darwin/notify.go`: UserNotifications, launch-time delegate, first-show authorization, notification IDs | Much of the native transport already exists |
| Windows notifications | `third_party/mygo/internal/windows/tray.go`: `Shell_NotifyIconW`, `NIF_INFO`, tray callback | Toast-like appearance does not establish registered WinRT notification activation or cold launch |

Current installation goroutines use `context.Background()` and their progress/completion callbacks have no install-generation check. Channel changes are disabled by the UI while downloading/ready, but a new cancellation setting must enforce the rule in the controller too. The updater's rollback is a failed-swap recovery attempt; it is not health-checked rollback of a release that launches badly. Some rollback errors are ignored, so automated installation should not claim guaranteed recovery without strengthening and testing this path.

## macOS registration and delivery

Use local `UNUserNotificationCenter` notifications owned by the main `Arion.app`, whose existing identifier is `si.astrafoundry.arion`. Do not register the IMK host as another notification sender. A packaged app with a stable bundle identity is required by the current MyGo adapter. Local reminders need no APNs token, push server or remote-notification registration.

The native sequence is:

1. Attach the notification delegate during launch, before launch completes. Install the application-level click handler early and queue any activation until bootstrap is ready.
2. Query `getNotificationSettings`; distinguish unknown, denied, permitted and API failure. Re-query after returning from system settings instead of caching permission forever.
3. Request unknown authorization once at a natural idle point after sign-in or an eligible update. Request alert presentation; updates should be silent. Do not request again automatically after denial or interrupt active input. There is no separate application notification switch.
4. Submit a local request with a stable update identifier. Use a category and action identifiers for “View update” and “Later” if native action buttons are required.
5. Handle responses on the UI thread. Body/view activation opens the existing update view; later changes reminder policy without activating an unrelated window. Remove obsolete requests after install/channel changes.

Apple documents permission and settings separately from submission: successful submission is not evidence that a banner was visible. Focus modes and user settings remain authoritative. See [authorization](https://developer.apple.com/documentation/usernotifications/asking-permission-to-use-notifications), [settings](https://developer.apple.com/documentation/usernotifications/unnotificationsettings/authorizationstatus), [local scheduling](https://developer.apple.com/documentation/usernotifications/scheduling-a-notification-locally-from-your-app), and [delegate callbacks](https://developer.apple.com/documentation/usernotifications/unusernotificationcenterdelegate).

MyGo gaps to address through a reviewed maintained patch:

- Expose explicit permission querying/requesting independently of `Show`. `NotificationsSupported()` currently means platform/bundle capability, not user authorization.
- Honor the authorization callback's `granted` and error values. The current backend ignores both and proceeds to submit; relying only on submission error is insufficient for accurate permission state.
- Expose action/category identifiers if notification buttons are adopted. The existing delegate forwards only the notification ID, losing the response action identifier.
- Verify cold-click delivery using `App.OnNotificationClick`, not only a per-instance `Notification.OnClick` callback: an old notification has no corresponding in-memory object after relaunch.

No private notification database edits or forced settings changes are proposed. Direct notification-settings navigation is optional and needs OS verification; a manual System Settings instruction is sufficient.

## Windows registration choices

### Recommended target: native WinRT toast with explicit desktop registration

For the existing unpackaged Go/MyGo application, implement a small Windows adapter using `Windows.UI.Notifications` and a COM activator. It can use a narrow native boundary without changing the UI shell. The installer and application must agree on one stable AUMID and activator CLSID.

Required lifecycle:

1. Installer creates a Start menu shortcut with `System.AppUserModel.ID` and `System.AppUserModel.ToastActivatorCLSID`.
2. Register the activator's executable command in the appropriate per-user `Software\Classes\CLSID\{CLSID}\LocalServer32` location for a per-user installation. Quote the executable path. Decide all-user installation behavior separately rather than writing inconsistent HKCU/HKLM identities.
3. At startup, initialize the COM/WinRT context and register the activator class factory; send through a notifier using the same AUMID. Setting only the process AppUserModelID does not supply this activation infrastructure.
4. Accept only fixed view/later action values and bounded channel/version identifiers. Route cold activation to the existing instance, or queue it until bootstrap is ready. Activation arguments never authorize downloading a URL or running a command.
5. Use tag/group replacement and notification history removal for stale reminders. On upgrade preserve identity and refresh executable registration; on uninstall remove owned notifications, shortcut and COM registration.

Microsoft's [desktop C++ registration guide](https://learn.microsoft.com/en-us/windows/apps/design/shell/tiles-and-notifications/send-local-toast-desktop-cpp-wrl) describes the shortcut and activator contract; its older URL was available in search results but returned 404 when opened directly. The [compat registration API](https://learn.microsoft.com/en-us/dotnet/api/communitytoolkit.winui.notifications.desktopnotificationmanagercompat.registeraumidandcomserver) independently documents AUMID/CLSID/LocalServer32 registration. These sources establish the architecture, not a tested Go binding. A Windows harness must validate the actual bridge, COM lifetime and installer before adoption.

### Alternative: Windows App SDK AppNotificationManager

`AppNotificationManager.Register()` handles COM registration and shell display identity for unpackaged apps. Subscribe to `NotificationInvoked` first. Normal exit uses `Unregister`; permanent removal uses `UnregisterAll`. Packaged MSIX apps instead declare the activation/COM extensions in the package manifest. See [Register](https://learn.microsoft.com/windows/windows-app-sdk/api/winrt/microsoft.windows.appnotifications.appnotificationmanager.register) and [UnregisterAll](https://learn.microsoft.com/en-us/windows/windows-app-sdk/api/winrt/microsoft.windows.appnotifications.appnotificationmanager.unregisterall).

This reduces registration code but adds Windows App SDK deployment and Singleton-package dependencies to the current Go distribution. It does not require adopting WinUI, but runtime packaging still needs evaluation. Prefer this route only if those dependencies are accepted in a decision record. The [quickstart](https://learn.microsoft.com/en-us/windows/apps/develop/notifications/app-notifications/app-notifications-quickstart) also excludes elevated apps and describes callback-based COM launch. Its prose and sample activation-kind branches are not fully aligned; test the chosen SDK version's real cold-start path instead of copying the sample blindly.

The current MyGo tray balloon can support an initial running-process reminder, but must be declared as that capability. Do not silently substitute it for registered actionable notifications after a registration failure. No Windows push service is needed for local update reminders.

## What Magpie actually does

Sources at the pinned revision:

- [update.go](https://github.com/yetone/magpie/blob/cec54c1d01f64cc0dd846d8561ab6dd92877b300/internal/gui/update.go)
- [update_wait.go](https://github.com/yetone/magpie/blob/cec54c1d01f64cc0dd846d8561ab6dd92877b300/internal/gui/update_wait.go)
- [automatic-check tests](https://github.com/yetone/magpie/blob/cec54c1d01f64cc0dd846d8561ab6dd92877b300/internal/gui/update_auto_test.go)
- [automatic-restart tests](https://github.com/yetone/magpie/blob/cec54c1d01f64cc0dd846d8561ab6dd92877b300/internal/gui/update_autorestart_test.go)

| Observed behavior | Relevance to Arion |
| --- | --- |
| `NoAutoUpdate`, interval, `NoAutoDownload`, and `AutoRestart` are separate controls | Keep checking, downloading/installing and restarting distinct |
| Initial five-second delay, then minute polling; test default is six hours | Reuse Arion's existing schedule; these exact timings are not requirements |
| `Stage`/`StageBinary` prepares the update; `install` swaps it later | This separation is the most useful implementation pattern |
| Busy detection includes requests/tools plus ten seconds of quiet; wait polls each second and expires after an hour | Use an Arion-specific busy gate, not Magpie's gateway predicate |
| `waitGen` invalidates canceled waits; `autoSkip` suppresses automatic restart for the canceled version | Use generation checks and version-scoped suppression; persist Arion reminder suppression |
| Tests cover ready-only restart, no concurrent waits, busy-to-idle transition, cancel and a newer version | Reproduce equivalent properties using Arion session/IME fixtures |
| Source has `blocked` versions and Windows launch-failure handling | Do not attribute this stronger recovery behavior to MyGo |

Limitations matter: `autoSkip` is an in-memory field in the inspected updater. Also, `autoTick` returning when the setting is disabled does not itself cancel an already-running wait; final authorization and busy rechecks should be explicit in Arion. The inspected wait checks busy before a potentially lengthy feed recheck, so Arion must close that race before acquiring its installation/restart gate. Magpie's “restart now” override and gateway handover are not appropriate defaults for a text input application. This inspection establishes update behavior, not Magpie's OS notification registration behavior.

## Proposed Arion behavior

Accepted defaults: retain automatic checks on; automatic download/install off; automatic restart off and outside the first implementation. OS notification settings are the sole reminder/product-notice control, without application switches. Unknown OS permission is requested once at a natural idle point after an eligible update or sign-in; denial is not requested again automatically. This accepted policy supersedes the earlier separate reminder opt-in proposal. See [update lifecycle](../project/update-lifecycle.md) and [ADR 026](../decisions/026-account-product-notifications.md).

Automatic check and automatic install remain separate persisted controls. With checks off, nothing schedules background checks; a manual check only discovers a version, and automatic installation additionally requires both switches on. The UI explains this dependency. OS notification permission controls native presentation and never determines whether an update exists.

### Reminder policy for #12

- Keep the persistent tray/menu entry. A nonactivating, dismissible post-input reminder supplies a discoverable surface even without OS permission; OS-authorized system notification is an alternative presentation surface.
- Evaluate pending reminders after successful input commit and during ordinary idle transitions. Never show during preparing, capturing, finalizing, reviewing or committing, or while an IME composition is owned.
- Proposed suppression key: channel plus semantic version plus phase (`available`/`ready`). Persist last submission, explicit later deadline and dismissal separately from updater progress. A proposed later period is 24 hours; it is a product parameter, not an OS guarantee.
- Avoid simultaneously showing the native reminder and in-app reminder. Record “submitted” rather than “seen”; do not repeatedly resubmit because Focus hid it. Switching channels or installing clears obsolete notifications.
- Clicking a stale notification opens the current update view and validates the current channel/version. It never installs the old notification's payload. “Later” defers; explicit in-app dismissal can suppress that version until the user returns to the update view.
- Reminder text contains only app/version/update state, no dictated text. All visible copy belongs in existing English/Simplified Chinese catalogs.

### Safe automatic installation for #13

Extend the maintained MyGo updater rather than duplicating download, delta, signature or extraction logic. Proposed API shape: `Prepare(ctx, progress) -> PreparedUpdate`, then `Apply(prepared)` under a safe gate, with explicit discard/recovery support. Existing `Install` can remain a composition of these operations for compatibility. This is a proposed new patch, not an already approved dependency change.

State sequence:

`available → downloading → verified/staged → waiting-for-safe-install → applying → ready-for-restart`

Failures identify check/download/verify/apply/relaunch; cancellation returns to an actionable state. Never label staged bytes as installed. Persist a bounded, versioned staging record containing channel, version, target identity and verified artifact identity; after a crash, revalidate it before reuse. Apply verifies target and staged content again, serializes ownership, and records rollback failures explicitly. An interrupted multi-file Windows swap requires journal/recovery testing before automatic installation ships.

The safety gate must include core session ownership, pending text insertion, active keyboard composition, worker launch/resource loading and IME install/update/removal transactions. “Microphone is off” is not sufficient. Model provisioning and other managed writes should also block restart until canceled or flushed safely.

Acquire the gate atomically with blocking new input starts, then recheck settings, generation, version and channel immediately before apply/relaunch. If busy, retain the staged update and wait without stealing focus. Before allowing input again after apply, ensure resources and worker launches remain coherent with the running host. The simple claim that an open executable keeps running does not prove that later model/helper/resource loads are safe after bundle replacement. Until this is proven, retain staging and perform apply in the protected restart transaction; present this clearly as “downloaded, installation pending restart.”

Turning automatic installation off cancels an active automatic download, invalidates late callbacks and prevents applying staged data. Manual installation remains possible. If the atomic apply has already begun, finish or roll back that transaction; the UI must not promise mid-swap cancellation. A ready installed version cannot be “uninstalled” by toggling the preference. Changing channels invalidates staged work and reminders; never switch the transaction's target halfway through apply.

Manual Restart also uses the same gate and can queue until input is safe. A later independent automatic-restart option must persist cancellation for a version, recheck consent at execution time, use a quiet interval and never force restart after a wait timeout.

## Ownership and delivery order

| Priority / dependency | Concrete work |
| --- | --- |
| P0: policy and acceptance | Add accepted reminder/install requirements to `specs/001-desktop-input.md`; update ADR 010 and privacy disclosure only when behavior is approved. Keep ADR 019's feed/trust contract |
| P0: lifecycle safety | Add one application-owned update activity gate; wire authoritative session/IME/worker state in bootstrap; route manual restart through it |
| P0: #12 | Reminder policy, persisted suppression, localized post-input UI, view/later routing; reuse existing update and tray state |
| P0: native notification harnesses | macOS explicit permission/cold-click/action support; Windows registration, COM warm/cold activation and uninstall harness before choosing the production bridge |
| P1: #13 | MyGo prepare/apply patch, rollback/recovery tests, persisted preference and staged state, progress/cancellation/generation control; safe apply and explicit restart |
| P2: optional automatic restart | Separate opt-in after native lifecycle acceptance; persistent per-version cancellation and final atomic gate |

Keep domain/session rules in core and OS-independent contracts free of SDK types. A desktop infrastructure notification adapter owns platform calls; bootstrap connects it to the application policy and UI runtime projection. The settings feature displays state and dispatches actions, without importing infrastructure. SDK handles, registry paths and notification XML never enter FSD entity state. Generic MyGo notification/updater patches require `third_party/README.md`, provenance/patch notices and their regression gates to be updated in the implementation change.

## Acceptance and verification plan

These are proposed checks, not passed results:

- Controller tests with a fake clock: startup/six-hour scheduling unchanged, no per-shortcut checks, later/version/channel suppression survives restart, off/manual combinations, cancellation and stale generations, retry and signature failure.
- Busy-race tests: input starts between idle observation and apply; IME preedit exists without microphone activity; commit is pending; disabling a queued restart; busy begins during a recheck. No new work enters after the gate is acquired.
- macOS signed installed `.app`: fresh allow/deny, permission change, Focus, foreground/background, notification-body/actions, quit then click, update then click an old reminder, no focus stealing, IME/worker resource coherence.
- Windows installed standard-user app: correct sender identity/icon, notifications disabled, warm/cold action activation, single-instance forwarding, upgrade path with spaces, Explorer restart, uninstall cleanup, locked file/antivirus and rollback/recovery failures. Validate each supported Windows version on that OS.
- Run `make updater-check` for the maintained patch and repository native/core checks for affected layers. Extend `docs/quality/e2e.md` with actual notification and active-input update scenarios; cross-compilation alone is not acceptance.

The research change itself requires only document/link/diff validation. No native test result or shipping readiness is claimed.
