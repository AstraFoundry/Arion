//go:build darwin

package e2e

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
)

// inFrontOf reports whether a shows in front of b.
func inFrontOf(a, b *mygo.Window) bool {
	var order []uintptr
	mygo.RunOnMain(func() { order = darwin.TestWindowOrder() })
	i, j := slices.Index(order, a.NativeHandle()), slices.Index(order, b.NativeHandle())
	return i >= 0 && j >= 0 && i < j
}

// pointFromEnv reads the point "x,y" of an environment variable, which
// places the windows of the child window tests where the person running
// them wants them, on a display of their choice.
func pointFromEnv(t *testing.T, name string) (p mygo.Point, ok bool) {
	t.Helper()
	s := os.Getenv(name)
	if s == "" {
		return mygo.Point{}, false
	}
	xs, ys, found := strings.Cut(s, ",")
	x, errX := strconv.Atoi(strings.TrimSpace(xs))
	y, errY := strconv.Atoi(strings.TrimSpace(ys))
	if !found || errX != nil || errY != nil {
		t.Fatalf("%s=%q, want x,y", name, s)
	}
	return mygo.Point{X: x, Y: y}, true
}

// childWindowsOrigin is where the child window tests put their first
// window, each next one 40 pixels further right and down:
// MYGO_E2E_ORIGIN=x,y, or near the top-left of the primary display's work
// area.
func childWindowsOrigin(t *testing.T) mygo.Point {
	t.Helper()
	if p, ok := pointFromEnv(t, "MYGO_E2E_ORIGIN"); ok {
		return p
	}
	work := mygo.Screen.PrimaryDisplay().WorkArea
	return mygo.Point{X: work.X + 80, Y: work.Y + 80}
}

// TestChildWindowAboveRaisedParent raises a child window's parent, which
// AppKit orders in front of the child, as it does when the parent is
// clicked: the child goes back above it, the key parent's too.
func TestChildWindowAboveRaisedParent(t *testing.T) {
	o := childWindowsOrigin(t)
	parent := newWindow(t, mygo.WindowOptions{Title: "Raised parent", Width: 500, Height: 400, X: o.X, Y: o.Y})
	child := newWindow(t, mygo.WindowOptions{Title: "Child of the raised parent", Parent: parent, Width: 260, Height: 200, X: o.X + 40, Y: o.Y + 40})
	waitOnScreen(child)
	eventually(t, "the child above its parent", func() bool { return inFrontOf(child, parent) })

	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	eventually(t, "the child above its focused parent", func() bool { return inFrontOf(child, parent) })

	parent.Focus() // the key window already
	eventually(t, "the child above its parent raised again", func() bool { return inFrontOf(child, parent) })
	if !parent.IsFocused() {
		t.Error("the parent lost the keyboard to its child")
	}
}

// TestChildWindowsOrderByFocus shows two child windows of a window, the
// second above the first, and focuses the first: it comes above the
// second, as any window the user clicks does, and both stay above their
// parent.
func TestChildWindowsOrderByFocus(t *testing.T) {
	o := childWindowsOrigin(t)
	parent := newWindow(t, mygo.WindowOptions{Title: "Children's parent", Width: 500, Height: 400, X: o.X, Y: o.Y})
	first := newWindow(t, mygo.WindowOptions{Title: "First child", Parent: parent, Width: 260, Height: 200, X: o.X + 40, Y: o.Y + 40})
	second := newWindow(t, mygo.WindowOptions{Title: "Second child", Parent: parent, Width: 260, Height: 200, X: o.X + 80, Y: o.Y + 80})
	waitOnScreen(first)
	waitOnScreen(second)
	eventually(t, "the second child above the first", func() bool { return inFrontOf(second, first) })

	first.Focus()
	eventually(t, "the first child focused", first.IsFocused)
	eventually(t, "the first child above the second", func() bool { return inFrontOf(first, second) })
	if !inFrontOf(first, parent) || !inFrontOf(second, parent) {
		t.Error("a child window is no longer above its parent")
	}

	// The parent raised keeps its children in the order they are in.
	parent.Focus()
	eventually(t, "both children above their raised parent, the first in front", func() bool {
		return inFrontOf(first, second) && inFrontOf(second, parent)
	})

	second.Focus()
	eventually(t, "the second child above the first again", func() bool { return inFrontOf(second, first) })
}

