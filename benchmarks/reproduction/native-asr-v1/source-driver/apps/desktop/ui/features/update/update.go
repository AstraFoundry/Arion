// Package update shows the newest release and installs it: a check on
// request, the download with its verification, and the restart into it.
package update

import (
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Card is the update status with its one action, and the automatic check
// preference.
func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	u := store.State.Update
	kit.Card(c, tokens).Children(func() {
		status(c, store, tokens)
		if u.Phase == runtime.UpdateUnsupported {
			return
		}
		kit.Divider(c, tokens)
		auto := store.State.Preferences.CheckUpdates
		kit.Row(c, tokens, messages.Text(copy.UpdateAuto), messages.Text(copy.UpdateAutoDetail), func() {
			if ui.Switch(c.Key("auto-update"), &auto).Label(messages.Text(copy.UpdateAuto)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.CheckUpdates = auto })
			}
		})
		kit.Divider(c, tokens)
		preview := store.State.Preferences.AllowPrereleaseUpdates
		locked := u.Phase == runtime.UpdateChecking || u.Phase == runtime.UpdateDownloading || u.Phase == runtime.UpdateReady
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
	case runtime.UpdateReady:
		kit.Row(c, tokens, messages.Text(copy.UpdateReady), messages.UpdateAvailable(u.Version), func() {
			button(kit.Primary, messages.Text(copy.UpdateRestart), true, actions.RestartToUpdate)
		})
	case runtime.UpdateFailed:
		if u.Failed == runtime.UpdateStepInstall {
			kit.Row(c, tokens, messages.Text(copy.UpdateInstallFailed), messages.UpdateAvailable(u.Version), func() {
				button(kit.Outline, messages.Text(copy.Retry), true, actions.InstallUpdate)
			})
			return
		}
		kit.Row(c, tokens, messages.Text(copy.UpdateCheckFailed), "", check)
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
