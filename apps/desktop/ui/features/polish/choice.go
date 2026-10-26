package polish

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Choice keeps download consent and progress with the local option. Its
// trailing check means the worker is ready, rather than merely selected.
func Choice(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	state := store.State
	const off, local, cloud = "off", "local", "cloud"
	selected := off
	switch {
	case state.Preferences.Polish && state.Preferences.PolishEngine == runtime.PolishCloud:
		selected = cloud
	case runtime.PolishOn(state):
		selected = local
	}
	localDetail := messages.DownloadSize(state.Polish.SizeBytes)
	unsupported := state.Polish.State == runtime.PolishUnsupported
	switch {
	case unsupported:
		localDetail = messages.Text(copy.PolishUnsupported)
	case state.Polish.Installed:
		localDetail = messages.Text(copy.PolishDetail)
	}
	busy := state.Polish.State == runtime.PolishProvisioning || state.Polish.State == runtime.PolishLoading
	failed := state.Polish.State == runtime.PolishFailed
	ready := state.Polish.State == runtime.PolishReady
	kit.Card(c, tokens).Children(func() {
		ui.RadioGroup(c, func() {
			for i, option := range []struct {
				value, title, detail string
				disabled             bool
			}{
				{off, messages.Text(copy.PolishOff), messages.Text(copy.PolishOffDetail), false},
				{local, messages.Text(copy.PolishEngineLocal), localDetail, unsupported},
				{cloud, messages.Text(copy.PolishEngineCloud), messages.Text(copy.PolishCloudOption), false},
			} {
				if i > 0 {
					kit.Divider(c, tokens)
				}
				isLocal := option.value == local
				chosen := selected == option.value
				header := ui.Row(c.Key("option-" + option.value)).AlignItems(ui.Center)
				if chosen && isLocal {
					header.Background(tokens.AccentSoft)
				}
				header.Children(func() {
					row := ui.RadioBase(c.Key("polish-"+option.value), &selected, option.value).Grow(1).Gap(12).
						Padding(12, 16).AlignItems(ui.Center).Label(option.title).Disabled(option.disabled)
					if row.Hovered() && !option.disabled && !(chosen && isLocal) {
						row.Background(tokens.AccentSoft.Alpha(0.5).Over(tokens.Card))
					}
					row.Children(func() {
						ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
							title := ui.Text(c, option.title).FontSize(theme.Body).FontWeight(500)
							if option.disabled {
								title.TextColor(tokens.Muted)
							}
							ui.Text(c, option.detail).FontSize(theme.Caption).TextColor(tokens.Muted)
						})
						if chosen && (!isLocal || ready) {
							check := ui.Icon(c, icons.Check).FontSize(16).TextColor(tokens.AccentInk)
							if isLocal {
								check.Label(messages.Text(copy.PolishReady))
							}
						}
					})
					if row.Changed() {
						value := option.value
						store.Actions.UpdatePreferences(func(p *runtime.Preferences) {
							p.Polish = value != off
							if value == cloud {
								p.PolishEngine = runtime.PolishCloud
							} else if value == local {
								p.PolishEngine = runtime.PolishLocal
							}
						})
					}
					if isLocal && chosen && !unsupported && !state.Polish.Installed && !busy && !failed && !ready {
						if kit.Button(c.Key("download"), tokens, kit.Primary, messages.Text(copy.PolishDownload), true).
							Margin(0, 16, 0, 0).Clicked() {
							store.Actions.DownloadPolish()
						}
					}
					if isLocal && chosen && state.Polish.State == runtime.PolishProvisioning {
						cancelDownload(c.Key("cancel-download"), store, tokens, kit.Primary).Margin(0, 16, 0, 0)
					}
				})
				if isLocal {
					localStatus(c.Key("local-status"), store, tokens, selected == local && (busy || failed))
				}
			}
		}).Gap(0).Label(messages.Text(copy.PolishLabel))
	})
}

// localStatus retains the last visible status while closing. AnimateTo
// retargets from its current value on interruption and honors Reduce Motion.
// Like MyGo's collapsible panel, its height follows measured content.
func localStatus(c *ui.Context, store runtime.Store, tokens theme.Tokens, open bool) {
	panel := ui.Column(c).Shrink(0).ClipY().Disabled(!open)
	last := ui.Local(panel, "status", func() runtime.Polish { return store.State.Polish })
	target := float32(0)
	if open {
		*last = store.State.Polish
		target = 1
	}
	progress := panel.AnimateTo("reveal", target, theme.Disclosure)
	if !open && progress == 0 {
		panel.Height(0)
		return
	}
	state := *store.State
	state.Polish = *last
	store.State = &state
	var content ui.Element
	panel.Opacity(progress).Children(func() {
		content = ui.Column(c).Shrink(0).Padding(theme.GapRelated, 0, 0, 0).
			Children(func() { status(c, store, tokens, false) })
	})
	if progress < 1 {
		panel.Height(progress * content.Bounds().H)
	}
}
