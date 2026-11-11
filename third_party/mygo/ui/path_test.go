package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
)

// TestPathsLargerThanAMaskDraw fills and strokes paths 2480 DIPs wide at
// twice the pixels, wider than the mask atlas: they draw from end to end,
// and a gradient spans the whole path.
func TestPathsLargerThanAMaskDraw(t *testing.T) {
	blue, red := RGB(20, 60, 230), RGB(230, 40, 20)
	img := coreRender(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var line, band, under Path
			line.MoveTo(10, 20).LineTo(2490, 20)
			p.StrokePath(&line, 4, blue)
			band.MoveTo(10, 40).LineTo(2490, 40).LineTo(2490, 60).LineTo(10, 60).Close()
			p.FillPath(&band, blue)
			under.MoveTo(10, 70).LineTo(2490, 70).LineTo(2490, 90).LineTo(10, 90).Close()
			p.FillPathGradient(&under, LinearGradient{Angle: 90, From: red, To: blue})
		})
	}, 2500, 100, 2)
	isBlue := func(c color.RGBA) bool { return c.B > 180 && c.R < 120 && c.G < 120 }
	isRed := func(c color.RGBA) bool { return c.R > 180 && c.B < 120 && c.G < 120 }
	for _, x := range []int{30, 2500, 4970} {
		if c := img.RGBAAt(x, 40); !isBlue(c) {
			t.Errorf("the stroke shows %v at (%d, 40)", c, x)
		}
		if c := img.RGBAAt(x, 100); !isBlue(c) {
			t.Errorf("the fill shows %v at (%d, 100)", c, x)
		}
	}
	if c := img.RGBAAt(30, 160); !isRed(c) {
		t.Errorf("the gradient starts with %v", c)
	}
	if c := img.RGBAAt(4970, 160); !isBlue(c) {
		t.Errorf("the gradient ends with %v", c)
	}
}

// TestTiledPathsMatchOneMask draws paths from one mask each, and again in
// tiles of 17 pixels: a fill whose subpaths overlap and wind around a
// hole (non-zero winding), a curved stroke crossing itself, a closed
// stroke, a dashed stroke with square caps and miter joins, whose caps and
// points reach past half its width, and fills and strokes in linear and
// conic gradients across their whole bounds. The pixels are the same,
// within a level of 255 where the rasterizer starts an edge at a tile's
// top or left rather than above or left of it.
func TestTiledPathsMatchOneMask(t *testing.T) {
	blue, red := RGB(20, 60, 230), RGB(230, 40, 20)
	view := func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var star, ring, wave, frame Path
			star.MoveTo(60, 8).LineTo(92, 110).LineTo(8, 46).LineTo(112, 46).LineTo(28, 110).Close()
			p.FillPath(&star, blue)
			ring.Circle(180, 60, 50).Circle(180, 60, 25).Circle(205.3, 60.2, 20)
			p.FillPath(&ring, red)
			wave.MoveTo(240, 100).CubeTo(270, -40, 300, 200, 330, 20).QuadTo(380, 140, 250, 30)
			p.StrokePath(&wave, 7.5, blue)
			frame.MoveTo(20.3, 130.6).LineTo(150.1, 140.2).LineTo(110.7, 200.9).Close()
			p.StrokePath(&frame, 3, red)
			var lens Path
			lens.Circle(220, 170, 40)
			p.FillPathGradient(&lens, LinearGradient{Angle: 135, From: red, To: blue})
			p.StrokePathGradient(&wave, 2, LinearGradient{Angle: 90, From: blue, To: red})
			var disc Path
			disc.Circle(320, 170, 35)
			p.FillPathConicGradient(&disc, ConicGradient{Stops: []GradientStop{{red, 0}, {blue, 0.5}, {red, 1}}})
			p.StrokePathConicGradient(&frame, 1.5, ConicGradient{Stops: []GradientStop{{blue, 0}, {red, 1}}})
			var spike Path
			spike.MoveTo(140, 200).LineTo(170, 150).LineTo(176, 210).LineTo(200, 160)
			p.StrokePath(spike.Cap(SquareCap).Join(MiterJoin).MiterLimit(8).Dash(2, 40, 6), 5, blue)
		})
	}
	whole := coverage(view, 380, 220)
	defer func(most, tile int) { pathMaskMax, pathTile = most, tile }(pathMaskMax, pathTile)
	pathMaskMax, pathTile = 0, 17
	tiled := coverage(view, 380, 220)
	worst := 0
	for i := range whole.Pix {
		d := int(whole.Pix[i]) - int(tiled.Pix[i])
		worst = max(worst, d, -d)
	}
	if worst > 1 {
		t.Errorf("tiled paths differ by up to %d levels from paths of one mask", worst)
	}
}

