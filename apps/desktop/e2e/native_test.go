//go:build arion_e2e && (darwin || windows)

// Package e2e exercises the production window on the actual OS event loop.
package e2e

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/desktop/bootstrap"
	"github.com/egoist/mygo"
)

func TestMain(m *testing.M) {
	if os.Getenv("ARION_NATIVE_E2E") != "1" {
		fmt.Fprintln(os.Stderr, "native E2E requires ARION_NATIVE_E2E=1 and an interactive desktop session")
		os.Exit(2)
	}
	mygo.App.OnWindowAllClosed(func() {})
	var code atomic.Int32
	code.Store(1)
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

func eventually(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("native window condition did not become true")
}

func TestProductionWindowLifecycle(t *testing.T) {
	options := bootstrap.WindowOptions()
	options.StateKey = "" // Keep test geometry out of user window preferences.
	window := mygo.NewWindow(options)
	t.Cleanup(func() {
		if !window.IsDestroyed() {
			window.Destroy()
		}
	})
	eventually(t, window.IsVisible)
	if window.Page() != nil {
		t.Fatal("production UI must use native content")
	}

	capture, err := window.CapturePage()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(capture))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() < 360 || img.Bounds().Dy() < 240 {
		t.Fatal("invalid native content dimensions")
	}
	// CapturePage renders Content; this is not an OS screenshot or GPU readback.
	first := img.At(0, 0)
	fr, fg, fb, fa := first.RGBA()
	different := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y && !different; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r != fr || g != fg || b != fb || a != fa {
				different = true
				break
			}
		}
	}
	if !different {
		t.Fatal("production content capture is blank")
	}
	if directory := os.Getenv("ARION_E2E_OUTPUT"); directory != "" {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "native-content.png"), capture, 0600); err != nil {
			t.Fatal(err)
		}
	}
	window.SetContentSize(560, 360)
	eventually(t, func() bool { width, height := window.ContentSize(); return width == 560 && height == 360 })
	var closed atomic.Int32
	window.OnClosed(func() { closed.Add(1) })
	window.Close()
	eventually(t, window.IsDestroyed)
	eventually(t, func() bool { return closed.Load() == 1 })
}
