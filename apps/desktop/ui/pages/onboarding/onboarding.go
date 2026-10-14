// Package onboarding is the first-run flow: permissions, the automatically
// chosen model, the shortcut, a first dictation and the choice between
// staying local and signing in for sync, each step beside an illustration
// of what it enables.
package onboarding

import (
	"time"

	"arion.local/desktop/ui/entities/account"
	"arion.local/desktop/ui/entities/onboarding"
	"arion.local/desktop/ui/features/microphone"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/features/shortcut"
	"arion.local/desktop/ui/features/signin"
	"arion.local/desktop/ui/features/uilanguage"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
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

// Render draws the current step beside its illustration.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	tokens := theme.Apply(c)
	state := store.State
	step := p.flow.Step()
	if p.entered.IsZero() {
		p.entered = c.Now()
	}
	p.sync(store, step)
	bar := c.TitleBar()
	top := max(bar.Height, 28) + 16
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(tokens.Canvas).Children(func() {
		ui.Column(c).Width(Pane).Children(func() {
			scene, levels := illustration(step, state)
			showcase.Render(c, tokens, store.Copy(), scene, state.Shortcut.Accelerator, levels)
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(top).DragWindow()
		})
		ui.Column(c).Grow(1).Children(func() {
			ui.Row(c).Height(top).Padding(0, Inset).AlignItems(ui.End).DragWindow().Children(func() {
				current, total := onboarding.Progress(p.flow)
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
					kit.Button(c.Key("back"), tokens, kit.Ghost, messages.Text(copy.Back), true).OnClick(func() { p.move(c, -1, state) })
				}
				ui.Spacer(c)
				switch step {
				case onboarding.Welcome:
					kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.OnbStart), true).OnClick(func() { p.move(c, 1, state) })
				case onboarding.Sync:
					// Signed out, the step's own buttons are the two paths.
					if account.StageOf(state.Account) != account.SignedOut {
						kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.Continue), onboarding.Ready(step, state)).OnClick(func() { p.move(c, 1, state) })
					}
				case onboarding.Finished:
					kit.Button(c.Key("finish"), tokens, kit.Primary, messages.Text(copy.OnbFinish), true).OnClick(func() {
						p.release(store)
						store.Actions.CompleteOnboarding()
					})
				default:
					kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(copy.Continue), onboarding.Ready(step, state)).OnClick(func() { p.move(c, 1, state) })
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
			p.move(c, 1, state)
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
	case onboarding.Shortcut:
		return showcase.Listening, nil
	case onboarding.Accessibility, onboarding.Finished:
		return showcase.Inserted, nil
	}
	return showcase.Dictating, nil
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

// sync runs the microphone monitor and shortcut test only on their steps,
// and reads whether a sign-in is stored on the sync step.
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
	if step == onboarding.Sync {
		signin.Check(store)
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
				ui.Text(c, messages.Text(copy.OnbWelcomeBody)).FontSize(theme.Heading).TextColor(tokens.Muted).MaxWidth(380)
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
				ui.Row(c).Gap(6).MaxWidth(380).Children(func() {
					ui.Icon(c, icons.TriangleAlert).FontSize(13).TextColor(tokens.Danger).Margin(1, 0, 0, 0)
					ui.Text(c, messages.Text(copy.OnbMicDenied)).FontSize(theme.Caption).Shrink(1)
				})
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
		// Recognition and the optional polishing are set up together.
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			modelcard.Render(c, store, tokens, true)
			kit.Card(c, tokens).Children(func() { polish.Row(c, store, tokens) })
		})
	case onboarding.Shortcut:
		heading(c, tokens, messages.Text(copy.OnbShortcutTitle), messages.Text(copy.OnbShortcutBody))
		heard := state.Shortcut.Pressed.After(p.entered)
		if !state.Shortcut.Registered {
			// The default is taken by another app: choose another here.
			ui.Column(c).Gap(8).AlignItems(ui.Start).Children(func() {
				ui.Text(c, messages.Text(copy.ShortcutTaken)).FontSize(theme.Caption).TextColor(tokens.Danger)
				shortcut.Field(c, store, tokens, &p.recorder)
			})
			return
		}
		ui.Column(c).Gap(12).Children(func() {
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
						cap.Border(1, tokens.AccentInk).Background(cap.AnimateColor("face", tokens.AccentSoft, theme.Quick))
					}
				}
			})
			if heard {
				kit.Status(c.Key("heard"), tokens, icons.CircleCheck, tokens.Success, messages.Text(copy.OnbShortcutHeard))
			} else {
				ui.Text(c.Key("press"), messages.Text(copy.OnbShortcutPress)).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.Muted)
			}
		})
	case onboarding.Try:
		heading(c, tokens, messages.Text(copy.OnbTryTitle), messages.Text(copy.OnbTryBody))
		ui.Column(c).Width(Content).Gap(10).Children(func() {
			ui.TextArea(c.Key("practice"), &p.practice).Placeholder(messages.Text(copy.OnbTryPlaceholder)).Height(96)
			if state.Model.State != runtime.ModelReady {
				modelcard.Render(c, store, tokens, true)
			}
		})
	case onboarding.Sync:
		heading(c, tokens, messages.Text(copy.OnbSyncTitle), messages.Text(copy.OnbSyncBody))
		ui.Column(c).Width(Content).Gap(12).Children(func() {
			if account.StageOf(state.Account) != account.SignedOut {
				signin.Card(c, store, tokens)
				if state.Account.SignedIn() {
					signin.HostedSync(c, store, tokens)
				}
				return
			}
			ui.Row(c).Gap(8).Children(func() {
				kit.Button(c.Key("local"), tokens, kit.Primary, messages.Text(copy.OnbContinueLocally), true).OnClick(func() { p.move(c, 1, state) })
				signin.Start(c, store, tokens, kit.Outline, messages.Text(copy.OnbSignInSync))
			})
			ui.Text(c, messages.Text(copy.AccNoDecrypt)).FontSize(theme.Caption).TextColor(tokens.Muted)
			signin.Notice(c, store, tokens)
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
