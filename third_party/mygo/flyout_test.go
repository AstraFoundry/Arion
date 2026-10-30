//go:build !js

package mygo

import (
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

// testFlyout creates a flyout and returns it with its fake native window.
func testFlyout(t *testing.T, opts FlyoutOptions) (*Window, *fake.Window) {
	t.Helper()
	fl := NewFlyout(opts)
	t.Cleanup(fl.Destroy)
	wins := fb.Windows()
	return fl, wins[len(wins)-1]
}

func TestFlyoutOptions(t *testing.T) {
	parent, _ := testWindow(t, WindowOptions{X: 100, Y: 100, Width: 400, Height: 300})
	fl, fw := testFlyout(t, FlyoutOptions{
		Parent:    parent,
		Anchor:    Rectangle{X: 10, Y: 20, Width: 80, Height: 24},
		Placement: PlacementRightEnd,
		Gap:       4,
		Width:     150,
		Focusable: true,
		Content:   ui.View(func(c *ui.Context) {}),
	})
	o := fw.Opts
	want := platform.Flyout{Anchor: platform.Rect{X: 10, Y: 20, Width: 80, Height: 24}, Side: platform.SideRight, Align: platform.AlignEnd, Gap: 4, Focusable: true}
	if o.Flyout == nil || *o.Flyout != want {
		t.Fatalf("platform flyout %+v, want %+v", o.Flyout, want)
	}
	if !o.Frameless || !o.Transparent || !o.SkipTaskbar || o.Resizable || o.HasShadow || o.Center || !o.Focusable || o.Parent == nil {
		t.Errorf("flyout window options %+v", o)
	}
	if o.Width != 150 || o.Height != 200 || !o.Surface {
		t.Errorf("size %dx%d, surface %v; want 150x200 of native UI", o.Width, o.Height, o.Surface)
	}
	if !fl.IsFlyout() || parent.IsFlyout() || fl.Parent() != parent {
		t.Errorf("IsFlyout %v, parent's %v", fl.IsFlyout(), parent.IsFlyout())
	}
	if !fl.IsVisible() || !fl.IsFocused() {
		t.Errorf("a focusable flyout shows focused: visible %v, focused %v", fl.IsVisible(), fl.IsFocused())
	}
	// Right of the anchor, their bottoms lined up, which would cross the
	// top of the work area (25): their tops instead.
	if got := fl.Bounds(); got != (Rectangle{X: 100 + 10 + 80 + 4, Y: 100 + 20, Width: 150, Height: 200}) {
		t.Errorf("flyout at %+v", got)
	}
}

func TestFlyoutPlacement(t *testing.T) {
	parent, pw := testWindow(t, WindowOptions{X: 100, Y: 100, Width: 400, Height: 300})
	anchor := Rectangle{X: 10, Y: 20, Width: 80, Height: 24}
	fl, fw := testFlyout(t, FlyoutOptions{Parent: parent, Anchor: anchor, Gap: 4, Width: 200, Height: 100})
	if fl.IsFocused() || !fl.IsVisible() {
		t.Errorf("a flyout shows without the keyboard: visible %v, focused %v", fl.IsVisible(), fl.IsFocused())
	}
	if got := fl.Bounds(); got != (Rectangle{X: 110, Y: 148, Width: 200, Height: 100}) {
		t.Errorf("flyout at %+v, want below its anchor", got)
	}

	// It follows its parent.
	parent.SetPosition(300, 200)
	if got := fl.Bounds(); got != (Rectangle{X: 310, Y: 248, Width: 200, Height: 100}) {
		t.Errorf("flyout at %+v after its parent moved", got)
	}
	// The fake display's work area ends at 900: above the anchor there.
	onMain(func() { pw.SetBounds(platform.Rect{X: 300, Y: 800, Width: 400, Height: 300}) })
	if got := fl.Bounds(); got != (Rectangle{X: 310, Y: 800 + 20 - 4 - 100, Width: 200, Height: 100}) {
		t.Errorf("flyout at %+v, want above its anchor", got)
	}

	parent.SetPosition(100, 100)
	fl.SetAnchor(Rectangle{X: 50, Y: 60})
	fl.SetSize(120, 90)
	if got := fl.Bounds(); got != (Rectangle{X: 150, Y: 164, Width: 120, Height: 90}) {
		t.Errorf("flyout at %+v after SetAnchor and SetSize", got)
	}
	fl.SetContentSize(100, 80)
	if f := fw.Flyout; f.Anchor != (platform.Rect{X: 50, Y: 60}) || fl.Bounds().Width != 100 {
		t.Errorf("flyout placed at %+v, %+v after SetContentSize", f, fl.Bounds())
	}

	// SetAnchor does nothing to other windows.
	before := parent.Bounds()
	parent.SetAnchor(Rectangle{X: 1, Y: 1})
	if parent.Bounds() != before {
		t.Error("SetAnchor moved a window that is not a flyout")
	}

	parent.Destroy()
	if !fl.IsDestroyed() {
		t.Error("the flyout outlived its parent")
	}
}

func TestFlyoutValidation(t *testing.T) {
	expectPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: no panic", name)
			}
		}()
		fn()
	}
	parent, _ := testWindow(t, WindowOptions{})
	expectPanic("no parent", func() { NewFlyout(FlyoutOptions{}) })
	expectPanic("unknown placement", func() { NewFlyout(FlyoutOptions{Parent: parent, Placement: "below"}) })
}

