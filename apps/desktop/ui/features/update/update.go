// Package update shows the newest release and installs it: a check on
// request, verified staging, reminders, and a protected restart.
package update

import (
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Card shows update progress and the separate check, download and reminder choices.
func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	u := store.State.Update
	kit.Card(c, tokens).Children(func() {
		status(c, store, tokens)
		if u.StateError {
			ui.Text(c, messages.Text(copy.UpdateStateFailed)).FontSize(theme.Caption).TextColor(tokens.Warning).Padding(0, 16, 10, 16)
		}
		if u.Phase == runtime.UpdateUnsupported || (u.Phase == runtime.UpdateFailed && u.Failed == runtime.UpdateStepRecovery) {
			return
		}
		kit.Divider(c, tokens)
		preferencesLocked := u.Phase == runtime.UpdateApplying || u.Phase == runtime.UpdateReady || (u.Phase == runtime.UpdateFailed && u.Failed == runtime.UpdateStepRestart)
		auto := store.State.Preferences.CheckUpdates
		kit.TipRow(c, tokens, messages.Text(copy.UpdateAuto), messages.Text(copy.UpdateAutoDetail), func() {
			if ui.Switch(c.Key("auto-update"), &auto).Label(messages.Text(copy.UpdateAuto)).Disabled(preferencesLocked).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.CheckUpdates = auto })
			}
		})
		kit.Divider(c, tokens)
		autoInstall := store.State.Preferences.AutoInstallUpdates
		kit.TipRow(c, tokens, messages.Text(copy.UpdateAutoInstall), messages.Text(copy.UpdateAutoInstallDetail), func() {
			if ui.Switch(c.Key("auto-install-update"), &autoInstall).Label(messages.Text(copy.UpdateAutoInstall)).Disabled(preferencesLocked).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.AutoInstallUpdates = autoInstall })
			}
		})
		kit.Divider(c, tokens)
		reminderPreference(c, store, tokens, preferencesLocked)
		kit.Divider(c, tokens)
		preview := store.State.Preferences.AllowPrereleaseUpdates
		locked := preferencesLocked || u.Phase == runtime.UpdateChecking || u.Phase == runtime.UpdateDownloading || u.Phase == runtime.UpdateApplying || u.Phase == runtime.UpdateReady || u.RestartQueued
		kit.Row(c, tokens, messages.Text(copy.UpdatePrerelease), "", func() {
			if ui.Switch(c.Key("prerelease-updates"), &preview).Label(messages.Text(copy.UpdatePrerelease)).Disabled(locked).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.AllowPrereleaseUpdates = preview })
			}
		})
	})
}

