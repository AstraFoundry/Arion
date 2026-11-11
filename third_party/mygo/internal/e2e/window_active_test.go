//go:build !js

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// inactiveMark is true in a page whose window is drawn inactive.
const inactiveMark = "document.documentElement.hasAttribute('data-mygo-window-inactive') && !mygo.windowActive"

// TestWindowActive switches the keyboard between two windows, a page and
// native UI, and checks that each is drawn active while it has it, and
// that a window stays active while its focusable flyout has it.
func TestWindowActive(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, X: 40, Y: 40})
	w.Page().LoadHTML(`<body>page</body>`, "")
	waitFor(t, w, "document.body")
	var active atomic.Bool
	native := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, X: 400, Y: 40, Content: ui.View(func(c *ui.Context) {
		active.Store(c.WindowActive())
		ui.Text(c, "native")
	})})

	native.Focus()
	eventually(t, "the native UI's window focused", func() bool { return mygo.FocusedWindow() == native })
	waitFor(t, w, inactiveMark)
	eventually(t, "the native UI drawn active", active.Load)

	w.Focus()
	eventually(t, "the page's window focused", func() bool { return mygo.FocusedWindow() == w })
	waitFor(t, w, "!document.documentElement.hasAttribute('data-mygo-window-inactive') && mygo.windowActive")
	eventually(t, "the native UI drawn inactive", func() bool { return !active.Load() })

	// The page's window keeps its look while its flyout has the keyboard.
	if _, err := w.Page().Eval(`window.marks = 0; new MutationObserver(() => window.marks++).observe(document.documentElement, {attributes: true, attributeFilter: ['data-mygo-window-inactive']}); true`); err != nil {
		t.Fatal(err)
	}
	fl := mygo.NewFlyout(mygo.FlyoutOptions{Parent: w, Anchor: mygo.Rectangle{X: 10, Y: 10}, Width: 200, Height: 120, Focusable: true})
	defer fl.Destroy()
	eventually(t, "the flyout focused", func() bool { return mygo.FocusedWindow() == fl })
	// Once what the focus change sent the page, which comes before a
	// second script, has run.
	_, _ = w.Page().Eval("1")
	waitFor(t, w, "window.marks === 0 && mygo.windowActive")
	if active.Load() {
		t.Error("the native UI drawn active while another window's flyout has the keyboard")
	}
}

// TestMaterialFollowsActive checks the state of macOS's materials: a
// window's follows whether it is active, as AppKit's sidebars do, while
// panels, flyouts and a window SetActiveAppearance keeps active keep
// theirs active, floating over whatever app is in front.
func TestMaterialFollowsActive(t *testing.T) {
	const follows, active = 0, 1 // NSVisualEffectState
	w := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, Vibrancy: mygo.VibrancySidebar, Transparent: true})
	state, ok := materialState(w)
	if !ok {
		t.Skip("the state of materials is macOS's")
	}
	if state != follows {
		t.Errorf("a window's material in state %d, want %d", state, follows)
	}
	panel := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, Panel: true, Vibrancy: mygo.VibrancyHUD, Transparent: true})
	fl := mygo.NewFlyout(mygo.FlyoutOptions{Parent: w, Anchor: mygo.Rectangle{X: 10, Y: 10}, Width: 200, Height: 120, Vibrancy: mygo.VibrancyPopover})
	defer fl.Destroy()
	for name, x := range map[string]*mygo.Window{"panel": panel, "flyout": fl} {
		if state, _ := materialState(x); state != active {
			t.Errorf("a %s's material in state %d, want %d", name, state, active)
		}
	}
	w.SetActiveAppearance(true)
	if state, _ := materialState(w); state != active {
		t.Errorf("kept active: state %d, want %d", state, active)
	}
	w.SetActiveAppearance(false)
	if state, _ := materialState(w); state != follows {
		t.Errorf("no longer kept active: state %d, want %d", state, follows)
	}
}
