# Update implementation validation

Scope: DSK-A16–19, maintained MyGo staging and native notification transport. Results below distinguish source/unit/build evidence from actual OS execution. No test replaces the production installation.

## Current OS-permission control revision

General and About no longer contain product-notice or system-reminder switches.
OS notification settings control delivery. The focused UI command below passes, covering both languages and all five OS
permission states with no notification controls, independent check/download
preferences, update delivery states and post-input reminder actions:

```sh
go -C apps/desktop test ./ui/app ./ui/shared/copy -run 'Test(NotificationControlsUseSystemSettings|UpdatePreferencesAreIndependent|UpdateDeliveryStates|PostInputUpdateReminder)' -count=1
```

`node scripts/check-i18n.mjs` and `node scripts/check-repository.mjs` also pass
(543 Markdown documents). Refreshed English/Chinese General/About, delivery and
reminder snapshots are under `.workspace/notification-os-permission-snapshots/`.
The full `go -C apps/desktop test ./ui/app -count=1` rerun passes (166 seconds).
Bootstrap, account-notice policy/client/store and update-state race tests pass,
including single in-flight authorization, one request per process, idle/profile
guards and deferred reminder persistence. `pnpm test` passes with the removed
notification preferences, including legacy JSON migration. Actual OS permission
presentation under the revised automatic scheduling and native Windows behavior
remain unverified. Earlier native observations below are historical evidence.

## Earlier verified implementation

- Core application race tests pass, including input exclusion, canceled-worker ownership and a canceled manual commit retaining ownership until its write finishes.
- Desktop bootstrap, update-state persistence and IME bridge race tests pass, including old-channel rejection, late download cancellation, staged-before-apply, persistence failure, completion-receipt acknowledgement failure and rollback retry/repair handling.
- Headless native UI tests pass, including both languages, staged/applying/queued/recovery states, the then-present notification enable actions and disabled controls during replacement.
- The owned macOS app fixture stages version 2 while version 1 and its resources remain unchanged; exits; cold-recovers verified bytes; applies; uses actual MyGo relaunch; and observes version 2 with its matching resource. An invalid signature leaves version 1 intact. Fixture apps are ad-hoc signed, not production notarized releases.

Repository, workflow, i18n, core and full desktop checks also pass. Native notification API/installer tests, all activation-envelope tests and Windows amd64/arm64 cross-builds pass; Windows-only COM/shortcut tests compile but were not executed.

## Native observations

- `pnpm e2e:native` passes on macOS arm64, including both new update scenarios: actual reminder windows retain the owned Settings window's focus, English/Chinese available/staged/ready states render, Later/Dismiss survive disk reload, and queued restart can be canceled. Settings staged/applying/queued states are captured. The first run exposed a nil update-source crash; the guard and regression test were added before the passing rerun.
- Existing fullscreen overlay, settings, onboarding and language native scenarios pass. Context capture, learning, save-dialog, system-paste and site-capture scenarios remain explicitly skipped by their existing permission/driver gates; this run does not claim those scenarios.
- The isolated macOS notification sender starts with unknown permission. Query and denied Show do not prompt; explicit request displays the actual OS permission prompt, and Allow changes permission to allowed. Notification submissions succeed. Banner display, action buttons and cold-click activation remain unverified because no visible notification was observed in this desktop session. Screen-sharing notification suppression was observed in system settings but is not established as the cause.
- The completed-receipt fixture cold-launches version 2, verifies the installed signed tree, clears the profile handle and acknowledges the receipt.

Local visual evidence is under `.workspace/e2e/darwin/`; isolated permission/event evidence is under `.workspace/e2e/notifications/current/`. These generated artifacts are not committed.

## Hosted CI fixture identities

Updater fixtures resolve their temporary installation directory before staging, matching `installTarget` in a relaunched process. Windows hosted runners expose temporary paths through an 8.3 user-name alias; comparing that alias with the resolved installation path previously prevented acknowledged-receipt cleanup in the fixtures. The native About capture uses the current display work-area height so AppKit can honor the requested bounds on smaller hosted displays. These fixture corrections retain the receipt-cleanup and exact window-size assertions. `node scripts/check-updater.mjs` and the complete `node scripts/run-native-e2e.mjs` pass on the development macOS arm64 host after the correction; existing permission/driver-gated scenarios remain skipped. Windows results still require the native CI run.

## Platform limits

Windows native notification delivery, cold activation, installer cleanup, locked-file/antivirus behavior and interactive application scenarios require a Windows host. Platform builds and API tests are not runtime acceptance. Interrupted swap refusal and preserved recovery data do not claim automatic repair. No production signed old-to-new Arion upgrade or clean-account OS permission matrix is claimed.
