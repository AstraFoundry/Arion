//go:build darwin

package e2e

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
)

const tabFocusPage = `<!doctype html><html><body style="margin:0;height:100vh">
<input id=i value=text><button id=b>Button</button><a id=a href="#">Link</a>
<script>window.ready = true</script></body></html>`

// tabFocusesLinks reports whether Tab reaches links and form controls in a
// window's page, whatever the page focused last.
func tabFocusesLinks(w *mygo.Window) bool {
	var links bool
	mygo.RunOnMain(func() { links = darwin.TestTabFocusesLinks(w.NativeHandle()) })
	return links
}

// keyboardNavigationOn reports whether the system's keyboard navigation
// setting, which makes Tab reach every control on its own, is on.
func keyboardNavigationOn(t *testing.T) bool {
	t.Helper()
	out, err := exec.Command("defaults", "read", "-g", "AppleKeyboardUIMode").Output()
	if err != nil {
		return false // unset: macOS's default, off
	}
	return strings.TrimSpace(string(out)) != "0"
}

// TestTabFocusesLinks tabs from an input of a window's page to a button and
// then a link with the system's keyboard navigation off, as a desktop app's
// page does, and checks that a page can ask for Safari's behavior instead.
func TestTabFocusesLinks(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Tab focus", Width: 400, Height: 300})
	if !tabFocusesLinks(w) {
		t.Fatal("the page's Tab does not reach links and form controls by default")
	}
	safari := newWindow(t, mygo.WindowOptions{Title: "Tab focus text fields", Width: 300, Height: 200, Page: mygo.PageOptions{SafariTabFocus: true}})
	if tabFocusesLinks(safari) {
		t.Error("SafariTabFocus still takes Tab to links and form controls")
	}
	if keyboardNavigationOn(t) {
		t.Skip("the system's keyboard navigation makes Tab reach every control already")
	}
	w.Page().LoadHTML(tabFocusPage, "")
	waitForPage(t, w.Page(), "window.ready")
	w.Focus()
	if !click(w, 30, 10) {
		t.Skip("click automation not available on this platform")
	}
	active := func() string {
		id, _ := mygo.EvalAs[string](w.Page(), "document.hasFocus() && document.activeElement ? document.activeElement.id : ''")
		return id
	}
	eventually(t, "the input focused", func() bool { return active() == "i" })
	tab := func() {
		t.Helper()
		var ok bool
		mygo.RunOnMain(func() { ok = darwin.TestPageKeyShift(w.NativeHandle(), 48, "\t", false) })
		if !ok {
			t.Skip("key automation not available on this platform")
		}
	}
	tab()
	eventually(t, "Tab to the button", func() bool { return active() == "b" })
	tab()
	eventually(t, "Tab to the link", func() bool { return active() == "a" })
}