// TestChildWindowHidesWithParent miniaturizes and hides a child window's
// parent, which hides the child, and restores and shows it, which shows
// the child again above it. A child hidden by itself stays hidden.
func TestChildWindowHidesWithParent(t *testing.T) {
	o := childWindowsOrigin(t)
	parent := newWindow(t, mygo.WindowOptions{Title: "Parent hiding", Width: 400, Height: 300, X: o.X, Y: o.Y})
	child := newWindow(t, mygo.WindowOptions{Title: "Child hiding with its parent", Parent: parent, Width: 240, Height: 180, X: o.X + 40, Y: o.Y + 40})
	hidden := newWindow(t, mygo.WindowOptions{Title: "Child hidden by itself", Parent: parent, Width: 240, Height: 180, X: o.X + 80, Y: o.Y + 80})
	waitOnScreen(child)
	waitOnScreen(hidden)
	hidden.Hide()

	parent.Minimize()
	eventually(t, "the parent minimized", parent.IsMinimized)
	eventually(t, "the child hidden with its minimized parent", func() bool { return !child.IsVisible() })
	parent.Restore()
	eventually(t, "the child back above its restored parent", func() bool { return child.IsVisible() && inFrontOf(child, parent) })

	parent.Hide()
	eventually(t, "the child hidden with its parent", func() bool { return !child.IsVisible() })
	parent.Show()
	eventually(t, "the child back above its parent", func() bool { return child.IsVisible() && inFrontOf(child, parent) })
	if hidden.IsVisible() {
		t.Error("the child hidden by itself showed with its parent")
	}
}

// TestChildWindowOnAnotherDisplay moves a child window to another display,
// where it stays, showing, rather than being pulled back to its parent's
// or ordered out, and goes above its parent as the parent is raised. It
// needs two displays; MYGO_E2E_OTHER_DISPLAY=x,y is where on the other
// one the child goes, else near the top-left of the work area of a display
// that is not the parent's.
func TestChildWindowOnAnotherDisplay(t *testing.T) {
	displays := mygo.Screen.Displays()
	if len(displays) < 2 {
		t.Skip("one display")
	}
	o := childWindowsOrigin(t)
	parent := newWindow(t, mygo.WindowOptions{Title: "Parent on the first display", Width: 400, Height: 300, X: o.X, Y: o.Y})
	child := newWindow(t, mygo.WindowOptions{Title: "Child moving across displays", Parent: parent, Width: 240, Height: 180, X: o.X + 40, Y: o.Y + 40})
	waitOnScreen(child)
	home := mygo.Screen.DisplayMatching(parent.Bounds())
	there, ok := pointFromEnv(t, "MYGO_E2E_OTHER_DISPLAY")
	if !ok {
		// The first display that is not the parent's, which need not be
		// the primary one, nor the first listed.
		i := slices.IndexFunc(displays, func(d mygo.Display) bool { return d.ID != home.ID })
		work := displays[i].WorkArea
		there = mygo.Point{X: work.X + 60, Y: work.Y + 60}
	}
	other := mygo.Screen.DisplayMatching(mygo.Rectangle{X: there.X, Y: there.Y, Width: 240, Height: 180})
	if other.ID == home.ID {
		t.Fatalf("the other display's point %+v is on the parent's display %d", there, home.ID)
	}

	child.SetPosition(there.X, there.Y)
	want := mygo.Rectangle{X: there.X, Y: there.Y, Width: child.Bounds().Width, Height: child.Bounds().Height}
	eventually(t, "the child at its place on the other display", func() bool { return child.Bounds() == want })
	waitOnScreen(child)

	parent.Focus()
	eventually(t, "the child above its raised parent", func() bool { return inFrontOf(child, parent) })
	if b := child.Bounds(); b != want {
		t.Errorf("the child moved to %+v as its parent was raised, want %+v", b, want)
	}
	if d := mygo.Screen.DisplayMatching(child.Bounds()); d.ID != other.ID {
		t.Errorf("the child is on display %d, want the other one, %d", d.ID, other.ID)
	}
	if !child.IsVisible() {
		t.Error("the child on the other display was ordered out")
	}
}

// TestChildPanelAboveRaisedParent raises the parent of a child window that
// is a panel (WindowOptions.Panel), which NSApp's orderedWindows leaves
// out: it goes back above its parent too, and hides and shows with it.
func TestChildPanelAboveRaisedParent(t *testing.T) {
	o := childWindowsOrigin(t)
	parent := newWindow(t, mygo.WindowOptions{Title: "Child panel's parent", Width: 500, Height: 400, X: o.X, Y: o.Y})
	panel := newWindow(t, mygo.WindowOptions{Title: "Child panel", Parent: parent, Panel: true, Width: 260, Height: 200, X: o.X + 40, Y: o.Y + 40})
	waitOnScreen(panel)
	eventually(t, "the panel above its parent", func() bool { return inFrontOf(panel, parent) })

	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	eventually(t, "the panel above its raised parent", func() bool { return inFrontOf(panel, parent) })

	parent.Hide()
	eventually(t, "the panel hidden with its parent", func() bool { return !panel.IsVisible() })
	parent.Show()
	eventually(t, "the panel back above its parent", func() bool { return panel.IsVisible() && inFrontOf(panel, parent) })
}

