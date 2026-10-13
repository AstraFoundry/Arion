// Package onboarding is the first-run flow: permissions, the automatically
// chosen model, the shortcut and a first dictation.
package onboarding

import (
	"math"
	"time"

	"arion.local/desktop/ui/entities/onboarding"
	"arion.local/desktop/ui/features/microphone"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/shortcut"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/modelcard"
	"arion.local/desktop/ui/widgets/stepper"
	"github.com/egoist/mygo/ui"
)

// Width and Height size the onboarding window.
const (
	Width  = 640
	Height = 540
)

// Page keeps the flow's own state.
type Page struct {
	flow        onboarding.Flow
	entered     time.Time
	monitoring  bool
	testing     bool
	practice    string
	axRequested bool
	axGranted   time.Time
	recorder    shortcut.Recorder
}

func NewPage() *Page { return &Page{} }

// Render draws the current step.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	state := store.State
	step := p.flow.Step()
	if p.entered.IsZero() {
		p.entered = c.Now()
	}
	p.sync(store, step)
	bar := c.TitleBar()
	ui.Column(c).Fill().Background(tokens.Canvas).Children(func() {
		ui.Row(c).Height(max(bar.Height, 28) + 16).Justify(ui.Center).AlignItems(ui.End).DragWindow().Children(func() {
			current, total := onboarding.Progress(p.flow)
			stepper.Render(c, tokens, current, total)
		})
		ui.Column(c).Grow(1).ClipX().Children(func() {
			enter := &ui.Motion{X: 36 * float32(p.flow.Direction)}
			if p.flow.Direction == 0 {
				enter = &ui.Motion{Y: 10}
			}
			ui.Column(c.Key(step)).Fill().Padding(28, 56).Gap(18).Justify(ui.Center).
				Transition(ui.ElementTransition{Animation: &theme.Page, Enter: enter, Exit: &ui.Motion{Scale: 0.98}}).
				Children(func() { p.step(c, store, tokens, step) })
		})
		ui.Row(c).Padding(16, 24, 20, 24).Gap(10).AlignItems(ui.Center).Children(func() {
			if p.flow.Index > 0 && step != onboarding.Finished {
				ui.Button(c.Key("back"), copy.Back).OnClick(func() { p.move(c, -1, state) })
			}
			ui.Spacer(c)
			switch step {
			case onboarding.Welcome:
				ui.PrimaryButton(c.Key("next"), copy.OnbStart).OnClick(func() { p.move(c, 1, state) })
			case onboarding.Finished:
				ui.PrimaryButton(c.Key("finish"), copy.OnbFinish).OnClick(func() {
					p.release(store)
					store.Actions.CompleteOnboarding()
				})
			default:
				ui.PrimaryButton(c.Key("next"), copy.Continue).Disabled(!onboarding.Ready(step, state)).OnClick(func() { p.move(c, 1, state) })
			}
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
			p.move(c, 1, state)
		}
	}
}

func (p *Page) move(c *ui.Context, delta int, state *runtime.State) {
	moved := false
	if delta > 0 {
		moved = p.flow.Next(state)
	} else if p.flow.Index > 0 {
		p.flow.Back()
		moved = true
	}
	if moved {
		p.entered = c.Now()
	}
}

// sync runs the microphone monitor and shortcut test only on their steps.
func (p *Page) sync(store runtime.Store, step onboarding.Step) {
	monitor := step == onboarding.Microphone && store.State.Permissions.Microphone == runtime.ConsentGranted
	if monitor != p.monitoring {
		p.monitoring = monitor
		if monitor {
			store.Actions.StartLevelMonitor()
		} else {
			store.Actions.StopLevelMonitor()
		}
	}
	testing := step == onboarding.Shortcut
	if testing != p.testing {
		p.testing = testing
		store.Actions.TestShortcut(testing)
	}
}

// Close releases resources if the window closes mid-flow.
func (p *Page) Close(store runtime.Store) { p.release(store) }

func (p *Page) release(store runtime.Store) {
	if p.monitoring {
		p.monitoring = false
		store.Actions.StopLevelMonitor()
	}
	if p.testing {
		p.testing = false
		store.Actions.TestShortcut(false)
	}
	if p.recorder.Listening {
		p.recorder.Listening = false
		store.Actions.SuspendShortcut(false)
	}
}

func heading(c *ui.Context, tokens theme.Tokens, title, body string) {
	ui.Column(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(theme.Title + 2).FontWeight(650).TextAlign(ui.Center)
		ui.Text(c, body).FontSize(theme.Body).TextColor(tokens.Muted).TextAlign(ui.Center).MaxWidth(420)
	})
}

