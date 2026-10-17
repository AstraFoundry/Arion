// Package settings is the preferences window: a sidebar of sections over
// the window material and grouped cards on the right.
package settings

import (
	"arion.local/desktop/ui/features/diagnostics"
	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/features/inputmode"
	"arion.local/desktop/ui/features/microphone"
	"arion.local/desktop/ui/features/modelchoice"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/features/shortcut"
	"arion.local/desktop/ui/features/signin"
	"arion.local/desktop/ui/features/uilanguage"
	"arion.local/desktop/ui/features/update"
	"arion.local/desktop/ui/features/vocabulary"
	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/modelcard"
	"arion.local/desktop/ui/widgets/showcase"
	"arion.local/desktop/ui/widgets/usage"
	"github.com/egoist/mygo/ui"
)

// Section IDs.
const (
	Overview   = "overview"
	General    = "general"
	Dictation  = "dictation"
	Microphone = "microphone"
	Model      = "model"
	Vocabulary = "vocabulary"
	Account    = "account"
	About      = "about"
)

// Page keeps the window's own state.
type Page struct {
	Section    string
	monitoring bool
	recorder   shortcut.Recorder
	draft      vocabulary.Draft
	delivery   runtime.Delivery
	mode       inputmode.State
	focus      int
	usage      usage.State
	// shown is the section drawn last, so entering one can replay its
	// entrance.
	shown string
}

func NewPage() *Page { return &Page{Section: Overview} }

// Render draws the settings window.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	if focus := store.State.Focus; focus.Seq != p.focus {
		p.focus, p.Section = focus.Seq, focus.Section
	}
	if p.Section == Account && store.State.Account.Disabled {
		p.Section = Overview
	}
	p.syncMonitor(store)
	if p.Section == Account {
		signin.Check(store)
	}
	if p.Section != p.shown {
		p.shown = p.Section
		if p.Section == Overview {
			p.usage.Replay()
		}
	}
	bar := c.TitleBar()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		side := ui.Column(c).Width(210).Padding(max(bar.Height, 12)+8, 10, 12, 10)
		if !c.Vibrancy() {
			side.Background(tokens.Card)
		}
		side.DragWindow().Children(func() {
			theme.Ink(c, tokens, func() { p.nav(c, store.Copy(), !store.State.Account.Disabled) })
		})
		ui.Column(c).Grow(1).Background(tokens.Canvas).Children(func() {
			ui.Row(c).Height(max(bar.Height, 12)+40).Padding(max(bar.Height, 12), theme.Gutter, 0, theme.Gutter).AlignItems(ui.Center).
				DragWindow().Children(func() {
				ui.Text(c, title(p.Section, store.Copy())).FontSize(theme.Title).FontWeight(600).ContentTransition(ui.ContentTransition{})
			})
			ui.Scroll(c).Grow(1).Children(func() {
				content := ui.Column(c.Key(p.Section)).Gap(theme.GapGroup).Padding(8, theme.Gutter, theme.Gutter, theme.Gutter).
					Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Y: 8}})
				content.Children(func() { p.section(c, store, tokens) })
			})
		})
	})
	// After the build: a scoped theme resets the root's background.
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
}

// syncMonitor runs the level monitor only while the microphone section shows.
func (p *Page) syncMonitor(store runtime.Store) {
	want := p.Section == Microphone && store.State.Permissions.Microphone == runtime.ConsentGranted
	if want != p.monitoring {
		p.monitoring = want
		if want {
			store.Actions.RefreshDevices()
			store.Actions.StartLevelMonitor()
		} else {
			store.Actions.StopLevelMonitor()
		}
	}
}

// Close releases resources the page holds.
func (p *Page) Close(store runtime.Store) {
	// The window opens again with its section's entrance.
	p.shown = ""
	if p.monitoring {
		p.monitoring = false
		store.Actions.StopLevelMonitor()
	}
	if p.recorder.Listening {
		store.Actions.SuspendShortcut(false)
	}
}

func title(section string, messages copy.Catalog) string {
	return map[string]string{Overview: messages.Text(copy.NavOverview), General: messages.Text(copy.NavGeneral), Dictation: messages.Text(copy.NavDictation), Microphone: messages.Text(copy.NavMicrophone),
		Model: messages.Text(copy.NavModel), Vocabulary: messages.Text(copy.NavVocabulary), Account: messages.Text(copy.NavAccount), About: messages.Text(copy.NavAbout)}[section]
}

