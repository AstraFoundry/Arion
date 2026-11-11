//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestShortcutRepeatedGTK holds a key down through XTEST, whose presses
// with no release between GTK reports with no flag of auto-repeat: the
// surface tells the repeats from the first press (keysDown).
func TestShortcutRepeatedGTK(t *testing.T) {
	var mu sync.Mutex
	var seen []bool
	frames := make(chan struct{}, 1)
	view := func(c *ui.Context) {
		if c.Shortcut(0, ui.KeyUp) {
			mu.Lock()
			seen = append(seen, c.ShortcutRepeated())
			mu.Unlock()
		}
		ui.Text(c, "hold Up")
		select {
		case frames <- struct{}{}:
		default:
		}
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Key repeat", Width: 300, Height: 200, Content: ui.View(view)})
	<-frames
	if !click(w, 150, 100) {
		t.Skip("pointer automation not available")
	}
	// Held, the key repeats by the X server's auto-repeat until released.
	if !fake(func() bool { return linux.TestHoldKey("Up", true) }) {
		t.Skip("key automation not available")
	}
	got := func() []bool {
		mu.Lock()
		defer mu.Unlock()
		return append([]bool(nil), seen...)
	}
	eventually(t, "the key to repeat", func() bool { return len(got()) >= 3 })
	fake(func() bool { return linux.TestHoldKey("Up", false) })
	s := got()
	if s[0] || !s[1] || !s[2] {
		t.Fatalf("repeats %v, want false then true", s)
	}
	// The repeats still queued come before the press after the release.
	pressKeys("Up")
	presses := func(s []bool) int { return len(s) - countTrue(s) }
	eventually(t, "a press after the release", func() bool { return presses(got()) == 2 })
	if s := got(); s[len(s)-1] {
		t.Fatalf("repeats %v: a press after the release came as a repeat", s)
	}
}

func countTrue(s []bool) (n int) {
	for _, v := range s {
		if v {
			n++
		}
	}
	return n
}
