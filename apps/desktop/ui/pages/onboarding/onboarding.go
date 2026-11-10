// Package onboarding guides account setup, offline dictation, polish,
// translation and selection actions.
package onboarding

import (
	"time"

	"arion.local/desktop/ui/entities/account"
	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/entities/onboarding"
	"arion.local/desktop/ui/features/cloud"
	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/features/selection"
	"arion.local/desktop/ui/features/signin"
	"arion.local/desktop/ui/features/uilanguage"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/keyfigure"
	"arion.local/desktop/ui/widgets/modelcard"
	"arion.local/desktop/ui/widgets/showcase"
	"arion.local/desktop/ui/widgets/stepper"
	"github.com/egoist/mygo/ui"
)

// Width and Height size the onboarding window; Pane is the illustration
// on the left, Inset the margin of the steps beside it, and Content the
// width of their cards and fields.
const (
	Width   = 840
	Height  = 560
	Pane    = 320
	Inset   = 44
	Content = 420
)

// Page keeps the flow's own state.
type Page struct {
	holdPicker        holdkey.PickerState
	translatePicker   holdkey.PickerState
	flow              onboarding.Flow
	dictatePractice   string
	translatePractice string
	chosen            runtime.HoldKey
	switched          time.Time
	cloud             cloud.Nav
	signin            signin.Choice
	// cloudOpen is the cloud setup layer of the polish step.
	cloudOpen bool
}

// NewPage starts setup. Cloud setup folds provider lists behind Change
// once one is in use, so the layer fits the window. Keyboard setup stays in Settings.
func NewPage() *Page { return &Page{cloud: cloud.Nav{Compact: true}, signin: signin.NewChoice()} }

// Render draws the current step beside its illustration.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	tokens := theme.Apply(c)
	state := store.State
	step := p.flow.Step()

	p.sync(c, store, step)
	bar := c.TitleBar()
	top := max(bar.Height, 28) + 16
	// Right-side caption buttons occupy the language picker's column.
	// Keep the whole header below their native hit area; left-side macOS
	// controls already sit over the illustration, away from this header.
	if bar.Right > 0 {
		top += bar.Height
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(tokens.Canvas).Children(func() {
		ui.Column(c).Width(Pane).Children(func() {
			key, label := state.Preferences.HoldKey, copy.HoldKeyFigureLabel
			if step == onboarding.Translate && holdkey.TranslationEnabled(store) {
				key, label = state.Preferences.TranslateKey, copy.TranslateKeyFigureLabel
			}
			if (step == onboarding.Dictate || step == onboarding.Translate) && keyfigure.Shows(key) && key != runtime.HoldOff {
				pressed := time.Time{}
				if state.Hold.PressedKey == key {
					pressed = state.Hold.Pressed
				}
				keyfigure.Render(c, tokens, messages, label, key, p.switched, pressed)
			} else {
				scene, levels := illustration(step, state)
				showcase.Render(c, tokens, messages, scene, holdname.Legend(state.Preferences.HoldKey), levels)
			}
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(top).DragWindow()
		})
		ui.Column(c).Grow(1).Children(func() {
			ui.Row(c).Height(top).Padding(0, Inset).AlignItems(ui.End).DragWindow().Children(func() {
				current, total := onboarding.Progress(p.flow, state)
				stepper.Render(c, tokens, store.Copy(), current, total)
				ui.Spacer(c)
				uilanguage.Picker(c, store)
			})
			ui.Column(c).Grow(1).ClipX().Children(func() {
				enter := &ui.Motion{X: 28 * float32(p.flow.Direction)}
				if p.flow.Direction == 0 {
					enter = &ui.Motion{Y: 10}
				}
				// The provider list is taller than the step. Start it at the
				// top and let it scroll; other steps stay centered.
				justify := ui.Center
				if step == onboarding.Polish && p.cloudOpen {
					justify = ui.Start
				}
				ui.Column(c.Key(step)).Fill().Padding(8, Inset).Gap(24).Justify(justify).AlignItems(ui.Start).
					Transition(ui.ElementTransition{Animation: &theme.Page, Enter: enter, Exit: &ui.Motion{Scale: 0.98}}).
					Children(func() { p.step(c, store, tokens, step) })
			})
			ui.Row(c).Padding(16, Inset, 24, Inset).Gap(8).AlignItems(ui.Center).Children(func() {
				if p.flow.Index > 0 {
					kit.Button(c.Key("back"), tokens, kit.Ghost, messages.Text(copy.Back), true).OnClick(func() {
						if step == onboarding.Polish && p.cloudOpen {
							if !p.cloud.Back() {
								p.cloudOpen = false
							}
							return
						}
						p.move(c, -1, store)
					})
				}
				ui.Spacer(c)
				if onboarding.Last(p.flow, state) {
					p.finish(c, store, tokens, messages)
					return
				}
				switch step {
				case onboarding.Welcome:
					label, variant := messages.Text(copy.Continue), kit.Primary
					if state.Account.Disabled {
						label = messages.Text(copy.OnbStart)
					} else if account.StageOf(state.Account) == account.SignedOut {
						label, variant = messages.Text(copy.OnbContinueLocally), kit.Ghost
					}
					button := kit.Button(c.Key("next"), tokens, variant, label, onboarding.Ready(step, state))
					if variant == kit.Ghost {
						button.TextColor(tokens.Muted)
					}
					button.OnClick(func() { p.move(c, 1, store) })
				case onboarding.Polish:
					if p.cloudOpen {
						label, variant := messages.Text(copy.Continue), kit.Primary
						if state.Cloud.Phase != runtime.CloudConnected || state.Preferences.CloudModel == "" {
							label, variant = messages.Text(copy.Skip), kit.Ghost
						}
						kit.Button(c.Key("next"), tokens, variant, label, !state.Cloud.Busy()).OnClick(func() {
							p.move(c, 1, store)
						})
						return
					}
					kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.Continue), onboarding.Ready(step, state)).OnClick(func() {
						if state.Preferences.Polish && state.Preferences.PolishEngine == runtime.PolishCloud {
							p.cloudOpen = true
							return
						}
						p.move(c, 1, store)
					})
				default:
					kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.Continue), onboarding.Ready(step, state)).OnClick(func() { p.move(c, 1, store) })
				}
			})
		})
	})
}