// TestFlyoutEscape closes a flyout on an Escape its page did not handle,
// as the user closing it, and no other window.
func TestFlyoutEscape(t *testing.T) {
	parent, pw := testWindow(t, WindowOptions{})
	fl, fw := testFlyout(t, FlyoutOptions{Parent: parent, Focusable: true})
	keep := true
	closes := 0
	fl.OnClose(func(e *CloseEvent) {
		closes++
		if keep {
			e.PreventDefault()
		}
	})
	escape := func(w *fake.Window) {
		page(w, `{"t":"escape"}`)
		onMain(func() {}) // the close posted after the message
	}
	escape(fw)
	if closes != 1 || fl.IsDestroyed() {
		t.Fatalf("OnClose heard %d closes, destroyed %v", closes, fl.IsDestroyed())
	}
	keep = false
	escape(fw)
	if !fl.IsDestroyed() {
		t.Error("Escape did not close the flyout")
	}
	escape(pw)
	if parent.IsDestroyed() {
		t.Error("Escape closed a window that is not a flyout")
	}
}

func TestTrayFlyout(t *testing.T) {
	tray, err := NewTray(TrayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer tray.Destroy()
	fl, fw := testFlyout(t, FlyoutOptions{Tray: tray, Placement: PlacementBottom, Gap: 2, Width: 300, Height: 200, Focusable: true})
	if fw.Opts.Parent != nil || fw.Opts.Flyout.Tray == nil {
		t.Errorf("a tray's flyout has parent %v, tray %v", fw.Opts.Parent, fw.Opts.Flyout.Tray)
	}
	// Below the icon, centered on it, and slid back into the work area.
	if got := fl.Bounds(); got != (Rectangle{X: 1140, Y: 27, Width: 300, Height: 200}) {
		t.Errorf("tray flyout at %+v", got)
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("a flyout with a parent and a tray did not panic")
			}
		}()
		parent, _ := testWindow(t, WindowOptions{})
		NewFlyout(FlyoutOptions{Parent: parent, Tray: tray})
	}()
}

func TestScreenAnchoredFlyout(t *testing.T) {
	anchor := Rectangle{X: 300, Y: 220, Width: 1, Height: 18}
	fl, fw := testFlyout(t, FlyoutOptions{ScreenAnchor: &anchor, Width: 270, Height: 198, Gap: 4, Focusable: true})
	if fw.Opts.Parent != nil || fw.Opts.Flyout.Tray != nil || !fl.IsFocused() {
		t.Fatal("screen flyout acquired an owner or failed to take the keyboard")
	}
	if got := fl.Bounds(); got != (Rectangle{X: 300, Y: 242, Width: 270, Height: 198}) {
		t.Fatalf("screen anchor placement: %+v", got)
	}
	fl.SetAnchor(Rectangle{X: 400, Y: 880})
	if got := fl.Bounds(); got.Y != 678 {
		t.Fatalf("screen flyout did not flip above work-area edge: %+v", got)
	}
	fl.SetContentSize(270, 100)
	if got := fl.Bounds(); got.Y != 776 || got.Height != 100 {
		t.Fatalf("screen flyout resize: %+v", got)
	}
}
