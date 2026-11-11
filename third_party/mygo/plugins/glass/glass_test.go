package glass

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// pixelAt returns the color of img at (x, y).
func pixelAt(img *image.RGBA, x, y int) [3]int {
	p := img.RGBAAt(x, y)
	return [3]int{int(p.R), int(p.G), int(p.B)}
}

func TestGlassPaintsOverWhatIsBehind(t *testing.T) {
	g := Glass{}
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(20, 120, 220)).Children(func() {
			ui.Row(c).Size(200, 44).Radius(22).Material(g).Children(func() {
				ui.Text(c, "On glass")
			})
		})
	}
	tt := ui.NewTester(view, 300, 100)
	tt.SetScale(2)
	if !tt.HasText("On glass") {
		t.Error("the glass's children did not paint")
	}
	// The glass lightens the blue behind it, which shows through.
	light := pixelAt(tt.Image(), 300, 70)
	if light[0] < 100 || light[2] < light[0] {
		t.Errorf("light glass over blue is %v", light)
	}
	tt.SetDark(true)
	if dark := pixelAt(tt.Image(), 300, 70); dark[2] >= light[2] || dark[0] >= light[0] {
		t.Errorf("dark glass over blue is %v, not darker than light glass's %v", dark, light)
	}
	tt.SetDark(false)
	g.Style = Clear
	tt.Frame()
	if clear := pixelAt(tt.Image(), 300, 70); clear[0] >= light[0] {
		t.Errorf("clear glass over blue is %v, not clearer than regular glass's %v", clear, light)
	}
}

func TestInteractiveGlassGrowsWhilePressed(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			ui.Box(c).Absolute().Left(50).Top(50).Size(200, 100).Radius(20).Material(Glass{Interactive: true})
		})
	}
	tt := ui.NewTester(view, 300, 200)
	tt.SetScale(2)
	// It grows at once.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	// The pixel left of the glass's left edge, in the middle.
	outside := func() int { return pixelAt(tt.Image(), 99, 200)[0] }
	before := outside()
	tt.Press(150, 100)
	tt.Frame()
	if grown := outside(); grown <= before+50 {
		t.Errorf("pressed, the pixel left of the glass is %d, as before (%d)", grown, before)
	}
	tt.Release(150, 100)
	tt.Frame()
	if after := outside(); after != before {
		t.Errorf("after the release, the pixel left of the glass is %d, not %d", after, before)
	}
}

func TestInteractiveGlassLightsOnHover(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			ui.Box(c).Absolute().Left(40).Top(30).Size(240, 100).Radius(20).Material(Glass{Interactive: true})
		})
	}
	tt := ui.NewTester(view, 320, 160)
	tt.SetScale(2)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	rim := func() int { return pixelAt(tt.Image(), 320, 62)[0] }
	before := rim()
	// The pointer enters the glass: its top rim lights.
	tt.Move(160, 80)
	tt.Frame()
	if hovered := rim(); hovered <= before {
		t.Errorf("the rim is %v while hovered, not lighter than %v", hovered, before)
	}
	// Leaving it restores the rim.
	tt.Move(10, 10)
	tt.Frame()
	if after := rim(); after != before {
		t.Errorf("the rim is %v after the pointer left, not %v", after, before)
	}
}

// Panes close together merge into one surface; far apart, they do not.
func TestGroupMergesNearPanes(t *testing.T) {
	gap := float32(8)
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			g := NewGroup(c)
			g.Radius = 22
			g.Gap(gap).Children(func() {
				g.Pane(ui.Box(c).Size(80, 44))
				g.Pane(ui.Box(c).Size(80, 44))
			})
		})
	}
	tt := ui.NewTester(view, 320, 100)
	tt.SetScale(2)
	// The first pane ends at 80, the second starts at 88: their gap's
	// middle is 84, which the glass bridges.
	if p := pixelAt(tt.Image(), 168, 44); p[0] < 30 {
		t.Errorf("the gap between close panes is %v, not glass", p)
	}
	// Far apart, nothing bridges the gap.
	gap = 40
	tt.Frame()
	if p := pixelAt(tt.Image(), 200, 44); p[0] > 10 {
		t.Errorf("the gap between far panes is %v, not the background", p)
	}
}

