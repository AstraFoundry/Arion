# Update lifecycle

Issues [#12](https://github.com/AstraFoundry/Arion/issues/12) and [#13](https://github.com/AstraFoundry/Arion/issues/13) are implemented through the existing center feed and maintained MyGo updater. Requirements are DSK-15–17 and acceptance DSK-A16–19. The [research](../research/update-notifications.md) records the source investigation and alternatives; this document describes the selected implementation.

## Controls and presentation

Automatic checks retain their existing default (on), launch check, six-hour interval and fifteen-minute feed polling. A separate one-second local policy tick handles idle reminders and queued restart; it does not check the feed on every shortcut. Automatic downloading defaults off. Both switches must be on for a scheduled check to prepare an update. Manual checking only discovers it. Downloading prepares and verifies; the About status explicitly says installation waits for restart. There is no automatic restart setting.

System reminders default off. Enable asks for OS authorization; normal notification delivery never prompts. The current OS permission is queried again when Arion becomes active. The persistent tray/About update entry remains available when OS notification delivery is disabled. The nonactivating in-app card and opted-in system notification are alternative presentation surfaces; neither repeats input text.

The content-free `updates.json` in the existing profile stores the staging handle/channel/version and up to 32 reminder decisions. A reminder key includes channel, version and available/ready phase. Submission suppresses repeats; Later allows one new reminder after 24 hours; Dismiss suppresses that phase. Visiting About clears dismissal but retains the last-submission suppression. A new version or newly downloaded phase is eligible independently. Submission is not a read receipt. Disk errors pause reminders and show a persistence warning.

## Transactions and safety

`bootstrap/updates.go` owns main-thread updater state and generations. `updatestate` owns private atomic persistence. `PreparedUpdate` owns signed archives/deltas and the staging directory next to the installation; it checks target/staged identities again before swapping. A completed receipt remains until the profile no longer refers to it and the application acknowledges it. Cold recovery validates the receipt rather than assuming that a missing stage means success. A failed acknowledgement restores the durable receipt handle; if restoration also fails, the updater enters explicit recovery. A cold process whose version and signed installed bytes match the receipt can retain normal input while receipt cleanup blocks further update transactions.

Restart acquires `Coordinator.TryMaintenance`, which shares the session lock with Start and counts asynchronous preparation, context, recognition, rewriting, commit and cleanup. The composition root also waits for model/resource jobs, microphone-meter shutdown, edit-tracker settlement and vocabulary writes. IME setup/removal cannot start during apply. Legacy input-method hosts do not report keyboard preedit, so an active editor client conservatively defers restart; opening About deactivates that client. The main-app update leaves the separately installed keyboard host running.

The maintenance lease remains held after bytes are replaced. A canceled MyGo quit sequence reports a failed restart with a retry action, retaining the lease so old code cannot load new resources. Quit is temporarily prevented while the swap or its protected preparation is in progress. Ordinary rollback discards the consumed stage and offers a new download. Interrupted transactions and rollback failure preserve their journal/backup and block further updates; the UI requests quitting and reinstalling the application. This is explicit repair handling, not a claim of automatic repair after an arbitrary crash.

## Native notifications

macOS uses the main app's existing bundle identity and UserNotifications. The maintained patch exposes permission query/request independently, custom actions and cold-launch action delivery. The main-app delegate handles a persisted notification ID even without an in-memory Notification object.

Windows uses native WinRT toast delivery and COM activation, with a deterministic AUMID/CLSID derived from the package identifier. `desktop.json` opts into installer registration through `windows.notifications`. The installer invokes exact registration/unregistration commands handled before application bootstrap; normal app operation is not elevated. The startup bridge forwards validated cold activation to an existing instance when necessary. A missing registration is an explicit error; there is no silent tray-balloon substitution.

Actions accept only previously persisted notification IDs and the fixed view/later actions. A stale notification opens the current About view; it cannot supply an installation URL or executable command. Channel changes and preference opt-out remove owned notifications; stale asynchronous completions remove their obsolete request again.

## Verification

- `pnpm updater:check`: maintained updater/signature/staging/rollback/receipt tests.
- `pnpm updater:e2e`: actual temporary native app packages, signed fixture feed, staging without replacement, process exit, recovery, apply, real relaunch and invalid-signature preservation. It never updates the installed Arion.
- `pnpm notifications:check`: notification API, payload/installer checks and platform builds.
- `pnpm notifications:prepare`: isolated notification probe. Follow its emitted commands for permission/delivery/cold-click observations; building it alone is not a native notification pass.
- `pnpm e2e:native`: actual production native reminder/settings windows, focus/capture and persisted Later/Dismiss, alongside existing desktop scenarios.
- `go test -race ./application` in `packages/core` and `go test -race ./bootstrap ./infrastructure/updatestate ./infrastructure/imebridge` in `apps/desktop`: controller/cancellation/resource lifetime regressions.

Native Windows delivery, cold activation, installer cleanup, real file-lock/antivirus behavior and recovery must be exercised on Windows. Cross-compilation does not establish these. Native fixture signing does not replace production notarized-package acceptance. Current measured results are in the [validation record](../quality/update-validation.md).