func (p *Page) section(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	prefs := store.State.Preferences
	switch p.Section {
	case Overview:
		usage.Overview(c, store, tokens, &p.usage, func() { p.Section = Vocabulary })
	case General:
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, messages.Text(copy.LanguageLabel), "", func() { uilanguage.Picker(c, store) })
		})
		kit.Group(c, tokens, messages.Text(copy.PermissionsHeader), func() {
			kit.Card(c, tokens).Children(func() {
				kit.Row(c, tokens, messages.Text(copy.MicPermission), messages.Text(copy.MicPermissionText), func() {
					permissions.Status(c, store, tokens, permissions.Microphone)
					permissions.Request(c, store, tokens, permissions.Microphone, messages.Text(copy.Allow), false)
				})
				kit.Divider(c, tokens)
				kit.Row(c, tokens, messages.Text(copy.AXPermission), messages.Text(copy.AXPermissionText), func() {
					permissions.Status(c, store, tokens, permissions.Accessibility)
					permissions.Request(c, store, tokens, permissions.Accessibility, messages.Text(copy.Allow), false)
				})
			})
		})
		kit.Card(c, tokens).Children(func() {
			launch := prefs.LaunchAtLogin
			kit.Row(c, tokens, messages.Text(copy.LaunchAtLogin), "", func() {
				if ui.Switch(c.Key("login"), &launch).Label(messages.Text(copy.LaunchAtLogin)).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.LaunchAtLogin = launch })
				}
			})
			kit.Divider(c, tokens)
			sounds := prefs.Sounds
			kit.Row(c, tokens, messages.Text(copy.Sounds), messages.Text(copy.SoundsDetail), func() {
				if ui.Switch(c.Key("sounds"), &sounds).Label(messages.Text(copy.Sounds)).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Sounds = sounds })
				}
			})
		})
	case Dictation:
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, messages.Text(copy.ShortcutLabel), messages.Text(copy.ShortcutDetail), func() {
				shortcut.Field(c, store, tokens, &p.recorder)
			})
			kit.Divider(c, tokens)
			kit.Row(c, tokens, messages.Text(copy.HoldKeyLabel), messages.Text(copy.HoldKeyDetail), func() {
				holdkey.Picker(c, store)
			})
			holdkey.Notes(c, store, tokens)
		})
		kit.Group(c, tokens, messages.Text(copy.ModeLabel), func() { inputmode.Choice(c, store, tokens, &p.mode) })
		// Insertion routes apply to shortcut mode; the input method commits
		// through the OS input-method protocol.
		if prefs.Mode == runtime.ModeShortcut {
			kit.Group(c, tokens, messages.Text(copy.DeliveryLabel), func() { p.deliveryCard(c, store, tokens) })
		}
		kit.Card(c, tokens).Children(func() {
			fillers := prefs.RemoveFillers
			kit.Row(c, tokens, messages.Text(copy.FillersLabel), messages.Text(copy.FillersDetail), func() {
				if ui.Switch(c.Key("fillers"), &fillers).Label(messages.Text(copy.FillersLabel)).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.RemoveFillers = fillers })
				}
			})
		})
	case Microphone:
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, messages.Text(copy.MicDevice), "", func() { microphone.Picker(c, store) })
			kit.Divider(c, tokens)
			kit.Row(c, tokens, messages.Text(copy.MicGain), messages.Text(copy.MicGainDetail), func() { microphone.GainValue(c, store, tokens) })
			// Inset so the end labels, centered on the end ticks, align with the row's text.
			ui.Column(c).Padding(0, 28, 14, 28).Children(func() { microphone.Gain(c, store, tokens) })
			kit.Divider(c, tokens)
			ui.Column(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
				if store.State.Permissions.Microphone == runtime.ConsentGranted {
					microphone.Meter(c, store, tokens)
					ui.Text(c, messages.Text(copy.MicTest)).FontSize(theme.Caption).TextColor(tokens.Muted)
				} else {
					permissions.Request(c, store, tokens, permissions.Microphone, messages.Text(copy.Allow), true)
				}
			})
		})
	case Model:
		kit.Group(c, tokens, messages.Text(copy.ModelRecognitionGroup), func() { modelcard.Render(c, store, tokens, false) })
		kit.Group(c, tokens, messages.Text(copy.ModelPolishGroup), func() {
			kit.Card(c, tokens).Children(func() {
				polish.Row(c, store, tokens)
				polish.MatchApp(c, store, tokens)
				polish.Context(c, store, tokens)
			})
		})
		if hasOthers(store) {
			kit.Group(c, tokens, messages.Text(copy.ModelOthers), func() { modelchoice.Others(c, store, tokens) })
		}
	case Vocabulary:
		vocabulary.Learning(c, store, tokens)
		vocabulary.Editor(c, store, tokens, &p.draft)
	case Account:
		signin.Card(c, store, tokens)
		if store.State.Account.SignedIn() {
			kit.Group(c, tokens, messages.Text(copy.AccSyncGroup), func() { signin.HostedSync(c, store, tokens) })
		}
	case About:
		kit.Card(c, tokens).Children(func() {
			ui.Column(c).Height(200).Children(func() {
				// The banner's top corners nest inside the card's border.
				inner := float32(theme.RadiusCard - 1)
				showcase.Render(c, tokens, store.Copy(), showcase.Dictating, store.State.Shortcut.Accelerator, nil, inner, inner, 0, 0)
			})
			ui.Row(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
				kit.Mark(c, 44).Margin(-4)
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, messages.Text(copy.Title)).FontSize(theme.Heading).FontWeight(600)
					ui.Text(c, messages.Version(store.State.Version)).FontSize(theme.Caption).TextColor(tokens.Muted).Selectable()
				})
			})
		})
		update.Card(c, store, tokens)
		diagnostics.Card(c, store, tokens)
		kit.Group(c, tokens, messages.Text(copy.NavPrivacy), func() { privacy(c, messages, tokens) })
		aboutLinks(c, messages, tokens)
	}
}

