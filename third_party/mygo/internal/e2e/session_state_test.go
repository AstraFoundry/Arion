//go:build !js

package e2e

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"html"
	"net/http"
	"path"
	"runtime"
	"testing"
	"unicode/utf16"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// sessionPage serves app://localhost/session/<name>: a page titled name,
// taller than its window, with a field at the top.
func sessionPage(w http.ResponseWriter, r *http.Request) {
	name := html.EscapeString(path.Base(r.URL.Path))
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!doctype html><title>%s</title><body style="margin:0"><input id=field name=field><div style="height:5000px">%[1]s</div>`, name)
}

// visit loads a session page, types into its field and scrolls it to y.
func visit(t *testing.T, w *mygo.Window, name, typed string, y int) {
	t.Helper()
	if err := w.Page().LoadURL("app://localhost/session/" + name); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, fmt.Sprintf(`document.title === %q && document.readyState === "complete"`, name))
	mustEval(t, w, fmt.Sprintf(`field.value = %q; scrollTo(0, %d)`, typed, y))
	waitFor(t, w, fmt.Sprintf(`scrollY === %d`, y))
}

// holds reports whether a session state holds s, in UTF-8 or UTF-16 as
// WebKit may encode it.
func holds(state []byte, s string) bool {
	u := make([]byte, 0, 2*len(s))
	for _, c := range utf16.Encode([]rune(s)) {
		u = append(u, byte(c), byte(c>>8))
	}
	return bytes.Contains(state, []byte(s)) || bytes.Contains(state, u)
}

// savedSession saves the session state of a window that visited "one",
// scrolled to 700, then "two", typed "typed second" into its field and
// scrolled to 1200, once WebKit saved the latter two.
func savedSession(t *testing.T) []byte {
	t.Helper()
	w := newWindow(t, mygo.WindowOptions{Title: "Session", Width: 300, Height: 200})
	visit(t, w, "one", "typed first", 700)
	visit(t, w, "two", "typed second", 1200)
	state, err := w.Page().SessionState()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		// WebKit saves the scroll position of the page it shows a moment
		// after it stopped scrolling, and its form fields as the page goes
		// out of view, so once it came into view: a window shows on
		// screen a while after it is shown, longer on a busy runner.
		// WebKit tells them to the UI process later.
		waitFor(t, w, `document.visibilityState === "visible"`)
		w.Hide()
		wait.Until(t, "WebKit saving the page hidden", func() bool {
			state, err = w.Page().SessionState()
			return err == nil && holds(state, "typed second") && holdsScroll(state, 0, 1200)
		})
	}
	w.Destroy()
	return state
}

// holdsScroll reports whether a session state holds the scroll position
// (x, y), as WebKit encodes it on macOS and in WebKitGTK: two 32-bit
// integers in a row, little-endian on the machines MyGo runs on.
func holdsScroll(state []byte, x, y int32) bool {
	return bytes.Contains(state, binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, uint32(x)), uint32(y)))
}

// A page's session state brings back its back/forward list in a new
// window, with what WebKit saved: the page's scroll position and form
// field as its window was hidden, and the entry before's scroll position
// as the page left it. On Windows it brings back the page alone.
//
// On macOS the page may load before the window server puts the new
// window on screen, and WebKit then puts it back at the top, as in
// TestSessionStateHiddenWindow: the backend puts it back where it was.
func TestSessionState(t *testing.T) {
	state := savedSession(t)
	r := newWindow(t, mygo.WindowOptions{Title: "Session restored", Width: 300, Height: 200, URL: "app://localhost/session/wrong",
		Page: mygo.PageOptions{SessionState: state}})
	waitFor(t, r, `document.title === "two" && document.readyState === "complete"`)
	if runtime.GOOS == "windows" {
		if r.Page().CanGoBack() {
			t.Error("CanGoBack = true where WebView2 brings back the URL alone")
		}
		return
	}
	t.Logf("restored: %v", mustEval(t, r, `JSON.stringify([scrollY, field.value, history.length, innerHeight, document.body.scrollHeight, document.visibilityState])`))
	if !r.Page().CanGoBack() {
		t.Fatal("CanGoBack = false in the page restored: it loaded the URL alone")
	}
	waitFor(t, r, `scrollY === 1200`)
	waitFor(t, r, `field.value === "typed second"`)
	r.Page().GoBack()
	waitFor(t, r, `document.title === "one" && document.readyState === "complete"`)
	waitFor(t, r, `scrollY === 700`)
	if !r.Page().CanGoForward() {
		t.Error("CanGoForward = false back in the page restored")
	}
}

// A session state brought back in a window made hidden keeps its scroll
// position once the window shows. WebKit on macOS puts a page that
// scrolled before its window first showed on screen back at the top as
// the window does; the backend puts it back where it was.
func TestSessionStateHiddenWindow(t *testing.T) {
	state := savedSession(t)
	r := newWindow(t, mygo.WindowOptions{Hidden: true, Title: "Session restored hidden", Width: 300, Height: 200, URL: "app://localhost/session/wrong",
		Page: mygo.PageOptions{SessionState: state}})
	waitFor(t, r, `document.title === "two" && document.readyState === "complete"`)
	if runtime.GOOS == "windows" {
		return
	}
	r.Show()
	waitFor(t, r, `document.visibilityState === "visible"`)
	waitFor(t, r, `scrollY === 1200`)
	waitFor(t, r, `field.value === "typed second"`)
}
