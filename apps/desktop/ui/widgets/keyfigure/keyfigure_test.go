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

// render draws the figure for key, pressed at press, after enough frames
// for its transitions to finish.
func render(t *testing.T, dark bool, key *runtime.HoldKey, press *time.Time) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		Render(c, theme.Resolve(dark), copy.For(copy.English), *key, *press)
	}, width, height)
	tt.SetDark(dark)
	rest(tt, 500*time.Millisecond)
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

// Every key of every platform has its keyboard and lit key in both
// appearances; a press shows, and choosing another key moves the light.
func TestFigureFollowsTheKey(t *testing.T) {
	for _, platform := range platforms {
		keys := runtime.HoldKeysFor(platform)
		for _, dark := range []bool{false, true} {
			appearance := map[bool]string{false: "light", true: "dark"}[dark]
			key, press := keys[0], time.Time{}
			tt := render(t, dark, &key, &press)
			if _, ok := tt.Find(copy.For(copy.English).Text(copy.HoldKeyFigureLabel)); !ok {
				t.Fatalf("%s %s: the figure is not labeled", platform, appearance)
			}
			before := clone(tt.Image())
			snapshot(t, tt, "keyfigure-"+string(key)+"-"+appearance)
			for _, next := range keys[1:] {
				key = next
				rest(tt, 600*time.Millisecond)
				after := clone(tt.Image())
				if differs(before, after) == 0 {
					t.Errorf("%s %s: choosing %s left the figure as it was", platform, appearance, next)
				}
				snapshot(t, tt, "keyfigure-"+string(key)+"-"+appearance)
				before = after
			}
			press = time.Now()
			rest(tt, 300*time.Millisecond)
			if differs(before, tt.Image()) == 0 {
				t.Errorf("%s %s: a press of %s did not show", platform, appearance, key)
			}
			snapshot(t, tt, "keyfigure-"+string(key)+"-"+appearance+"-pressed")
			// The press is let go: the figure is at rest again.
			rest(tt, 900*time.Millisecond)
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
