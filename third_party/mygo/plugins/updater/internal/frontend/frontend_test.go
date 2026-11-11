package frontend

import (
	"testing"

	"github.com/egoist/mygo"
)

type fakeWindow struct {
	visible, focused, inactive bool
	closed                     []func()
}

func (w *fakeWindow) Show()           { w.visible = true }
func (w *fakeWindow) ShowInactive()   { w.visible, w.inactive = true, true }
func (w *fakeWindow) Focus()          { w.focused = true }
func (w *fakeWindow) IsVisible() bool { return w.visible }
func (w *fakeWindow) OnClosed(fn func()) func() {
	w.closed = append(w.closed, fn)
	return func() { w.closed = nil }
}

// fakeApp is the app as Show sees it, on GOOS os, until the test ends.
type fakeApp struct {
	active, focused bool
	policy          mygo.ActivationPolicy
	listeners       []func()
}

func useApp(t *testing.T, os string) *fakeApp {
	a := &fakeApp{policy: mygo.ActivationPolicyRegular}
	prev := [...]any{goos, appActive, activationPolicy, focusApp, onAppActive}
	goos = os
	appActive = func() bool { return a.active }
	activationPolicy = func() mygo.ActivationPolicy { return a.policy }
	focusApp = func() { a.focused = true }
	onAppActive = func(fn func()) func() {
		a.listeners = append(a.listeners, fn)
		return func() { a.listeners = nil }
	}
	t.Cleanup(func() {
		goos = prev[0].(string)
		appActive = prev[1].(func() bool)
		activationPolicy = prev[2].(func() mygo.ActivationPolicy)
		focusApp = prev[3].(func())
		onAppActive = prev[4].(func(func()) func())
	})
	return a
}

func (a *fakeApp) activate() {
	a.active = true
	for _, fn := range a.listeners {
		fn()
	}
}

// A background check while the app is active shows the window in front, with
// the keyboard, as Sparkle does; it used to show it behind the app's window.
func TestShowActiveApp(t *testing.T) {
	for _, os := range []string{"darwin", "windows", "linux"} {
		a := useApp(t, os)
		a.active = true
		w := &fakeWindow{}
		show(w, false)
		if !w.visible || !w.focused {
			t.Errorf("%s: visible %v, focused %v; want the window in front", os, w.visible, w.focused)
		}
	}
}

// On macOS a background check while another app is in front waits for the
// app to become active, then shows the window in front, once.
func TestShowWaitsForActivation(t *testing.T) {
	a := useApp(t, "darwin")
	w := &fakeWindow{}
	show(w, false)
	if w.visible {
		t.Fatal("the window showed while the app was in the background")
	}
	a.activate()
	if !w.visible || !w.focused || w.inactive {
		t.Fatalf("visible %v, focused %v, inactive %v once active; want it in front", w.visible, w.focused, w.inactive)
	}
	if len(a.listeners) != 0 || len(w.closed) != 0 {
		t.Error("still watching the app after showing the window")
	}
}

// A window closed while it waits for the app, as when its session ends,
// never shows.
func TestShowClosedWhileWaiting(t *testing.T) {
	a := useApp(t, "darwin")
	w := &fakeWindow{}
	show(w, false)
	for _, fn := range w.closed {
		fn()
	}
	a.activate()
	if w.visible {
		t.Error("a closed window showed when the app became active")
	}
}

// An app without a Dock icon may never become active by itself, so a
// background check brings it to the front with the window, as Sparkle does.
func TestShowAccessoryApp(t *testing.T) {
	a := useApp(t, "darwin")
	a.policy = mygo.ActivationPolicyAccessory
	w := &fakeWindow{}
	show(w, false)
	if !a.focused || !w.visible || !w.focused {
		t.Errorf("app focused %v, window visible %v, focused %v; want both in front", a.focused, w.visible, w.focused)
	}
}

// Elsewhere a background check while another app is in front shows the
// window without the keyboard.
func TestShowInactiveElsewhere(t *testing.T) {
	useApp(t, "windows")
	w := &fakeWindow{}
	show(w, false)
	if !w.visible || w.focused {
		t.Errorf("visible %v, focused %v; want it shown without the keyboard", w.visible, w.focused)
	}
}
