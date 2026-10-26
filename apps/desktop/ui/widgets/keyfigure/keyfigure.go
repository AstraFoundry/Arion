// Package keyfigure illustrates the dictation key: an isometric laptop,
// drawn live with the Anatomy kit's rules (ui/shared/iso). It arrives
// closed with its back to the viewer, turns round as its lid opens, and
// the camera closes in on the chosen key, which is lit. Choosing another
// key pulls the camera back a little on its way there; a press sinks the
// cap.
package keyfigure

import (
	"image"
	"math"
	"slices"
	"sort"
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/iso"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// platforms have a laptop each in the figure.
var platforms = []string{"darwin", "windows"}

var layouts = func() map[string][]key {
	out := map[string][]key{}
	for _, platform := range platforms {
		out[platform] = keysOf(platform)
	}
	return out
}()

// keyboardOf is the platform whose keyboard carries key.
func keyboardOf(key runtime.HoldKey) string {
	for _, platform := range platforms {
		if slices.Contains(runtime.HoldKeysFor(platform), key) {
			return platform
		}
	}
	return ""
}

// Shows reports whether the figure has key, so callers keep another
// illustration for a platform without one.
func Shows(key runtime.HoldKey) bool { return keyboardOf(key) != "" }

// The camera: where it ends up, how close, and how far it pulls back
// between keys.
const (
	azimuth   = 32
	elevation = 40
	keyPitch  = 46   // DIPs a key pitch spans once zoomed in
	pullBack  = 0.78 // of the zoom, halfway between two keys
)

// The arrival, from the first frame.
var (
	turnFrames = []ui.Keyframe{{Value: 0}, {At: 250 * time.Millisecond, Value: 0}, {At: 1350 * time.Millisecond, Value: 1, Ease: ui.EaseInOut}}
	openFrames = []ui.Keyframe{{Value: 0}, {At: 650 * time.Millisecond, Value: 0}, {At: 1650 * time.Millisecond, Value: 1, Ease: ui.EaseInOut}}
	zoomFrames = []ui.Keyframe{{Value: 0}, {At: 1000 * time.Millisecond, Value: 0}, {At: 2100 * time.Millisecond, Value: 1, Ease: ui.EaseInOut}}
	dipFrames  = []ui.Keyframe{{Value: 1}, {At: 180 * time.Millisecond, Value: pullBack, Ease: ui.EaseOut}, {At: 620 * time.Millisecond, Value: 1, Ease: ui.EaseInOut}}
	// Down quickly, held a moment, back up softly.
	pressFrames = []ui.Keyframe{{Value: 0}, {At: 80 * time.Millisecond, Value: 1, Ease: ui.EaseOut}, {At: 480 * time.Millisecond, Value: 1}, {At: 700 * time.Millisecond, Value: 0, Ease: ui.EaseOut}}
)

// pose is the figure at one frame.
type pose struct {
	turn, open, zoom, dip, press float32
	focus                        iso.Vec
}

// Render fills its box with the figure, described by label. Changed is
// when another key was chosen, and pressed when the key last went down,
// each zero until then.
func Render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, label copy.Key, chosen runtime.HoldKey, changed, pressed time.Time) {
	platform := keyboardOf(chosen)
	keys := layouts[platform]
	i := slices.IndexFunc(keys, func(k key) bool { return k.id == string(chosen) })
	target := keys[i].center(0)
	box := ui.Box(c).Fill().Clip().Background(tokens.Card).Role(ui.RoleImage).Label(messages.Text(label))
	at := pose{
		turn: box.Keyframes("turn", turnFrames),
		open: box.Keyframes("open", openFrames),
		zoom: box.Keyframes("zoom", zoomFrames),
		dip:  1,
		// The camera glides to the chosen key.
		focus: iso.Vec{box.AnimateTo("fx", target[0], theme.Settle), box.AnimateTo("fy", target[1], theme.Settle), target[2]},
	}
	if !changed.IsZero() {
		at.dip = box.Keyframes("dip", dipFrames, ui.KeyframeOptions{Run: uint64(changed.UnixNano())})
	}
	if !pressed.IsZero() {
		at.press = box.Keyframes("press", pressFrames, ui.KeyframeOptions{Run: uint64(pressed.UnixNano())})
	}
	colors := colorsOf(tokens)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Image(view.show(r, p.Scale(), frame{colors, platform, string(chosen), at}), r, ui.FillBox)
	})
}

// frame is everything a picture depends on: the same frame is not drawn
// twice, so a figure at rest costs nothing.
type frame struct {
	ink      colors
	platform string
	chosen   string
	at       pose
}