// coverage draws a view w×h DIPs at 2x with the coverage of masks as they
// are rasterized: on Windows, renderers correct it for paths as for text,
// as Direct2D does, which can make a level of difference two.
func coverage(view func(c *context), w, h int) *image.RGBA {
	tt := coreNewTester(view, w, h)
	tt.SetScale(2)
	s := tt.h.last
	s.Text = scene.TextParams{}
	var r raster.Renderer
	r.Render(s)
	return &image.RGBA{Pix: r.Image.RGBA(), Stride: 4 * r.Image.W, Rect: image.Rect(0, 0, r.Image.W, r.Image.H)}
}

// TestLargePathsDrawOnlyWhatShows draws paths tens of thousands of pixels
// across in a small window: a horizontal line, a diagonal one and a square
// around the window. They draw only the tiles that show and that they
// reach, and the square's tiles inside it share their masks.
func TestLargePathsDrawOnlyWhatShows(t *testing.T) {
	blue := RGB(20, 60, 230)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var square, line, diagonal Path
			square.MoveTo(-9000, -9000).LineTo(9000, -9000).LineTo(9000, 9000).LineTo(-9000, 9000).Close()
			p.FillPath(&square, RGB(230, 230, 230))
			line.MoveTo(-20000, 100).LineTo(20000, 100)
			p.StrokePath(&line, 4, blue)
			diagonal.MoveTo(-10000, -10000).LineTo(10000, 10000)
			p.StrokePath(&diagonal, 4, blue)
		})
	}, 400, 200)
	tt.SetScale(2)
	img := tt.Image()
	for _, pt := range [][2]int{{4, 200}, {796, 200}, {300, 300}, {10, 10}} {
		if c := img.RGBAAt(pt[0], pt[1]); c.B < 180 || c.R > 120 {
			t.Errorf("(%d, %d) is %v, not the lines' blue", pt[0], pt[1], c)
		}
	}
	if c := img.RGBAAt(700, 50); c != (color.RGBA{230, 230, 230, 255}) {
		t.Errorf("the square shows %v", c)
	}
	// 800×400 pixels show, in two by two tiles of the square's mask and
	// the diagonal's (511 pixels, from their masks' corners), one by two of
	// the line's. The square has no edge there: its tiles share one mask.
	tiles, masks := map[uint8]int{}, map[[2]uint16]bool{}
	for _, g := range tt.rt.scene.Glyphs {
		tiles[g.Color.G]++
		masks[[2]uint16{g.U, g.V}] = true
	}
	if tiles[230] != 4 || tiles[60] > 2+4 || len(masks) > 1+2+4 {
		t.Errorf("the square draws %d tiles, the lines %d, from %d masks", tiles[230], tiles[60], len(masks))
	}
}

// TestPathsTheAtlasHoldsDrawFromOneMask draws a path 3000 pixels wide, as
// it is and under a transform off the pixel grid: the atlas holds its
// mask, so it draws from one, as before tiles.
func TestPathsTheAtlasHoldsDrawFromOneMask(t *testing.T) {
	for _, moved := range []bool{false, true} {
		tt := coreNewTester(func(c *context) {
			coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
				draw := func() {
					var band Path
					band.MoveTo(0, 10).LineTo(1500, 10).LineTo(1500, 60).LineTo(0, 60).Close()
					p.FillPath(&band, RGB(20, 60, 230))
				}
				if moved {
					p.Transform(Translation(0.3, 0), 0, 0, draw)
				} else {
					draw()
				}
			})
		}, 1600, 100)
		tt.SetScale(2)
		if n := len(tt.rt.scene.Glyphs); n != 1 {
			t.Errorf("transformed %v, the path draws %d masks", moved, n)
		}
	}
}

