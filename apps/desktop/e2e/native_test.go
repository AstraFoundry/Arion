//go:build arion_e2e && (darwin || windows)

// Package e2e exercises the production windows on the actual OS event loop.
package e2e

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/desktop/bootstrap"
	preferences "arion.local/desktop/infrastructure/settings"
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/pages/settings"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

var app *bootstrap.App
var profileDir string

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
	profileDir = data
	// An isolated profile: no stored preferences, no models, no onboarding.
	app, err = bootstrap.New(bootstrap.Options{DataDir: data, SkipOnboarding: true, Offline: true})
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
	os.RemoveAll(data)
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

// The pill is an owned status surface, not a document that can be closed.
// If its native window is destroyed, the next session must restore it.
func TestOverlayRestoredAfterDestroy(t *testing.T) {
	var original *mygo.Window
	eventually(t, "overlay exists", func() bool {
		original = app.Windows()["overlay"]
		return original != nil
	})
	if original.IsClosable() {
		t.Fatal("status overlay exposes a close action")
	}
	original.Destroy()
	eventually(t, "destroyed overlay released", func() bool { return app.Windows()["overlay"] == nil })
	app.Coordinator().Start()
	var restored *mygo.Window
	eventually(t, "next session restores overlay", func() bool {
		restored = app.Windows()["overlay"]
		return restored != nil && restored != original && restored.IsVisible()
	})
	if restored.IsFocused() || restored.IsClosable() {
		t.Fatal("restored overlay can take focus or be closed")
	}
}

func TestSettingsWindowLifecycle(t *testing.T) {
	app.OpenSettings()
	var settings *mygo.Window
	eventually(t, "settings opened", func() bool { settings = app.Windows()["settings"]; return settings != nil && settings.IsVisible() })
	if !nonblank(capture(t, settings, "settings", 640, 440)) {
		t.Fatal("settings content blank")
	}
	// Outer bounds: MyGo v0.4.6 on Windows maps content size to the outer
	// size of a hidden-title-bar window, which Windows 10 then narrows by
	// the resize frame (docs/research/mygo.md).
	settings.SetSize(820, 600)
	if !within(5*time.Second, func() bool { w, h := settings.Size(); return w == 820 && h == 600 }) {
		w, h := settings.Size()
		t.Fatalf("settings size %dx%d after resize to 820x600", w, h)
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

// DSK-A15: finishing setup keeps Arion visible and saves completion.
func TestOnboardingCompletionOpensSettings(t *testing.T) {
	app.OpenOnboarding()
	var onboarding *mygo.Window
	eventually(t, "onboarding opened", func() bool {
		onboarding = app.Windows()["onboarding"]
		return onboarding != nil && onboarding.IsVisible()
	})
	mygo.RunOnMain(func() { app.Store().Actions.CompleteOnboarding() })
	eventually(t, "onboarding destroyed", onboarding.IsDestroyed)
	var window *mygo.Window
	eventually(t, "settings visible and focused after setup", func() bool {
		window = app.Windows()["settings"]
		return window != nil && window.IsVisible() && window.IsFocused()
	})
	defer window.Close()
	if !nonblank(capture(t, window, "settings-after-onboarding", uiapp.SettingsWidth, uiapp.SettingsHeight)) {
		t.Fatal("settings content blank after setup")
	}
	prefs, _, err := (preferences.Store{Path: filepath.Join(profileDir, "preferences.json")}).Load()
	if err != nil || !prefs.OnboardingCompleted {
		t.Fatalf("onboarding completion not persisted: completed=%v, err=%v", prefs.OnboardingCompleted, err)
	}
}

// DSK-A09: switching interface language updates existing native windows while
// leaving the shared dictation/recognition preferences intact.
func TestInterfaceLanguageUpdatesNativeWindows(t *testing.T) {
	app.OpenSettings()
	app.OpenOnboarding()
	var settings, onboarding *mygo.Window
	eventually(t, "language test windows opened", func() bool {
		windows := app.Windows()
		settings, onboarding = windows["settings"], windows["onboarding"]
		return settings != nil && onboarding != nil && settings.IsVisible() && onboarding.IsVisible()
	})
	defer mygo.RunOnMain(func() {
		app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = arion.LanguageEnglish })
		settings.Close()
		onboarding.Close()
	})
	var before arion.Preferences
	app.Update(func(s *arion.State) {
		before = s.Preferences
		app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = arion.LanguageChinese })
	})
	eventually(t, "localized native settings title", func() bool { return settings.Title() == "Arion 设置" })
	if !nonblank(capture(t, settings, "settings-zh-CN", 640, 440)) {
		t.Fatal("Chinese settings content blank")
	}
	if !nonblank(capture(t, onboarding, "onboarding-zh-CN", uiapp.OnboardingWidth, uiapp.OnboardingHeight)) {
		t.Fatal("Chinese onboarding content blank")
	}
	app.Update(func(s *arion.State) {
		want := before
		want.UILanguage = arion.LanguageChinese
		if !reflect.DeepEqual(s.Preferences, want) {
			t.Errorf("language switch changed other preferences: %v", s.Preferences)
		}
		app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = arion.LanguageEnglish })
	})
	eventually(t, "English native settings title restored", func() bool { return settings.Title() == "Arion Settings" })
}

// DSK-A14: opt-in because a UI driver must answer the real OS save sheets.
// The expected ZIP path is confined to an isolated test output directory.
func TestDiagnosticExportNativeDialog(t *testing.T) {
	if os.Getenv("ARION_E2E_DIAGNOSTICS") != "1" {
		t.Skip("requires a UI driver for native save sheets")
	}
	destination := os.Getenv("ARION_E2E_DIAGNOSTICS_EXPORT")
	if destination == "" {
		t.Fatal("ARION_E2E_DIAGNOSTICS_EXPORT is required")
	}
	t.Logf("native export build version: %q", mygo.App.Version())
	app.OpenSettings()
	for _, step := range []struct {
		name string
		want arion.DiagnosticsPhase
	}{{"cancel", arion.DiagnosticsIdle}, {"save", arion.DiagnosticsSaved}} {
		app.Update(func(s *arion.State) {
			s.Focus = arion.SettingsFocus{Section: settings.About, Seq: s.Focus.Seq + 1}
			app.Store().Actions.ExportDiagnostics()
		})
		t.Logf("native diagnostic chooser: %s", step.name)
		var phase arion.DiagnosticsPhase
		if !within(60*time.Second, func() bool {
			app.Update(func(s *arion.State) { phase = s.Diagnostics.Phase })
			return phase != arion.DiagnosticsExporting
		}) {
			t.Fatal("native save sheet was not answered")
		}
		if phase != step.want {
			t.Fatalf("%s: phase %s", step.name, phase)
		}
		if step.name == "cancel" {
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("cancel created export: %v", err)
			}
		}
	}
	archive, err := zip.OpenReader(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	manifest, logs := false, false
	for _, file := range archive.File {
		manifest = manifest || file.Name == "manifest.json"
		logs = logs || strings.HasPrefix(file.Name, "events.")
	}
	if !manifest || !logs {
		t.Fatal("native export missing manifest/logs")
	}
}
