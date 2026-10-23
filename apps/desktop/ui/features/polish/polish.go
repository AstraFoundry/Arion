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
	kit.TipRow(c, tokens, messages.Text(copy.PolishLabel), messages.Text(detail), func() {
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
	kit.TipRow(c, tokens, messages.Text(copy.PolishEngineLabel), "", func() {
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

// Punctuation chooses how polished text is punctuated. It is a row in the
// Polish card, label on the left and the mode menu on the right, like
// Runs on. The control stays visible while polish is off and does not
// change the preference then. English lists three modes. Spaces remains
// a saved preference and a Simplified Chinese option; the English menu
// does not show it and does not replace a saved spaces value.
func Punctuation(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	modes := []runtime.Punctuation{runtime.PunctuationSmart, runtime.PunctuationFull, runtime.PunctuationOmit}
	keys := []copy.Key{copy.PunctuationSmart, copy.PunctuationFull, copy.PunctuationOmit}
	if store.State.Preferences.UILanguage == runtime.LanguageChinese {
		modes = append(modes, runtime.PunctuationSpaces)
		keys = append(keys, copy.PunctuationSpaces)
	}
	labels := make([]string, len(modes))
	selected := ""
	for i, mode := range modes {
		labels[i] = messages.Text(keys[i])
		if store.State.Preferences.Punctuation == mode {
			selected = labels[i]
		}
	}
	kit.Divider(c, tokens)
	kit.TipRow(c, tokens, messages.Text(copy.PunctuationLabel), "", func() {
		menu := ui.Select(c.Key("punctuation"), &selected, labels).Label(messages.Text(copy.PunctuationLabel))
		if !store.State.Preferences.Polish {
			menu.Disabled(true)
		}
		if menu.Changed() && store.State.Preferences.Polish && selected != "" {
			for i, label := range labels {
				if label == selected {
					mode := modes[i]
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Punctuation = mode })
				}
			}
		}
	})
}

// MatchApp is the switch row letting polishing follow the focused app,
// field and web site. It is shown only while polishing is on.
func MatchApp(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !runtime.PolishOn(store.State) {
		return
	}
	messages := store.Copy()
	on := store.State.Preferences.MatchApp
	kit.TipRow(c, tokens, messages.Text(copy.MatchAppLabel), messages.Text(copy.MatchAppDetail), func() {
		if ui.Switch(c.Key("match-app"), &on).Label(messages.Text(copy.MatchAppLabel)).Changed() {
			store.Actions.UpdatePreferences(func(prefs *runtime.Preferences) { prefs.MatchApp = on })
		}
	})
}

// Shown reports whether polishing reads context: it is on, on this device.
// The cloud never receives context (ADR 022).
func ContextShown(store runtime.Store) bool {
	return runtime.PolishOn(store.State) && store.State.Preferences.PolishEngine != runtime.PolishCloud
}

// Context is the card of switches for the context polishing may read,
// each explained in its tooltip. Visible window text uses Accessibility.
func Context(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !ContextShown(store) {
		return
	}
	messages := store.Copy()
	prefs := store.State.Preferences
	kit.Card(c, tokens).Children(func() {
		row := func(key string, label, detail copy.Key, value bool, set func(*runtime.Preferences, bool), extra func()) {
			on := value
			if key != "context-cursor" {
				kit.Divider(c, tokens)
			}
			kit.TipRow(c, tokens, messages.Text(label), messages.Text(detail), func() {
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
	})
}

// Summary is the polish setting in words: off, this device, or the cloud
// with its model.
func Summary(store runtime.Store) string {
	messages := store.Copy()
	prefs := store.State.Preferences
	switch {
	case !runtime.PolishOn(store.State):
		return messages.Text(copy.PolishOff)
	case prefs.PolishEngine == runtime.PolishCloud && prefs.CloudModel != "":
		return messages.Text(copy.PolishEngineCloud) + " · " + prefs.CloudModel
	case prefs.PolishEngine == runtime.PolishCloud:
		return messages.Text(copy.PolishEngineCloud)
	}
	return messages.Text(copy.PolishEngineLocal)
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

// Choice is the onboarding page's choice of polishing: off, on this device
// or in the cloud, one option per row. Choosing this device starts the
// model's download, whose progress follows.
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
				row := ui.RadioBase(c.Key("polish-"+option.value), &selected, option.value).Gap(12).Padding(12, 16).AlignItems(ui.Center).
					Label(option.title).Disabled(option.disabled)
				if row.Hovered() && !option.disabled {
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
					if selected == option.value {
						ui.Icon(c, icons.Check).FontSize(16).TextColor(tokens.AccentInk)
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
			}
		}).Label(messages.Text(copy.PolishLabel))
		if selected == local {
			Status(c, store, tokens)
		}
	})
}
