package terminal

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// A program that starts a synchronized update and writes nothing more, its
// cursor hidden, has the update shown after a second all the same: the view
// asks for the frame that lets it go, as no blink, output or input will.
func TestSynchronizedUpdateShowsAfterASecondWithoutOutput(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), NoBlink: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	term.Feed([]byte("\x1b[?25lbefore\x1b[?2026hafter"))
	tt.Frame()
	held := func() bool {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.held
	}
	if !held() {
		t.Fatal("the synchronized update did not hold the screen")
	}
	// The second has gone by, as far as paint's clock tells; the Tester's
	// runs only the frames asked for.
	term.mu.Lock()
	term.rmu.Lock()
	term.heldSince = time.Now().Add(-2 * time.Second)
	term.rmu.Unlock()
	term.mu.Unlock()
	tt.Advance(2 * time.Second)
	if held() {
		t.Fatal("a synchronized update with no more output stayed held: no frame came to let it go")
	}
}