// picture keeps the figure's bitmap. Two images take turns, as the GPU
// may still read the one last shown.
type picture struct {
	canvas iso.Canvas
	images [2]*image.RGBA
	turn   int
	bitmap *ui.Bitmap
	last   frame
	size   image.Point
}

// view is the one figure onboarding shows.
var view picture

// show returns the bitmap of f in a box r DIPs at scale pixels to a DIP,
// drawing it if it changed.
func (v *picture) show(r ui.Rect, scale float32, f frame) *ui.Bitmap {
	size := image.Pt(int(math.Ceil(float64(r.W*scale))), int(math.Ceil(float64(r.H*scale))))
	if v.bitmap != nil && size == v.size && f == v.last {
		return v.bitmap
	}
	if size != v.size {
		v.images = [2]*image.RGBA{image.NewRGBA(image.Rectangle{Max: size}), image.NewRGBA(image.Rectangle{Max: size})}
		v.size = size
	}
	v.turn = 1 - v.turn
	img := v.images[v.turn]
	v.canvas.Begin(img, ui.Rect{W: r.W, H: r.H}, scale)
	draw(&v.canvas, ui.Rect{W: r.W, H: r.H}, f.ink, layouts[f.platform], f.chosen, f.at)
	v.canvas.End()
	if v.bitmap == nil {
		v.bitmap = ui.NewBitmap(img)
	} else {
		v.bitmap.Update(img)
	}
	v.last = f
	return v.bitmap
}

// colors are the figure's inks, all from the Sunbeam tokens.
type colors struct {
	body, lit                    iso.Palette
	well, pad, screen, halo, ink ui.Color
	litInk, lens                 ui.Color
}

func colorsOf(t theme.Tokens) colors {
	shade, well, top, edge := float32(0.16), float32(0.1), t.Card, t.Faint
	if t.Dark {
		shade, well, top, edge = 0.5, 0.5, t.Card.Mix(t.Text, 0.05), t.Faint.Mix(t.Card, 0.35)
	}
	screen := t.PillSurface
	screen.A = 255
	body := iso.Palette{Top: top, Shadow: t.Card.Mix(t.Shade, shade), Lit: t.Card.Mix(t.Shade, shade/4), Edge: edge, Crease: t.CardBorder}
	lit := iso.Palette{Top: top.Mix(t.Accent, 0.24), Shadow: t.Card.Mix(t.Accent, 0.3).Mix(t.Shade, shade), Lit: t.Card.Mix(t.Accent, 0.3), Edge: t.AccentInk, Crease: t.AccentInk.Alpha(0.5)}
	return colors{
		body: body, lit: lit,
		well: t.Card.Mix(t.Shade, well), pad: top.Mix(t.Shade, well/3), screen: screen,
		halo: t.Shade.Alpha(map[bool]float32{false: 0.08, true: 0.22}[t.Dark]),
		ink:  edge, litInk: t.AccentInk, lens: t.Faint,
	}
}

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

