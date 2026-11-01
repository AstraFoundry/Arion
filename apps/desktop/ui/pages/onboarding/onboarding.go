// Package onboarding prepares offline dictation before optional Settings.
package onboarding

import (
	"time"

	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/entities/onboarding"
	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/features/microphone"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/uilanguage"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
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
	holdPicker  holdkey.PickerState
	flow        onboarding.Flow
	monitoring  bool
	practice    string
	axRequested bool
	axGranted   time.Time
	chosen      runtime.HoldKey
	switched    time.Time
}

// NewPage starts the required setup. Optional configuration stays in Settings.
func NewPage() *Page { return &Page{} }

// Render draws the current step beside its illustration.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	tokens := theme.Apply(c)
	state := store.State
	step := p.flow.Step()
	if step == onboarding.Finished && !onboarding.Ready(step, state) {
		p.flow.Move(-1, state)
		step = p.flow.Step()
	}

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
			key := state.Preferences.HoldKey
			if step == onboarding.Try && keyfigure.Shows(key) {
				pressed := time.Time{}
				if state.Hold.PressedKey == key {
					pressed = state.Hold.Pressed
				}
				keyfigure.Render(c, tokens, messages, copy.HoldKeyFigureLabel, key, p.switched, pressed)
			} else {
				scene, levels := illustration(step, state)
				showcase.Render(c, tokens, messages, scene, holdname.Legend(key), levels)
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
				ui.Column(c.Key(step)).Fill().Padding(8, Inset).Gap(24).Justify(ui.Center).AlignItems(ui.Start).
					Transition(ui.ElementTransition{Animation: &theme.Page, Enter: enter, Exit: &ui.Motion{Scale: 0.98}}).
					Children(func() { p.step(c, store, tokens, step) })
			})
			ui.Row(c).Padding(16, Inset, 24, Inset).Gap(8).AlignItems(ui.Center).Children(func() {
				if p.flow.Index > 0 && step != onboarding.Finished {
					kit.Button(c.Key("back"), tokens, kit.Ghost, messages.Text(copy.Back), true).OnClick(func() {
						p.move(c, -1, store)
					})
				}
				ui.Spacer(c)
				switch step {
				case onboarding.Welcome:
					kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.OnbStart), true).OnClick(func() { p.move(c, 1, store) })
				case onboarding.Finished:
					kit.Button(c.Key("finish"), tokens, kit.Primary, messages.Text(copy.OnbFinish), onboarding.Ready(step, state)).OnClick(func() {
						p.release(store)
						store.Actions.CompleteOnboarding()
					})
				default:
					kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.Continue), onboarding.Ready(step, state)).OnClick(func() { p.move(c, 1, store) })
				}
			})
		})
	})
	// Accessibility is granted in System Settings; advance once it arrives.
	if step == onboarding.Accessibility && onboarding.Ready(step, state) && p.axRequested {
		if p.axGranted.IsZero() {
			p.axGranted = c.Now()
		}
		if wait := 700*time.Millisecond - c.Now().Sub(p.axGranted); wait > 0 {
			c.After(wait)
		} else {
			p.axRequested = false
			p.move(c, 1, store)
		}
	}
}

// illustration picks the scene that shows what a step enables; the
// microphone step shows the live level once it is allowed.
func illustration(step onboarding.Step, state *runtime.State) (showcase.Scene, []float32) {
	switch step {
	case onboarding.Microphone:
		if state.Permissions.Microphone == runtime.ConsentGranted {
			return showcase.Listening, state.MonitorLevels
		}
		return showcase.Listening, nil
	case onboarding.Try:
		return showcase.Listening, nil
	case onboarding.Accessibility, onboarding.Finished:
		return showcase.Inserted, nil
	}
	return showcase.Dictating, nil
}

func (p *Page) move(c *ui.Context, delta int, store runtime.Store) {
	if p.flow.Move(delta, store.State) {
		p.axGranted = time.Time{}
	}
}

// sync limits microphone monitoring and shortcut recording to their pages.
func (p *Page) sync(c *ui.Context, store runtime.Store, step onboarding.Step) {
	monitor := step == onboarding.Microphone && store.State.Permissions.Microphone == runtime.ConsentGranted
	if monitor != p.monitoring {
		p.monitoring = monitor
		if monitor {
			store.Actions.StartLevelMonitor()
		} else {
			store.Actions.StopLevelMonitor()
		}
	}
	if step != onboarding.Try {
		p.holdPicker.Close(store)
		p.chosen = ""
		p.switched = time.Time{}
	} else if key := store.State.Preferences.HoldKey; key != p.chosen {
		if p.chosen != "" {
			p.switched = c.Now()
		}
		p.chosen = key
	}
}

