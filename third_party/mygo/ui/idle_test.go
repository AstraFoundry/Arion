package ui

import (
	"testing"
	"time"
)

// TestSettledWindowSleeps checks that a window whose content stopped
// changing asks for nothing more, whatever settled it: no frame, and no
// timer that would wake the app for one. A window left open all day
// otherwise spends battery on frames that draw what shows already.
func TestSettledWindowSleeps(t *testing.T) {
	n, on, name := 0, false, ""
	var input Rect
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Center().Gap(16).Children(func() {
			Textf(c, "%d", n).FontSize(56).Bold()
			Row(c).Gap(8).Children(func() {
				if Button(c.Key("decrement"), "−").Label("Decrement").Width(44).Clicked() {
					n--
				}
				if PrimaryButton(c.Key("increment"), "+").Label("Increment").Width(44).Tooltip("Count up").Clicked() {
					n++
				}
			})
			Switch(c, &on)
			input = TextInput(c, &name).Width(160).Bounds()
		})
	}, 360, 300)
	steps := []struct {
		what string
		do   func()
	}{
		{"opened", func() {}},
		{"pointed at a button with a tooltip", func() {
			r, _ := tt.Find("+")
			tt.Move(r.X+r.W/2, r.Y+r.H/2)
		}},
		{"clicked", func() { tt.Click("+") }},
		{"toggled a switch", func() { tt.Key(0, KeyTab); tt.Key(0, KeySpace) }},
		{"typed into a text input", func() {
			tt.ClickAt(input.X+input.W/2, input.Y+input.H/2)
			tt.Type("hi")
		}},
		{"left with the caret in a text input", func() { tt.SetFocused(false) }},
		{"focused again", func() { tt.SetFocused(true) }},
		{"drawn inactive", func() { tt.SetWindowActive(false) }},
		{"drawn active again", func() { tt.SetWindowActive(true) }},
		{"switched to dark", func() { tt.SetDark(true) }},
		{"resized", func() { tt.SetSize(400, 320) }},
	}
	for _, step := range steps {
		step.do()
		// Transitions, the tooltip's delay and the caret's blinking end
		// within a minute.
		tt.Advance(time.Minute)
		// Each frame or timer due wakes the app.
		wakeups := 0
		for end := tt.Now().Add(time.Hour); ; wakeups++ {
			at, ok := tt.due()
			if !ok || at.After(end) {
				break
			}
			tt.Advance(at.Sub(tt.Now()))
		}
		if wakeups != 0 {
			t.Errorf("%s, then settled: %d wakeups in an hour", step.what, wakeups)
		}
		if tt.FramePending() || !tt.h.afterAt.IsZero() || !tt.h.repaintAt.IsZero() || tt.rt.animating || tt.rt.repainting {
			t.Errorf("%s, then settled: a frame asked for %v, timers at %v and %v, animating %v, repainting %v",
				step.what, tt.FramePending(), tt.h.afterAt, tt.h.repaintAt, tt.rt.animating, tt.rt.repainting)
		}
		if step.what == "pointed at a button with a tooltip" && !tt.HasText("Count up") {
			t.Error("the tooltip did not show")
		}
	}
	if n != 1 || !on || name != "hi" {
		t.Errorf("the steps counted %d, switched %v, typed %q", n, on, name)
	}
}
