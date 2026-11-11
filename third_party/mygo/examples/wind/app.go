package main

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// app is the window's state: the texture, and the draft in the composer.
type app struct {
	tex *texture
	// draft is the composer's text, and typed what the texture last saw
	// of it: typing keeps the wind blowing.
	draft, typed string
	// controls are where the controls were in the last frame, in the
	// pane: a press there is theirs, and blows no ripple.
	controls []ui.Rect
}

func newApp(seed uint32) *app {
	return &app{tex: newTexture(seed, nil)}
}

func (s *app) view(c *ui.Context) {
	t := c.Theme()
	// While the desktop asks for less motion, the texture shows the wind
	// as it stands.
	s.tex.still = c.Preferences().ReduceMotion
	if c.Shortcut(ui.Cmd, ui.KeyR) {
		s.tex.reseed(rand.Uint32())
	}
	if s.draft != s.typed {
		s.typed = s.draft
		s.tex.wake()
	}
	w, _ := c.Size()

	pane := ui.Column(c).Fill().AlignItems(ui.Center).Background(t.Background)
	pane.Draw(func(p *ui.Painter, r ui.Rect) { s.tex.draw(p, r, t.Dark) })
	// The pointer stirs the air wherever it is over the pane, but for the
	// controls, which take it themselves.
	pane.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputPointerMove:
			calm := s.tex.calm
			s.tex.move(ev.X, ev.Y, time.Now())
			// A frame, to wake the texture as it rests.
			return calm
		case ui.InputPointerDown:
			if ev.Button == 0 && !s.onControl(ev.X, ev.Y) {
				s.tex.tap(ev.X, ev.Y, time.Now())
				return true
			}
		case ui.InputKeyDown:
			s.tex.wake()
		}
		return false
	})
	pane.Children(func() {
		// The card's middle lies where the texture is calmest, 45% down.
		ui.Spacer(c).Grow(45)
		ui.Column(c).Width(min(640, w-48)).Gap(12).Children(func() {
			input := ui.TextInput(c.Key("draft"), &s.draft).Placeholder("Describe a task or ask a question").
				Label("Task").Padding(14, 16).Radius(14).Background(t.Surface).
				Shadow(0, 4, 18, 0, ui.Hex("#0000000f"))
			s.controls = append(s.controls[:0], input.Bounds())
			ui.Text(c, fmt.Sprintf("Move the pointer to warm the air, click to blow a ripple (every third a heart), %s for another wind.", shortcutLabel())).
				FontSize(12).TextColor(t.TextMuted).TextAlign(ui.Center)
		})
		ui.Spacer(c).Grow(55)
		ui.Row(c).AlignSelf(ui.End).Padding(0, 14, 10, 0).Children(func() {
			ui.Text(c, fmt.Sprintf("%s/%08x", s.tex.w.look, s.tex.w.seed)).FontSize(11).Font(family).TextColor(t.TextMuted)
		})
	})
}

// shortcutLabel returns how the platform writes the shortcut of another
// wind.
func shortcutLabel() string {
	if ui.Cmd == ui.Super {
		return "⌘R"
	}
	return "Ctrl+R"
}

// onControl reports whether (x, y) in the pane lies on a control.
func (s *app) onControl(x, y float32) bool {
	for _, r := range s.controls {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return true
		}
	}
	return false
}
