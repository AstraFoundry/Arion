//go:build !js

package e2e

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

// watchLeftBehind notes what a package's tests share and one can change for
// those after it: the keyboard, whose lock keys outlive a window and even
// the process on an X server, and the windows open. The function it returns
// reports what changed since and puts it back, so that the test at fault
// fails, by name, rather than one that runs much later and did nothing wrong.
func watchLeftBehind() (check func(report func(format string, args ...any))) {
	keyboard, restore := keyboardState()
	windows := mygo.Windows()
	return func(report func(format string, args ...any)) {
		if now, _ := keyboardState(); now != keyboard {
			report("the test left the keyboard with %s, and found it with %s: tests after it would get other keys; a test that presses a lock key or holds one lets it go in a cleanup", now, keyboard)
			restore()
		}
		for _, w := range mygo.Windows() {
			if !slices.Contains(windows, w) {
				report("the test left a window open, titled %q: it can take the keyboard from the tests after it", w.Title())
				w.Destroy()
			}
		}
	}
}

// TestLeftBehindWindow leaves a window open: the check says so and closes it.
func TestLeftBehindWindow(t *testing.T) {
	check := watchLeftBehind()
	w := mygo.NewWindow(mygo.WindowOptions{Title: "Left open", Hidden: true})
	var got []string
	check(func(format string, args ...any) { got = append(got, format) })
	if len(got) != 1 || !strings.Contains(got[0], "window") {
		t.Errorf("reported %q, want the window", got)
	}
	if !w.IsDestroyed() {
		t.Error("the window is still open")
		w.Destroy()
	}
}
