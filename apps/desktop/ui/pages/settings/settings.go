// Package settings is the preferences window: a sidebar of sections over
// the window material and grouped cards on the right.
package settings

import (
	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/features/cloud"
	"arion.local/desktop/ui/features/contributors"
	"arion.local/desktop/ui/features/diagnostics"
	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/features/keyboard"
	"arion.local/desktop/ui/features/microphone"
	"arion.local/desktop/ui/features/modelchoice"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/features/provision"
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
	Keyboard   = "keyboard"
	Microphone = "microphone"
	Model      = "model"
	Vocabulary = "vocabulary"
	Account    = "account"
	About      = "about"
)

// modelPage is a page under Model: the section itself, or the page of
// one of its models. Settings stay on the section; only the models' own
// details and choices take a page.
type modelPage string

const (
	modelIndex       modelPage = ""
	modelRecognition modelPage = "recognition"
	modelCloud       modelPage = "cloud"
)

// aboutPage is a page under About: the section itself or its contributors.
type aboutPage string

const (
	aboutIndex        aboutPage = ""
	aboutContributors aboutPage = "contributors"
)

// Page keeps the window's own state.
type Page struct {
	holdPicker      holdkey.PickerState
	translatePicker holdkey.PickerState
	Section         string
	monitoring      bool
	draft           vocabulary.Draft
	// model is the page shown under Model; cloud is where the cloud
	// model's pages are.
	model    modelPage
	cloud    cloud.Nav
	about    aboutPage
	keyboard keyboard.State
	focus    int
	usage    usage.State
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
	if s, ok := sectionOf(p.Section); !ok || !s.visible(store) {
		p.Section = Overview
	}
	p.syncMonitor(store)
	if p.Section == Account {
		signin.Check(store)
	}
	if p.Section != p.shown {
		if p.shown == Dictation {
			p.holdPicker.Close(store)
			p.translatePicker.Close(store)
		}
		if p.shown == Keyboard {
			store.Actions.CancelInputMethodSetup()
		}
		p.shown = p.Section
		p.model, p.cloud, p.about = modelIndex, cloud.Nav{}, aboutIndex
		if p.Section == Overview {
			p.usage.Replay()
		}
	}
	if p.model == modelCloud && store.State.Preferences.PolishEngine != runtime.PolishCloud {
		p.model, p.cloud = modelIndex, cloud.Nav{}
	}
	sub := p.subPage()
	bar := c.TitleBar()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		side := ui.Column(c).Width(210).Padding(max(bar.Height, 12)+8, 10, 12, 10)
		if !c.Vibrancy() {
			side.Background(tokens.Card)
		}
		side.DragWindow().Children(func() {
			theme.Ink(c, tokens, func() { p.nav(c, store, tokens) })
		})
		ui.Column(c).Grow(1).Background(tokens.Canvas).Children(func() {
			ui.Row(c).Height(max(bar.Height, 12)+40).Padding(max(bar.Height, 12), theme.Gutter, 0, theme.Gutter).AlignItems(ui.Center).
				DragWindow().Gap(4).Children(func() {
				heading := title(p.Section, store.Copy())
				if sub {
					// A page under a section: back steps out one page at a time.
					heading = p.subTitle(store)
					if kit.IconButton(c.Key("back"), tokens, icons.ChevronLeft, store.Copy().Text(copy.Back)).Margin(0, 0, 0, -6).Clicked() {
						p.back()
					}
				}
				ui.Text(c, heading).FontSize(theme.Title).FontWeight(600).ContentTransition(ui.ContentTransition{})
			})
			ui.Scroll(c).Grow(1).Children(func() {
				key := p.Section
				if sub {
					key += "/" + string(p.model) + "/" + p.cloud.Provider + "/" + string(p.about)
				}
				content := ui.Column(c.Key(key)).Gap(theme.GapGroup).Padding(8, theme.Gutter, theme.Gutter, theme.Gutter).
					Transition(ui.ElementTransition{Animation: &theme.Settle, Position: true, Enter: &ui.Motion{Y: 8}})
				content.Children(func() {
					p.section(c, store, tokens)
				})
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
	p.holdPicker.Close(store)
	p.translatePicker.Close(store)
	if p.Section == Keyboard {
		store.Actions.CancelInputMethodSetup()
	}
	// The window opens again with its section's entrance.
	p.shown = ""
	if p.monitoring {
		p.monitoring = false
		store.Actions.StopLevelMonitor()
	}
}

func title(id string, messages copy.Catalog) string {
	s, _ := sectionOf(id)
	return messages.Text(s.title)
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
				kit.TipRow(c, tokens, messages.Text(copy.MicPermission), messages.Text(copy.MicPermissionText), func() {
					permissions.Status(c, store, tokens, permissions.Microphone)
					permissions.Request(c, store, tokens, permissions.Microphone, messages.Text(copy.Allow), false)
				})
				kit.Divider(c, tokens)
				kit.TipRow(c, tokens, messages.Text(copy.AXPermission), messages.Text(copy.AXPermissionText), func() {
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
			kit.TipRow(c, tokens, messages.Text(copy.Sounds), messages.Text(copy.SoundsDetail), func() {
				if ui.Switch(c.Key("sounds"), &sounds).Label(messages.Text(copy.Sounds)).Changed() {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Sounds = sounds })
				}
			})
		})
	case Dictation:
		holdkey.ListenerStatus(c, store, tokens)
		kit.Card(c.Key("dictation-shortcut"), tokens).Children(func() {
			holdkey.Setting(c, store, &p.holdPicker)
		})

		kit.Card(c.Key("translation-settings"), tokens).Children(func() {
			ui.Column(c).Padding(12, 16).Children(func() { holdkey.TranslationToggle(c, store, &p.translatePicker) })
			kit.Disclosure(c.Key("translation-details"), holdkey.TranslationEnabled(store), func() {
				kit.Divider(c, tokens)
				holdkey.TranslateSetting(c, store, &p.translatePicker)
				kit.Divider(c, tokens)
				kit.Row(c, tokens, messages.Text(copy.TranslateLanguageLabel), "", func() { holdkey.TranslateLanguagePicker(c, store) })
			})
		})
	case Keyboard:
		kit.Group(c, tokens, "", func() { keyboard.Source(c, store, tokens, &p.keyboard) })
		kit.Group(c, tokens, messages.Text(copy.KbdCandidatesGroup), func() { keyboard.Candidates(c, store, tokens) })
		kit.Group(c, tokens, messages.Text(copy.KbdKeysGroup), func() { keyboard.Keys(c, store, tokens) })
		kit.Group(c, tokens, messages.Text(copy.KbdPhoneticsGroup), func() { keyboard.Phonetics(c, store, tokens) })
		keyboard.Neural(c, store, tokens)
	case Microphone:
		kit.Card(c, tokens).Children(func() {
			kit.Row(c, tokens, messages.Text(copy.MicDevice), "", func() { microphone.Picker(c, store) })
			kit.Divider(c, tokens)
			kit.TipRow(c, tokens, messages.Text(copy.MicGain), messages.Text(copy.MicGainDetail), func() { microphone.GainValue(c, store, tokens) })
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
		p.modelSection(c, store, tokens)
	case Vocabulary:
		vocabulary.Learning(c, store, tokens)
		vocabulary.Editor(c, store, tokens, &p.draft)
	case Account:
		signin.Card(c, store, tokens)
		if store.State.Account.SignedIn() {
			kit.Group(c, tokens, messages.Text(copy.AccSyncGroup), func() { signin.HostedSync(c, store, tokens) })
		}
		signin.Portal(c, store, tokens)
	case About:
		if p.about == aboutContributors {
			contributors.Page(c, tokens)
			return
		}
		kit.Card(c, tokens).Children(func() {
			ui.Column(c).Height(200).Children(func() {
				// The banner's top corners nest inside the card's border.
				inner := float32(theme.RadiusCard - 1)
				showcase.Render(c, tokens, store.Copy(), showcase.Dictating, holdname.Legend(store.State.Preferences.HoldKey), nil, inner, inner, 0, 0)
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
		if aboutLinks(c, store, tokens) {
			p.about = aboutContributors
		}
	}
}

// privacy states what stays on this device and when Arion uses the network.
func privacy(c *ui.Context, messages copy.Catalog, tokens theme.Tokens) {
	kit.Card(c, tokens).Children(func() {
		for i, item := range []struct {
			icon       *ui.SVG
			title, tip string
		}{{icons.Cpu, messages.Text(copy.PrivacyLocal), messages.Text(copy.PrivacyLocalTxt)}, {icons.MicOff, messages.Text(copy.PrivacyAudio), messages.Text(copy.PrivacyAudioTxt)},
			{icons.Lock, messages.Text(copy.PrivacyLogs), messages.Text(copy.PrivacyLogsTxt)}, {icons.HardDriveDownload, messages.Text(copy.PrivacyNetwork), messages.Text(copy.PrivacyNetTxt)}} {
			if i > 0 {
				kit.Divider(c, tokens)
			}
			ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, item.icon).FontSize(16).TextColor(tokens.Muted)
				ui.Row(c).Grow(1).Shrink(1).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.title).FontSize(theme.Body).FontWeight(500).Shrink(1)
					kit.Hint(c, tokens, item.tip)
				})
			})
		}
	})
}

// aboutLinks names the team and where the project, its news, bug reports
// and license live, and ends with the contributors. It reports a click on
// the contributors.
func aboutLinks(c *ui.Context, store runtime.Store, tokens theme.Tokens) bool {
	messages := store.Copy()
	open := false
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.AboutTeam), "", func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				// The avatar keeps GitHub's white tile so the dark mark reads in both appearances.
				ui.Image(c, assets.Team).Size(20, 20).Fit(ui.Cover).Radius(theme.RadiusMD).Border(1, tokens.Hairline).Clip()
				ui.Link(c, config.Team, config.TeamURL)
			})
		})
		links := func(list ...[3]string) {
			for _, link := range list {
				kit.Divider(c, tokens)
				kit.Row(c, tokens, link[0], "", func() { ui.Link(c, link[1], link[2]) })
			}
		}
		links([3]string{messages.Text(copy.AboutSource), config.Repository, config.RepositoryURL},
			[3]string{messages.Text(copy.AboutX), config.XHandle, config.XURL})
		kit.Divider(c, tokens)
		kit.Row(c, tokens, messages.Text(copy.AboutCommunity), "", func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				for _, app := range []struct {
					logo      ui.ImageSource
					name, url string
				}{{icons.Discord, messages.Text(copy.AboutDiscord), config.DiscordURL}, {assets.Feishu, messages.Text(copy.AboutFeishu), config.FeishuURL}} {
					if kit.AppButton(c.Key(app.url), tokens, app.logo, app.name) {
						c.OpenURL(app.url)
					}
				}
			})
		})
		links([3]string{messages.Text(copy.AboutFeedback), messages.Text(copy.AboutFeedbackLink), config.IssuesURL},
			[3]string{messages.Text(copy.AboutLicense), config.License, config.LicenseURL})
		kit.Divider(c, tokens)
		open = contributors.Row(c, store, tokens)
	})
	return open
}