// illustration picks the scene that shows what a step enables; the
// microphone request shows the live level once it is allowed.
func illustration(step onboarding.Step, state *runtime.State) (showcase.Scene, []float32) {
	switch step {
	case onboarding.Permissions:
		if state.Permissions.Microphone == runtime.ConsentGranted && state.Permissions.Accessibility == runtime.ConsentGranted {
			return showcase.Inserted, nil
		}
		return showcase.Listening, nil
	case onboarding.Dictate, onboarding.Translate:
		return showcase.Listening, nil
	}
	return showcase.Dictating, nil
}

func shownKey(step onboarding.Step, store runtime.Store) runtime.HoldKey {
	if step == onboarding.Translate && holdkey.TranslationEnabled(store) {
		return store.State.Preferences.TranslateKey
	}
	return store.State.Preferences.HoldKey
}

func (p *Page) finish(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog) {
	kit.Button(c.Key("finish"), tokens, kit.Primary, messages.Text(copy.OnbFinish), onboarding.Ready(p.flow.Step(), store.State)).OnClick(func() {
		p.release(store)
		store.Actions.CompleteOnboarding()
	})
}

func (p *Page) move(c *ui.Context, delta int, store runtime.Store) {
	if p.flow.Move(delta, store.State) {
		p.cloudOpen = false
	}
}

// sync limits microphone monitoring and shortcut recording to their pages.
func (p *Page) sync(c *ui.Context, store runtime.Store, step onboarding.Step) {
	if step == onboarding.Welcome && !store.State.Account.Disabled {
		signin.Check(store)
	}
	if step != onboarding.Dictate {
		p.holdPicker.Close(store)
	}
	if step != onboarding.Translate {
		p.translatePicker.Close(store)
	}
	if step != onboarding.Dictate && step != onboarding.Translate {
		p.chosen = ""
		p.switched = time.Time{}
	} else if key := shownKey(step, store); key != p.chosen {
		if p.chosen != "" {
			p.switched = c.Now()
		}
		p.chosen = key
	}
	if step != onboarding.Polish {
		p.cloudOpen = false
	}
}

// Close releases resources if the window closes mid-flow.
func (p *Page) Close(store runtime.Store) { p.release(store) }

func (p *Page) release(store runtime.Store) {
	p.holdPicker.Close(store)
	p.translatePicker.Close(store)
}

func heading(c *ui.Context, tokens theme.Tokens, title, body string) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, title).FontSize(theme.Title + 4).FontWeight(650)
		ui.Text(c, body).FontSize(theme.Body).TextColor(tokens.Muted).MaxWidth(380)
	})
}

