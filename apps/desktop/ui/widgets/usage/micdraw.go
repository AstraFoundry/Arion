package usage

import (
	"math"
	"slices"

	"arion.local/desktop/ui/shared/iso"
	"github.com/egoist/mygo/ui"
)

// The microphone's measures.
const (
	baseR, baseH = 20, 5
	stemR        = 2.2
	collarZ      = 22
	// The yoke: a crossbar under the capsule and two arms up to the
	// pivots, armOut from the axis, with a knurled thumb screw outside each.
	pivotZ        = 56
	barZ, barH    = 40, 2.4
	armOut        = capsuleR + 1.8
	armT, armW    = 1.6, 3.4
	knobR, knobH  = 3, 2.2
	capsuleR      = 9
	capsuleBottom = -9 // along the capsule's axis from the pivot
	capsuleTop    = 15
	bandAt, bandH = -1.5, 2.4
)

// head is the yoke and capsule in one pose: facing is the floor direction
// the microphone faces, side the pivot axis, axis the capsule's.
type head struct {
	pivot, facing, side, axis iso.Vec
	// across is the capsule's second cross direction, axis × side.
	across iso.Vec
}

func headOf(at micPose) head {
	yaw := float64(micAzimuth+at.yaw) * math.Pi / 180
	tilt := float64(at.tilt) * math.Pi / 180
	facing := iso.Vec{float32(math.Cos(yaw)), float32(math.Sin(yaw)), 0}
	side := iso.Vec{-facing[1], facing[0], 0}
	axis := iso.Vec{0, 0, float32(math.Cos(tilt))}.Add(facing.Scale(float32(math.Sin(tilt))))
	return head{pivot: iso.Vec{0, 0, pivotZ}, facing: facing, side: side, axis: axis, across: axis.Cross(side)}
}

// at is the point off along the capsule's axis from the pivot.
func (h head) at(off float32) iso.Vec { return h.pivot.Add(h.axis.Scale(off)) }

// ring is the capsule's cross-section of radius r off along its axis.
func (h head) ring(off, r float32, steps int) []iso.Vec {
	c := h.at(off)
	out := make([]iso.Vec, steps)
	for i := range out {
		a := float64(i) / float64(steps) * 2 * math.Pi
		out[i] = c.Add(h.side.Scale(r * float32(math.Cos(a)))).Add(h.across.Scale(r * float32(math.Sin(a))))
	}
	return out
}

// barrel is a length of the capsule from off, len long, of radius r.
func (h head) barrel(off, length, r float32) iso.Prism {
	return iso.Prism{Ring: h.ring(off, r, 40), Up: h.axis.Scale(length)}
}

// box is a block centered on c with half-extents a and b across its
// base and up as its height.
func box(c, a, b, up iso.Vec) iso.Prism {
	base := c.Sub(up.Scale(0.5))
	return iso.Prism{Ring: []iso.Vec{base.Sub(a).Sub(b), base.Add(a).Sub(b), base.Add(a).Add(b), base.Sub(a).Add(b)}, Up: up}
}

