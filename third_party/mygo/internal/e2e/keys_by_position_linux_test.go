//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestKeysByPositionGTK presses keys through XTEST, as a keyboard does, and sees
// them reach an element by position, down and up, with no name for them.
func TestKeysByPositionGTK(t *testing.T) {
	type seen struct {
		kind ui.InputKind
		code ui.KeyCode
		mods ui.Modifiers
	}
	var mu sync.Mutex
	var got []seen
	frames := make(chan struct{}, 1)
	view := func(c *ui.Context) {
		ui.Box(c).Grow(1).Focusable().AutoFocus().HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputKeyDown || ev.Kind == ui.InputKeyUp {
				mu.Lock()
				got = append(got, seen{ev.Kind, ev.Code, ev.Mods})
				mu.Unlock()
				return true
			}
			return false
		})
		select {
		case frames <- struct{}{}:
		default:
		}
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Keys by position", Width: 300, Height: 200, Content: ui.View(view)})
	<-frames
	if !click(w, 150, 100) {
		t.Skip("pointer automation not available")
	}
	for _, c := range []struct {
		keysym string
		code   ui.KeyCode
		mods   ui.Modifiers // while pressed
	}{
		{"Super_L", ui.CodeSuperLeft, ui.Super},
		{"Super_R", ui.CodeSuperRight, ui.Super},
		{"Shift_L", ui.CodeShiftLeft, ui.Shift},
		{"Shift_R", ui.CodeShiftRight, ui.Shift},
		{"Control_R", ui.CodeControlRight, ui.Ctrl},
		{"Alt_L", ui.CodeAltLeft, ui.Alt},
		{"Caps_Lock", ui.CodeCapsLock, 0},
		{"Scroll_Lock", ui.CodeScrollLock, 0},
		{"Pause", ui.CodePause, 0},
		{"KP_Add", ui.CodeNumpadAdd, 0},
	} {
		mu.Lock()
		got = nil
		mu.Unlock()
		tap := func() bool {
			return fake(func() bool { return linux.TestHoldKey(c.keysym, true) && linux.TestHoldKey(c.keysym, false) })
		}
		if !tap() {
			t.Skip("key automation not available")
		}
		if c.keysym == "Caps_Lock" || c.keysym == "Scroll_Lock" {
			// The X server outlives the test: a lock left on changes the keys
			// of the tests after this one, as Caps Lock turns l into L.
			t.Cleanup(func() { tap() })
		}
		eventually(t, c.keysym+" down and up", func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(got) >= 2
		})
		mu.Lock()
		down, up := got[0], got[len(got)-1]
		mu.Unlock()
		if down.kind != ui.InputKeyDown || down.code != c.code || down.mods != c.mods {
			t.Errorf("%s pressed: %+v, want code %#x, mods %v", c.keysym, down, uint16(c.code), c.mods)
		}
		if up.kind != ui.InputKeyUp || up.code != c.code || up.mods != 0 {
			t.Errorf("%s released: %+v, want code %#x, no mods", c.keysym, up, uint16(c.code))
		}
	}
}