// TestLongStrokesDrawWhenTheAtlasIsCrowded strokes eight long wires,
// 3540×706 pixels across, that change shape every frame, as wires to a
// node dragged across a canvas: each mask would take a shelf of the atlas
// of its own, and the largest atlas holds five of them. Every wire draws
// in every frame, as tiles once the atlas is crowded.
func TestLongStrokesDrawWhenTheAtlasIsCrowded(t *testing.T) {
	const n = 8
	wire := func(i, frame int) (p Path, at func(t float32) (float32, float32)) {
		x0, y0 := 10+float32(frame)*0.37, float32(10+i*12)
		k := float32(i) * 3.1
		pts := [4][2]float32{{x0, y0}, {900 - k, y0}, {900 + k, y0 + 353}, {1780, y0 + 353}}
		p.MoveTo(pts[0][0], pts[0][1]).CubeTo(pts[1][0], pts[1][1], pts[2][0], pts[2][1], pts[3][0], pts[3][1])
		return p, func(t float32) (float32, float32) {
			u := 1 - t
			w := [4]float32{u * u * u, 3 * u * u * t, 3 * u * t * t, t * t * t}
			var x, y float32
			for j, pt := range pts {
				x, y = x+w[j]*pt[0], y+w[j]*pt[1]
			}
			return x, y
		}
	}
	frame, wires := 0, n
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			for i := range wires {
				path, _ := wire(i, frame)
				p.StrokePath(&path, 2, RGB(20, 60, 230))
			}
		})
	}, 1800, 560)
	tt.SetScale(2)
	for frame = 1; frame <= 2; frame++ {
		tt.Frame()
		img := tt.Image()
		for i := range n {
			_, at := wire(i, frame)
			for _, along := range []float32{0.1, 0.3, 0.5, 0.7, 0.9} {
				x, y := at(along)
				found := false
				for dy := -2; dy <= 2 && !found; dy++ {
					for dx := -2; dx <= 2 && !found; dx++ {
						c := img.RGBAAt(int(2*x)+dx, int(2*y)+dy)
						found = c.B > 150 && c.R < 120
					}
				}
				if !found {
					t.Errorf("frame %d: wire %d is missing %v of the way along, at (%.0f, %.0f) pixels", frame, i, along, 2*x, 2*y)
				}
			}
		}
	}
	if g := len(tt.rt.scene.Glyphs); g <= n {
		t.Errorf("crowded, %d wires draw from %d masks, not as tiles", n, g)
	}
}

// TestCrowdedPathsDrawAsTilesForAWhile strokes a wire 1160×280 pixels
// across and a short one after the atlas was crowded: the long one draws
// as tiles, with the pixels of its mask within a level of 255, and from
// one mask once pathCrowdedPaints paints have gone by; the short one, no
// larger than a tile, from one mask throughout.
func TestCrowdedPathsDrawAsTilesForAWhile(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var long, short Path
			long.MoveTo(10.3, 10).CubeTo(300, 10, 300, 150, 590, 150)
			p.StrokePath(&long, 2, RGB(20, 60, 230))
			short.MoveTo(10, 180).LineTo(100, 190)
			p.StrokePath(&short, 2, RGB(230, 40, 20))
		})
	}, 600, 200)
	tt.SetScale(2)
	// The masks of the frame last painted, and its pixels as masks are
	// rasterized (coverage).
	masks := func() (long, short int) {
		for _, g := range tt.rt.scene.Glyphs {
			if g.Color.R > 200 {
				short++
			} else {
				long++
			}
		}
		return long, short
	}
	pixels := func() []byte {
		s := tt.h.last
		s.Text = scene.TextParams{}
		var r raster.Renderer
		r.Render(s)
		return r.Image.RGBA()
	}
	whole := pixels()
	tt.rt.paths.crowded = tt.rt.paints + pathCrowdedPaints
	tt.Frame()
	if long, short := masks(); long != 3 || short != 1 {
		t.Errorf("crowded, the long wire draws from %d masks, the short one from %d", long, short)
	}
	worst := 0
	for i, v := range pixels() {
		d := int(v) - int(whole[i])
		worst = max(worst, d, -d)
	}
	if worst > 1 {
		t.Errorf("the tiles differ by up to %d levels from one mask", worst)
	}
	tt.rt.paints = tt.rt.paths.crowded
	tt.Frame()
	if long, short := masks(); long != 1 || short != 1 {
		t.Errorf("with room again, the long wire draws from %d masks, the short one from %d", long, short)
	}
}

