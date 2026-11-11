//go:build linux && (amd64 || arm64)

package e2e

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
)

func keyboardState() (state string, restore func()) {
	var k linux.TestKeyboard
	var ok bool
	mygo.RunOnMain(func() { k, ok = linux.TestKeyboardState() })
	if !ok {
		return "", func() {}
	}
	return k.String(), func() { mygo.RunOnMain(func() { linux.TestRestoreKeyboard(k) }) }
}

// TestLeftBehindKeyboard turns Caps Lock on and holds a key, as a test that
// forgets to undo them does: the check says so, and undoes them.
func TestLeftBehindKeyboard(t *testing.T) {
	// The server has the keyboard's keys once a window showed.
	w := newWindow(t, mygo.WindowOptions{Title: "Left behind", Width: 200, Height: 100})
	w.Focus()
	eventually(t, "the window to have the keyboard", w.IsFocused)
	before, _ := keyboardState()
	if before == "" {
		t.Skip("key automation not available")
	}
	check := watchLeftBehind()
	// Up first: the X server then has the layout of the keyboard XTEST
	// fakes, whatever the desktop's own does with Caps Lock.
	if !fake(func() bool { return linux.TestHoldKey("Up", true) }) {
		t.Skip("key automation not available")
	}
	if !fake(func() bool { return linux.TestHoldKey("Caps_Lock", true) && linux.TestHoldKey("Caps_Lock", false) }) {
		fake(func() bool { return linux.TestHoldKey("Up", false) })
		t.Skip("no Caps Lock key")
	}
	var got []string
	check(func(format string, args ...any) {
		for _, a := range args {
			if s, ok := a.(string); ok {
				got = append(got, s)
			}
		}
	})
	if len(got) == 0 || !strings.Contains(got[0], "Caps_Lock") || !strings.Contains(got[0], "holding Up") {
		t.Errorf("reported %q, want Caps Lock on and Up held", got)
	}
	if after, _ := keyboardState(); after != before {
		t.Errorf("the keyboard is left with %s, want %s", after, before)
	}
}
