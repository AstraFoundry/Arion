package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The grid's cells are 6.5 DIPs wide, in whole pixels, twice as high,
// centered on the pane and overhanging it; a pane too small has none.
func TestGrid(t *testing.T) {
	g, ok := gridOf(2200, 1440, 2)
	if !ok || g.cellW != 13 || g.cellH != 26 || g.cols != 170 || g.rows != 56 {
		t.Fatalf("grid of 2200×1440 at 2: %+v, want 170×56 cells of 13×26", g)
	}
	if g.left != -5 || g.top != -8 {
		t.Errorf("the grid starts at %d, %d, want it centered at -5, -8", g.left, g.top)
	}
	if g, _ := gridOf(1100, 720, 1); g.cellW != 7 {
		t.Errorf("cells %d pixels wide at 1, want 7 (6.5 rounded)", g.cellW)
	}
	if _, ok := gridOf(100, 100, 1); ok {
		t.Error("a grid in a pane of 100×100 DIPs")
	}
}

// A seed picks the same wind every time, heading down the pane, and a
// look that the last visits did not show.
func TestWind(t *testing.T) {
	a, b := newWind(42, "mixed"), newWind(42, "mixed")
	if a != b {
		t.Fatal("one seed, two winds")
	}
	for seed := range uint32(200) {
		w := newWind(seed, "mixed")
		if w.heading <= 0 || w.heading >= 3.1416 || w.speed < 42 || w.speed > 64 || w.gustWidth < 120 || w.gustWidth > 190 {
			t.Fatalf("wind of %d out of range: %+v", seed, w)
		}
		if l := pickLook(seed, []string{"mixed", "figured"}); l != "lettered" {
			t.Fatalf("look %q after mixed and figured, want lettered", l)
		}
		if l := pickLook(seed, []string{"mixed", "figured", "lettered"}); l == "mixed" {
			t.Fatal("the latest look again")
		}
	}
}

// Clicks on the pane blow ripples from the pointer, every third a heart;
// a click on the composer blows none.
func TestClicks(t *testing.T) {
	// Without an atlas the texture draws nothing: the clicks are what is
	// tested.
	s := newApp(7)
	tt := ui.NewTester(s.view, windowW, windowH)

	for i := range 3 {
		tt.ClickAt(100+float32(i)*50, 120)
	}
	taps := s.tex.hand.taps
	if len(taps) != 3 || taps[0].x != 100 || taps[0].y != 120 {
		t.Fatalf("taps %+v, want 3 from (100, 120) on", taps)
	}
	if taps[0].force <= 0 || taps[1].force <= 0 || taps[2].force >= 0 {
		t.Errorf("forces %v, %v, %v, want the third a heart's (negative)", taps[0].force, taps[1].force, taps[2].force)
	}
	if err := tt.Click("Task"); err != nil {
		t.Fatal(err)
	}
	if len(s.tex.hand.taps) != 3 {
		t.Error("a click on the composer blew a ripple")
	}
}

// The wind blows for its budget, settles and fades out; whatever the user
// does next brings it back in.
func TestSettles(t *testing.T) {
	tex := newTexture(7, nil)
	now := time.Unix(1000, 0)
	for range int((budget + settleTime + 1) / 0.04) {
		now = now.Add(40 * time.Millisecond)
		tex.step(now)
	}
	if tex.energy != 0 {
		t.Fatalf("energy %v after the budget, want 0", tex.energy)
	}
	frozen := tex.windTime
	tex.step(now.Add(time.Second))
	if tex.windTime != frozen {
		t.Error("the wind moved once settled")
	}
	tex.calm = true
	tex.move(10, 10, now)
	if tex.calm || tex.revealFrom != tex.active || tex.until != tex.active+budget {
		t.Errorf("woken: calm %v, comes in from %v, blows until %v, at %v", tex.calm, tex.revealFrom, tex.until, tex.active)
	}
}

// stillTexture returns a texture whose wind stands still at a time, in a
// pane of 300×200 DIPs at 1, shown long ago: drawn on the CPU too when
// soft, and with a click a second old when click.
func stillTexture(soft, click bool) *texture {
	tex := newTexture(0x3fa1c2d9, nil)
	tex.soft = soft
	tex.windTime = 21.5
	tex.calm, tex.active, tex.revealFrom, tex.energy = true, 10, 0, 1
	tex.atlases[atlasKey{7, 1}] = newAtlas(7, 1)
	tex.shown = time.Now().Add(-time.Hour)
	if click {
		tex.hand.tap(150, 100, 0)
		tex.start = time.Now().Add(-time.Second)
	}
	return tex
}

// render draws tex in a pane of 300×200 DIPs, on the CPU, the shaders
// interpreted, and returns how many pixels show the plain ink and how many
// the ripples' tint.
func render(tex *texture) (plain, tinted int) {
	img := ui.Render(func(c *ui.Context) {
		c.Root().Background(ui.Hex("#ffffff"))
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { tex.draw(p, r, false) })
	}, 300, 200, 1)
	for i := 0; i < len(img.Pix); i += 4 {
		r, g, b := int(img.Pix[i]), int(img.Pix[i+1]), int(img.Pix[i+2])
		switch {
		case r == 255 && g == 255 && b == 255:
		case b > r+20:
			tinted++
		default:
			plain++
		}
	}
	return plain, tinted
}

// The shaders draw glyphs in the plain ink where the wind blows, and a
// click's ripple tints those its front reaches.
func TestRender(t *testing.T) {
	if plain, tinted := render(stillTexture(true, false)); plain == 0 || tinted != 0 {
		t.Errorf("%d plain and %d tinted pixels, want some plain and none tinted", plain, tinted)
	}
	if _, tinted := render(stillTexture(true, true)); tinted == 0 {
		t.Error("no pixel tinted a second after a click")
	}
}

// Where the frame does not draw on a GPU, as here, the texture shows
// nothing, as the app's on software renderers.
func TestNotAccelerated(t *testing.T) {
	tex := stillTexture(false, true)
	if plain, tinted := render(tex); plain+tinted != 0 || tex.windValues != nil {
		t.Errorf("%d pixels drawn without a GPU, want none", plain+tinted)
	}
}

// Settled, the texture has faded out; asking for less motion then shows
// the wind as it stands, at once.
func TestStillWhileCalm(t *testing.T) {
	tex := stillTexture(true, false)
	tex.energy = 0
	if plain, _ := render(tex); plain != 0 {
		t.Fatalf("%d pixels drawn once settled, want none", plain)
	}
	tex.still = true
	if plain, _ := render(tex); plain == 0 {
		t.Error("nothing drawn once still")
	}
	tex.still = false
	if plain, _ := render(tex); plain != 0 {
		t.Errorf("%d pixels drawn once moving again, settled, want none", plain)
	}
}