// TestWiresDrawInTheFrameTheAtlasFills strokes nine long wires that stay
// as they are, 3545×405 pixels across, which the largest atlas keeps in
// one mask each, filling it, then has a tenth as long, which changes shape
// every frame, as a wire to a node dragged across a canvas: its mask finds
// no room, and in that frame and those after, every wire draws, as tiles,
// though the first paint of the frame drew the others' masks, which the
// atlas making room for it would keep, as the frame drew them, leaving
// the tiles no room.
func TestWiresDrawInTheFrameTheAtlasFills(t *testing.T) {
	const n = 9
	frame, long := 0, false
	wire := func(i int) (p Path, at func(t float32) (float32, float32)) {
		x0, y0, x1, y1 := float32(10), float32(10+i*12), float32(1780), float32(10+i*12+200)
		if i == n {
			// Dragged: short until it is long, and moving.
			x0, y0, x1, y1 = 10+float32(frame)*0.37, float32(10+i*12), 200, float32(10+i*12+20)
			if long {
				x1, y1 = 1780, y0+200
			}
		}
		k := float32(i) * 3.1
		pts := [4][2]float32{{x0, y0}, {(x0+x1)/2 - k, y0}, {(x0+x1)/2 + k, y1}, {x1, y1}}
		p.MoveTo(pts[0][0], pts[0][1]).CubeTo(pts[1][0], pts[1][1], pts[2][0], pts[2][1], pts[3][0], pts[3][1])
		return p, func(t float32) (float32, float32) {
			u := 1 - t
			w := [4]float32{u * u * u, 3 * u * u * t, 3 * u * t * t, t * t * t}
			var x, y float32
			for j, pt := range pts {
				x, y = x+w[j]*pt[0], y+w[j]*pt[1]
			}
			return x, y
		}
	}
	empty := true
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			if empty {
				// A line whose mask, 4093 pixels tall, only an empty atlas
				// of the largest holds: the atlas grows to it, and lets go
				// of what earlier tests drew.
				var line Path
				line.MoveTo(2, 1).LineTo(2, 2045)
				p.StrokePath(&line, 2, RGB(20, 60, 230))
				return
			}
			for i := range n + 1 {
				path, _ := wire(i)
				p.StrokePath(&path, 2, RGB(20, 60, 230))
			}
		})
	}, 1800, 360)
	// At twice the pixels, the line tries one mask, not tiles, though the
	// first frame's did not fit the atlas then.
	tt.rt.paths.crowded = 0
	tt.SetScale(2)
	empty, tt.rt.paths.crowded = false, 0
	// Until the wires that stay draw from one mask each, which the atlas
	// keeps.
	for frame = 1; frame <= 3; frame++ {
		tt.Frame()
	}
	if g := len(tt.rt.scene.Glyphs); g != n+1 {
		t.Fatalf("with room, %d wires draw from %d masks", n+1, g)
	}
	long = true
	for end := frame + 3; frame < end; frame++ {
		tt.Frame()
		img := tt.Image()
		for i := range n + 1 {
			_, at := wire(i)
			for _, along := range []float32{0.1, 0.3, 0.5, 0.7, 0.9} {
				x, y := at(along)
				found := false
				for dy := -2; dy <= 2 && !found; dy++ {
					for dx := -2; dx <= 2 && !found; dx++ {
						c := img.RGBAAt(int(2*x)+dx, int(2*y)+dy)
						found = c.B > 150 && c.R < 120
					}
				}
				if !found {
					t.Errorf("frame %d: wire %d is missing %v of the way along, at (%.0f, %.0f) pixels", frame, i, along, 2*x, 2*y)
				}
			}
		}
	}
}

