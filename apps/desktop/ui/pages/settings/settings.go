// Package settings is the preferences window: a sidebar of sections over
// the window material and grouped cards on the right.
package settings

import (
	"arion.local/desktop/ui/features/inputmode"
	"arion.local/desktop/ui/features/microphone"
	"arion.local/desktop/ui/features/modelchoice"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/shortcut"
	"arion.local/desktop/ui/features/vocabulary"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/modelcard"
	"github.com/egoist/mygo/ui"
)

// Section IDs.
const (
	General    = "general"
	Dictation  = "dictation"
	Microphone = "microphone"
	Model      = "model"
	Vocabulary = "vocabulary"
	Privacy    = "privacy"
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
}

func NewPage() *Page { return &Page{Section: General} }

// Render draws the settings window.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	p.syncMonitor(store)
	bar := c.TitleBar()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		side := ui.Column(c).Width(210).Padding(max(bar.Height, 12)+8, 10, 12, 10)
		if !c.Vibrancy() {
			side.Background(tokens.Card)
		}
		side.DragWindow().Children(func() {
			ui.Sidebar(c.Key("nav"), &p.Section, func() {
				ui.SidebarItem(c, General, icons.Settings, copy.NavGeneral)
				ui.SidebarItem(c, Dictation, icons.Keyboard, copy.NavDictation)
				ui.SidebarItem(c, Microphone, icons.Mic, copy.NavMicrophone)
				ui.SidebarItem(c, Model, icons.Cpu, copy.NavModel)
				ui.SidebarItem(c, Vocabulary, icons.BookOpen, copy.NavVocabulary)
				ui.SidebarItem(c, Privacy, icons.ShieldCheck, copy.NavPrivacy)
				ui.SidebarItem(c, About, icons.Info, copy.NavAbout)
			}).Grow(1)
		})
		ui.Column(c).Grow(1).Background(tokens.Canvas).Children(func() {
			ui.Row(c).Height(max(bar.Height, 12)+40).Padding(max(bar.Height, 12), theme.Gutter, 0, theme.Gutter).AlignItems(ui.Center).
				DragWindow().Children(func() {
				ui.Text(c, title(p.Section)).FontSize(theme.Title).FontWeight(600).ContentTransition(ui.ContentTransition{})
			})
			ui.Scroll(c).Grow(1).Children(func() {
				content := ui.Column(c.Key(p.Section)).Gap(18).Padding(8, theme.Gutter, theme.Gutter, theme.Gutter).
					Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Y: 8}})
				content.Children(func() { p.section(c, store, tokens) })
			})
		})
	})
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
	if p.monitoring {
		p.monitoring = false
		store.Actions.StopLevelMonitor()
	}
	if p.recorder.Listening {
		store.Actions.SuspendShortcut(false)
	}
}

func title(section string) string {
	return map[string]string{General: copy.NavGeneral, Dictation: copy.NavDictation, Microphone: copy.NavMicrophone,
		Model: copy.NavModel, Vocabulary: copy.NavVocabulary, Privacy: copy.NavPrivacy, About: copy.NavAbout}[section]
}

