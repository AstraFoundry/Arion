//go:build arion_e2e && (darwin || windows)

// Package e2e exercises the production windows on the actual OS event loop.
package e2e

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/desktop/bootstrap"
	uiapp "arion.local/desktop/ui/app"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

var app *bootstrap.App

func TestMain(m *testing.M) {
	if os.Getenv("ARION_NATIVE_E2E") != "1" {
		fmt.Fprintln(os.Stderr, "native E2E requires ARION_NATIVE_E2E=1 and an interactive desktop session")
		os.Exit(2)
	}
	data, err := os.MkdirTemp("", "arion-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(data)
	// An isolated profile: no stored preferences, no models, no onboarding.
	app, err = bootstrap.New(bootstrap.Options{DataDir: data, SkipOnboarding: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var code atomic.Int32
	code.Store(1)
	app.Start()
	mygo.App.WhenReady(func() {
		go func() {
			code.Store(int32(m.Run()))
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(int(code.Load()))
}

// within polls predicate until it holds or the timeout passes.
func within(timeout time.Duration, predicate func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func eventually(t *testing.T, what string, predicate func() bool) {
	t.Helper()
	if !within(5*time.Second, predicate) {
		t.Fatalf("native condition not met: %s", what)
	}
}

// capture checks a window renders nonblank native content and saves it.
func capture(t *testing.T, window *mygo.Window, name string, minW, minH int) image.Image {
	t.Helper()
	if window.Page() != nil {
		t.Fatal("production UI must use native content")
	}
	data, err := window.CapturePage()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() < minW || img.Bounds().Dy() < minH {
		t.Fatalf("%s: invalid content size %v", name, img.Bounds())
	}
	if directory := os.Getenv("ARION_E2E_OUTPUT"); directory != "" {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, name+".png"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return img
}

func nonblank(img image.Image) bool {
	first := img.At(img.Bounds().Min.X, img.Bounds().Min.Y)
	fr, fg, fb, fa := first.RGBA()
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 2 {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 2 {
			r, g, b, a := img.At(x, y).RGBA()
			if r != fr || g != fg || b != fb || a != fa {
				return true
			}
		}
	}
	return false
}

// Dictating in a fresh profile shows the overlay without taking focus and
// an explicit failure: PERMISSION_DENIED without Accessibility trust (before
// any microphone use, DSK-A01), otherwise MODEL_MISSING since no model is
// provisioned in the isolated profile (DSK-09).
func TestOverlayExplainsFailure(t *testing.T) {
	var overlay *mygo.Window
	eventually(t, "overlay created", func() bool { overlay = app.Windows()["overlay"]; return overlay != nil })
	var want arion.ErrorCode = arion.CodeModelMissing
	app.Update(func(s *arion.State) {
		if s.Permissions.Accessibility != arion.ConsentGranted {
			want = arion.CodePermissionDenied
		}
	})
	app.Coordinator().Start()
	eventually(t, "explicit failure "+string(want), func() bool {
		var failure arion.ErrorCode
		app.Update(func(s *arion.State) { failure = s.Dictation.Failure })
		// A headless CI desktop may have no foreground editor: that is an
		// explicit TARGET_UNSUPPORTED, also before any microphone use.
		return failure == want || (want == arion.CodeModelMissing && failure == arion.CodeTargetUnsupported)
	})
	eventually(t, "overlay shown", overlay.IsVisible)
	if overlay.IsFocused() {
		t.Fatal("overlay took focus")
	}
	eventually(t, "overlay pill drawn", func() bool {
		return nonblank(capture(t, overlay, "overlay-failure", uiapp.OverlayWidth, uiapp.OverlayHeight))
	})
	time.Sleep(600 * time.Millisecond)
	capture(t, overlay, "overlay-failure", uiapp.OverlayWidth, uiapp.OverlayHeight)
}

func TestSettingsWindowLifecycle(t *testing.T) {
	app.OpenSettings()
	var settings *mygo.Window
	eventually(t, "settings opened", func() bool { settings = app.Windows()["settings"]; return settings != nil && settings.IsVisible() })
	if !nonblank(capture(t, settings, "settings", 640, 440)) {
		t.Fatal("settings content blank")
	}
	settings.SetContentSize(820, 600)
	if !within(5*time.Second, func() bool { w, h := settings.ContentSize(); return w == 820 && h == 600 }) {
		w, h := settings.ContentSize()
		t.Fatalf("settings content size %dx%d after resize to 820x600", w, h)
	}
	var closed atomic.Int32
	settings.OnClosed(func() { closed.Add(1) })
	settings.Close()
	eventually(t, "settings destroyed", settings.IsDestroyed)
	eventually(t, "one close callback", func() bool { return closed.Load() == 1 })
	eventually(t, "settings released", func() bool { return app.Windows()["settings"] == nil })
}

func TestOnboardingWindowLifecycle(t *testing.T) {
	app.OpenOnboarding()
	var onboarding *mygo.Window
	eventually(t, "onboarding opened", func() bool {
		onboarding = app.Windows()["onboarding"]
		return onboarding != nil && onboarding.IsVisible()
	})
	if !nonblank(capture(t, onboarding, "onboarding", uiapp.OnboardingWidth, uiapp.OnboardingHeight)) {
		t.Fatal("onboarding content blank")
	}
	onboarding.Close()
	eventually(t, "onboarding destroyed", onboarding.IsDestroyed)
}
