package usage

import (
	"image"
	"math"
	"time"

	"arion.local/desktop/ui/shared/iso"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The first-run figure: a desk microphone and nothing else, drawn live with the Anatomy
// kit's rules (ui/shared/iso). In a loop its switch slides on, the band
// round its capsule lights, three dashed rings of sound spread from the
// band and fade, and the switch slides off again: it is ready, and
// nothing has been said yet.
//
// World units are about a millimetre: x and y on the floor, z up, the
// microphone standing on the origin.

// The camera, as the kit's default, turned a little toward the switch.
const (
	micAzimuth   = 40
	micElevation = 30
)

// The microphone's measures.
const (
	baseR, baseH   = 20, 5
	stemR          = 2.2
	collarZ        = 30
	capsuleR       = 9
	capsuleZ0      = 46
	capsuleZ1      = 70
	bandZ, bandH   = 54, 2.4
	ringReach      = 16 // how far past the band a ring of sound spreads
	rings          = 3
	ringFor        = 1.4  // seconds a ring takes to spread
	ringFirst      = 0.8  // seconds into the loop the first ring leaves
	ringStagger    = 0.45 // seconds between rings
	micLoopSeconds = 3.6
)

const micLoop = micLoopSeconds * float64(time.Second)

// micPose is the figure at one moment.
type micPose struct {
	// on is how far the switch has slid and the band is lit, 0 to 1.
	on float32
	// rings is how far each ring of sound has spread, 0 to 1, or -1
	// while it is not out.
	rings [rings]float32
}

// micAt is the pose at t into the loop. The still pose, under reduced
// motion and at t = 0, is the microphone switched off and quiet.
func micAt(t time.Duration) micPose {
	s := float32(math.Mod(float64(t), micLoop) / float64(time.Second))
	ramp := func(a, b float32) float32 { return ease(clamp((s - a) / (b - a))) }
	p := micPose{on: ramp(0.3, 0.7) - ramp(2.6, 3.0)}
	for i := range p.rings {
		u := (s - ringFirst - float32(i)*ringStagger) / ringFor
		if u < 0 || u > 1 {
			u = -1
		}
		p.rings[i] = u
	}
	return p
}

func clamp(v float32) float32 { return max(0, min(v, 1)) }

// ease is a cubic ease in and out.
func ease(t float32) float32 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := 2*t - 2
	return 1 + u*u*u/2
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
// technology. It loops while drawn and stands still under reduced motion.
func Microphone(c *ui.Context, tokens theme.Tokens, label string) ui.Element {
	still := c.Preferences().ReduceMotion
	ink := micInk(tokens)
	return ui.Box(c).Role(ui.RoleImage).Label(label).Draw(func(p *ui.Painter, r ui.Rect) {
		pose := micAt(0)
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

// micBounds are points round the space the figure fills, the widest ring
// and the capsule's rim included, so that the camera does not move with the rings.
func micBounds() []iso.Vec {
	reach := float32(capsuleR + 1 + ringReach)
	ring := float32(bandZ + bandH/2)
	var out []iso.Vec
	for i := 0; i < 32; i++ {
		a := float64(i) / 32 * 2 * math.Pi
		cos, sin := float32(math.Cos(a)), float32(math.Sin(a))
		out = append(out, iso.Vec{reach * cos, reach * sin, ring}, iso.Vec{(baseR + 5) * cos, (baseR + 5) * sin, -2},
			iso.Vec{capsuleR * cos, capsuleR * sin, capsuleZ1})
	}
	return out
}

// fit returns the camera that shows micBounds within view, pad DIPs in,
// at the left: an empty state reads from its left edge.
func fit(view ui.Rect, pad float32) iso.Camera {
	bounds := micBounds()
	extent := func(cam iso.Camera) (x0, y0, x1, y1 float32) {
		x0, y0, x1, y1 = math.MaxFloat32, math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32
		for _, p := range bounds {
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

// toward is the floor direction to the viewer, and across the one to its
// right, for parts placed on the near side.
func toward() (iso.Vec, iso.Vec) {
	a := float64(micAzimuth) * math.Pi / 180
	t := iso.Vec{float32(math.Cos(a)), float32(math.Sin(a)), 0}
	return t, iso.Vec{-t[1], t[0], 0}
}

// drawMic paints the figure back to front, bottom to top.
func drawMic(k *iso.Canvas, view ui.Rect, ink iso.Ink, at micPose) {
	cam := fit(view, 2)
	const width = 1
	body := ink.Body
	rubber := iso.Palette{Top: body.Shadow, Shadow: body.Shadow, Lit: body.Shadow, Edge: ink.Edge, Crease: ink.Edge}
	solid := func(pr iso.Prism, pal iso.Palette, top func()) { pr.Draw(k, cam, pal, width, top) }
	line := func(points []iso.Vec, color ui.Color) { k.Stroke(iso.Poly(cam, points, false), width, color) }
	dot := func(p iso.Vec, r float32, color ui.Color) {
		k.Fill(iso.Poly(cam, iso.On(iso.Circle(p[0], p[1], r, 12), iso.Floor(p[2])), true), color)
	}
	round := func(z, r, h float32) iso.Prism {
		return iso.Prism{Ring: iso.On(iso.Circle(0, 0, r, 48), iso.Floor(z)), Up: iso.Vec{0, 0, h}}
	}
	near, right := toward()
	a := float64(micAzimuth) * math.Pi / 180

	k.Fill(iso.Poly(cam, iso.On(iso.Circle(0, 0, baseR+5, 48), iso.Floor(-2)), true), ink.Halo)

	// The base: a rubber ring, the weighted disc with a groove and screws,
	// and a bevel.
	solid(round(-1.5, baseR-1, 1.5), rubber, nil)
	solid(round(0, baseR, baseH), body, func() {
		k.Stroke(iso.Outline(cam, iso.On(iso.Circle(0, 0, baseR-3.5, 48), iso.Floor(baseH))), width, body.Crease)
		for i := 0; i < 3; i++ {
			angle := a + math.Pi/2 + float64(i)*2*math.Pi/3
			p := iso.Vec{15 * float32(math.Cos(angle)) * 0.86, 15 * float32(math.Sin(angle)) * 0.86, baseH}
			dot(p, 1.1, body.Crease)
			dot(p, 0.5, ink.Edge)
		}
	})
	solid(round(baseH, baseR-2, 1), body, nil)

	// The switch on the near side: a recessed track and a slider.
	sw := near.Scale(13)
	plate := func(w, d, z float32) []iso.Vec {
		out := []iso.Vec{sw.Add(right.Scale(-w)).Add(near.Scale(-d)), sw.Add(right.Scale(w)).Add(near.Scale(-d)), sw.Add(right.Scale(w)).Add(near.Scale(d)), sw.Add(right.Scale(-w)).Add(near.Scale(d))}
		for i := range out {
			out[i][2] = z
		}
		return out
	}
	solid(iso.Prism{Ring: plate(3.6, 1.9, baseH+1), Up: iso.Vec{0, 0, 0.6}}, body, func() {
		k.Fill(iso.Poly(cam, plate(2.6, 0.7, baseH+1.6), true), body.Shadow)
	})
	slider := sw.Add(right.Scale(at.on*2.6 - 1.3))
	knob := []iso.Vec{slider.Add(right.Scale(-1.1)).Add(near.Scale(-1)), slider.Add(right.Scale(1.1)).Add(near.Scale(-1)), slider.Add(right.Scale(1.1)).Add(near.Scale(1)), slider.Add(right.Scale(-1.1)).Add(near.Scale(1))}
	for i := range knob {
		knob[i][2] = baseH + 1.6
	}
	solid(iso.Prism{Ring: knob, Up: iso.Vec{0, 0, 1.2}}, body, nil)

	// The stem through a knurled collar.
	solid(round(baseH+1, stemR, capsuleZ0-baseH-1), body, nil)
	solid(round(collarZ, 3.6, 4), body, nil)
	for i := -6; i <= 6; i++ {
		angle := a + float64(i)*math.Pi/14
		p := iso.Vec{3.6 * float32(math.Cos(angle)), 3.6 * float32(math.Sin(angle)), collarZ + 0.6}
		line([]iso.Vec{p, p.Add(iso.Vec{0, 0, 2.8})}, body.Crease)
	}

	// The rings of sound: their far halves behind the capsule, their near
	// halves in front.
	soundRings(k, cam, ink, at, a, false)

	// The capsule: a lower shell, the band, the grille, and a bevelled
	// top with its mesh.
	solid(round(capsuleZ0, capsuleR-1.2, 2), body, nil)
	solid(round(capsuleZ0+2, capsuleR, bandZ-capsuleZ0-2), body, nil)
	solid(round(bandZ, capsuleR+0.3, bandH), body.Mix(ink.Lit, at.on), nil)
	grille := float32(bandZ + bandH)
	solid(round(grille, capsuleR, capsuleZ1-grille-1), body, nil)
	for z := grille + 2; z < capsuleZ1-2; z += 2.2 {
		line(arc(capsuleR, z, a-math.Pi/2, a+math.Pi/2), body.Crease)
	}
	solid(round(capsuleZ1-1, capsuleR-1, 1), body, func() {
		for _, o := range [][2]float32{{0, 0}, {3, 0}, {-3, 0}, {0, 3}, {0, -3}, {2.2, 2.2}, {-2.2, 2.2}, {2.2, -2.2}, {-2.2, -2.2}, {5.2, 0}, {-5.2, 0}, {0, 5.2}, {0, -5.2}} {
			dot(iso.Vec{o[0], o[1], capsuleZ1}, 0.45, ink.Edge)
		}
	})
	soundRings(k, cam, ink, at, a, true)
}

// arc is the circle of radius r at height z from angle a0 to a1.
func arc(r, z float32, a0, a1 float64) []iso.Vec {
	var out []iso.Vec
	for s := 0; s <= 32; s++ {
		angle := a0 + float64(s)/32*(a1-a0)
		out = append(out, iso.Vec{r * float32(math.Cos(angle)), r * float32(math.Sin(angle)), z})
	}
	return out
}

// soundRings draws the near or far halves of the rings out: each spreads
// from the band, easing out, as a dashed circle that fades as it goes.
func soundRings(k *iso.Canvas, cam iso.Camera, ink iso.Ink, at micPose, a float64, nearHalf bool) {
	z := float32(bandZ + bandH/2)
	from, to := a+math.Pi/2, a+3*math.Pi/2
	if nearHalf {
		from, to = a-math.Pi/2, a+math.Pi/2
	}
	for _, u := range at.rings {
		if u < 0 {
			continue
		}
		r := capsuleR + 1 + ringReach*(1-(1-u)*(1-u))
		alpha := min(1, u*6) * (1 - u)
		var dashes iso.Shape
		for _, dash := range dashed(arc(r, z, from, to), 2.6, 2) {
			dashes = append(dashes, iso.Poly(cam, dash, false)...)
		}
		k.Stroke(dashes, 1, ink.Edge.Alpha(alpha))
	}
}

// dashed cuts the polyline path into dashes on world units long, off
// apart, starting on a dash. A dash turns the corners it reaches.
func dashed(path []iso.Vec, on, off float32) [][]iso.Vec {
	var out [][]iso.Vec
	var dash []iso.Vec
	phase := float32(0) // distance into the current dash and gap
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		d := b.Sub(a)
		l := float32(math.Sqrt(float64(d.Dot(d))))
		for s := float32(0); s < l-1e-4; {
			if phase < on {
				step := min(l-s, on-phase)
				if dash == nil {
					dash = []iso.Vec{a.Lerp(b, s/l)}
				}
				s, phase = s+step, phase+step
				dash = append(dash, a.Lerp(b, s/l))
				if phase >= on {
					out, dash = append(out, dash), nil
				}
			} else {
				step := min(l-s, on+off-phase)
				s, phase = s+step, phase+step
				if phase >= on+off {
					phase = 0
				}
			}
		}
	}
	if len(dash) > 1 {
		out = append(out, dash)
	}
	return out
}