func draw(k *iso.Canvas, r ui.Rect, ink colors, keys []key, chosen string, at pose) {
	overview := iso.Vec{base.x + base.w/2, base.y + base.d/2, baseH}
	fit := iso.ScaleFor(r.W-32, float32(math.Hypot(float64(base.w), float64(base.d))))
	close := iso.ScaleFor(keyPitch, U)
	// Zooming is even in scale ratio, not in scale.
	scale := fit * float32(math.Pow(float64(close/fit), float64(at.zoom))) * at.dip
	cam := iso.NewCamera(azimuth+180*(1-at.turn), elevation, scale, overview.Lerp(at.focus, at.zoom), r)
	width := lerp(0.6, 1, at.zoom)
	lid := lidAt(lidOpen * at.open)
	stroke := func(lines [][]iso.Vec, w float32, color ui.Color) {
		var s iso.Shape
		for _, l := range lines {
			s = append(s, iso.Poly(cam, l, false)...)
		}
		k.Stroke(s, w, color)
	}
	shape := func(points []iso.Vec, fill, edge ui.Color) {
		k.Fill(iso.Poly(cam, points, true), fill)
		k.Stroke(iso.Outline(cam, points), width, edge)
	}

	// A soft contact shadow: the footprint grown in steps, each faint.
	// Only a rim shows past the base, so each step is a ring: the
	// footprint, wound the other way, cuts its middle out.
	footprint := iso.On(iso.RoundedRect(base.x, base.y, base.w, base.d, 10, 6), iso.Floor(0))
	slices.Reverse(footprint)
	for i := 0; i < 2; i++ {
		grow := float32(i)*6 + 2
		ring := iso.Poly(cam, iso.On(iso.RoundedRect(base.x-grow, base.y-grow, base.w+2*grow, base.d+2*grow, 10+grow, 6), iso.Floor(0)), true)
		k.Fill(append(ring, iso.Poly(cam, footprint, true)...), ink.halo)
	}

	// The base, with the keyboard well, trackpad and thumb scoop on its deck.
	deck := iso.Floor(baseH)
	slab := iso.Prism{Ring: iso.On(iso.RoundedRect(base.x, base.y, base.w, base.d, 10, 6), iso.Floor(0)), Up: iso.Vec{0, 0, baseH}}
	slab.Draw(k, cam, ink.body, width, func() {
		shape(iso.On(iso.RoundedRect(-3, -3, 15*U+6, keysDepth+6, 4, 4), deck), ink.well, ink.body.Crease)
		padY := keysDepth + padGap
		shape(iso.On(iso.RoundedRect(7.5*U-padW/2, padY, padW, padD, 6, 6), deck), ink.pad, ink.body.Crease)
		stroke([][]iso.Vec{iso.On(arc(7.5*U, base.y+base.d, 16, 5, 180, 360, 16), deck)}, width, ink.body.Crease)
	})
	// Ports on whichever side shows.
	for _, side := range []float32{-1, 1} {
		if !cam.Sees(iso.Vec{side, 0, 0}) {
			continue
		}
		x := base.x + base.w/2 + side*base.w/2
		for _, y := range []float32{base.y + 40, base.y + 58} {
			shape(iso.On(iso.RoundedRect(y, 3.5, 12, 3, 1.5, 3), func(u, v float32) iso.Vec { return iso.Vec{x, u, v} }), ink.well, ink.body.Edge)
		}
	}

	// The lid, its screen, and the hinge barrel.
	shell := iso.Prism{Ring: iso.On(iso.RoundedRect(base.x, 0, base.w, base.d, 10, 6), func(u, v float32) iso.Vec { return lid(u, v, 0) }), Up: lid(0, 0, lidH).Sub(lid(0, 0, 0))}
	inner := lid(0, 0, 0).Sub(lid(0, 0, 1))
	drawLid := func() {
		shell.Draw(k, cam, ink.body, width, nil)
		if cam.Sees(inner) {
			face := func(u, v float32) iso.Vec { return lid(u, v, 0) }
			shape(iso.On(iso.RoundedRect(base.x+9, 14, base.w-18, base.d-23, 3, 4), face), ink.screen, ink.body.Edge)
			k.Fill(iso.Poly(cam, iso.On(iso.Circle(base.x+base.w/2, base.d-4.5, 1.1, 12), face), true), ink.lens)
		}
	}
	barrel := iso.Prism{
		Ring: iso.On(iso.Circle(base.y, baseH+lidGap-0.6, hingeR, 18), func(u, v float32) iso.Vec { return iso.Vec{base.x + hingeEdge, u, v} }),
		Up:   iso.Vec{base.w - 2*hingeEdge, 0, 0},
	}

	// The caps, back to front, skipping those out of view.
	canvas := k
	drawKeys := func() {
		order := make([]int, 0, len(keys))
		for i, k := range keys {
			if !k.inView(cam, r) {
				continue
			}
			order = append(order, i)
		}
		sort.Slice(order, func(a, b int) bool { return cam.Depth(keys[order[a]].center(0)) < cam.Depth(keys[order[b]].center(0)) })
		for _, i := range order {
			k := keys[i]
			pal, legendInk, sink := ink.body, ink.ink, float32(0)
			if k.id == chosen {
				pal, legendInk, sink = ink.lit, ink.litInk, travel*at.press
				pal.Top = pal.Top.Mix(ink.lit.Edge, 0.25*at.press)
			}
			top := k.center(sink)
			k.prism(sink).Draw(canvas, cam, pal, width, func() {
				if k.mark != "" {
					stroke(legend(k.mark, top), width*1.25, legendInk)
				}
				if k.homing {
					stroke([][]iso.Vec{{{top[0] - 2.2, k.y + k.d - 3, top[2]}, {top[0] + 2.2, k.y + k.d - 3, top[2]}}}, width*1.4, legendInk)
				}
				if k.touch {
					canvas.Stroke(iso.Outline(cam, iso.On(iso.Circle(top[0], top[1], 4, 24), iso.Floor(top[2]))), width, legendInk)
				}
			})
		}
	}

	// Seen from inside the lid, the caps stand in front of it; seen from
	// outside, the lid covers them.
	if cam.Sees(inner) {
		drawLid()
		barrel.Draw(k, cam, ink.body, width, nil)
		drawKeys()
	} else {
		drawKeys()
		barrel.Draw(k, cam, ink.body, width, nil)
		drawLid()
	}
}