func hasOthers(store runtime.Store) bool {
	for _, bundle := range store.State.Model.Bundles {
		if bundle.Evaluated && bundle.ID != store.State.Model.Active.ID {
			return true
		}
	}
	return false
}

// nav lists the sections, with the account card at the foot of the
// sidebar.
func (p *Page) nav(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	ui.Sidebar(c.Key("nav"), &p.Section, func() {
		for _, s := range sections {
			if s.card || !s.visible(store) {
				continue
			}
			row := ui.SidebarItem(c, s.id, s.icon, messages.Text(s.title))
			if s.waiting(store) {
				row.Children(func() { kit.AttentionDot(c, tokens) })
			}
		}
	}).Grow(1)
	if account, _ := sectionOf(Account); account.visible(store) && signin.SidebarCard(c, store, tokens, p.Section == Account) {
		p.Section = Account
	}
}

// subPage reports a page under the current section.
func (p *Page) subPage() bool {
	return (p.Section == Model && p.model != modelIndex) || (p.Section == About && p.about != aboutIndex)
}

func (p *Page) subTitle(store runtime.Store) string {
	if p.Section == About {
		return store.Copy().Text(copy.AboutContributors)
	}
	return p.modelTitle(store)
}

// back leaves the current page for the one above it.
func (p *Page) back() {
	if p.Section == About {
		p.about = aboutIndex
		return
	}
	p.modelBack()
}