// TestStrokeStylesMatchSVG strokes paths with butt and square caps, miter
// and bevel joins, a miter limit and dashes, and MyGo's SVG renderer
// strokes them with the same stroke-linecap, stroke-linejoin,
// stroke-miterlimit, stroke-dasharray and stroke-dashoffset: their pixels
// are the same within a level of 255. The paths share their center lines,
// at whole DIPs apart, so that each style needs a mask of its own, and a
// corner too sharp for its miter limit is bevelled, while one within it
// reaches as far as it allows.
func TestStrokeStylesMatchSVG(t *testing.T) {
	zig, zigD := func(p *Path, x, y float32) {
		p.MoveTo(x+10.375, y+70.125).LineTo(x+40.25, y+10.625).LineTo(x+70.5, y+70.875).LineTo(x+110.125, y+5.375).LineTo(x+112.75, y+60.25)
	}, "M10.375 70.125L40.25 10.625L70.5 70.875L110.125 5.375L112.75 60.25"
	quad, quadD := func(p *Path, x, y float32) {
		p.MoveTo(x+10.375, y+10.125).LineTo(x+100.625, y+12.25).LineTo(x+100.25, y+60.5).LineTo(x+10.875, y+60.375).Close()
	}, "M10.375 10.125L100.625 12.25L100.25 60.5L10.875 60.375Z"
	spike, spikeD := func(p *Path, x, y float32) {
		p.MoveTo(x+10.5, y+20.25).LineTo(x+110.125, y+35.5).LineTo(x+10.25, y+50.75).MoveTo(x+10.5, y+60.25).LineTo(x+60.125, y+66.5).LineTo(x+10.25, y+72.75)
	}, "M10.5 20.25L110.125 35.5L10.25 50.75M10.5 60.25L60.125 66.5L10.25 72.75"
	styles := []struct {
		path        func(p *Path, x, y float32)
		d           string
		cap         LineCap
		join        LineJoin
		limit       float32
		dash        []float32
		offset      float32
		width       float32
		svgCap      string
		svgJoin     string
		svgDash     string
		svgMiterMax string
	}{
		{zig, zigD, ButtCap, MiterJoin, 0, nil, 0, 9, "butt", "miter", "none", "4"},
		{zig, zigD, SquareCap, BevelJoin, 0, nil, 0, 9, "square", "bevel", "none", "4"},
		{zig, zigD, ButtCap, MiterJoin, 0, []float32{14, 6}, 3, 5, "butt", "miter", "14 6", "4"},
		{quad, quadD, SquareCap, MiterJoin, 0, []float32{17, 9, 3}, -4, 5, "square", "miter", "17 9 3", "4"},
		{quad, quadD, ButtCap, BevelJoin, 0, nil, 0, 7, "butt", "bevel", "none", "4"},
		{spike, spikeD, ButtCap, MiterJoin, 10, nil, 0, 4, "butt", "miter", "none", "10"},
	}
	const cell = 130
	w, h := 2*cell, (len(styles)+1)/2*90
	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" fill="none" stroke="black">`, w, h, w, h)
	for i, s := range styles {
		fmt.Fprintf(&svg, `<path transform="translate(%d %d)" d="%s" stroke-width="%v" stroke-linecap="%s" stroke-linejoin="%s" stroke-miterlimit="%s" stroke-dasharray="%s" stroke-dashoffset="%v"/>`,
			i%2*cell, i/2*90, s.d, s.width, s.svgCap, s.svgJoin, s.svgMiterMax, s.svgDash, s.offset)
	}
	svg.WriteString(`</svg>`)
	icon := MustParseSVG([]byte(svg.String()))
	draw := func(viaSVG bool) func(c *context) {
		return func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
				if viaSVG {
					p.Icon(icon, Rect{0, 0, float32(w), float32(h)}, RGB(0, 0, 0))
					return
				}
				for i, s := range styles {
					var path Path
					s.path(&path, float32(i%2*cell), float32(i/2*90))
					path.Cap(s.cap).Join(s.join)
					if s.limit > 0 {
						path.MiterLimit(s.limit)
					}
					if s.dash != nil {
						path.Dash(s.offset, s.dash...)
					}
					p.StrokePath(&path, s.width, RGB(0, 0, 0))
				}
			})
		}
	}
	got, want := coverage(draw(false), w, h), coverage(draw(true), w, h)
	worst, ink := 0, 0
	for i := range got.Pix {
		d := int(got.Pix[i]) - int(want.Pix[i])
		worst = max(worst, d, -d)
		if got.Pix[i] < 128 {
			ink++
		}
	}
	if worst > 1 || ink < 10000 {
		t.Errorf("the strokes differ from the SVG's by up to %d levels, with %d dark channels", worst, ink)
	}
}