func TestPaintInDrawing(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(0, 0, 0)).Draw(func(p *ui.Painter, r ui.Rect) {
			Paint(p, ui.Rect{X: 20, Y: 20, W: 100, H: 60}, 20, Glass{})
		})
	}
	tt := ui.NewTester(view, 200, 120)
	// Light glass over black is mid gray.
	if g := pixelAt(tt.Image(), 70, 50); g[0] < 100 || g[0] > 180 {
		t.Errorf("light glass over black is %v", g)
	}
}

func TestString(t *testing.T) {
	if s := (Glass{Style: Clear, Tint: ui.RGB(1, 2, 3), Interactive: true}).String(); s != "glass clear tint(#010203ff) interactive" {
		t.Errorf("String: %q", s)
	}
}

// With Reduce Transparency, a glass pane, a blur and a scroll edge are
// opaque: the theme's surfaces, not what is behind them.
func TestReduceTransparency(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(20, 120, 220)).Children(func() {
			ui.Box(c).Absolute().Left(50).Top(20).Size(200, 60).Radius(12).Material(Glass{})
			ui.Box(c).Absolute().Left(50).Top(100).Size(200, 40).Radius(8).Material(ScrollEdge{Hard: true})
			ui.Box(c).Absolute().Left(50).Top(160).Size(200, 40).Radius(8).Material(Blur{Radius: 8})
		})
	}
	tt := ui.NewTester(view, 320, 220)
	tt.SetScale(2)
	glassAt := func() [3]int { return pixelAt(tt.Image(), 300, 100) }
	before := glassAt()
	if before[0] > 200 && before[1] > 200 {
		t.Fatalf("glass before Reduce Transparency is already the surface: %v", before)
	}
	tt.SetPreferences(ui.Preferences{ReduceTransparency: true})
	tt.Frame()
	theme := ui.LightTheme()
	surface := [3]int{int(theme.Surface.R), int(theme.Surface.G), int(theme.Surface.B)}
	background := [3]int{int(theme.Background.R), int(theme.Background.G), int(theme.Background.B)}
	near := func(got, want [3]int) bool {
		for i := range got {
			if got[i] < want[i]-2 || got[i] > want[i]+2 {
				return false
			}
		}
		return true
	}
	if got := glassAt(); !near(got, surface) {
		t.Errorf("glass is %v, not the opaque surface %v", got, surface)
	}
	if got := pixelAt(tt.Image(), 300, 240); !near(got, background) {
		t.Errorf("scroll edge is %v, not the background %v", got, background)
	}
	if got := pixelAt(tt.Image(), 300, 360); !near(got, background) {
		t.Errorf("blur is %v, not the background %v", got, background)
	}
	// The desktop's setting back off restores the backdrop.
	tt.SetPreferences(ui.Preferences{})
	tt.Frame()
	if after := glassAt(); near(after, surface) {
		t.Errorf("glass stayed opaque after Reduce Transparency: %v", after)
	}
}

// High Contrast lights the glass's rim more.
func TestHighContrastGlass(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(0, 0, 0)).Children(func() {
			ui.Box(c).Absolute().Left(40).Top(30).Size(240, 100).Radius(20).Material(Glass{})
		})
	}
	tt := ui.NewTester(view, 320, 160)
	tt.SetScale(2)
	tt.Frame()
	normal := pixelAt(tt.Image(), 320, 62)
	tt.SetPreferences(ui.Preferences{HighContrast: true})
	tt.Frame()
	contrast := pixelAt(tt.Image(), 320, 62)
	if contrast[0] <= normal[0] {
		t.Errorf("the rim is %v with high contrast, not lighter than %v", contrast, normal)
	}
}