func (p *Page) step(c *ui.Context, store runtime.Store, tokens theme.Tokens, step onboarding.Step) {
	state := store.State
	switch step {
	case onboarding.Welcome:
		ui.Column(c).Gap(22).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Size(84, 84).Radius(24).Background(tokens.Accent).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				c.AnimationFrame()
				kit.LevelBars(c, breathing(c.Now()), 7, 4, 40, 4, tokens.AccentText)
			})
			ui.Text(c, copy.OnbWelcomeTitle).FontSize(theme.Display).FontWeight(700).TextAlign(ui.Center)
			ui.Text(c, copy.OnbWelcomeBody).FontSize(theme.Heading).TextColor(tokens.Muted).TextAlign(ui.Center).MaxWidth(420)
		})
	case onboarding.Microphone:
		heading(c, tokens, copy.OnbMicTitle, copy.OnbMicBody)
		ui.Column(c).Gap(12).AlignItems(ui.Center).Children(func() {
			switch state.Permissions.Microphone {
			case runtime.ConsentGranted:
				microphone.Meter(c, store, tokens)
				ui.Text(c, copy.OnbMicTest).FontSize(theme.Caption).TextColor(tokens.Muted)
			case runtime.ConsentDenied, runtime.ConsentRestricted:
				ui.Text(c, copy.OnbMicDenied).FontSize(theme.Caption).TextColor(tokens.Danger).TextAlign(ui.Center)
				permissions.Request(c, store, permissions.Microphone, copy.OnbMicAllow, true)
			default:
				permissions.Request(c, store, permissions.Microphone, copy.OnbMicAllow, true)
			}
		})
	case onboarding.Accessibility:
		heading(c, tokens, copy.OnbAXTitle, copy.OnbAXBody)
		ui.Column(c).Gap(12).AlignItems(ui.Center).Children(func() {
			if state.Permissions.Accessibility == runtime.ConsentGranted {
				ui.Row(c.Key("granted")).Gap(6).AlignItems(ui.Center).TextColor(tokens.Success).
					Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.8}}).Children(func() {
					ui.Icon(c, icons.CircleCheck).FontSize(16)
					ui.Text(c, copy.OnbAXGranted).FontSize(theme.Body).FontWeight(500)
				})
				return
			}
			if ui.PrimaryButton(c.Key("ax"), copy.OnbAXAllow).Clicked() {
				p.axRequested = true
				store.Actions.RequestAccessibility()
			}
			if p.axRequested {
				ui.Row(c.Key("waiting")).Gap(8).AlignItems(ui.Center).Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
					ui.Spinner(c).Size(14, 14)
					ui.Text(c, copy.OnbAXWaiting).FontSize(theme.Caption).TextColor(tokens.Muted)
				})
			}
		})
	case onboarding.Model:
		heading(c, tokens, copy.OnbModelTitle, copy.OnbModelBody)
		ui.Column(c).Width(460).Margin(0, ui.Auto).Children(func() { modelcard.Render(c, store, tokens, false) })
	case onboarding.Shortcut:
		heading(c, tokens, copy.OnbShortcutTitle, copy.OnbShortcutBody)
		heard := state.Shortcut.Pressed.After(p.entered)
		if !state.Shortcut.Registered {
			// The default is taken by another app: choose another here.
			ui.Column(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, copy.ShortcutTaken).FontSize(theme.Caption).TextColor(tokens.Danger)
				shortcut.Field(c, store, tokens, &p.recorder)
			})
			return
		}
		ui.Column(c).Gap(14).AlignItems(ui.Center).Children(func() {
			keys := ui.Row(c).Gap(6)
			if heard {
				lift := keys.Keyframes("press", []ui.Keyframe{{Value: 0}, {At: 90 * time.Millisecond, Value: 3, Ease: ui.EaseOut}, {At: 260 * time.Millisecond, Value: 0, Ease: ui.EaseOut}},
					ui.KeyframeOptions{Run: uint64(state.Shortcut.Pressed.UnixNano())})
				keys.Translate(0, lift)
			}
			keys.Children(func() {
				for _, key := range kit.Keys(state.Shortcut.Accelerator) {
					cap := kit.Keycap(c, tokens, key, false).Height(40).MinWidth(44).Padding(0, 14)
					if heard {
						cap.Border(1, tokens.Accent).Background(cap.AnimateColor("face", tokens.AccentSoft, theme.Quick))
					}
				}
			})
			label, color := copy.OnbShortcutPress, tokens.Muted
			if heard {
				label, color = copy.OnbShortcutHeard, tokens.Success
			}
			ui.Text(c, label).FontSize(theme.Caption).FontWeight(500).TextColor(color).ContentTransition(ui.ContentTransition{})
		})
	case onboarding.Try:
		heading(c, tokens, copy.OnbTryTitle, copy.OnbTryBody)
		ui.Column(c).Width(460).Margin(0, ui.Auto).Gap(10).Children(func() {
			ui.TextArea(c.Key("practice"), &p.practice).Placeholder(copy.OnbTryPlaceholder).Height(96)
			if state.Model.State != runtime.ModelReady {
				modelcard.Render(c, store, tokens, true)
			}
		})
	case onboarding.Finished:
		ui.Column(c).Gap(16).AlignItems(ui.Center).Children(func() {
			mark := ui.Box(c).Size(64, 64).Radius(32).Background(tokens.Success)
			drawn := mark.Keyframes("draw", []ui.Keyframe{{Value: 0}, {At: 420 * time.Millisecond, Value: 1, Ease: ui.EaseOut}}, ui.KeyframeOptions{Delay: 120 * time.Millisecond})
			mark.Draw(func(pt *ui.Painter, r ui.Rect) {
				check := (&ui.Path{}).MoveTo(r.X+19, r.Y+33).LineTo(r.X+28, r.Y+42).LineTo(r.X+46, r.Y+23)
				pt.StrokePath(check.Trim(0, drawn), 4, ui.RGB(255, 255, 255))
			})
			ui.Text(c, copy.OnbDoneTitle).FontSize(theme.Display).FontWeight(700)
			ui.Text(c, copy.OnbDoneBody).FontSize(theme.Body).TextColor(tokens.Muted).TextAlign(ui.Center).MaxWidth(380)
		})
	}
}

// breathing is an idle voice-like level pattern for the welcome mark.
func breathing(now time.Time) []float32 {
	t := float64(now.UnixMilli()) / 1000
	levels := make([]float32, 7)
	for i := range levels {
		v := 0.35 + 0.3*math.Sin(t*2.1+float64(i)*0.9) + 0.2*math.Sin(t*3.7+float64(i)*1.7)
		levels[i] = float32(math.Max(0.08, math.Min(1, v)))
	}
	return levels
}
