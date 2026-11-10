package usage

import (
	"image"
	"math"
	"time"

	"arion.local/desktop/ui/shared/iso"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The first-run figure: a desk microphone and nothing else, drawn live
// with the Anatomy kit's rules (ui/shared/iso). Its capsule hangs in a
// yoke on two thumb-screw pivots. In a loop the microphone turns from
// facing away round to the viewer, the capsule tips up from leaning back
// and settles aimed at the speaker, the band lights, then goes out and the
// microphone drifts back: it is ready, and nothing has been said yet.
//
// World units are about a millimetre: x and y on the floor, z up, the
// microphone standing on the origin.

// The camera, as the kit's default, a little to the right of the front.
const (
	micAzimuth   = 40
	micElevation = 30
)

// The poses the loop moves between, in degrees: away is turned off to the
// side and leaning back, aimed faces the viewer, tipped a little forward.
const (
	awayYaw, awayTilt   = -62, -30
	aimedYaw, aimedTilt = 0, 14
	micLoopSeconds      = 4.4
)

const micLoop = micLoopSeconds * float64(time.Second)

// micPose is the figure at one moment.
type micPose struct {
	// yaw turns the whole microphone about its stem, in degrees from
	// facing the viewer.
	yaw float32
	// tilt leans the capsule on its pivots, in degrees from upright,
	// forward positive.
	tilt float32
	// on is how far the switch has slid and the band is lit, 0 to 1.
	on float32
}

// aimed is the pose held under reduced motion: ready, facing the viewer.
var aimed = micPose{yaw: aimedYaw, tilt: aimedTilt}

// micAt is the pose at t into the loop, which starts and ends turned away.
func micAt(t time.Duration) micPose {
	s := float32(math.Mod(float64(t), micLoop) / float64(time.Second))
	span := func(a, b float32) float32 { return clamp((s - a) / (b - a)) }
	// Turning round and tipping up overlap; the tip settles past its mark
	// and back, as a capsule does on a stiff pivot. Both drift back
	// together at the end.
	turn := ease(span(0.4, 1.6)) - ease(span(3.4, 4.4))
	tip := settle(span(0.7, 2.0)) - ease(span(3.4, 4.4))
	return micPose{
		yaw:  lerp(awayYaw, aimedYaw, turn),
		tilt: lerp(awayTilt, aimedTilt, tip),
		on:   ease(span(1.8, 2.1)) - ease(span(3.0, 3.3)),
	}
}

func clamp(v float32) float32 { return max(0, min(v, 1)) }

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

// ease is a cubic ease in and out.
func ease(t float32) float32 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := 2*t - 2
	return 1 + u*u*u/2
}

// settle eases in and out to 1, carried past it by a few percent on the
// way and back: a capsule settling on a stiff pivot. It starts and ends
// at rest.
func settle(t float32) float32 {
	const over = 0.08
	bump := float32(math.Sin(math.Pi * float64(clamp((t-0.5)/0.5))))
	return ease(t) + over*bump*bump
}

// micFrame is everything one picture depends on, so the same frame is
// never drawn twice.
type micFrame struct {
	ink  iso.Ink
	pose micPose
}

// micPicture keeps the figure's bitmap. Two images take turns, as the
// GPU may still read the one last shown.
type micPicture struct {
	canvas iso.Canvas
	images [2]*image.RGBA
	turn   int
	bitmap *ui.Bitmap
	last   micFrame
	size   image.Point
}

// mic is the one figure the overview shows.
var mic micPicture

// Microphone fills its box with the figure, labeled for assistive
// technology. It loops while drawn and holds the aimed pose under reduced
// motion.
func Microphone(c *ui.Context, tokens theme.Tokens, label string) ui.Element {
	still := c.Preferences().ReduceMotion
	ink := micInk(tokens)
	return ui.Box(c).Role(ui.RoleImage).Label(label).Draw(func(p *ui.Painter, r ui.Rect) {
		pose := aimed
		if !still {
			// Repaint only: the loop is a pure function of the clock.
			p.AnimationFrame()
			pose = micAt(time.Duration(p.Now().UnixNano()))
		}
		p.Image(mic.show(r, p.Scale(), micFrame{ink, pose}), r, ui.FillBox)
	})
}

// micInk is the kit's inks with outlines drawn nearer Text: the figure is
// small, and at one pixel a DIP the Faint outline fades into the card.
func micInk(tokens theme.Tokens) iso.Ink {
	ink := iso.InkOf(tokens)
	ink.Edge = ink.Edge.Mix(tokens.Text, 0.3)
	ink.Body.Edge = ink.Edge
	return ink
}

// show returns the bitmap of f in a box r DIPs at scale pixels to a DIP,
// drawing it only if it changed.
func (v *micPicture) show(r ui.Rect, scale float32, f micFrame) *ui.Bitmap {
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
	view := ui.Rect{W: r.W, H: r.H}
	v.canvas.Begin(img, view, scale)
	drawMic(&v.canvas, view, f.ink, f.pose)
	v.canvas.End()
	if v.bitmap == nil {
		v.bitmap = ui.NewBitmap(img)
	} else {
		v.bitmap.Update(img)
	}
	v.last = f
	return v.bitmap
}

// micBounds are points round the space the figure fills over the whole
// loop, so that the camera does not move with the capsule.
var micBounds = func() []iso.Vec {
	var out []iso.Vec
	for i := 0; i < 32; i++ {
		a := float64(i) / 32 * 2 * math.Pi
		out = append(out, iso.Vec{(baseR + 5) * float32(math.Cos(a)), (baseR + 5) * float32(math.Sin(a)), -2})
	}
	for ms := 0; ms < int(micLoop/float64(time.Millisecond)); ms += 50 {
		h := headOf(micAt(time.Duration(ms) * time.Millisecond))
		for _, off := range []float32{capsuleBottom, capsuleTop} {
			out = append(out, h.ring(off, capsuleR+0.4, 16)...)
		}
		for _, side := range []float32{-1, 1} {
			out = append(out, h.pivot.Add(h.side.Scale(side*(armOut+knobH))).Add(iso.Vec{0, 0, knobR}))
		}
	}
	return out
}()

// fit returns the camera that shows micBounds within view, pad DIPs in,
// at the left: an empty state reads from its left edge.
func fit(view ui.Rect, pad float32) iso.Camera {
	extent := func(cam iso.Camera) (x0, y0, x1, y1 float32) {
		x0, y0, x1, y1 = math.MaxFloat32, math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32
		for _, p := range micBounds {
			x, y := cam.Project(p)
			x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
		}
		return
	}
	x0, y0, x1, y1 := extent(iso.NewCamera(micAzimuth, micElevation, 1, iso.Vec{}, view))
	scale := min((view.W-2*pad)/(x1-x0), (view.H-2*pad)/(y1-y0))
	cam := iso.NewCamera(micAzimuth, micElevation, scale, iso.Vec{}, view)
	x0, y0, _, y1 = extent(cam)
	cam.Center[0] += view.X + pad - x0
	cam.Center[1] += view.Y + (view.H-(y1-y0))/2 - y0
	return cam
}
