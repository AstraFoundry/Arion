package main

import (
	_ "embed"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// The grid: cells cellSize DIPs wide and twice as high, at least minCols
// by minRows of them, and no more than maxPixels pixels in all.
const (
	cellSize         = 6.5
	minCols, minRows = 24, 8
	maxPixels        = 21e6
)

// The clock: a frame every frameTime, or rippleTime while a click's blow
// runs out; the wind blows for budget seconds after the last thing the
// user did and settles over settleTime, gathers speed again over
// resumeTime, and the texture comes in over revealTime seconds of wind
// and fades in over fadeTime.
const (
	frameTime, rippleTime          = 40 * time.Millisecond, 10 * time.Millisecond
	budget, settleTime, resumeTime = 180, 8, 3
	revealTime                     = 4
	fadeTime                       = 3 * time.Second
	// strength is how strong the canvas shows: its light is drawn
	// headroom times weaker, leaving room for the ripples' color above it.
	strength, headroom = 0.26, 3
)

var (
	// inkColor is the plain glyphs' color, glowLight and glowDark the
	// ripples' on a light and a dark theme.
	inkColor  = ui.Hex("#898781")
	glowLight = [3]float32{0, 0.25, 1}
	glowDark  = [3]float32{0.1, 0.68, 1}
	// fadeEase is CSS's ease-out.
	fadeEase = ui.CubicBezier(0, 0, 0.58, 1)
)

// grid is the cells of a pane: their size and count, and where the first
// lies, in pixels; the grid is centered on the pane and overhangs it.
type grid struct {
	cellW, cellH, cols, rows int
	left, top                int
	scale                    float32
}

// gridOf returns the grid of a pane of w×h pixels at scale pixels a DIP,
// or false for a pane too small for one.
func gridOf(w, h int, scale float32) (grid, bool) {
	if w <= 0 || h <= 0 || scale <= 0 {
		return grid{}, false
	}
	cw := max(5, int(math.Round(cellSize*float64(scale))))
	g := grid{cellW: cw, cellH: 2 * cw, scale: scale}
	g.cols, g.rows = (w+cw-1)/cw, (h+2*cw-1)/(2*cw)
	if g.cols < minCols || g.rows < minRows {
		return grid{}, false
	}
	if k := float64(g.cols*g.cellW*g.rows*g.cellH) / maxPixels; k > 1 {
		g.cols, g.rows = int(float64(g.cols)/math.Sqrt(k)), int(float64(g.rows)/math.Sqrt(k))
	}
	g.left = int(math.Round(float64(w-g.cols*g.cellW) / 2))
	g.top = int(math.Round(float64(h-g.rows*g.cellH) / 2))
	return g, true
}

// cellX and cellY return the middle of a cell's column and row in the
// pane, in DIPs.
func (g grid) cellX(cx int) float32 {
	return (float32(g.left) + (float32(cx)+0.5)*float32(g.cellW)) / g.scale
}
func (g grid) cellY(cy int) float32 {
	return (float32(g.top) + (float32(cy)+0.5)*float32(g.cellH)) / g.scale
}

// atlasKey is what an atlas is drawn for.
type atlasKey struct {
	cellW int
	scale float32
}

// texture is the glyphs a pane shows, blown by a wind.
type texture struct {
	w      wind
	recent []string // the looks of the last visits, latest first
	// later runs a function outside the window's frames, where the
	// atlases are drawn (ui.Render draws a frame of its own).
	later   func(fn func())
	atlases map[atlasKey]*atlas
	asked   map[atlasKey]bool
	hand    hand
	// The clock: when the texture started, which the hand's times count
	// from, its last frame, and when it first showed.
	start, last, shown time.Time
	// windTime is the wind's time; active the seconds the texture moved;
	// revealFrom when it last began to come in, resumedAt when the wind
	// last resumed, and until when it blows; calm that it has settled.
	windTime, active             float64
	revealFrom, resumedAt, until float64
	energy                       float64
	calm                         bool
	// still draws a frame that does not move, while the desktop asks for
	// less motion.
	still bool
	// soft draws the texture where the frame does not draw on a GPU too,
	// for tests: the app shows none on software renderers.
	soft bool
	// The frame shown: the wind's field, drawn by wind.kage, one pixel a
	// sample, and the uniforms of the shaders (windValues for wind.kage,
	// glyphValues for glyphs.kage), for the grid drawn.
	field                   *ui.Canvas
	windValues, glyphValues map[string]any
	drawn                   grid
	// When the frame shown was worked out, on which theme and whether
	// still; and whether the field is drawn for it.
	rendered         time.Time
	dark, stillDrawn bool
	fieldDone        bool
}

// newTexture returns a texture blown by the wind of seed.
func newTexture(seed uint32, later func(fn func())) *texture {
	t := &texture{later: later, atlases: map[atlasKey]*atlas{}, asked: map[atlasKey]bool{}}
	t.reseed(seed)
	return t
}

// reseed starts a new visit, with the wind of seed, in a look the last
// visits did not show.
func (t *texture) reseed(seed uint32) {
	look := pickLook(seed, t.recent)
	t.recent = append([]string{look}, t.recent...)
	if len(t.recent) > 3 {
		t.recent = t.recent[:3]
	}
	t.w = newWind(seed, look)
	t.windTime = float64(seed) / 4294967296 * 64
	t.active, t.revealFrom, t.resumedAt, t.until = 0, 0, -resumeTime, budget
	t.energy, t.calm = 1, false
	t.last, t.shown = time.Time{}, time.Time{}
	t.hand = hand{}
}

// pickLook returns the look of seed among those not in recent, or not the
// latest of them.
func pickLook(seed uint32, recent []string) string {
	var fresh, notLast, all []string
	for _, l := range looks {
		all = append(all, l.name)
		if !contains(recent, l.name) {
			fresh = append(fresh, l.name)
		}
		if len(recent) == 0 || recent[0] != l.name {
			notLast = append(notLast, l.name)
		}
	}
	pick := all
	if len(fresh) > 0 {
		pick = fresh
	} else if len(notLast) > 0 {
		pick = notLast
	}
	return pick[int(random(seed, "look")()*float64(len(pick)))]
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// clock returns the seconds since the texture started, as the hand counts
// them.
func (t *texture) clock(now time.Time) float32 {
	if t.start.IsZero() {
		t.start = now
	}
	return float32(now.Sub(t.start).Seconds())
}

// wake has the wind blow again for its whole budget, after the user did
// something: settled, it comes in anew.
func (t *texture) wake() {
	if t.calm {
		t.calm = false
		t.revealFrom, t.resumedAt, t.until = t.active, t.active, t.active+budget
		t.last = time.Time{}
	} else if t.active < t.until {
		t.until = t.active + budget
	}
}

// move and tap note the pointer's moves and clicks at (x, y) in the pane,
// in DIPs.
func (t *texture) move(x, y float32, now time.Time) {
	if t.still {
		return
	}
	t.hand.move(x, y, t.clock(now))
	t.wake()
}

func (t *texture) tap(x, y float32, now time.Time) {
	if t.still {
		return
	}
	t.hand.tap(x, y, t.clock(now))
	t.wake()
}

// step moves the clock to now: the wind blows as long as its budget
// lasts, then settles, the texture fading with it.
func (t *texture) step(now time.Time) {
	dt := 0.0
	if !t.last.IsZero() {
		dt = min(max(now.Sub(t.last).Seconds(), 0), 0.25)
	}
	t.last = now
	t.active += dt
	t.energy = 1 - smooth((t.active-t.until)/settleTime)
	speed := smooth((t.active - t.resumedAt) / resumeTime)
	t.windTime += dt * t.energy * speed
}

func smooth(v float64) float64 { return float64(smoothstep(0, 1, float32(v))) }

// The shaders: the wind's field, and the glyphs it lights.
var (
	//go:embed wind.kage
	windSource string
	//go:embed glyphs.kage
	glyphSource string

	windShader  = shader.MustCompile(windSource)
	glyphShader = shader.MustCompile(glyphSource)
)

// draw paints the texture in r, the pane, with p, moving it on when its
// next frame is due: the shaders draw it, the wind's field into a canvas
// of a pixel a sample, then the glyphs, which read it. Where the frame
// does not draw on a GPU, it draws nothing, as the app shows no texture on
// software renderers: there the shaders would take seconds a frame.
func (t *texture) draw(p *ui.Painter, r ui.Rect, dark bool) {
	if !p.Accelerated() && !t.soft {
		return
	}
	scale := p.Scale()
	g, ok := gridOf(int(math.Round(float64(r.W*scale))), int(math.Round(float64(r.H*scale))), scale)
	if !ok {
		return
	}
	key := atlasKey{g.cellW, scale}
	a := t.atlases[key]
	if a == nil {
		if !t.asked[key] && t.later != nil {
			t.asked[key] = true
			t.later(func() { t.atlases[key] = newAtlas(key.cellW, key.scale) })
		}
		// Until the window can run it, ask again.
		if t.later == nil {
			p.After(frameTime)
		}
		return
	}
	now := p.Now()
	moving := !t.still && !t.calm
	// A frame may paint this more than once, and other repaints do too, as
	// the caret blinks: the texture moves on at its own pace.
	due := t.windValues == nil || g != t.drawn || dark != t.dark || t.still != t.stillDrawn || t.shown.IsZero() ||
		!now.Equal(t.rendered) && (moving && now.Sub(t.last) >= t.interval(now)/2 ||
			now.Sub(t.shown) < fadeTime+frameTime)
	if due {
		if t.shown.IsZero() {
			t.shown = now
		}
		if moving {
			t.step(now)
		}
		t.uniforms(g, a, r, dark, now)
		t.rendered, t.dark, t.stillDrawn, t.drawn, t.fieldDone = now, dark, t.still, g, false
		if moving && t.energy == 0 && !t.hand.busy(t.clock(now)) {
			t.calm = true
		}
	}

	if w, h := g.cols, 2*g.rows; t.field == nil || !sameSize(t.field, w, h) {
		t.field, t.fieldDone = ui.NewCanvas(w, h), false
	}
	if !t.fieldDone {
		t.fieldDone = true
		w, h := t.field.Size()
		t.field.Draw(p, func(cp *ui.Painter) {
			windShader.Draw(cp, ui.Rect{W: float32(w), H: float32(h)}, shader.Options{Uniforms: t.windValues})
		})
	}
	k := 1 / scale
	dst := ui.Rect{X: r.X + float32(g.left)*k, Y: r.Y + float32(g.top)*k, W: float32(g.cols*g.cellW) * k, H: float32(g.rows*g.cellH) * k}
	p.Clip(r, 0, func() {
		glyphShader.Draw(p, dst, shader.Options{Uniforms: t.glyphValues, Images: [4]*ui.Bitmap{t.field.Bitmap(), a.bitmap}})
	})

	switch {
	case !t.still && !t.calm:
		p.After(t.interval(now))
	case now.Sub(t.shown) < fadeTime:
		p.After(frameTime)
	}
}

func sameSize(c *ui.Canvas, w, h int) bool {
	cw, ch := c.Size()
	return cw == w && ch == h
}

// interval returns the time between frames at now.
func (t *texture) interval(now time.Time) time.Duration {
	if t.hand.rippling(t.clock(now)) {
		return rippleTime
	}
	return frameTime
}

// uniforms works out the shaders' uniforms for a frame of grid g in pane
// r, at now: the wind's time and its reveal, the hand's blows, and the
// light, as the app's vertex shaders and its code around them do.
func (t *texture) uniforms(g grid, a *atlas, r ui.Rect, dark bool, now time.Time) {
	reveal := smooth((t.active-t.revealFrom)/revealTime) * t.energy
	if t.still {
		reveal = 1
		t.hand = hand{}
	}
	h := &t.hand
	h.frame(t.clock(now))
	k := 1 / g.scale
	wt := float32(t.windTime)
	cw, ch := float32(g.cols*g.cellW)*k, float32(g.rows*g.cellH)*k

	taps := make([]float32, 0, 4*maxTaps)
	for _, tp := range h.taps {
		taps = append(taps, tp.x, tp.y, tp.at, tp.force)
	}
	wakes := make([]float32, 0, 4*maxWakes)
	ways := make([]float32, 0, 4*maxWakes)
	for _, w := range h.warm {
		wakes = append(wakes, w.x, w.y, w.heat, w.reach)
		ways = append(ways, w.dirX, w.dirY, w.half, 0)
	}
	t.windValues = map[string]any{
		"Seed":       t.w.seed,
		"Time":       wt,
		"Reveal":     float32(reveal),
		"Heading":    []float32{cos(t.w.heading), float32(math.Sin(float64(t.w.heading)))},
		"Carried":    t.w.carried(wt),
		"Span":       t.w.span(cw, ch),
		"GustLength": t.w.gustLength,
		"GustWidth":  t.w.gustWidth,
		"Bend":       t.w.bend,
		"BendScale":  t.w.bendScale,
		"PaneSize":   []float32{r.W, r.H},
		"Origin":     []float32{float32(g.left) * k, float32(g.top) * k},
		"Step":       []float32{float32(g.cellW) * k, float32(g.cellH) * k / 2},
		"Now":        h.now,
		"TapCount":   len(h.taps),
		"Taps":       taps,
		"WakeCount":  len(h.warm),
		"Wakes":      wakes,
		"Ways":       ways,
	}

	seq := a.sequence(t.w.look)
	boxes := make([]float32, 0, 4*len(seq))
	for _, c := range seq {
		b := a.boxes[c]
		boxes = append(boxes, b[:]...)
	}
	glow := glowLight
	if dark {
		glow = glowDark
	}
	// The canvas shows headroom times as strong as its plain light, and
	// fades in with the pane holding it, both over fadeTime.
	fade := float32(1)
	if since := now.Sub(t.shown); since < fadeTime {
		e := fadeEase(float32(since) / float32(fadeTime))
		fade = e * e
	}
	t.glyphValues = map[string]any{
		"Seed":     t.w.seed,
		"Time":     wt,
		"Grid":     []float32{float32(g.cellW), float32(g.cellH), float32(g.cols), float32(g.rows)},
		"Cells":    []float32{g.cellX(0), g.cellY(0), float32(g.cellW) * k, float32(g.cellH) * k},
		"Seq":      seq,
		"SeqLen":   len(seq),
		"Boxes":    boxes,
		"Now":      h.now,
		"TapCount": len(h.taps),
		"Taps":     taps,
		"Ink":      []float32{float32(inkColor.R) / 255, float32(inkColor.G) / 255, float32(inkColor.B) / 255},
		"InkAlpha": float32(inkColor.A) / 255,
		"Glow":     glow[:],
		"Opacity":  min(1, strength*headroom) * fade,
	}
}