// drawMic paints the figure back to front, bottom to top.
func drawMic(k *iso.Canvas, view ui.Rect, ink iso.Ink, at micPose) {
	cam := fit(view, 2)
	const width = 1
	body := ink.Body
	rubber := iso.Palette{Top: body.Shadow, Shadow: body.Shadow, Lit: body.Shadow, Edge: ink.Edge, Crease: ink.Edge}
	solid := func(pr iso.Prism, pal iso.Palette, top func()) { pr.Draw(k, cam, pal, width, top) }
	line := func(points []iso.Vec, color ui.Color) { k.Stroke(iso.Poly(cam, points, false), width, color) }
	disc := func(c, u, v iso.Vec, r float32, color ui.Color) {
		points := make([]iso.Vec, 12)
		for i := range points {
			a := float64(i) / 12 * 2 * math.Pi
			points[i] = c.Add(u.Scale(r * float32(math.Cos(a)))).Add(v.Scale(r * float32(math.Sin(a))))
		}
		k.Fill(iso.Poly(cam, points, true), color)
	}
	x, y := iso.Vec{1, 0, 0}, iso.Vec{0, 1, 0}
	round := func(z, r, h float32) iso.Prism {
		return iso.Prism{Ring: iso.On(iso.Circle(0, 0, r, 48), iso.Floor(z)), Up: iso.Vec{0, 0, h}}
	}
	h := headOf(at)
	up := iso.Vec{0, 0, 1}

	// The halo and the base: a rubber ring, the weighted disc with a
	// groove and three screws, which turn with it, and a bevel.
	k.Fill(iso.Poly(cam, iso.On(iso.Circle(0, 0, baseR+5, 48), iso.Floor(-2)), true), ink.Halo)
	solid(round(-1.5, baseR-1, 1.5), rubber, nil)
	solid(round(0, baseR, baseH), body, func() {
		k.Stroke(iso.Outline(cam, iso.On(iso.Circle(0, 0, baseR-3.5, 48), iso.Floor(baseH))), width, body.Crease)
		for i := 0; i < 3; i++ {
			yaw := float64(micAzimuth+at.yaw)*math.Pi/180 + math.Pi/3 + float64(i)*2*math.Pi/3
			p := iso.Vec{13 * float32(math.Cos(yaw)), 13 * float32(math.Sin(yaw)), baseH}
			disc(p, x, y, 1.1, body.Crease)
			disc(p, x, y, 0.5, ink.Edge)
		}
	})
	solid(round(baseH, baseR-2, 1), body, nil)

	// The switch on the front of the base: a recessed track and a slider.
	sw := h.facing.Scale(12).Add(iso.Vec{0, 0, baseH + 1.3})
	solid(box(sw, h.side.Scale(3.6), h.facing.Scale(1.9), up.Scale(0.6)), body, func() {
		k.Fill(iso.Poly(cam, box(sw, h.side.Scale(2.6), h.facing.Scale(0.7), up.Scale(0.6)).Ring, true), body.Shadow)
	})
	slider := sw.Add(h.side.Scale(at.on*2.6 - 1.3)).Add(up.Scale(0.9))
	solid(box(slider, h.side.Scale(1.1), h.facing.Scale(1), up.Scale(1.2)), body, nil)

	// The stem through a knurled collar, up to the yoke's crossbar.
	solid(round(baseH+1, stemR, barZ-barH/2-baseH-1), body, nil)
	solid(round(collarZ, 3.6, 4), body, nil)
	a := float64(micAzimuth) * math.Pi / 180
	for i := -6; i <= 6; i++ {
		angle := a + float64(i)*math.Pi/14
		p := iso.Vec{3.6 * float32(math.Cos(angle)), 3.6 * float32(math.Sin(angle)), collarZ + 0.6}
		line([]iso.Vec{p, p.Add(up.Scale(2.8))}, body.Crease)
	}
	solid(box(iso.Vec{0, 0, barZ}, h.side.Scale(armOut+armT/2), h.facing.Scale(armW/2), up.Scale(barH)), body, nil)

	// The arms with their thumb screws and the capsule, far to near.
	type part struct {
		depth float32
		draw  func()
	}
	parts := []part{{cam.Depth(h.pivot), func() { capsule(k, cam, ink, h, at, solid, disc) }}}
	for _, s := range []float32{-1, 1} {
		out := h.side.Scale(s)
		armAt := iso.Vec{0, 0, (barZ + barH/2 + pivotZ + armW/2) / 2}.Add(out.Scale(armOut))
		arm := box(armAt, out.Scale(armT/2), h.facing.Scale(armW/2), up.Scale(pivotZ+armW/2-barZ-barH/2))
		knob := iso.Prism{Ring: iso.On(iso.Circle(0, 0, knobR, 28), func(u, v float32) iso.Vec {
			return h.pivot.Add(out.Scale(armOut + armT/2)).Add(h.facing.Scale(u)).Add(up.Scale(v))
		}), Up: out.Scale(knobH)}
		knurl := func() {
			for i := 0; i < 16; i++ {
				angle := float64(i) / 16 * 2 * math.Pi
				p := h.pivot.Add(out.Scale(armOut + armT/2)).Add(h.facing.Scale(knobR * float32(math.Cos(angle)))).Add(up.Scale(knobR * float32(math.Sin(angle))))
				n := h.facing.Scale(float32(math.Cos(angle))).Add(up.Scale(float32(math.Sin(angle))))
				if cam.Sees(n) {
					line([]iso.Vec{p.Add(out.Scale(0.3)), p.Add(out.Scale(knobH - 0.3))}, body.Crease)
				}
			}
		}
		farKnob := !cam.Sees(out)
		parts = append(parts, part{cam.Depth(armAt), func() {
			if farKnob {
				solid(knob, body, nil)
				knurl()
			}
			solid(arm, body, nil)
			if !farKnob {
				solid(knob, body, nil)
				knurl()
			}
		}})
	}
	slices.SortFunc(parts, func(p, q part) int {
		switch {
		case p.depth < q.depth:
			return -1
		case p.depth > q.depth:
			return 1
		}
		return 0
	})
	for _, p := range parts {
		p.draw()
	}
}

