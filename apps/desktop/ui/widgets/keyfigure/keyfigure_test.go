package keyfigure

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

const width, height = 320, 560

// figure is the widget under test with the inputs a test changes.
type figure struct {
	key              runtime.HoldKey
	changed, pressed time.Time
}

func render(dark bool, f *figure) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		Render(c, theme.Resolve(dark), copy.For(copy.English), f.key, f.changed, f.pressed)
	}, width, height)
	tt.SetDark(dark)
	return tt
}

func rest(tt *ui.Tester, d time.Duration) {
	for deadline := time.Now().Add(d); time.Now().Before(deadline); {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
}

// differs counts the pixels two frames do not share, past the rounding
// of compositing a transparent layer.
func differs(a, b *image.RGBA) int {
	n := 0
	for i := 0; i+3 < len(a.Pix); i += 4 {
		for j := i; j < i+3; j++ {
			if d := int(a.Pix[j]) - int(b.Pix[j]); d > 2 || d < -2 {
				n++
				break
			}
		}
	}
	return n
}

func clone(img *image.RGBA) *image.RGBA {
	out := *img
	out.Pix = append([]byte(nil), img.Pix...)
	return &out
}

func snapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	directory := os.Getenv("ARION_UI_SNAPSHOTS")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name+".png"), buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The laptop arrives closed and opens zoomed in on the key; choosing
// another key moves there, pulling back on the way; a press shows and goes.
func TestFigureFollowsTheKey(t *testing.T) {
	for _, platform := range platforms {
		keys := runtime.HoldKeysFor(platform)
		for _, dark := range []bool{false, true} {
			appearance := map[bool]string{false: "light", true: "dark"}[dark]
			name := func(stage string) string { return "keyfigure-" + platform + "-" + appearance + "-" + stage }
			f := &figure{key: keys[0]}
			tt := render(dark, f)
			tt.Frame()
			if _, ok := tt.Find(copy.For(copy.English).Text(copy.HoldKeyFigureLabel)); !ok {
				t.Fatalf("%s %s: the figure is not labeled", platform, appearance)
			}
			closed := clone(tt.Image())
			snapshot(t, tt, name("0-closed"))
			for i, at := range []time.Duration{700, 1200, 1700} {
				rest(tt, at*time.Millisecond-time.Duration([]int{0, 700, 1200}[i])*time.Millisecond)
				snapshot(t, tt, name("1-arriving-"+string(rune('a'+i))))
			}
			rest(tt, 900*time.Millisecond)
			before := clone(tt.Image())
			if differs(closed, before) == 0 {
				t.Errorf("%s %s: the laptop did not open", platform, appearance)
			}
			snapshot(t, tt, name("2-"+string(f.key)))
			for _, next := range keys[1:] {
				f.key, f.changed = next, time.Now()
				rest(tt, 200*time.Millisecond)
				snapshot(t, tt, name("3-moving-to-"+string(next)))
				rest(tt, 700*time.Millisecond)
				after := clone(tt.Image())
				if differs(before, after) == 0 {
					t.Errorf("%s %s: choosing %s left the figure as it was", platform, appearance, next)
				}
				snapshot(t, tt, name("4-"+string(next)))
				before = after
			}
			f.pressed = time.Now()
			rest(tt, 200*time.Millisecond)
			if differs(before, tt.Image()) == 0 {
				t.Errorf("%s %s: a press of %s did not show", platform, appearance, f.key)
			}
			snapshot(t, tt, name("5-pressed"))
			rest(tt, 700*time.Millisecond)
			if n := differs(before, tt.Image()); n != 0 {
				t.Errorf("%s %s: %d pixels still differ after the press", platform, appearance, n)
			}
		}
	}
}

func TestShows(t *testing.T) {
	for _, key := range append(runtime.HoldKeysFor("darwin"), runtime.HoldKeysFor("windows")...) {
		if !Shows(key) {
			t.Errorf("no figure for %s", key)
		}
	}
	if Shows("") {
		t.Error("a figure for no key")
	}
}
