package usage

import (
	"image"
	"math"
	"time"

	"arion.local/desktop/ui/shared/iso"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The first-run figure: a desk microphone wired to a letter tray of paper,
// drawn live with the Anatomy kit's rules (ui/shared/iso). A dashed sheet
// waits over the stack for the first dictation. In a loop the microphone's
// band lights, the sheet settles onto the stack as paper, the stack gives
// a little, lines of text are typed across it, and the sheet lifts away
// as a dashed outline again: nothing has been kept yet.
//
// World units are about a millimetre: x and y on the floor, z up.

// The camera, as the kit's default but turned to show the tray's front.
const (
	trayAzimuth   = 40
	trayElevation = 30
)

// The tray and its paper.
const (
	trayW, trayD, trayH = 120, 90, 6 // base plate
	wallH, wallT        = 14, 4      // the back and left walls
	lipH, lipT          = 4, 4       // the front and right lips
	sheetX, sheetY      = 14, 12
	sheetW, sheetD      = 92, 68
	sheetT              = 2.2
	sheets              = 4
	hover               = 18 // how high the dashed sheet waits
)

// The microphone, standing to the tray's right.
var micAt = iso.Vec{156, 22, 0}

const (
	micBaseR, micBaseH = 20, 5
	stemR              = 2.2
	capsuleR           = 9
	capsuleZ0          = 46
	capsuleZ1          = 70
	bandZ, bandH       = 54, 2.4
)

// trayLoop is one cycle of the figure.
const trayLoop = 4200 * time.Millisecond

// trayPose is the figure at one moment.
type trayPose struct {
	// lift is how high the top sheet is over the stack, 0 to 1 of hover.
	lift float32
	// paper is how solid the top sheet is: 0 a dashed outline, 1 paper.
	paper float32
	// typed is how much of its text is written, 0 to 1.
	typed float32
	// live is how lit the microphone's band is.
	live float32
	// give is how far the stack is pressed down, 0 to 1.
	give float32
}

// trayAt is the pose at t into the loop. The still pose, under reduced
// motion and at t = 0, is the dashed sheet waiting.
func trayAt(t time.Duration) trayPose {
	s := float32(t%trayLoop) / float32(time.Second)
	ramp := func(a, b float32) float32 { return ease(clamp((s - a) / (b - a))) }
	p := trayPose{lift: 1}
	// The band lights as the sheet comes down and goes out once the text
	// is written.
	p.live = ramp(0.5, 0.9) - ramp(2.7, 3.1)
	p.lift = 1 - ramp(0.9, 1.7) + ramp(3.4, 4.2)
	p.paper = ramp(1.2, 1.7) - ramp(3.4, 3.9)
	p.typed = ramp(1.8, 2.7) - ramp(3.4, 3.8)
	// The stack gives as the sheet lands and springs back without bounce.
	if s > 1.55 && s < 2.25 {
		u := (s - 1.55) / 0.7
		p.give = float32(math.Sin(float64(u)*math.Pi)) * (1 - u)
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

// trayFrame is everything one picture depends on, so the same frame is
// never drawn twice.
type trayFrame struct {
	ink  iso.Ink
	pose trayPose
}

// trayPicture keeps the figure's bitmap. Two images take turns, as the
// GPU may still read the one last shown.
type trayPicture struct {
	canvas iso.Canvas
	images [2]*image.RGBA
	turn   int
	bitmap *ui.Bitmap
	last   trayFrame
	size   image.Point
}

// tray is the one figure the overview shows.
var tray trayPicture

// Tray fills its box with the figure, labeled for assistive technology.
// It loops while drawn and holds the waiting sheet under reduced motion.
func Tray(c *ui.Context, tokens theme.Tokens, label string) ui.Element {
	still := c.Preferences().ReduceMotion
	ink := trayInk(tokens)
	return ui.Box(c).Role(ui.RoleImage).Label(label).Draw(func(p *ui.Painter, r ui.Rect) {
		pose := trayAt(0)
		if !still {
			// Repaint only: the loop is a pure function of the clock.
			p.AnimationFrame()
			pose = trayAt(time.Duration(p.Now().UnixNano()))
		}
		p.Image(tray.show(r, p.Scale(), trayFrame{ink, pose}), r, ui.FillBox)
	})
}

// trayInk is the kit's inks with outlines drawn nearer Text: the figure
// is small, and at one pixel a DIP the Faint outline fades into the card.
func trayInk(tokens theme.Tokens) iso.Ink {
	ink := iso.InkOf(tokens)
	ink.Edge = ink.Edge.Mix(tokens.Text, 0.3)
	ink.Body.Edge = ink.Edge
	return ink
}

// show returns the bitmap of f in a box r DIPs at scale pixels to a DIP,
// drawing it only if it changed.
func (v *trayPicture) show(r ui.Rect, scale float32, f trayFrame) *ui.Bitmap {
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
	drawTray(&v.canvas, view, f.ink, f.pose)
	v.canvas.End()
	if v.bitmap == nil {
		v.bitmap = ui.NewBitmap(img)
	} else {
		v.bitmap.Update(img)
	}
	v.last = f
	return v.bitmap
}

// trayBounds are the corners of the space the figure fills, hover
// included, so that the camera does not move with the sheet.
var trayBounds = []iso.Vec{
	{-4, -4, -2}, {trayW + 4, -4, -2}, {-4, trayD + 4, -2}, {micAt[0] + micBaseR, micAt[1] + micBaseR, -2},
	{micAt[0] + micBaseR, micAt[1] - micBaseR, -2}, {micAt[0] - micBaseR, micAt[1] + micBaseR, -2},
	{0, 0, trayH + wallH}, {micAt[0], micAt[1], capsuleZ1 + 1}, {sheetX, sheetY + sheetD, trayH + sheets*sheetT + hover + sheetT},
}

// fit returns the camera that shows trayBounds within view, pad DIPs in,
// at the left: an empty state reads from its left edge.
func fit(view ui.Rect, pad float32) iso.Camera {
	cam := iso.NewCamera(trayAzimuth, trayElevation, 1, iso.Vec{}, view)
	x0, y0, x1, y1 := float32(math.MaxFloat32), float32(math.MaxFloat32), -float32(math.MaxFloat32), -float32(math.MaxFloat32)
	for _, p := range trayBounds {
		x, y := cam.Project(p)
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	scale := min((view.W-2*pad)/(x1-x0), (view.H-2*pad)/(y1-y0))
	cam = iso.NewCamera(trayAzimuth, trayElevation, scale, iso.Vec{}, view)
	x0, y0 = cam.Project(trayBounds[0])
	y1 = y0
	for _, p := range trayBounds {
		x, y := cam.Project(p)
		x0, y0, y1 = min(x0, x), min(y0, y), max(y1, y)
	}
	cam.Center[0] += view.X + pad - x0
	cam.Center[1] += view.Y + (view.H-(y1-y0))/2 - y0
	return cam
}

// drawTray paints the figure back to front, bottom to top.
func drawTray(k *iso.Canvas, view ui.Rect, ink iso.Ink, at trayPose) {
	cam := fit(view, 2)
	width := float32(1)
	body := ink.Body
	foot := iso.Palette{Top: ink.Body.Shadow, Shadow: ink.Body.Shadow, Lit: ink.Body.Shadow, Edge: ink.Edge, Crease: ink.Edge}
	solid := func(pr iso.Prism, pal iso.Palette, top func()) { pr.Draw(k, cam, pal, width, top) }
	line := func(points []iso.Vec, color ui.Color) { k.Stroke(iso.Poly(cam, points, false), width, color) }
	dot := func(p iso.Vec, r float32, color ui.Color) {
		k.Fill(iso.Poly(cam, iso.On(iso.Circle(p[0], p[1], r, 12), iso.Floor(p[2])), true), color)
	}
	slab := func(x, y, w, d, r, z, h float32) iso.Prism {
		return iso.Prism{Ring: iso.On(iso.RoundedRect(x, y, w, d, r, 4), iso.Floor(z)), Up: iso.Vec{0, 0, h}}
	}
	round := func(c iso.Vec, r, h float32) iso.Prism {
		return iso.Prism{Ring: iso.On(iso.Circle(c[0], c[1], r, 40), iso.Floor(c[2])), Up: iso.Vec{0, 0, h}}
	}

	// The halo under the tray and the microphone.
	k.Fill(iso.Poly(cam, iso.On(iso.RoundedRect(-6, -6, trayW+12, trayD+12, 12, 6), iso.Floor(-2)), true), ink.Halo)
	k.Fill(iso.Poly(cam, iso.On(iso.Circle(micAt[0], micAt[1], micBaseR+5, 40), iso.Floor(-2)), true), ink.Halo)

	// The tray: feet, a base plate with a label plate on its front, two
	// walls at the back, the stack, then the lips in front of it.
	for _, f := range [][2]float32{{9, 9}, {trayW - 9, 9}, {9, trayD - 9}, {trayW - 9, trayD - 9}} {
		solid(round(iso.Vec{f[0], f[1], -2}, 4, 2), foot, nil)
	}
	solid(slab(0, 0, trayW, trayD, 5, 0, trayH), body, func() {
		// Screws at the plate's corners, inside the walls and lips.
		for _, s := range [][2]float32{{trayW - 8, 8}, {8, trayD - 8}, {trayW - 8, trayD - 8}} {
			dot(iso.Vec{s[0], s[1], trayH}, 1.4, ink.Body.Crease)
			dot(iso.Vec{s[0], s[1], trayH}, 0.6, ink.Edge)
		}
	})
	front := func(u, v float32) iso.Vec { return iso.Vec{u, trayD, v} }
	plate := iso.On(iso.RoundedRect(44, 1.2, 32, 3.6, 0.8, 2), front)
	k.Fill(iso.Poly(cam, plate, true), ink.Top)
	k.Stroke(iso.Outline(cam, plate), width, ink.Edge)
	for i, w := range []float32{20, 14} {
		v := float32(2.5) + float32(i)*1.2
		line([]iso.Vec{front(48, v), front(48+w, v)}, ink.Body.Crease)
	}
	solid(slab(0, 0, trayW, wallT, 1.5, trayH, wallH), body, func() {
		line([]iso.Vec{{3, wallT / 2, trayH + wallH}, {trayW - 3, wallT / 2, trayH + wallH}}, ink.Body.Crease)
	})
	solid(slab(0, wallT, wallT, trayD-wallT, 1.5, trayH, wallH), body, func() {
		line([]iso.Vec{{wallT / 2, wallT + 1, trayH + wallH}, {wallT / 2, trayD - 3, trayH + wallH}}, ink.Body.Crease)
	})

	// The stack: each sheet a little off square, pressed down as the new
	// one lands.
	t := sheetT * (1 - 0.18*at.give)
	z := float32(trayH)
	for _, o := range [sheets][2]float32{{1, -1}, {-1.2, 0.6}, {0.6, 1}, {-0.4, -0.3}} {
		solid(slab(sheetX+o[0], sheetY+o[1], sheetW, sheetD, 1, z, t), body, nil)
		z += t
	}
	sheet(k, cam, ink, at, z, width)

	// The lips in front of the stack, the right one ruled every 5 with a
	// longer tick every 25.
	solid(slab(lipT+2, trayD-lipT, trayW-lipT-4, lipT, 1, trayH, lipH), body, nil)
	solid(slab(trayW-lipT, wallT+2, lipT, trayD-wallT-lipT-4, 1, trayH, lipH), body, func() {
		for y := float32(wallT + 5); y < trayD-lipT-4; y += 5 {
			l := float32(1.4)
			color := ink.Body.Crease
			if int(y-wallT)%25 == 0 {
				l, color = 2.6, ink.Edge
			}
			line([]iso.Vec{{trayW, y, trayH + lipH}, {trayW - l, y, trayH + lipH}}, color)
		}
	})

	// The cable runs from the microphone's base along the floor into a
	// socket on the tray's right side.
	socket := slab(trayW, 52, 3, 10, 0.8, 0.6, 4.4)
	solid(socket, body, nil)
	cable := []iso.Vec{{trayW + 3, 57, 2.8}, {trayW + 8, 57, 1.4}, {trayW + 13, 52, 1.4}, {trayW + 13, 36, 1.4}, {trayW + 18, 31, 1.4}, {micAt[0] - micBaseR + 6, micAt[1] + 4, 1.4}}
	k.Stroke(iso.Poly(cam, cable, false), 3, ink.Edge)
	k.Stroke(iso.Poly(cam, cable, false), 1.4, ink.Body.Shadow)
	dot(iso.Vec{trayW + 3.2, 57, 2.8}, 0.9, ink.Edge)

	microphone(k, cam, ink, at, width, solid, line, dot, round)
}

// sheet draws the top sheet at height z over the stack: a dashed outline
// while it waits, paper with lines of text once it has landed.
func sheet(k *iso.Canvas, cam iso.Camera, ink iso.Ink, at trayPose, z, width float32) {
	lift := at.lift * hover
	base := z + lift
	plan := iso.RoundedRect(sheetX, sheetY, sheetW, sheetD, 1, 4)
	if at.paper > 0 {
		pr := iso.Prism{Ring: iso.On(plan, iso.Floor(base)), Up: iso.Vec{0, 0, sheetT}}
		pr.Draw(k, cam, ink.Body.Alpha(at.paper), width, nil)
	}
	top := base + sheetT
	if at.paper < 1 {
		// The dashes run round the outline at a fixed pitch, so they do
		// not crawl as the sheet moves.
		ring := iso.On(plan, iso.Floor(top))
		var dashes iso.Shape
		for _, dash := range dashed(append(ring, ring[0]), 3.2, 2.4) {
			dashes = append(dashes, iso.Poly(cam, dash, false)...)
		}
		k.Stroke(dashes, width, ink.Edge.Alpha(1-at.paper))
	}
	if at.typed > 0 {
		// Lines of text, typed left to right, line after line.
		lines := []float32{70, 78, 62, 74, 40}
		for i, w := range lines {
			share := clamp(at.typed*float32(len(lines)) - float32(i))
			if share == 0 {
				break
			}
			y := float32(sheetY + 10 + i*10)
			x := float32(sheetX + 10)
			k.Stroke(iso.Poly(cam, []iso.Vec{{x, y, top}, {x + w*share, y, top}}, false), width*1.4, ink.Edge.Alpha(at.paper))
		}
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
		if l == 0 {
			continue
		}
		for s := float32(0); s < l; {
			if phase < on {
				step := min(l-s, on-phase)
				if dash == nil {
					dash = []iso.Vec{a.Lerp(b, s/l)}
				}
				s += step
				phase += step
				dash = append(dash, a.Lerp(b, s/l))
				if phase >= on {
					out, dash = append(out, dash), nil
				}
			} else {
				step := min(l-s, on+off-phase)
				s += step
				phase += step
				if phase >= on+off {
					phase = 0
				}
			}
			if s >= l-1e-4 {
				break
			}
		}
	}
	if len(dash) > 1 {
		out = append(out, dash)
	}
	return out
}