// capsule draws the capsule along its axis, end on end from the one
// facing away: a lower shell, the body, the band that lights while it is
// on, the grille with its rings, and a bevelled top with its mesh.
func capsule(k *iso.Canvas, cam iso.Camera, ink iso.Ink, h head, at micPose,
	solid func(iso.Prism, iso.Palette, func()), disc func(c, u, v iso.Vec, r float32, color ui.Color)) {
	body := ink.Body
	grille := float32(bandAt + bandH)
	rings := func() {
		for off := grille + 2; off < capsuleTop-2; off += 2.2 {
			ring := h.ring(off, capsuleR, 48)
			var run []iso.Vec
			flush := func() {
				if len(run) > 1 {
					k.Stroke(iso.Poly(cam, run, false), 1, body.Crease)
				}
				run = nil
			}
			for i := 0; i <= len(ring); i++ {
				p := ring[i%len(ring)]
				if cam.Sees(p.Sub(h.at(off))) {
					run = append(run, p)
				} else {
					flush()
				}
			}
			flush()
		}
	}
	mesh := func() {
		c := h.at(capsuleTop)
		for _, o := range [][2]float32{{0, 0}, {3, 0}, {-3, 0}, {0, 3}, {0, -3}, {2.2, 2.2}, {-2.2, 2.2}, {2.2, -2.2}, {-2.2, -2.2}, {5.2, 0}, {-5.2, 0}, {0, 5.2}, {0, -5.2}} {
			disc(c.Add(h.side.Scale(o[0])).Add(h.across.Scale(o[1])), h.side, h.across, 0.45, ink.Edge)
		}
	}
	pieces := []func(){
		func() { solid(h.barrel(capsuleBottom, 2, capsuleR-1.2), body, nil) },
		func() { solid(h.barrel(capsuleBottom+2, bandAt-capsuleBottom-2, capsuleR), body, nil) },
		func() { solid(h.barrel(bandAt, bandH, capsuleR+0.3), body.Mix(ink.Lit, at.on), nil) },
		func() { solid(h.barrel(grille, capsuleTop-1-grille, capsuleR), body, nil); rings() },
		func() { solid(h.barrel(capsuleTop-1, 1, capsuleR-1), body, mesh) },
	}
	if !cam.Sees(h.axis) {
		slices.Reverse(pieces)
	}
	for _, draw := range pieces {
		draw()
	}
}