// privacy states what stays on this device and the network's two uses.
func privacy(c *ui.Context, messages copy.Catalog, tokens theme.Tokens) {
	kit.Card(c, tokens).Children(func() {
		for i, item := range []struct {
			icon          *ui.SVG
			title, detail string
		}{{icons.Cpu, messages.Text(copy.PrivacyLocal), messages.Text(copy.PrivacyLocalTxt)}, {icons.MicOff, messages.Text(copy.PrivacyAudio), messages.Text(copy.PrivacyAudioTxt)},
			{icons.Lock, messages.Text(copy.PrivacyLogs), messages.Text(copy.PrivacyLogsTxt)}, {icons.HardDriveDownload, messages.Text(copy.PrivacyNetwork), messages.Text(copy.PrivacyNetTxt)}} {
			if i > 0 {
				kit.Divider(c, tokens)
			}
			ui.Row(c).Gap(12).Padding(12, 16).Children(func() {
				ui.Icon(c, item.icon).FontSize(16).TextColor(tokens.Muted).Margin(2, 0, 0, 0)
				ui.Column(c).Gap(2).Grow(1).Children(func() {
					ui.Text(c, item.title).FontSize(theme.Body).FontWeight(500)
					ui.Text(c, item.detail).FontSize(theme.Caption).TextColor(tokens.Muted)
				})
			})
		}
	})
}

// aboutLinks names the team and where the project, its news, bug reports
// and license live.
func aboutLinks(c *ui.Context, messages copy.Catalog, tokens theme.Tokens) {
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.AboutTeam), "", func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				// The avatar keeps GitHub's white tile so the dark mark reads in both appearances.
				ui.Image(c, assets.Team).Size(20, 20).Fit(ui.Cover).Radius(theme.RadiusMD).Border(1, tokens.Hairline).Clip()
				ui.Link(c, config.Team, config.TeamURL)
			})
		})
		for _, link := range []struct{ title, label, url string }{
			{messages.Text(copy.AboutSource), config.Repository, config.RepositoryURL},
			{messages.Text(copy.AboutX), config.XHandle, config.XURL},
			{messages.Text(copy.AboutFeedback), messages.Text(copy.AboutFeedbackLink), config.IssuesURL},
			{messages.Text(copy.AboutLicense), config.License, config.LicenseURL},
		} {
			kit.Divider(c, tokens)
			kit.Row(c, tokens, link.title, "", func() { ui.Link(c, link.label, link.url) })
		}
	})
}

func hasOthers(store runtime.Store) bool {
	for _, bundle := range store.State.Model.Bundles {
		if bundle.Evaluated && bundle.ID != store.State.Model.Active.ID {
			return true
		}
	}
	return false
}

// deliveryCard chooses how shortcut-mode text reaches the target app.
func (p *Page) deliveryCard(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	kit.Card(c, tokens).Children(func() {
		p.delivery = store.State.Preferences.Delivery
		ui.RadioGroup(c.Key("delivery"), func() {
			options := []struct {
				value         runtime.Delivery
				label, detail string
			}{{runtime.DeliveryAutomatic, messages.Text(copy.DeliveryAutomatic), messages.Text(copy.DeliveryAutoDetail)}, {runtime.DeliveryClipboard, messages.Text(copy.DeliveryClipboard), messages.Text(copy.DeliveryClipDetail)}}
			for i, option := range options {
				if i > 0 {
					kit.Divider(c, tokens)
				}
				value := option.value
				ui.Column(c).Padding(12, 16).Gap(3).Children(func() {
					if ui.Radio(c.Key(value), &p.delivery, value, option.label).Changed() {
						store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Delivery = value })
					}
					ui.Text(c, option.detail).FontSize(theme.Caption).TextColor(tokens.Muted).Margin(0, 0, 0, 24)
				})
			}
		})
	})
}

// nav lists the sections.
func (p *Page) nav(c *ui.Context, messages copy.Catalog, accountEnabled bool) {
	ui.Sidebar(c.Key("nav"), &p.Section, func() {
		ui.SidebarItem(c, Overview, icons.ChartLine, messages.Text(copy.NavOverview))
		ui.SidebarItem(c, General, icons.Settings, messages.Text(copy.NavGeneral))
		ui.SidebarItem(c, Dictation, icons.Keyboard, messages.Text(copy.NavDictation))
		ui.SidebarItem(c, Microphone, icons.Mic, messages.Text(copy.NavMicrophone))
		ui.SidebarItem(c, Model, icons.Cpu, messages.Text(copy.NavModel))
		ui.SidebarItem(c, Vocabulary, icons.BookOpen, messages.Text(copy.NavVocabulary))
		if accountEnabled {
			ui.SidebarItem(c, Account, icons.CircleUser, messages.Text(copy.NavAccount))
		}
		ui.SidebarItem(c, About, icons.Info, messages.Text(copy.NavAbout))
	}).Grow(1)
}