// TestDashesStayAsTrimmed trims a dashed line, from its start and from its
// middle: what it keeps of its dashes shows where they were, and nothing
// past it shows.
func TestDashesStayAsTrimmed(t *testing.T) {
	var from, to float32 = 0, 1
	view := func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var line Path
			line.MoveTo(10, 10).LineTo(110, 10).LineTo(110, 30).Dash(3, 7, 4).Cap(ButtCap).Trim(from, to)
			p.StrokePath(&line, 4, RGB(0, 0, 0))
		})
	}
	whole := coverage(view, 120, 40)
	// along returns how far along the line, 120 DIPs long, the pixel
	// (x, y) is, but for those around its corner.
	along := func(x, y int) (float32, bool) {
		px, py := (float32(x)+0.5)/2, (float32(y)+0.5)/2
		switch {
		case px < 106 && py < 14:
			return px - 10, true
		case px > 106 && py > 14:
			return 100 + py - 10, true
		}
		return 0, false
	}
	for _, trim := range [][2]float32{{0, 0.5}, {0.25, 0.9}} {
		from, to = trim[0], trim[1]
		trimmed := coverage(view, 120, 40)
		for y := range 80 {
			for x := range 240 {
				at, ok := along(x, y)
				a, b := whole.RGBAAt(x, y), trimmed.RGBAAt(x, y)
				switch {
				case !ok:
				case at > from*120+1 && at < to*120-1 && a != b:
					t.Fatalf("trimmed from %v to %v, (%d, %d) shows %v, not %v", from, to, x, y, b, a)
				case (at < from*120-1 || at > to*120+1) && b != (color.RGBA{255, 255, 255, 255}):
					t.Fatalf("trimmed from %v to %v, (%d, %d) shows %v", from, to, x, y, b)
				}
			}
		}
	}
}

// TestStrokeStylesHaveMasksOfTheirOwn strokes a path at whole DIPs apart,
// as wide each time and reaching as far, with round caps and joins, butt
// caps, bevel joins and dashes: each stroke draws from a mask of its own.
func TestStrokeStylesHaveMasksOfTheirOwn(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
			for i, style := range []func(*Path){
				func(*Path) {},
				func(p *Path) { p.Cap(ButtCap) },
				func(p *Path) { p.Join(BevelJoin) },
				func(p *Path) { p.Dash(0, 6, 3) },
				func(p *Path) { p.Dash(1, 6, 3) },
			} {
				var path Path
				x := float32(i * 50)
				path.MoveTo(x+10, 10).LineTo(x+40, 20).LineTo(x+10, 30)
				style(&path)
				p.StrokePath(&path, 4, RGB(20, 60, 230))
			}
		})
	}, 260, 40)
	masks := map[[2]uint16]bool{}
	for _, g := range tt.rt.scene.Glyphs {
		masks[[2]uint16{g.U, g.V}] = true
	}
	if len(masks) != 5 {
		t.Errorf("five strokes draw from %d masks", len(masks))
	}
}

// TestMiterLimitsOfOtherJoinsShareAMask strokes a path with round joins
// and with bevel joins, each with miter limits of 4 and 10: the limit
// shapes miter joins alone, so each join draws from one mask.
func TestMiterLimitsOfOtherJoinsShareAMask(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
			for i, join := range []LineJoin{RoundJoin, RoundJoin, BevelJoin, BevelJoin} {
				var path Path
				y := float32(10 + 30*i)
				path.MoveTo(10, y).LineTo(40, y+10).LineTo(10, y+20).Join(join).MiterLimit(float32(4 + 6*(i%2)))
				p.StrokePath(&path, 4, RGB(20, 60, 230))
			}
		})
	}, 60, 130)
	masks := map[[2]uint16]bool{}
	for _, g := range tt.rt.scene.Glyphs {
		masks[[2]uint16{g.U, g.V}] = true
	}
	if len(masks) != 2 {
		t.Errorf("two joins draw from %d masks", len(masks))
	}
}