// onActiveSpace reports whether w is on the Space that shows.
func onActiveSpace(w *mygo.Window) (on bool) {
	mygo.RunOnMain(func() { on = darwin.TestOnActiveSpace(w.NativeHandle()) })
	return on
}

// windowLevel returns w's window level.
func windowLevel(w *mygo.Window) (l int) {
	mygo.RunOnMain(func() { l = darwin.TestWindowLevel(w.NativeHandle()) })
	return l
}

// TestChildWindowAboveOnTopParent shows a child window of a window that is
// always on top: the child takes its level, staying above it and above the
// windows its parent is above, and follows it as its parent stops being
// on top and is again, while it does not report being on top itself.
func TestChildWindowAboveOnTopParent(t *testing.T) {
	o := childWindowsOrigin(t)
	other := newWindow(t, mygo.WindowOptions{Title: "Window under the on-top parent", Width: 500, Height: 400, X: o.X, Y: o.Y})
	parent := newWindow(t, mygo.WindowOptions{Title: "On-top parent", AlwaysOnTop: true, Width: 400, Height: 300, X: o.X + 40, Y: o.Y + 40})
	child := newWindow(t, mygo.WindowOptions{Title: "Child of the on-top parent", Parent: parent, Width: 240, Height: 180, X: o.X + 80, Y: o.Y + 80})
	waitOnScreen(other)
	waitOnScreen(child)
	eventually(t, "the child above its on-top parent", func() bool { return inFrontOf(child, parent) })
	if l, pl := windowLevel(child), windowLevel(parent); l < pl {
		t.Errorf("the child's level %d is below its parent's %d", l, pl)
	}
	if child.IsAlwaysOnTop() {
		t.Error("the child reports being on top, as its parent is")
	}

	other.Focus()
	eventually(t, "the other window focused", other.IsFocused)
	eventually(t, "the on-top parent and its child above the focused window", func() bool {
		return inFrontOf(child, parent) && inFrontOf(parent, other)
	})

	parent.Focus()
	eventually(t, "the child above its raised on-top parent", func() bool { return inFrontOf(child, parent) })

	parent.SetAlwaysOnTop(false)
	eventually(t, "the parent no longer on top", func() bool { return !parent.IsAlwaysOnTop() })
	if l, pl := windowLevel(child), windowLevel(parent); l != pl {
		t.Errorf("the child's level %d once its parent is no longer on top, want its parent's %d", l, pl)
	}
	other.Focus()
	eventually(t, "the focused window above the parent no longer on top, and its child", func() bool {
		return inFrontOf(other, child) && inFrontOf(child, parent)
	})

	parent.SetAlwaysOnTop(true)
	eventually(t, "the child above its parent on top again, and above the other window", func() bool {
		return inFrontOf(child, parent) && inFrontOf(child, other)
	})
}

// TestChildWindowFollowsFullScreenParent puts a child window's parent in
// full screen, a Space of its own: the child goes there with it, above it,
// and comes back with it.
func TestChildWindowFollowsFullScreenParent(t *testing.T) {
	o := childWindowsOrigin(t)
	parent := newWindow(t, mygo.WindowOptions{Title: "Full-screen parent", Width: 400, Height: 300, X: o.X, Y: o.Y})
	child := newWindow(t, mygo.WindowOptions{Title: "Child of the full-screen parent", Parent: parent, Width: 240, Height: 180, X: o.X + 40, Y: o.Y + 40})
	waitOnScreen(child)
	eventually(t, "the child above its parent", func() bool { return inFrontOf(child, parent) })

	parent.SetFullScreen(true)
	eventually(t, "the parent in full screen", parent.IsFullScreen)
	eventually(t, "the child on its full-screen parent's Space, above it", func() bool {
		return onActiveSpace(parent) && onActiveSpace(child) && child.IsVisible() && inFrontOf(child, parent)
	})

	parent.SetFullScreen(false)
	eventually(t, "the parent out of full screen", func() bool { return !parent.IsFullScreen() })
	eventually(t, "the child back with its parent, above it", func() bool {
		return onActiveSpace(parent) && onActiveSpace(child) && child.IsVisible() && inFrontOf(child, parent)
	})
	parent.Focus()
	eventually(t, "the child above its raised parent after full screen", func() bool { return inFrontOf(child, parent) })
}