func status(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	u := store.State.Update
	actions := store.Actions
	button := func(variant kit.Variant, label string, enabled bool, action func()) {
		if kit.Button(c.Key(label), tokens, variant, label, enabled).Clicked() {
			action()
		}
	}
	check := func() { button(kit.Outline, messages.Text(copy.UpdateCheck), true, actions.CheckForUpdates) }
	available := func(action func()) {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Link(c, messages.Text(copy.UpdateNotes), config.ReleaseURL+u.Version).FontSize(theme.Caption)
			button(kit.Primary, messages.Text(copy.UpdateInstall), true, action)
		})
	}
	switch u.Phase {
	case runtime.UpdateUnsupported:
		kit.Row(c, tokens, messages.Text(copy.AboutUpdates), messages.Text(copy.UpdateUnsupported), nil)
	case runtime.UpdateChecking:
		kit.Row(c, tokens, messages.Text(copy.AboutUpdates), "", func() {
			button(kit.Outline, messages.Text(copy.UpdateChecking), false, func() {})
		})
	case runtime.UpdateNoRelease:
		kit.Row(c, tokens, messages.Text(copy.UpdateNoRelease), "", check)
	case runtime.UpdateCurrent:
		kit.Row(c, tokens, messages.Text(copy.UpdateCurrent), "", check)
	case runtime.UpdateAvailable:
		kit.Row(c, tokens, messages.UpdateAvailable(u.Version), "", func() { available(actions.InstallUpdate) })
	case runtime.UpdateDownloading:
		ui.Column(c).Padding(12, 16).Gap(8).Children(func() {
			ui.Text(c, messages.UpdateAvailable(u.Version)).FontSize(theme.Body).FontWeight(500)
			fraction := -1.0
			if u.Total > 0 {
				fraction = float64(u.Downloaded) / float64(u.Total)
			}
			ui.Progress(c.Key("update-bar"), fraction).Label(messages.Text(copy.UpdateDownloading))
			ui.Row(c).Children(func() {
				ui.Text(c, messages.Text(copy.UpdateDownloading)).FontSize(theme.Caption).TextColor(tokens.Muted)
				ui.Spacer(c)
				if u.Total > 0 {
					ui.Text(c, messages.Progress(u.Downloaded, u.Total)).FontSize(theme.Caption).TextColor(tokens.Muted).FontFeatures("tnum")
				}
			})
		})
	case runtime.UpdateStaged:
		if u.RestartQueued {
			restartQueued(c, store, tokens)
			return
		}
		kit.Row(c, tokens, messages.Text(copy.UpdateStaged), messages.UpdateAvailable(u.Version), func() {
			button(kit.Primary, messages.Text(copy.UpdateRestart), true, actions.RestartToUpdate)
		})
	case runtime.UpdateApplying:
		kit.Row(c, tokens, messages.Text(copy.UpdateApplying), messages.Text(copy.UpdateApplyingDetail), nil)
	case runtime.UpdateReady:
		if u.RestartQueued {
			restartQueued(c, store, tokens)
			return
		}
		kit.Row(c, tokens, messages.Text(copy.UpdateReady), messages.UpdateAvailable(u.Version), func() {
			button(kit.Primary, messages.Text(copy.UpdateRestart), true, actions.RestartToUpdate)
		})
	case runtime.UpdateFailed:
		if u.Failed == runtime.UpdateStepRecovery {
			kit.Row(c, tokens, messages.Text(copy.UpdateRecoveryRequired), messages.Text(copy.UpdateRecoveryDetail), func() {
				button(kit.Outline, messages.Text(copy.Quit), true, actions.Quit)
			})
			return
		}
		label, action := copy.UpdateCheckFailed, actions.CheckForUpdates
		switch u.Failed {
		case runtime.UpdateStepInstall:
			label, action = copy.UpdateInstallFailed, actions.InstallUpdate
		case runtime.UpdateStepPrepare:
			label, action = copy.UpdatePrepareFailed, actions.InstallUpdate
		case runtime.UpdateStepApply:
			label, action = copy.UpdateApplyFailed, actions.RestartToUpdate
		case runtime.UpdateStepRestart:
			label, action = copy.UpdateRestartFailed, actions.RestartToUpdate
		}
		detail := ""
		if u.Version != "" {
			detail = messages.UpdateAvailable(u.Version)
		}
		if u.Failed == runtime.UpdateStepRestart {
			detail = messages.Text(copy.UpdateRestartFailedDetail)
		}
		kit.Row(c, tokens, messages.Text(label), detail, func() {
			if u.Failed == runtime.UpdateStepCheck || u.Failed == "" {
				check()
			} else {
				button(kit.Outline, messages.Text(copy.Retry), true, action)
			}
		})
	default:
		kit.Row(c, tokens, messages.Text(copy.AboutUpdates), "", check)
	}
}

// Notice is the menu bar panel's way to an update found in the background:
// it opens Settings › About. Nothing shows without a pending update.
func Notice(store runtime.Store) (label string, ok bool) {
	u := store.State.Update
	if !u.Pending() {
		return "", false
	}
	return store.Copy().UpdateTo(u.Version), true
}

func restartQueued(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	kit.Row(c, tokens, messages.Text(copy.UpdateRestartQueued), messages.Text(copy.UpdateRestartQueuedDetail), func() {
		if kit.Button(c.Key("cancel-restart"), tokens, kit.Outline, messages.Text(copy.UpdateCancelRestart), true).Clicked() {
			store.Actions.CancelUpdateRestart()
		}
	})
}

func reminderPreference(c *ui.Context, store runtime.Store, tokens theme.Tokens, locked bool) {
	messages := store.Copy()
	permission := store.State.UpdateNotifications.Permission
	enabled := store.State.Preferences.SystemUpdateReminders
	detail := copy.UpdateSystemRemindersDetail
	switch permission {
	case runtime.UpdateNotificationsDenied:
		detail = copy.UpdatePermissionDenied
	case runtime.UpdateNotificationsFailed:
		detail = copy.UpdatePermissionFailed
	case runtime.UpdateNotificationsUnsupported:
		detail = copy.UpdatePermissionUnsupported
	}
	kit.TipRow(c, tokens, messages.Text(copy.UpdateSystemReminders), messages.Text(detail), func() {
		if ui.Switch(c.Key("system-update-reminders"), &enabled).Label(messages.Text(copy.UpdateSystemReminders)).Disabled(locked || permission == runtime.UpdateNotificationsUnsupported).Changed() {
			if enabled {
				store.Actions.EnableUpdateReminders()
			} else {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.SystemUpdateReminders = false })
			}
		}
	})
	if permission == runtime.UpdateNotificationsDenied || permission == runtime.UpdateNotificationsFailed {
		ui.Text(c, messages.Text(detail)).FontSize(theme.Caption).TextColor(tokens.Muted).Padding(0, 16, 10, 16)
	}
}
