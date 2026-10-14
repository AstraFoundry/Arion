// Package polish lets the user turn on live rewriting by the local LLM and
// shows its one-time download and readiness.
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

// Row is the switch row; the switch is disabled where the runtime is not
// shipped.
func Row(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	p := store.State.Polish
	// Where the runtime is not shipped the capability cannot be on,
	// whatever an earlier profile saved.
	on := store.State.Preferences.Polish && p.State != runtime.PolishUnsupported
	kit.Row(c, tokens, messages.Text(copy.PolishLabel), messages.Text(copy.PolishDetail), func() {
		toggle := ui.Switch(c.Key("polish"), &on).Label(messages.Text(copy.PolishLabel))
		if p.State == runtime.PolishUnsupported {
			toggle.Disabled(true)
		}
		if toggle.Changed() {
			store.Actions.UpdatePreferences(func(prefs *runtime.Preferences) { prefs.Polish = on })
		}
	})
	Status(c, store, tokens)
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