func (p *Page) section(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	prefs := store.State.Preferences
	switch p.Section {
	case General:
		kit.SectionTitle(c, tokens, copy.PermissionsHeader)
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, copy.MicPermission, copy.MicPermissionText, func() {
				permissions.Status(c, store, tokens, permissions.Microphone)
				permissions.Request(c, store, permissions.Microphone, copy.Allow, false)
			})
			kit.Divider(c, tokens)
			kit.Row(c, tokens, copy.AXPermission, copy.AXPermissionText, func() {
				permissions.Status(c, store, tokens, permissions.Accessibility)
				permissions.Request(c, store, permissions.Accessibility, copy.Allow, false)
			})
		})
		kit.Card(c, tokens).Children(func() {
			launch := prefs.LaunchAtLogin
			kit.Row(c, tokens, copy.LaunchAtLogin, "", func() {
				if ui.Switch(c.Key("login"), &launch).Label(copy.LaunchAtLogin).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.LaunchAtLogin = launch })
				}
			})
			kit.Divider(c, tokens)
			sounds := prefs.Sounds
			kit.Row(c, tokens, copy.Sounds, copy.SoundsDetail, func() {
				if ui.Switch(c.Key("sounds"), &sounds).Label(copy.Sounds).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Sounds = sounds })
				}
			})
		})
	case Dictation:
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, copy.ShortcutLabel, copy.ShortcutDetail, func() {
				shortcut.Field(c, store, tokens, &p.recorder)
			})
		})
		kit.SectionTitle(c, tokens, copy.ModeLabel)
		inputmode.Choice(c, store, tokens, &p.mode)
		// Insertion routes apply to shortcut mode; the input method commits
		// through the OS input-method protocol.
		if prefs.Mode == runtime.ModeShortcut {
			kit.SectionTitle(c, tokens, copy.DeliveryLabel)
			kit.Card(c, tokens).Children(func() {
				p.delivery = prefs.Delivery
				ui.RadioGroup(c.Key("delivery"), func() {
					options := []struct {
						value         runtime.Delivery
						label, detail string
					}{{runtime.DeliveryAutomatic, copy.DeliveryAutomatic, copy.DeliveryAutoDetail}, {runtime.DeliveryClipboard, copy.DeliveryClipboard, copy.DeliveryClipDetail}}
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
		kit.Card(c, tokens).Children(func() {
			fillers := prefs.RemoveFillers
			kit.Row(c, tokens, copy.FillersLabel, copy.FillersDetail, func() {
				if ui.Switch(c.Key("fillers"), &fillers).Label(copy.FillersLabel).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.RemoveFillers = fillers })
				}
			})
		})
	case Microphone:
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, copy.MicDevice, "", func() { microphone.Picker(c, store) })
			kit.Divider(c, tokens)
			ui.Column(c).Padding(16).Gap(10).AlignItems(ui.Center).Children(func() {
				if store.State.Permissions.Microphone == runtime.ConsentGranted {
					microphone.Meter(c, store, tokens)
					ui.Text(c, copy.MicTest).FontSize(theme.Caption).TextColor(tokens.Muted)
				} else {
					permissions.Request(c, store, permissions.Microphone, copy.Allow, true)
				}
			})
		})
	case Model:
		modelcard.Render(c, store, tokens, false)
		if hasOthers(store) {
			kit.SectionTitle(c, tokens, copy.ModelOthers)
			modelchoice.Others(c, store, tokens)
		}
	case Vocabulary:
		vocabulary.Editor(c, store, tokens, &p.draft)
	case Privacy:
		kit.Card(c, tokens).Children(func() {
			for i, item := range []struct {
				icon          *ui.SVG
				title, detail string
			}{{icons.Cpu, copy.PrivacyLocal, copy.PrivacyLocalTxt}, {icons.MicOff, copy.PrivacyAudio, copy.PrivacyAudioTxt},
				{icons.Lock, copy.PrivacyLogs, copy.PrivacyLogsTxt}, {icons.HardDriveDownload, copy.PrivacyNetwork, copy.PrivacyNetTxt}} {
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
	case About:
		kit.Card(c, tokens).Padding(16).Gap(10).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Size(36, 36).Radius(10).Background(tokens.Accent).Justify(ui.Center).AlignItems(ui.Center).
					TextColor(tokens.AccentText).Children(func() { ui.Icon(c, icons.AudioLines).FontSize(20) })
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, copy.Title).FontSize(theme.Heading).FontWeight(600)
					ui.Text(c, copy.Version(store.State.Version)).FontSize(theme.Caption).TextColor(tokens.Muted).Selectable()
				})
			})
			ui.Text(c, copy.AboutCredits).FontSize(theme.Caption).TextColor(tokens.Muted)
		})
	}
}

func hasOthers(store runtime.Store) bool {
	for _, bundle := range store.State.Model.Bundles {
		if bundle.Evaluated && bundle.ID != store.State.Model.Active.ID {
			return true
		}
	}
	return false
}