// TestCapsOfLinesOfNoLength strokes subpaths of no length: a round cap
// draws a dot, which adds up with a line it lies on rather than cutting a
// hole in it, a square cap a square, and a butt cap nothing. Trimmed to
// almost nothing, a stroke with butt caps shows what it keeps, without
// fading as caps that would show a dot do.
func TestCapsOfLinesOfNoLength(t *testing.T) {
	img := coverage(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var dot, square, none Path
			dot.MoveTo(10, 10).LineTo(50, 10).MoveTo(30, 10).LineTo(30, 10)
			p.StrokePath(&dot, 6, RGB(0, 0, 0))
			square.MoveTo(70, 10).LineTo(70, 10).Cap(SquareCap)
			p.StrokePath(&square, 6, RGB(0, 0, 0))
			none.MoveTo(90, 10).LineTo(90, 10).Cap(ButtCap)
			p.StrokePath(&none, 6, RGB(0, 0, 0))
		})
	}, 100, 20)
	black, white := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{{60, 20, black}, {61, 21, black}, {137, 17, black}, {180, 20, white}} {
		if got := img.RGBAAt(c.x, c.y); got != c.want {
			t.Errorf("(%d, %d) is %v, not %v", c.x, c.y, got, c.want)
		}
	}
	f := flatPath{trimmed: true, kept: 0.5, cap: ButtCap}
	if k := f.capFade(4); k != 1 {
		t.Errorf("trimmed short, a stroke with butt caps shows %v as opaque", k)
	}
}

// TestMiterMasksReachAsFarAsTheirPoints strokes a square with miter joins
// and a miter limit of 100: its mask reaches as far as its corners' points
// do, √2 half widths from them, not as far as the limit would let them.
func TestMiterMasksReachAsFarAsTheirPoints(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
			var square Path
			square.MoveTo(100, 100).LineTo(200, 100).LineTo(200, 200).LineTo(100, 200).Close()
			p.StrokePath(square.Join(MiterJoin).MiterLimit(100), 10, RGB(20, 60, 230))
		})
	}, 300, 300)
	g := tt.rt.scene.Glyphs
	// 100 DIPs and √2 times 5 to each side, and a pixel or two of
	// rounding, where the limit would make it 1100.
	if len(g) != 1 || g[0].W < 114 || g[0].W > 118 || g[0].H < 114 || g[0].H > 118 {
		t.Errorf("the square's mask is %v", g)
	}
}

// TestDottedLines dashes lines with dashes of no length, which show as
// dots with round caps, and starts a line's pattern where a dash ends,
// which is in the gap after it, as Skia has it: no dot there.
func TestDottedLines(t *testing.T) {
	img := coverage(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var dots, gap Path
			dots.MoveTo(10, 10).LineTo(90, 10).Dash(0, 0, 8)
			p.StrokePath(&dots, 4, RGB(0, 0, 0))
			gap.MoveTo(10, 30).LineTo(90, 30).Dash(5, 5, 5)
			p.StrokePath(&gap, 4, RGB(0, 0, 0))
		})
	}, 100, 40)
	black, white := color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}
	for x := 10; x < 90; x += 8 {
		if c := img.RGBAAt(2*x, 20); c != black {
			t.Errorf("the dot at %d is %v", x, c)
		}
		if c := img.RGBAAt(2*x+8, 20); c != white {
			t.Errorf("between dots, %d is %v", x+4, c)
		}
	}
	if c := img.RGBAAt(20, 60); c != white {
		t.Errorf("starting in the gap, the line shows %v", c)
	}
	if c := img.RGBAAt(36, 60); c != black {
		t.Errorf("its first dash shows %v", c)
	}
}