// Close releases resources if the window closes mid-flow.
func (p *Page) Close(store runtime.Store) { p.release(store) }

func (p *Page) release(store runtime.Store) {
	p.holdPicker.Close(store)
	if p.monitoring {
		p.monitoring = false
		store.Actions.StopLevelMonitor()
	}
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
	case onboarding.Microphone:
		heading(c, tokens, messages.Text(copy.OnbMicTitle), messages.Text(copy.OnbMicBody))
		ui.Column(c).Gap(10).Children(func() {
			switch state.Permissions.Microphone {
			case runtime.ConsentGranted:
				microphone.Meter(c, store, tokens)
				ui.Text(c, messages.Text(copy.OnbMicTest)).FontSize(theme.Caption).TextColor(tokens.Muted)
			case runtime.ConsentDenied, runtime.ConsentRestricted:
				kit.Notice(c, tokens, tokens.Danger, messages.Text(copy.OnbMicDenied), nil).MaxWidth(380)
				permissions.Request(c, store, tokens, permissions.Microphone, messages.Text(copy.OnbMicAllow), true)
			default:
				permissions.Request(c, store, tokens, permissions.Microphone, messages.Text(copy.OnbMicAllow), true)
			}
		})
	case onboarding.Accessibility:
		heading(c, tokens, messages.Text(copy.OnbAXTitle), messages.Text(copy.OnbAXBody))
		ui.Column(c).Gap(10).Children(func() {
			if state.Permissions.Accessibility == runtime.ConsentGranted {
				kit.Status(c.Key("granted"), tokens, icons.CircleCheck, tokens.Success, messages.Text(copy.OnbAXGranted)).
					Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.8}})
				return
			}
			if kit.Button(c.Key("ax"), tokens, kit.Primary, messages.Text(copy.OnbAXAllow), true).Clicked() {
				p.axRequested = true
				store.Actions.RequestAccessibility()
			}
			if p.axRequested {
				ui.Row(c.Key("waiting")).Gap(8).AlignItems(ui.Center).Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
					ui.Spinner(c).Size(14, 14)
					ui.Text(c, messages.Text(copy.OnbAXWaiting)).FontSize(theme.Caption).TextColor(tokens.Muted)
				})
			}
		})
	case onboarding.Model:
		heading(c, tokens, messages.Text(copy.OnbModelTitle), messages.Text(copy.OnbModelBody))
		// Explicit download confirmation precedes permission setup.
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			modelcard.Render(c, store, tokens, true)
		})
	case onboarding.Try:
		heading(c, tokens, messages.Text(copy.OnbTryTitle), messages.Text(copy.OnbTryBody))
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			ui.Row(c).Children(func() { holdkey.Picker(c, store, &p.holdPicker) })
			holdkey.ListenerStatus(c, store, tokens)
			if state.Permissions.Microphone != runtime.ConsentGranted {
				permissions.Request(c, store, tokens, permissions.Microphone, messages.Text(copy.OnbMicAllow), true)
			}
			if state.Permissions.Accessibility != runtime.ConsentGranted {
				if kit.Button(c, tokens, kit.Primary, messages.Text(copy.OnbAXAllow), true).Clicked() {
					store.Actions.RequestAccessibility()
				}
			}
			if state.Model.State == runtime.ModelReady {
				ui.TextArea(c.Key("practice"), &p.practice).Label(messages.Text(copy.OnbTryPlaceholder)).Placeholder(messages.Text(copy.OnbTryPlaceholder)).Height(96)
			} else {
				modelcard.Render(c, store, tokens, true)
			}
		})
	case onboarding.Finished:
		ui.Column(c).Gap(16).Children(func() {
			mark := ui.Box(c).Size(56, 56).Radius(28).Background(tokens.Success)
			drawn := mark.Keyframes("draw", []ui.Keyframe{{Value: 0}, {At: 420 * time.Millisecond, Value: 1, Ease: ui.EaseOut}}, ui.KeyframeOptions{Delay: 120 * time.Millisecond})
			mark.Draw(func(pt *ui.Painter, r ui.Rect) {
				check := (&ui.Path{}).MoveTo(r.X+17, r.Y+29).LineTo(r.X+25, r.Y+37).LineTo(r.X+40, r.Y+20)
				pt.StrokePath(check.Trim(0, drawn), 3.5, tokens.OnFill)
			})
			ui.Column(c).Gap(10).Children(func() {
				ui.Text(c, messages.Text(copy.OnbDoneTitle)).FontSize(theme.Display).FontWeight(700)
				ui.Text(c, messages.Text(copy.OnbDoneBody)).FontSize(theme.Body).TextColor(tokens.Muted).MaxWidth(380)
			})
		})
	}
}