// Panes up to 100 DIPs, as controls, keep the measured material,
// whatever the clarity; from 200, as sidebars, theirs is the clarity's;
// between, it eases from one to the other.
func TestPaneMaterial(t *testing.T) {
	near := func(a, b float32) bool { return a-b < 1e-5 && b-a < 1e-5 }
	for _, k := range []float32{0, 0.25, 0.5, 1} {
		for _, m := range []float32{20, 44, 72, 100} {
			for _, s := range []float32{1, 2} {
				bezel := min(36, m/2)
				blur := min(max(m*0.035, 1), 10) * s
				light := paneMaterial(m, s, k, false, false)
				if light.blur != blur || light.bezel != bezel*s || light.refraction != 1.6*bezel*s || light.rim != 0.35 ||
					light.low != 0.541 || light.high != 1 {
					t.Errorf("light %v DIPs at %vx, clarity %v: %+v", m, s, k, light)
				}
				dark := paneMaterial(m, s, k, false, true)
				if dark.blur != blur || dark.rim != 0.35 || dark.low != 0.15 || dark.high != 0.51 {
					t.Errorf("dark %v DIPs at %vx, clarity %v: %+v", m, s, k, dark)
				}
			}
		}
	}
	for _, m := range []float32{200, 300, 900} {
		light, dark := paneMaterial(m, 2, 0.5, false, false), paneMaterial(m, 2, 0.5, false, true)
		if !near(light.blur, 10) || !near(light.bezel, 96) || !near(light.rim, 0.5) || !near(light.low, 0.35) || light.high != 1 {
			t.Errorf("light %v DIPs: %+v", m, light)
		}
		if !near(dark.blur, 10) || !near(dark.high, 0.6) || dark.low != 0.15 || !near(dark.rim, 0.5) {
			t.Errorf("dark %v DIPs: %+v", m, dark)
		}
	}
	// Between, every parameter lies between the two and moves one way.
	prev := paneMaterial(100, 1, 0.5, false, false)
	for m := float32(110); m <= 200; m += 10 {
		mat := paneMaterial(m, 1, 0.5, false, false)
		if mat.low > prev.low || mat.rim < prev.rim || mat.bezel < prev.bezel || mat.low < 0.35 || mat.rim > 0.5 {
			t.Errorf("%v DIPs: %+v after %+v", m, mat, prev)
		}
		prev = mat
	}
	if mid := paneMaterial(150, 1, 0.5, false, false); !near(mid.low, (0.541+0.35)/2) || !near(mid.rim, 0.425) {
		t.Errorf("150 DIPs, half way: %+v", mid)
	}
	// Clear glass's tone is the same at any size.
	if small, large := paneMaterial(60, 1, 0.5, true, false), paneMaterial(400, 1, 0.5, true, false); small.low != large.low || small.high != large.high || small.blur != large.blur {
		t.Errorf("clear: %+v and %+v", small, large)
	}
}

// A large pane's clarity runs from the measured material, at 0, through
// a clearer one at 0.5, to one nearly clear at 1, every parameter moving
// one way; out of range it is the nearest end.
func TestPaneClarity(t *testing.T) {
	near := func(a, b float32) bool { return a-b < 1e-5 && b-a < 1e-5 }
	type tone struct{ blur, light, darkLow, darkHigh, rim, bezel float32 }
	at := func(k float32) tone {
		l, d := paneMaterial(400, 1, k, false, false), paneMaterial(400, 1, k, false, true)
		if l.blur != d.blur || l.rim != d.rim || l.bezel != d.bezel {
			t.Errorf("clarity %v: light %+v and dark %+v differ", k, l, d)
		}
		return tone{l.blur, l.low, d.low, d.high, l.rim, l.bezel}
	}
	same := func(a, b tone) bool {
		return near(a.blur, b.blur) && near(a.light, b.light) && near(a.darkLow, b.darkLow) &&
			near(a.darkHigh, b.darkHigh) && near(a.rim, b.rim) && near(a.bezel, b.bezel)
	}
	for _, c := range []struct {
		k    float32
		want tone
	}{
		{0, tone{10, 0.541, 0.15, 0.51, 0.35, 36}},
		{0.25, tone{7.5, 0.4455, 0.15, 0.555, 0.425, 42}},
		{0.5, tone{5, 0.35, 0.15, 0.6, 0.5, 48}},
		{0.75, tone{3.5, 0.275, 0.125, 0.725, 0.5, 48}},
		{1, tone{2, 0.2, 0.1, 0.85, 0.5, 48}},
		{-1, tone{10, 0.541, 0.15, 0.51, 0.35, 36}},
		{2, tone{2, 0.2, 0.1, 0.85, 0.5, 48}},
	} {
		if got := at(c.k); !same(got, c.want) {
			t.Errorf("clarity %v: %+v, want %+v", c.k, got, c.want)
		}
	}
	// Clearer, it blurs less, lightens and darkens less, its rim brighter
	// and its bezel wider, or as before.
	prev := at(0)
	for k := float32(0.05); k <= 1.001; k += 0.05 {
		cur := at(k)
		if cur.blur > prev.blur || cur.light > prev.light || cur.darkLow > prev.darkLow || cur.darkHigh < prev.darkHigh ||
			cur.rim < prev.rim || cur.bezel < prev.bezel {
			t.Errorf("clarity %v: %+v after %+v", k, cur, prev)
		}
		prev = cur
	}
	// A pane between a control's size and a sidebar's eases toward it.
	for _, k := range []float32{0, 0.5, 1} {
		small, mid, large := paneMaterial(100, 1, k, false, false), paneMaterial(150, 1, k, false, false), paneMaterial(200, 1, k, false, false)
		if !near(mid.low, (small.low+large.low)/2) || !near(mid.rim, (small.rim+large.rim)/2) {
			t.Errorf("clarity %v, 150 DIPs: %+v, not half way between %+v and %+v", k, mid, small, large)
		}
	}
}