// TestDashesTurnAroundClosedSubpaths dashes squares with butt caps and
// miter joins: a dash through the first corner turns it as one dash, its
// point filled, and a dash longer than the square strokes it as it is
// stroked whole.
func TestDashesTurnAroundClosedSubpaths(t *testing.T) {
	square := func(p *Path, x float32) *Path {
		return p.MoveTo(x+10, 10).LineTo(x+50, 10).LineTo(x+50, 50).LineTo(x+10, 50).Close().Cap(ButtCap).Join(MiterJoin)
	}
	img := coverage(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var through, long, whole Path
			// 160 DIPs around, ten patterns: the first starts 4 into a
			// dash, and the last ends there.
			p.StrokePath(square(&through, 0).Dash(4, 10, 6), 6, RGB(0, 0, 0))
			p.StrokePath(square(&long, 60).Dash(0, 1000, 10), 6, RGB(0, 0, 0))
			p.StrokePath(square(&whole, 120), 6, RGB(0, 0, 0))
		})
	}, 180, 60)
	if c := img.RGBAAt(2*10-4, 2*10-4); c != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("the point of the first corner is %v", c)
	}
	for y := range 120 {
		for x := 130; x < 236; x++ {
			if a, b := img.RGBAAt(x+120, y), img.RGBAAt(x, y); a != b {
				t.Fatalf("a dash around the square differs from it whole at (%d, %d): %v, %v", x, y, b, a)
			}
		}
	}
}

// TestStrokesPastTheCutsDrawWhole cuts a dashed line into more dashes and
// gaps than strokeCutsMax: the rest of it, and the path's other subpaths,
// are stroked whole rather than left out.
func TestStrokesPastTheCutsDrawWhole(t *testing.T) {
	defer func(most int) { strokeCutsMax = most }(strokeCutsMax)
	strokeCutsMax = 10
	img := coverage(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			var lines Path
			lines.MoveTo(10, 10).LineTo(90, 10).MoveTo(10, 30).LineTo(90, 30).Dash(0, 2, 2).Cap(ButtCap)
			p.StrokePath(&lines, 4, RGB(0, 0, 0))
		})
	}, 100, 40)
	// Ten cuts take the line to 30 DIPs; past them, gaps are stroked.
	for _, pt := range [][2]int{{2*15 + 1, 20}, {2*80 + 1, 20}, {2*15 + 1, 60}, {2*80 + 1, 60}} {
		if c := img.RGBAAt(pt[0], pt[1]); c != (color.RGBA{0, 0, 0, 255}) {
			t.Errorf("(%d, %d) is %v", pt[0], pt[1], c)
		}
	}
	if c := img.RGBAAt(2*12+1, 20); c != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("a gap before the cuts run out is %v", c)
	}
}

// TestDashOffsetsAPatternApartShareAMask strokes a dashed line with
// offsets a whole pattern apart, as an offset animating around it does:
// they draw from one mask.
func TestDashOffsetsAPatternApartShareAMask(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
			for i, offset := range []float32{1, 10, -8} {
				var line Path
				y := float32(10 + 20*i)
				p.StrokePath(line.MoveTo(10, y).LineTo(90, y).Dash(offset, 6, 3), 2, RGB(20, 60, 230))
			}
		})
	}, 100, 70)
	masks := map[[2]uint16]bool{}
	for _, g := range tt.rt.scene.Glyphs {
		masks[[2]uint16{g.U, g.V}] = true
	}
	if len(masks) != 1 {
		t.Errorf("three lines dashed alike draw from %d masks", len(masks))
	}
}

// TestResetRemovesTheStrokesStyle resets a path whose stroke has caps,
// joins, a miter limit and dashes: it strokes round and whole again.
func TestResetRemovesTheStrokesStyle(t *testing.T) {
	var p Path
	p.Cap(SquareCap).Join(BevelJoin).MiterLimit(9).Dash(2, 3, 4).Reset()
	if p.cap != RoundCap || p.join != RoundJoin || p.miter != 0 || len(p.dash) != 0 || p.dashOffset != 0 {
		t.Errorf("reset, the path keeps %v %v %v %v %v", p.cap, p.join, p.miter, p.dash, p.dashOffset)
	}
	for _, lengths := range [][]float32{{}, {3, -1}, {0, 0}} {
		if p.Dash(0, lengths...); len(p.dash) != 0 {
			t.Errorf("dashed %v, the path keeps %v", lengths, p.dash)
		}
	}
}
