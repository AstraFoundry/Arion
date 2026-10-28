package kit

import (
	"testing"
	"time"

	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func TestDisclosureReservesSpaceThroughOpeningAndClosing(t *testing.T) {
	open := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			Disclosure(c.Key("details"), open, func() { ui.Box(c).Height(120) })
			ui.Text(c, "After")
		})
	}, 300, 300)
	y := func() float32 {
		t.Helper()
		r, ok := tt.Find("After")
		if !ok {
			t.Fatal("following content missing")
		}
		return r.Y
	}
	compact := y()
	open = true
	tt.Frame()
	time.Sleep(30 * time.Millisecond)
	tt.Frame()
	if at := y(); at <= compact || at >= compact+120 {
		t.Fatalf("opening did not reserve intermediate space: %v", at)
	}
	time.Sleep(theme.Disclosure.Duration)
	tt.Frame()
	if y() != compact+120 {
		t.Fatal("opening did not reach content height")
	}
	open = false
	tt.Frame()
	time.Sleep(30 * time.Millisecond)
	tt.Frame()
	if at := y(); at <= compact || at >= compact+120 {
		t.Fatalf("closing removed space immediately: %v", at)
	}
	open = true
	tt.Frame()
	time.Sleep(theme.Disclosure.Duration)
	tt.Frame()
	if y() != compact+120 {
		t.Fatal("reversing the close did not restore content height")
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	open = false
	tt.Frame()
	if y() != compact {
		t.Fatal("reduced motion retained animated space")
	}
}