func (p *Page) modelTitle(store runtime.Store) string {
	if p.model == modelRecognition {
		return store.Copy().Text(copy.ModelRecognitionPage)
	}
	return cloud.Title(store, &p.cloud)
}

// modelBack leaves the current page under Model for the one above it.
func (p *Page) modelBack() {
	if p.model != modelCloud || !p.cloud.Back() {
		p.model = modelIndex
	}
}

// modelSection is Speech & Polish in the order the text goes:
// recognition, polishing and where it runs, its formatting, then the
// context it reads, the last two only while polishing runs. The
// recognition model and the cloud model each take a page.
func (p *Page) modelSection(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	switch p.model {
	case modelIndex:
		kit.Group(c, tokens, messages.Text(copy.ModelRecognitionGroup), func() {
			kit.Card(c, tokens).Children(func() {
				source := messages.Text(copy.PanelAutomatic)
				if !store.State.Model.Automatic {
					source = messages.Text(copy.PanelChosen)
				}
				if kit.LinkRow(c.Key("recognition"), tokens, messages.Text(copy.ModelRecognitionPage), source, nil) {
					p.model = modelRecognition
				}
				if model.Busy(store.State.Model) || store.State.Model.State == runtime.ModelFailed {
					ui.Column(c).Padding(0, 16, 12, 16).Children(func() { provision.Progress(c, store, tokens) })
				}
			})
		})
		kit.Group(c, tokens, messages.Text(copy.ModelPolishGroup), func() {
			kit.Card(c, tokens).Children(func() {
				polish.Row(c, store, tokens)
				polish.Engine(c, store, tokens)
				if store.State.Preferences.PolishEngine == runtime.PolishCloud && cloud.Summary(c, store, tokens) {
					p.model = modelCloud
				}
			})
		})
		// What polishing does with the text, then what it may read: both
		// only while it runs.
		if runtime.PolishOn(store.State) {
			kit.Group(c, tokens, messages.Text(copy.ModelFormatGroup), func() {
				kit.Card(c, tokens).Children(func() {
					polish.Punctuation(c, store, tokens)
					kit.Divider(c, tokens)
					polish.MatchApp(c, store, tokens)
				})
			})
			kit.Group(c, tokens, messages.Text(copy.ContextGroup), func() { polish.Context(c, store, tokens) })
		}
	case modelRecognition:
		modelcard.Render(c, store, tokens, false)
		if hasOthers(store) {
			kit.Group(c, tokens, messages.Text(copy.ModelOthers), func() { modelchoice.Others(c, store, tokens) })
		}
	case modelCloud:
		ui.Column(c).Gap(theme.GapGroup).Children(func() { cloud.Pages(c, store, tokens, &p.cloud) })
	}
}