// Glass of a clarity paints a large pane clearer: black under it
// lighter in light mode the less clear, the same for small panes.
func TestClarityPaints(t *testing.T) {
	gray := func(k float32, w, h float32) int {
		view := func(c *ui.Context) {
			ui.Box(c).Fill().Background(ui.RGB(0, 0, 0)).Draw(func(p *ui.Painter, r ui.Rect) {
				Paint(p, ui.Rect{X: 10, Y: 10, W: w, H: h}, 20, Glass{Clarity: k})
			})
		}
		tt := ui.NewTester(view, 420, 320)
		return pixelAt(tt.Image(), int(10+w/2), int(10+h/2))[0]
	}
	frost, mid, clear := gray(0, 400, 300), gray(0.5, 400, 300), gray(1, 400, 300)
	if !(frost > mid && mid > clear) {
		t.Errorf("black under a large pane at clarity 0, 0.5, 1: %d, %d, %d", frost, mid, clear)
	}
	if a, b := gray(0, 100, 60), gray(1, 100, 60); a != b {
		t.Errorf("black under a small pane at clarity 0 and 1: %d and %d", a, b)
	}
}

func TestStringClarity(t *testing.T) {
	if s := (Glass{Clarity: 0.5}).String(); s != "glass clarity(0.5)" {
		t.Errorf("String: %q", s)
	}
}

// At clarity 0 every pane, of any size and style, is exactly the
// material it was before clarity: the measured one.
func TestClarityZero(t *testing.T) {
	for _, m := range []float32{1, 20, 72, 100, 150, 199, 200, 260, 900, 4000} {
		for _, s := range []float32{1, 1.5, 2} {
			bezel := min(36, m/2)
			want := material{bezel: bezel * s, refraction: 1.6 * bezel * s, rim: 0.35, rimWidth: 1, light: math.Pi / 2}
			for _, c := range []struct {
				clear, dark                        bool
				blur, low, high, curve, saturation float32
			}{
				{true, false, s, 0.125, 1.082, 1, 1},
				{true, true, s, 0.125, 1.082, 1, 1},
				{false, true, min(max(m*0.035, 1), 10) * s, 0.15, 0.51, 2, 2.2},
				{false, false, min(max(m*0.035, 1), 10) * s, 0.541, 1, 1.2, 1},
			} {
				w := want
				w.blur, w.low, w.high, w.curve, w.saturation = c.blur, c.low, c.high, c.curve, c.saturation
				if got := paneMaterial(m, s, 0, c.clear, c.dark); got != w {
					t.Errorf("%v DIPs at %vx, clear %v, dark %v: %+v, want %+v", m, s, c.clear, c.dark, got, w)
				}
			}
		}
	}
}

// On clear glass, clarity leaves the tone as it is and only lights the
// rim and curves the bezel.
func TestClarityClear(t *testing.T) {
	frost, clear := paneMaterial(400, 1, 0, true, false), paneMaterial(400, 1, 1, true, false)
	if frost.blur != clear.blur || frost.low != clear.low || frost.high != clear.high {
		t.Errorf("tone: %+v and %+v", frost, clear)
	}
	if clear.rim != 0.5 || clear.bezel != 48 {
		t.Errorf("rim and bezel: %+v", clear)
	}
}
