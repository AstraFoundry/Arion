package loading

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// tester shows the indicator s alone, named by its name, with o.
func tester(s Spinner, o *Options) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Center().Background(ui.RGB(255, 255, 255)).Children(func() {
			s.New(c, *o).Label(s.Name)
		})
	}, 64, 64)
}

// at returns the tester's frame at its time.
func at(tt *ui.Tester) *image.RGBA {
	tt.Frame()
	return clone(tt.Image())
}

func clone(img *image.RGBA) *image.RGBA {
	c := image.NewRGBA(img.Rect)
	copy(c.Pix, img.Pix)
	return c
}

// ink counts the pixels not white inside r, and outside it.
func ink(img *image.RGBA, r image.Rectangle) (in, out int) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := img.PixOffset(x, y)
			if img.Pix[p] > 250 && img.Pix[p+1] > 250 && img.Pix[p+2] > 250 {
				continue
			}
			if image.Pt(x, y).In(r) {
				in++
			} else {
				out++
			}
		}
	}
	return in, out
}

// differ reports whether two frames differ by more than rounding.
func differ(a, b *image.RGBA) bool {
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		if d > 2 || d < -2 {
			return true
		}
	}
	return false
}

func TestAll(t *testing.T) {
	all := All()
	if len(all) != 29 {
		t.Fatalf("%d indicators, want loading.dev's 29", len(all))
	}
	names := map[string]bool{}
	for _, s := range all {
		if names[s.Name] || s.New == nil {
			t.Errorf("indicator %q repeated or without its function", s.Name)
		}
		names[s.Name] = true
	}
}

// Every indicator draws in its box, at every point of its cycle, and
// nothing outside it.
func TestDrawsInItsBox(t *testing.T) {
	o := Options{Size: 40, Duration: time.Second}
	for _, s := range All() {
		tt := tester(s, &o)
		r, ok := tt.Find(s.Name)
		if !ok {
			t.Fatalf("%s: not found", s.Name)
		}
		var aspect float32 = 1
		switch s.Name {
		case "Bouncing dots":
			aspect = 1.15
		case "Linear dots":
			aspect = 0.9375
		}
		if r.H != 40 || r.W != 40*aspect {
			t.Errorf("%s: %v×%v, want %v×40", s.Name, r.W, r.H, 40*aspect)
		}
		box := image.Rect(int(r.X)-1, int(r.Y)-1, int(r.X+r.W)+2, int(r.Y+r.H)+2)
		for step := range 8 {
			// Flip is edge on a quarter and three quarters into its cycle.
			if s.Name == "Flip" && step%4 == 2 {
				tt.Advance(time.Second / 8)
				continue
			}
			in, out := ink(at(tt), box)
			if in < 20 || out > 0 {
				t.Errorf("%s at %d/8: %d pixels drawn in its box, %d outside", s.Name, step, in, out)
			}
			tt.Advance(time.Second / 8)
		}
	}
}

// Every indicator moves, asking for frames as it does, and comes back
// where it was after two cycles: Bouncing dots's go up in one and down in
// the next.
func TestMovesInCycles(t *testing.T) {
	o := Options{Duration: time.Second}
	for _, s := range All() {
		tt := tester(s, &o)
		start := at(tt)
		if !tt.FramePending() {
			t.Errorf("%s: asks for no frame", s.Name)
		}
		tt.Advance(300 * time.Millisecond)
		a := at(tt)
		tt.Advance(2 * time.Second)
		b := at(tt)
		if !differ(start, a) {
			t.Errorf("%s: the same 0.3 cycles in", s.Name)
		}
		if differ(a, b) {
			t.Errorf("%s: not the same two cycles later", s.Name)
		}
	}
}

// Where the desktop asks for less motion, an indicator stands still and
// asks for no frames.
func TestReduceMotion(t *testing.T) {
	var o Options
	for _, s := range All() {
		tt := tester(s, &o)
		tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
		still := at(tt)
		if tt.FramePending() {
			t.Errorf("%s: asks for frames at rest", s.Name)
		}
		tt.Advance(time.Second)
		if differ(still, at(tt)) {
			t.Errorf("%s: moves at rest", s.Name)
		}
		if in, _ := ink(still, still.Bounds()); in == 0 {
			t.Errorf("%s: draws nothing at rest", s.Name)
		}
	}
}

// Paused, an indicator stops where it is, and goes on from there.
func TestPaused(t *testing.T) {
	o := Options{Duration: time.Second}
	for _, s := range All() {
		tt := tester(s, &o)
		tt.Advance(300 * time.Millisecond)
		o.Paused = true
		held := at(tt)
		if tt.FramePending() {
			t.Errorf("%s: asks for frames paused", s.Name)
		}
		tt.Advance(400 * time.Millisecond)
		if differ(held, at(tt)) {
			t.Errorf("%s: moves paused", s.Name)
		}
		o.Paused = false
		if differ(held, at(tt)) {
			t.Errorf("%s: not where it stopped as it goes on", s.Name)
		}
		tt.Advance(200 * time.Millisecond)
		if !differ(held, at(tt)) {
			t.Errorf("%s: does not go on", s.Name)
		}
		o.Paused = false
	}
}

// An indicator draws in the color given, and in the theme's text color
// without one.
func TestColor(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	for _, color := range []ui.Color{{}, red} {
		var text ui.Color
		img := ui.Render(func(c *ui.Context) {
			text = c.Theme().Text
			ui.Column(c).Fill().Center().Children(func() { Morph(c, Options{Size: 40, Color: color}) })
		}, 64, 64, 1)
		want := color
		if want == (ui.Color{}) {
			want = text
		}
		p := img.PixOffset(32, 32)
		if got := (ui.Color{R: img.Pix[p], G: img.Pix[p+1], B: img.Pix[p+2], A: 255}); got != want {
			t.Errorf("color %v: drawn in %v, want %v", color, got, want)
		}
	}
}

func TestOneOptions(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("two Options did not panic")
		}
	}()
	ui.Render(func(c *ui.Context) { Arc(c, Options{}, Options{}) }, 32, 32, 1)
}

func TestBezier(t *testing.T) {
	for _, c := range []struct{ t, want float64 }{{0, 0}, {0.5, 0.5}, {1, 1}, {0.25, 0.1291}} {
		if got := easeInOut(c.t); got < c.want-1e-3 || got > c.want+1e-3 {
			t.Errorf("ease-in-out(%v) = %v, want %v", c.t, got, c.want)
		}
	}
	// Cascade's overshoots below 0 and past 1.
	if overshoot(0.1) >= 0 || overshoot(0.9) <= 1 {
		t.Errorf("overshoot: %v at 0.1, %v at 0.9", overshoot(0.1), overshoot(0.9))
	}
}
