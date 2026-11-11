package permissionflow

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPermissions(t *testing.T) {
	seen := map[Permission]bool{}
	for _, p := range All() {
		if seen[p] {
			t.Errorf("%s twice", p)
		}
		seen[p] = true
		if !p.valid() || p.Title() == "" || p.Title() == string(p) {
			t.Errorf("%s has no title", p)
		}
		if !strings.HasPrefix(macAnchors[p], "Privacy_") {
			t.Errorf("%s has no page in System Settings", p)
		}
	}
	if len(seen) != len(titles) || len(macAnchors) != len(titles) {
		t.Errorf("All has %d permissions, titles %d, anchors %d", len(seen), len(titles), len(macAnchors))
	}
	if got, want := macSettingsURL(ScreenRecording), "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"; got != want {
		t.Errorf("URL %q, want %q", got, want)
	}
	for p, c := range windowsCapabilities {
		if !seen[p] || !strings.HasPrefix(c.uri, "ms-settings:privacy-") || c.capability == "" {
			t.Errorf("Windows' %s: %+v", p, c)
		}
	}
	if macDraggable(Microphone) || !macDraggable(Accessibility) || !macDraggable(FullDiskAccess) {
		t.Error("the wrong pages take drags")
	}
}

func TestUnknownPermission(t *testing.T) {
	p := Permission("telepathy")
	if _, err := Check(p); err == nil {
		t.Error("Check took an unknown permission")
	}
	if _, err := Request(context.Background(), p); err == nil {
		t.Error("Request took an unknown permission")
	}
	if err := OpenSettings(p); err == nil {
		t.Error("OpenSettings took an unknown permission")
	}
	if _, err := Guide(context.Background(), p, GuideOptions{}); err == nil {
		t.Error("Guide took an unknown permission")
	}
	if Allowed(p) {
		t.Error("an unknown permission is allowed")
	}
}

func TestUnsupported(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("the system has permissions")
	}
	for _, p := range All() {
		if _, err := Check(p); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("Check(%s): %v", p, err)
		}
		if err := OpenSettings(p); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("OpenSettings(%s): %v", p, err)
		}
		if !Allowed(p) {
			t.Errorf("%s isn't allowed, which nothing keeps", p)
		}
	}
}

func TestWindowsConsent(t *testing.T) {
	for _, c := range []struct {
		values []string
		want   Status
	}{
		{nil, Granted},
		{[]string{"", "", ""}, Granted},
		{[]string{"Allow", "Allow", "Allow"}, Granted},
		{[]string{"Deny", "Allow", "Allow"}, Denied},
		{[]string{"Allow", "Allow", "deny"}, Denied},
		{[]string{"", "Deny", ""}, Denied},
	} {
		if got := windowsConsent(c.values...); got != c.want {
			t.Errorf("windowsConsent(%q) = %v, want %v", c.values, got, c.want)
		}
	}
}

func TestWait(t *testing.T) {
	ctx := context.Background()
	n := 0
	st, err := wait(ctx, func() (Status, error) {
		if n++; n < 3 {
			return Denied, nil
		}
		return Granted, nil
	}, time.Nanosecond)
	if st != Granted || err != nil || n != 3 {
		t.Errorf("wait = %v, %v after %d checks", st, err, n)
	}

	st, err = wait(ctx, func() (Status, error) { return Unknown, nil }, time.Nanosecond)
	if st != Unknown || err != nil {
		t.Errorf("wait of what can't be told = %v, %v", st, err)
	}

	boom := errors.New("boom")
	if _, err := wait(ctx, func() (Status, error) { return Unknown, boom }, time.Nanosecond); err != boom {
		t.Errorf("wait's error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	checks := 0
	st, err = wait(cancelled, func() (Status, error) {
		if checks++; checks == 2 {
			cancel()
		}
		return NotDetermined, nil
	}, time.Nanosecond)
	if st != NotDetermined || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled wait = %v, %v", st, err)
	}
}

func TestPanelText(t *testing.T) {
	var b strings.Builder
	for _, r := range panelText("MyApp", "Accessibility") {
		if r.bold {
			b.WriteString("**" + r.text + "**")
		} else {
			b.WriteString(r.text)
		}
	}
	if got, want := b.String(), "Drag **MyApp** to the list above to allow **Accessibility**."; got != want {
		t.Errorf("panel says %q, want %q", got, want)
	}
	for path, want := range map[string]string{
		"/Applications/My App.app":  "My App",
		"/Applications/My App.app/": "My App",
		"/usr/local/bin/tool":       "tool",
	} {
		if got := appName(path); got != want {
			t.Errorf("appName(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestPanelFrame(t *testing.T) {
	screen := rect{0, 0, 1512, 944} // the visible part, under the menu bar
	main := 982.0                   // the main display's height

	// System Settings' window, 715×600, at (300, 100) from the top left.
	settings := flipRect(rect{300, 100, 715, 600}, main)
	if want := (rect{300, 282, 715, 600}); settings != want {
		t.Fatalf("flipped %v, want %v", settings, want)
	}
	got := panelFrame(settings, screen, 100)
	if want := (rect{530, 182, 485, 100}); got != want {
		t.Errorf("panel %v, want %v: under the page, to its right edge", got, want)
	}

	// No room under the window: the panel stays on the screen.
	low := flipRect(rect{300, 380, 715, 600}, main)
	if got := panelFrame(low, screen, 100); got.Y != screenInset {
		t.Errorf("panel at %v, off the bottom of the screen", got)
	}

	// Pushed against the right edge.
	right := flipRect(rect{1300, 100, 715, 600}, main)
	if got := panelFrame(right, screen, 100); got.maxX() != screen.maxX()-screenInset {
		t.Errorf("panel %v leaves the screen", got)
	}

	// A narrow window still gets a panel wide enough to read.
	narrow := flipRect(rect{300, 100, 300, 600}, main)
	if got := panelFrame(narrow, screen, 100); got.W != panelMinWidth {
		t.Errorf("panel %v under a narrow window", got)
	}
}

func TestScreenOf(t *testing.T) {
	left := rect{0, 0, 1512, 944}
	right := rect{1512, -200, 1920, 1050}
	if got := screenOf(rect{1600, 100, 700, 500}, []rect{left, right}); got != right {
		t.Errorf("screen %v, want the right one", got)
	}
	if got := screenOf(rect{1400, 100, 200, 500}, []rect{left, right}); got != left {
		t.Errorf("screen %v, want the left one, with most of the window", got)
	}
	if got := screenOf(rect{-5000, 0, 10, 10}, []rect{left, right}); got != left {
		t.Errorf("screen %v for a window off every screen, want the first", got)
	}
}

func TestMainSettingsWindow(t *testing.T) {
	main := rect{300, 100, 715, 600}
	got, ok := mainSettingsWindow([]settingsWindow{
		{bounds: rect{0, 0, 30, 30}, alpha: 1},                // a helper
		{bounds: rect{0, 0, 2000, 2000}, layer: 25, alpha: 1}, // a layer above
		{bounds: rect{0, 0, 1000, 1000}, alpha: 0},            // not showing
		{bounds: main, alpha: 1},
	})
	if !ok || got != main {
		t.Errorf("window %v, %v, want %v", got, ok, main)
	}
	if _, ok := mainSettingsWindow(nil); ok {
		t.Error("a window among none")
	}
}