func (p *Page) step(c *ui.Context, store runtime.Store, tokens theme.Tokens, step onboarding.Step) {
	messages := store.Copy()
	state := store.State
	switch step {
	case onboarding.Welcome:
		ui.Column(c).Gap(16).Children(func() {
			kit.Mark(c, 64).Margin(-6)
			ui.Column(c).Gap(10).Children(func() {
				ui.Text(c, messages.Text(copy.OnbWelcomeTitle)).FontSize(theme.Display).FontWeight(700)
				ui.Text(c, messages.Text(copy.OnbWelcomeBody)).FontSize(theme.Heading).TextColor(tokens.Muted).MaxWidth(Content)
			})
		})
		if !state.Account.Disabled {
			p.account(c, store, tokens)
		}
	case onboarding.Model:
		heading(c, tokens, messages.Text(copy.OnbModelTitle), messages.Text(copy.OnbModelBody))
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			modelcard.Render(c, store, tokens, true)
		})
	case onboarding.Polish:
		if p.cloudOpen {
			ui.Column(c).Grow(1).Width(Content).Gap(2 * theme.GapRelated).Children(func() {
				ui.Text(c.Key("cloud-title"), cloud.Title(store, &p.cloud)).FontSize(theme.Title + 4).FontWeight(650).ContentTransition(ui.ContentTransition{})
				ui.Scroll(c.Key("cloud-scroll")).Grow(1).Children(func() {
					ui.Column(c.Key("cloud-" + p.cloud.Provider)).Width(Content).Gap(2 * theme.GapRelated).Transition(ui.ElementTransition{Animation: &theme.Page, Enter: &ui.Motion{Y: 8}}).
						Children(func() { cloud.Pages(c, store, tokens, &p.cloud) })
				})
			})
			return
		}
		heading(c, tokens, messages.Text(copy.OnbPolishTitle), messages.Text(copy.OnbPolishBody))
		ui.Column(c).Width(Content).Children(func() { polish.Choice(c, store, tokens) })
	case onboarding.Permissions:
		ui.Column(c).Width(Content).Gap(24).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				ui.Text(c, messages.Text(copy.OnbPermissionsTitle)).FontSize(theme.Title + 4).FontWeight(650)
				ui.Text(c, messages.Text(copy.OnbPermissionsBody)).FontSize(theme.Body).TextColor(tokens.Muted).MaxWidth(Content)
			})
			p.permissions(c, store, tokens)
		})
	case onboarding.Dictate:
		heading(c, tokens, messages.Text(copy.OnbDictateTitle), messages.Text(copy.OnbDictateBody))
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			kit.Card(c, tokens).Children(func() { holdkey.PolishShortcut(c, store, &p.holdPicker) })
			holdkey.ListenerStatus(c, store, tokens)
			if permissions.Missing(store) {
				p.permissions(c, store, tokens)
			}
			p.practiceField(c, store, tokens, messages, &p.dictatePractice)
		})
	case onboarding.Translate:
		heading(c, tokens, messages.Text(copy.OnbTranslateTitle), messages.Text(copy.OnbTranslateBody))
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			kit.Card(c, tokens).Children(func() {
				kit.RowFrame(ui.Column(c)).Justify(ui.Center).Children(func() { holdkey.TranslationToggle(c, store, &p.translatePicker) })
				kit.Disclosure(c.Key("translation-details"), holdkey.TranslationEnabled(store), func() {
					kit.Divider(c, tokens)
					holdkey.TranslateSetting(c, store, &p.translatePicker)
					kit.Divider(c, tokens)
					kit.Row(c, tokens, messages.Text(copy.TranslateLanguageLabel), "", func() { holdkey.TranslateLanguagePicker(c, store) })
				})
			})
			holdkey.ListenerStatus(c, store, tokens)
			if permissions.Missing(store) {
				p.permissions(c, store, tokens)
			}
			p.practiceField(c, store, tokens, messages, &p.translatePractice)
			if onboarding.Last(p.flow, state) {
				done(c, tokens, messages)
			}
		})
	case onboarding.Selection:
		heading(c, tokens, messages.Text(copy.OnbSelectionTitle), messages.Text(copy.OnbSelectionBody))
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			kit.Card(c.Key("selection-settings"), tokens).Children(func() { selection.Settings(c, store, tokens) })
			done(c, tokens, messages)
		})
	}
}

// done says where Arion stays once setup completes.
func done(c *ui.Context, tokens theme.Tokens, messages copy.Catalog) {
	ui.Text(c, messages.Text(copy.OnbDoneBody)).FontSize(theme.Caption).TextColor(tokens.Muted).MaxWidth(380)
}

func (p *Page) account(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	state := store.State
	ui.Column(c).Width(Content).Gap(12).Children(func() {
		ui.Text(c, messages.Text(copy.OnbAccountTitle)).FontSize(theme.Heading).FontWeight(650)
		if account.StageOf(state.Account) != account.SignedOut {
			signin.Card(c, store, tokens)
			if state.Account.SignedIn() {
				signin.HostedSync(c, store, tokens)
			}
			return
		}
		signin.Onboarding(c, store, tokens, &p.signin)
		signin.Notice(c, store, tokens)
	})
}

func (p *Page) permissions(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	row := func(title, tip string, kind permissions.Kind) {
		kit.TipRow(c, tokens, title, tip, func() {
			if permissions.Granted(store, kind) {
				permissions.Status(c, store, tokens, kind)
				return
			}
			permissions.Request(c, store, tokens, kind, messages.Text(copy.Allow), false)
		})
	}
	kit.Card(c, tokens).Width(Content).AlignItems(ui.Stretch).Children(func() {
		row(messages.Text(copy.MicPermission), messages.Text(copy.MicPermissionText), permissions.Microphone)
		kit.Divider(c, tokens)
		row(messages.Text(copy.AXPermission), messages.Text(copy.AXPermissionText), permissions.Accessibility)
	})
}

func (p *Page) practiceField(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, text *string) {
	if store.State.Model.State == runtime.ModelReady {
		ui.TextArea(c.Key("practice"), text).Label(messages.Text(copy.OnbTryPlaceholder)).Placeholder(messages.Text(copy.OnbTryPlaceholder)).Height(96)
		return
	}
	modelcard.Render(c, store, tokens, true)
}
