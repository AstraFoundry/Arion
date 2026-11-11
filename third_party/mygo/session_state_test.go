//go:build !js

package mygo

import (
	"errors"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/ui"
)

// restored returns what the fake engine put back of a page's session
// state, and the URL the page is at.
func restored(fw *fake.Window) (state, url string) {
	onMain(func() { state, url = string(fw.Restored), fw.URL() })
	return state, url
}

// A page's session state brings back its back/forward list in a new
// window, in place of the window's URL.
func TestSessionState(t *testing.T) {
	w, _ := testWindow(t, WindowOptions{Hidden: true, URL: "app://localhost/docs?page=2"})
	state, err := w.Page().SessionState()
	if err != nil {
		t.Fatal(err)
	}
	goos, url, native, ok := decodeSessionState(state)
	if !ok || goos != runtime.GOOS || url != "app://localhost/docs?page=2" || string(native) != "fake session of app://localhost/docs?page=2" {
		t.Fatalf("SessionState = %q, %q, %q, %v", goos, url, native, ok)
	}

	_, fw := testWindow(t, WindowOptions{Hidden: true, URL: "app://localhost/", Page: PageOptions{SessionState: state}})
	if got, url := restored(fw); got != "fake session of app://localhost/docs?page=2" || url != "app://localhost/docs?page=2" {
		t.Errorf("restored %q, at %q; want the saved state, at its page", got, url)
	}
}

// Where the platform keeps no session state, as on Windows, or can't
// read the one saved, the page loads the URL the state saved; a state
// that isn't one loads the window's URL.
func TestSessionStateFallback(t *testing.T) {
	const saved = "app://localhost/saved"
	check := func(name string, state []byte, want string) {
		t.Helper()
		_, fw := testWindow(t, WindowOptions{Hidden: true, URL: "app://localhost/", Page: PageOptions{SessionState: state}})
		if got, url := restored(fw); got != "" || url != want {
			t.Errorf("%s: restored %q, at %q; want nothing restored, at %q", name, got, url, want)
		}
	}
	check("another platform's", encodeSessionState("plan9", saved, []byte("fake session of app://localhost/other")), saved)
	check("one the engine can't read", encodeSessionState(runtime.GOOS, saved, []byte("WebKit 1")), saved)
	check("one of a blank page", encodeSessionState(runtime.GOOS, "", nil), "app://localhost/")
	check("not a state", []byte("app://localhost/saved"), "app://localhost/")
	later := encodeSessionState(runtime.GOOS, saved, nil)
	later[len(sessionStateMagic)] = sessionStateVersion + 1
	check("a later version's", later, "app://localhost/")

	fb.SetNoSessionState(true)
	defer fb.SetNoSessionState(false)
	w, _ := testWindow(t, WindowOptions{Hidden: true, URL: saved})
	state, err := w.Page().SessionState()
	if err != nil {
		t.Fatal(err)
	}
	if goos, url, native, ok := decodeSessionState(state); !ok || goos != runtime.GOOS || url != saved || len(native) != 0 {
		t.Fatalf("SessionState without the engine's = %q, %q, %q, %v; want the URL alone", goos, url, native, ok)
	}
	check("a platform without session state", state, saved)
	check("the engine's, on a platform without session state", encodeSessionState(runtime.GOOS, saved, []byte("fake session of app://localhost/other")), saved)
}

// Cut states are not states, and don't panic.
func TestDecodeSessionStateCut(t *testing.T) {
	state := encodeSessionState("linux", "app://localhost/", []byte{1, 2, 3})
	for n := range len(state) {
		_, _, native, ok := decodeSessionState(state[:n])
		if ok && n < len(state)-3 {
			t.Errorf("a state cut to %d of %d bytes decodes, with %q", n, len(state), native)
		}
	}
}

func TestSessionStateWebView(t *testing.T) {
	w, _ := testWindow(t, WindowOptions{Hidden: true, URL: "app://localhost/tab"})
	state, err := w.Page().SessionState()
	if err != nil {
		t.Fatal(err)
	}
	show := true
	_, v, _, fv := webViewWindowOpts(t, &show, WebViewOptions{URL: "app://localhost/", Page: PageOptions{SessionState: state}})
	if got, url := restored(fv.Window); got != "fake session of app://localhost/tab" || url != "app://localhost/tab" {
		t.Errorf("web view restored %q, at %q; want the saved state, at its page", got, url)
	}
	again, err := v.Page().SessionState()
	if err != nil {
		t.Fatal(err)
	}
	if _, url, native, _ := decodeSessionState(again); url != "app://localhost/tab" || string(native) != "fake session of app://localhost/tab" {
		t.Errorf("the web view's SessionState = %q, %q; want its own page's", url, native)
	}
}

func TestSessionStateErrors(t *testing.T) {
	c := NewWindow(WindowOptions{Hidden: true, Content: ui.View(func(*ui.Context) {})})
	t.Cleanup(c.Destroy)
	if state, err := c.pg.SessionState(); !errors.Is(err, errNoPage) || state != nil {
		t.Errorf("SessionState of a window of native UI = %q, %v; want %v", state, err, errNoPage)
	}
	w, _ := testWindow(t, WindowOptions{Hidden: true})
	w.Destroy()
	if state, err := w.Page().SessionState(); !errors.Is(err, errDestroyed) || state != nil {
		t.Errorf("SessionState after Destroy = %q, %v; want %v", state, err, errDestroyed)
	}
}
