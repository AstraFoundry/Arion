// Package polish lets the user turn on polishing, choose whether it runs on
// this device or in the cloud, and shows the local model's one-time
// download and readiness. The cloud connection is features/cloud.
package polish

import (
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Row is the switch row; the switch is disabled while the chosen engine
// cannot run here. The local model's status follows it.
func Row(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	// Where the local runtime is not shipped, local polishing cannot be on,
	// whatever an earlier profile saved.
	on := runtime.PolishOn(store.State)
	cloud := store.State.Preferences.PolishEngine == runtime.PolishCloud
	detail := copy.PolishDetail
	if cloud {
		detail = copy.PolishCloudDetail
	}
	kit.Row(c, tokens, messages.Text(copy.PolishLabel), messages.Text(detail), func() {
		toggle := ui.Switch(c.Key("polish"), &on).Label(messages.Text(copy.PolishLabel))
		if !runtime.PolishAvailable(store.State) {
			toggle.Disabled(true)
		}
		if toggle.Changed() {
			store.Actions.UpdatePreferences(func(prefs *runtime.Preferences) { prefs.Polish = on })
		}
	})
	if !cloud {
		Status(c, store, tokens)
	}
}

// Engine chooses where polishing runs. It shows while polishing is off
// too: choosing the cloud is what makes polishing available where the
// local runtime is not shipped.
func Engine(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	selected := 0
	if store.State.Preferences.PolishEngine == runtime.PolishCloud {
		selected = 1
	}
	kit.Divider(c, tokens)
	kit.Row(c, tokens, messages.Text(copy.PolishEngineLabel), "", func() {
		if ui.Segmented(c.Key("polish-engine"), &selected, messages.Text(copy.PolishEngineLocal), messages.Text(copy.PolishEngineCloud)).
			Label(messages.Text(copy.PolishEngineLabel)).Changed() {
			engine := runtime.PolishLocal
			if selected == 1 {
				engine = runtime.PolishCloud
			}
			store.Actions.UpdatePreferences(func(prefs *runtime.Preferences) { prefs.PolishEngine = engine })
		}
	})
}

// MatchApp is the switch letting polishing follow the focused app, field and
// web site. It is shown only while polishing is on.
func MatchApp(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !runtime.PolishOn(store.State) {
		return
	}
	messages := store.Copy()
	on := store.State.Preferences.MatchApp
	kit.Divider(c, tokens)
	kit.Row(c, tokens, messages.Text(copy.MatchAppLabel), messages.Text(copy.MatchAppDetail), func() {
		if ui.Switch(c.Key("match-app"), &on).Label(messages.Text(copy.MatchAppLabel)).Changed() {
			store.Actions.UpdatePreferences(func(prefs *runtime.Preferences) { prefs.MatchApp = on })
		}
	})
}

// Context lists the switches for the context polishing may read. Shown
// only while polishing is on this device: the cloud never receives
// context (ADR 022). Visible window text uses Accessibility.
func Context(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !runtime.PolishOn(store.State) || store.State.Preferences.PolishEngine == runtime.PolishCloud {
		return
	}
	messages := store.Copy()
	prefs := store.State.Preferences
	row := func(key string, label, detail copy.Key, value bool, set func(*runtime.Preferences, bool), extra func()) {
		on := value
		kit.Divider(c, tokens)
		kit.Row(c, tokens, messages.Text(label), messages.Text(detail), func() {
			if ui.Switch(c.Key(key), &on).Label(messages.Text(label)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { set(p, on) })
			}
		})
		if extra != nil {
			extra()
		}
	}
	row("context-cursor", copy.ContextCursorLabel, copy.ContextCursorDetail, prefs.ContextCursor,
		func(p *runtime.Preferences, on bool) { p.ContextCursor = on }, nil)
	row("context-screen", copy.ContextScreenLabel, copy.ContextScreenDetail, prefs.ContextScreen,
		func(p *runtime.Preferences, on bool) { p.ContextScreen = on }, func() {
			if !prefs.ContextScreen || store.State.Permissions.Accessibility == runtime.ConsentGranted {
				return
			}
			ui.Row(c.Key("screen-permission")).Padding(0, 16, 12, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.TriangleAlert).FontSize(14).TextColor(tokens.Warning)
				ui.Text(c, messages.Text(copy.ContextScreenPermission)).FontSize(theme.Caption).Grow(1)
				if kit.Button(c, tokens, kit.Outline, messages.Text(copy.Allow), true).Clicked() {
					store.Actions.RequestAccessibility()
				}
			})
		})
	row("context-history", copy.ContextHistoryLabel, copy.ContextHistoryDetail, prefs.ContextHistory,
		func(p *runtime.Preferences, on bool) { p.ContextHistory = on }, nil)
}

// Status is the line under the row: what enabling costs, progress, or a
// failure with its retry.
func Status(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	p := store.State.Polish
	on := store.State.Preferences.Polish
	ui.Column(c).Padding(0, 16, 12, 16).Gap(8).Children(func() {
		switch {
		case p.State == runtime.PolishUnsupported:
			ui.Text(c, messages.Text(copy.PolishUnsupported)).FontSize(theme.Caption).TextColor(tokens.Muted)
		case p.State == runtime.PolishFailed:
			ui.Row(c.Key("failed")).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.TriangleAlert).FontSize(14).TextColor(tokens.Danger)
				ui.Text(c, messages.Text(model.FailureKey(p.Failure))).FontSize(theme.Caption).Grow(1)
				if kit.Button(c, tokens, kit.Outline, messages.Text(copy.Retry), true).Clicked() {
					store.Actions.RetryPolish()
				}
			})
		case p.State == runtime.PolishProvisioning || p.State == runtime.PolishLoading:
			label := messages.Text(model.PolishPhaseLabel(p))
			ui.Progress(c.Key("bar"), model.PolishFraction(p)).Label(label)
			ui.Row(c).Children(func() {
				ui.Text(c, label).FontSize(theme.Caption).TextColor(tokens.Muted).ContentTransition(ui.ContentTransition{})
				ui.Spacer(c)
				if p.State == runtime.PolishProvisioning && p.Phase == runtime.PhaseDownloading {
					ui.Text(c, messages.Progress(p.Done, p.Total)).FontSize(theme.Caption).TextColor(tokens.Muted).FontFeatures("tnum")
				}
			})
		case p.State == runtime.PolishReady:
			kit.Status(c.Key("ready"), tokens, icons.ShieldCheck, tokens.Success, messages.Text(copy.PolishReady))
		case !on && !p.Installed:
			ui.Text(c, messages.DownloadSize(p.SizeBytes)).FontSize(theme.Caption).TextColor(tokens.Muted)
		}
	})
}
