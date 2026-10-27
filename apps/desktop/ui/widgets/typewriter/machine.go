package typewriter

import (
	"math"

	"arion.local/desktop/ui/shared/iso"
	"github.com/egoist/mygo/ui"
)

// The machine, in world units of about a millimetre: x across, y from the
// back toward the typist, z up. The carriage, platen and paper slide along
// x as it types; the type bars strike at typeX.
const (
	feetH      = 6
	baseTop    = 18
	housingTop = 80
	deckBack   = 150 // where the keyboard's slope meets the housing
	deckFront  = 272
	deckHigh   = 74
	deckLow    = 30

	frameY0, frameY1, frameTop = 8, 40, 90
	platenY, platenZ, platenR  = 26, 106, 15
	knobR                      = 12

	paperX, paperW, paperH = 70, 160, 140 // on the carriage
	paperY, paperZ         = 34, 121.5    // the paper's foot
	tilt                   = 14           // degrees the paper leans back
	margin                 = 14
	em                     = 6   // the type's size
	lead                   = 8.4 // between lines
	baseline               = 10  // the typing line, above the paper's foot
	typeX                  = 150

	bailY, bailZ   = 46, 111
	ribbonY        = 41.5
	ribbonZ, ribbH = 115, 5
	spoolZ         = 112
)

// The mic stands on the desk to the machine's front left.
var micAt = [2]float32{-100, 236}

const micBase = 28

// rows are the keys' legends, back to front, in the Greek layout.
var rows = []string{"1234567890-", ";ςΕΡΤΥΘΙΟΠ", "ΑΣΔΦΓΗΞΚΛ΄", "ΖΧΨΩΒΝΜ,."}

type key struct {
	x, y   float32
	legend rune
}

var keys = func() []key {
	var out []key
	for r, row := range rows {
		legends := []rune(row)
		x0 := typeX - float32(len(legends)-1)/2*23 + float32(r)*4 - 6
		for i, l := range legends {
			out = append(out, key{x: x0 + float32(i)*23, y: 166 + float32(r)*25, legend: l})
		}
	}
	return out
}()

// keyOf is the key a character is typed with, or -1: punctuation the
// layout puts on a shifted key types on that key; a space on the bar.
func keyOf(r rune) int {
	switch r {
	case ' ':
		return -1
	case '·', '·':
		r = ';'
	case '(':
		r = '9'
	case ')':
		r = '0'
	case 'ς':
	default:
		r = base(r)
	}
	for i, k := range keys {
		if k.legend == r {
			return i
		}
	}
	return -1
}

// slope is the height of the keyboard's slope at y.
func slope(y float32) float32 {
	return deckHigh + (y-deckBack)*(deckLow-deckHigh)/(deckFront-deckBack)
}

var (
	sinTilt = float32(math.Sin(tilt * math.Pi / 180))
	cosTilt = float32(math.Cos(tilt * math.Pi / 180))
)

// onPaper places a point of the paper: u across from its left edge, v up
// from its foot, with the carriage moved by off.
func onPaper(off float32) func(u, v float32) iso.Vec {
	return func(u, v float32) iso.Vec { return iso.Vec{paperX + off + u, paperY - v*sinTilt, paperZ + v*cosTilt} }
}

// carriage is how far the carriage stands from its centre with the pen at
// pen ems: the character being typed lands at typeX.
func carriage(pen float32) float32 { return typeX - (paperX + margin + pen*em) }

// rod is a cylinder of radius r from a to b.
func rod(a, b iso.Vec, r float32, steps int) iso.Prism {
	d := b.Sub(a).Unit()
	u := d.Cross(iso.Vec{0, 0, 1})
	if u.Dot(u) < 1e-6 {
		u = iso.Vec{1, 0, 0}
	}
	u = u.Unit()
	w := d.Cross(u)
	ring := make([]iso.Vec, steps)
	for i := range ring {
		t := float64(i) / float64(steps) * 2 * math.Pi
		ring[i] = a.Add(u.Scale(r * float32(math.Cos(t)))).Add(w.Scale(r * float32(math.Sin(t))))
	}
	return iso.Prism{Ring: ring, Up: b.Sub(a)}
}

// post is a vertical cylinder of radius r standing on (x, y, z0) to z1.
func post(x, y, r, z0, z1 float32) iso.Prism {
	return iso.Prism{Ring: iso.On(iso.Circle(x, y, r, 24), iso.Floor(z0)), Up: iso.Vec{0, 0, z1 - z0}}
}

func slab(x, y, w, d, r, z0, z1 float32) iso.Prism {
	return iso.Prism{Ring: iso.On(iso.RoundedRect(x, y, w, d, r, 4), iso.Floor(z0)), Up: iso.Vec{0, 0, z1 - z0}}
}

// part is one solid, or a few drawn together, in a world box: painted
// back to front, after the part it rests on.
type part struct {
	lo, hi iso.Vec
	on     int
	draw   func()
}

func boxOf(points ...iso.Vec) (iso.Vec, iso.Vec) {
	lo, hi := points[0], points[0]
	for _, p := range points[1:] {
		for i := range 3 {
			lo[i], hi[i] = min(lo[i], p[i]), max(hi[i], p[i])
		}
	}
	return lo, hi
}

func prismBox(pr iso.Prism) (iso.Vec, iso.Vec) {
	points := append([]iso.Vec{}, pr.Ring...)
	for _, p := range pr.Ring {
		points = append(points, p.Add(pr.Up))
	}
	return boxOf(points...)
}

// order sorts parts back to front. Two parts that overlap on screen are
// ordered by a plane between their boxes when there is one, else by the
// part one rests on, else by the depth of their middles.
func order(cam iso.Camera, parts []part) []int {
	n := len(parts)
	toward := cam.Toward()
	type rect struct{ x0, y0, x1, y1 float32 }
	screens := make([]rect, n)
	depth := make([]float32, n)
	for i, p := range parts {
		r := rect{math.MaxFloat32, math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
		for c := range 8 {
			x, y := cam.Project(iso.Vec{pick(c&1, p.lo[0], p.hi[0]), pick(c&2, p.lo[1], p.hi[1]), pick(c&4, p.lo[2], p.hi[2])})
			r = rect{min(r.x0, x), min(r.y0, y), max(r.x1, x), max(r.y1, y)}
		}
		screens[i] = r
		depth[i] = cam.Depth(p.lo.Add(p.hi).Scale(0.5))
	}
	// before reports whether a is painted before b.
	before := func(a, b int) bool {
		pa, pb := parts[a], parts[b]
		if pb.on == a {
			return true
		}
		if pa.on == b {
			return false
		}
		const eps = 0.01
		for axis := range 3 {
			if math.Abs(float64(toward[axis])) < 1e-3 {
				continue
			}
			if pa.hi[axis] <= pb.lo[axis]+eps {
				return toward[axis] > 0
			}
			if pb.hi[axis] <= pa.lo[axis]+eps {
				return toward[axis] < 0
			}
		}
		return depth[a] < depth[b]
	}
	after := make([][]int, n)
	waiting := make([]int, n)
	for a := range n {
		for b := a + 1; b < n; b++ {
			sa, sb := screens[a], screens[b]
			if sa.x1 <= sb.x0 || sb.x1 <= sa.x0 || sa.y1 <= sb.y0 || sb.y1 <= sa.y0 {
				continue
			}
			first, second := a, b
			if !before(a, b) {
				first, second = b, a
			}
			after[first] = append(after[first], second)
			waiting[second]++
		}
	}
	// Kahn's sort, the deepest ready part first; a cycle gives way at its
	// deepest part.
	done := make([]bool, n)
	out := make([]int, 0, n)
	for len(out) < n {
		pick := -1
		for i := range n {
			if !done[i] && waiting[i] == 0 && (pick < 0 || depth[i] < depth[pick]) {
				pick = i
			}
		}
		if pick < 0 {
			for i := range n {
				if !done[i] && (pick < 0 || depth[i] < depth[pick]) {
					pick = i
				}
			}
		}
		done[pick] = true
		out = append(out, pick)
		for _, j := range after[pick] {
			waiting[j]--
		}
	}
	return out
}

func pick(bit int, lo, hi float32) float32 {
	if bit != 0 {
		return hi
	}
	return lo
}

// cameraAt is the camera for the arrival's progress in r.
func cameraAt(r ui.Rect, at pose) iso.Camera {
	// In front, the camera keeps the sheet whole: it glides with the
	// carriage along each line rather than stepping with every stroke.
	overview := iso.Vec{110, 150, 70}
	focus := onPaper(carriage(at.typed.glide))(paperW/2, baseline+36)
	fit := iso.ScaleFor(r.W-24, 470)
	close := iso.ScaleFor(r.W, frontSpan)
	scale := fit * float32(math.Pow(float64(close/fit), float64(at.zoom)))
	azimuth := lerp(overviewAzimuth, frontAzimuth, at.front) + 180*(1-at.turn)
	elevation := lerp(overviewElevation, frontElevation, at.front)
	return iso.NewCamera(azimuth, elevation, scale, overview.Lerp(focus, at.zoom), r)
}

func draw(k *iso.Canvas, r ui.Rect, ink colors, at pose) {
	cam := cameraAt(r, at)
	unit := cam.Unit()
	width := lerp(0.55, 0.9, at.zoom)
	typed := at.typed
	off := carriage(typed.pen)
	toward := cam.Toward()

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
	solid := func(pr iso.Prism, pal iso.Palette, top func()) { pr.Draw(k, cam, pal, width, top) }
	// A rod drawn as a line, w world units thick, with its outline.
	wire := func(points []iso.Vec, w float32, pal iso.Palette) {
		stroke([][]iso.Vec{points}, w*unit+2*width, pal.Edge)
		stroke([][]iso.Vec{points}, w*unit, pal.Lit)
	}
	// The visible half of a ring of radius r around a vertical axis.
	rim := func(x, y, r, z float32) []iso.Vec {
		facing := math.Atan2(float64(toward[1]), float64(toward[0]))
		out := make([]iso.Vec, 25)
		for i := range out {
			a := facing - math.Pi/2 + float64(i)/24*math.Pi
			out[i] = iso.Vec{x + r*float32(math.Cos(a)), y + r*float32(math.Sin(a)), z}
		}
		return out
	}

	var parts []part
	add := func(lo, hi iso.Vec, on int, draw func()) int {
		parts = append(parts, part{lo, hi, on, draw})
		return len(parts) - 1
	}
	addSolid := func(pr iso.Prism, pal iso.Palette, on int, top func()) int {
		lo, hi := prismBox(pr)
		return add(lo, hi, on, func() { solid(pr, pal, top) })
	}

	// Soft contact shadows under the machine and the mic.
	for i := 0; i < 2; i++ {
		grow := float32(i)*6 + 2
		k.Fill(iso.Poly(cam, iso.On(iso.RoundedRect(-grow, -grow, 300+2*grow, 280+2*grow, 12+grow, 6), iso.Floor(0)), true), ink.halo)
		k.Fill(iso.Poly(cam, iso.On(iso.Circle(micAt[0], micAt[1], micBase+grow, 36), iso.Floor(0)), true), ink.halo)
	}

	// Feet, base and the jack on its left side.
	for _, f := range [][2]float32{{26, 22}, {274, 22}, {26, 258}, {274, 258}} {
		addSolid(post(f[0], f[1], 9, 0, feetH), ink.rubber, -1, nil)
	}
	base := slab(0, 0, 300, 280, 12, feetH, baseTop)
	lo, hi := prismBox(base)
	baseID := add(lo, hi, -1, func() {
		solid(base, ink.body, nil)
		if cam.Sees(iso.Vec{-1, 0, 0}) {
			side := func(u, v float32) iso.Vec { return iso.Vec{0, u, v} }
			shape(iso.On(iso.RoundedRect(190, 8, 20, 8, 2, 3), side), ink.well, ink.body.Edge)
			for _, y := range []float32{186, 214} {
				k.Fill(iso.Poly(cam, iso.On(iso.Circle(y, 12, 0.8, 8), side), true), ink.body.Edge)
			}
		}
	})

	// The housing, with the type-bar basket on its top and a name plate
	// on each side.
	housing := slab(8, 4, 284, 146, 10, baseTop, housingTop)
	bars := len(keys)
	basket := iso.Vec{typeX, 146, housingTop}
	barAt := func(i int, r float32) iso.Vec {
		a := math.Pi + (float64(i)+0.5)/float64(bars)*math.Pi
		return iso.Vec{basket[0] + r*float32(math.Cos(a)), basket[1] + r*float32(math.Sin(a)), housingTop}
	}
	struck := -1
	if typed.since < strike && typed.count > 0 {
		l := text.lines[typed.line%len(text.lines)]
		struck = keyOf(l.strokes[typed.count-1].r)
	}
	lo, hi = prismBox(housing)
	housingID := add(lo, hi, baseID, func() {
		solid(housing, ink.body, func() {
			deck := iso.Floor(housingTop)
			var well [][2]float32
			for i := 0; i <= 32; i++ {
				a := math.Pi + float64(i)/32*math.Pi
				well = append(well, [2]float32{basket[0] + 62*float32(math.Cos(a)), basket[1] + 62*float32(math.Sin(a))})
			}
			shape(iso.On(well, deck), ink.well, ink.body.Crease)
			var rest [][]iso.Vec
			for i := range bars {
				if i != struck {
					rest = append(rest, []iso.Vec{barAt(i, 56), barAt(i, 24)})
				}
			}
			stroke(rest, width*1.1, ink.metal.Edge)
			for _, radius := range []float32{56, 24} {
				var arc []iso.Vec
				for i := 0; i <= 32; i++ {
					a := math.Pi + float64(i)/32*math.Pi
					arc = append(arc, iso.Vec{basket[0] + radius*float32(math.Cos(a)), basket[1] + radius*float32(math.Sin(a)), housingTop})
				}
				stroke([][]iso.Vec{arc}, width*1.4, ink.metal.Edge)
			}
		})
		for _, x := range []float32{8, 292} {
			normal := iso.Vec{x - 150, 0, 0}.Unit()
			if !cam.Sees(normal) {
				continue
			}
			side := func(u, v float32) iso.Vec { return iso.Vec{x + normal[0]*0.01, u, v} }
			shape(iso.On(iso.RoundedRect(40, 36, 74, 30, 3, 3), side), ink.metal.Top, ink.body.Edge)
			var rules [][]iso.Vec
			for _, v := range []float32{58, 51, 44} {
				rules = append(rules, []iso.Vec{side(48, v), side(lerp(106, 80, (58-v)/14), v)})
			}
			stroke(rules, width, ink.body.Crease)
			for _, c := range [][2]float32{{44, 40}, {110, 40}, {44, 62}, {110, 62}} {
				k.Fill(iso.Poly(cam, iso.On(iso.Circle(c[0], c[1], 0.9, 8), side), true), ink.body.Edge)
			}
		}
	})

	// The keyboard's slope, its keys and the space bar.
	deck := iso.Prism{Ring: []iso.Vec{{8, deckBack, baseTop}, {8, deckBack, deckHigh}, {8, deckFront, deckLow}, {8, deckFront, baseTop}}, Up: iso.Vec{284, 0, 0}}
	lo, hi = prismBox(deck)
	deckID := add(lo, hi, baseID, func() {
		solid(deck, ink.body, nil)
		slopeNormal := iso.Vec{0, deckHigh - deckLow, deckFront - deckBack}.Unit()
		if cam.Sees(slopeNormal) {
			y := float32(deckFront - 6)
			stroke([][]iso.Vec{{{8, y, slope(y)}, {292, y, slope(y)}}}, width, ink.body.Crease)
		}
	})
	for i, kk := range keys {
		sink := float32(0)
		if i == struck {
			sink = 3 * float32(math.Sin(math.Pi*float64(typed.since)/float64(strike)))
		}
		floor := slope(kk.y)
		capZ := floor + 9 - sink
		stem := post(kk.x, kk.y, 1.4, floor-1, capZ)
		cap := post(kk.x, kk.y, 7.5, capZ, capZ+2.6)
		top := capZ + 2.6
		legend := kk.legend
		lo, hi := prismBox(cap)
		lo[2] = floor
		add(lo, hi, deckID, func() {
			solid(stem, ink.metal, nil)
			solid(cap, ink.metal, func() {
				k.Fill(iso.Poly(cam, iso.On(iso.Circle(kk.x, kk.y, 6, 20), iso.Floor(top)), true), ink.paper)
				k.Stroke(iso.Outline(cam, iso.On(iso.Circle(kk.x, kk.y, 6, 20), iso.Floor(top))), width*0.8, ink.metal.Crease)
				g := serif.glyph(legend)
				size := float32(6)
				var s iso.Shape
				for _, c := range g.contours {
					s = append(s, iso.Poly(cam, iso.On(c, func(u, v float32) iso.Vec {
						return iso.Vec{kk.x + (u-g.advance/2)*size, kk.y - (v-0.34)*size, top}
					}), true)...)
				}
				k.Fill(s, ink.legend)
			})
		})
	}
	spaceSink := float32(0)
	if typed.since < strike && typed.count > 0 && struck < 0 {
		spaceSink = 2 * float32(math.Sin(math.Pi*float64(typed.since)/float64(strike)))
	}
	spaceZ := slope(262) + 8 - spaceSink
	space := slab(85, 258, 130, 9, 4, spaceZ, spaceZ+2.6)
	lo, hi = prismBox(space)
	lo[2] = slope(266)
	add(lo, hi, deckID, func() {
		for _, x := range []float32{100, 200} {
			solid(post(x, 262.5, 1.4, slope(262)-1, spaceZ), ink.metal, nil)
		}
		solid(space, ink.metal, nil)
	})

	// The carriage: its frame with the margin scale, the platen between
	// its knobs, the return lever and the paper bail.
	frame := slab(off, frameY0, 300, frameY1-frameY0, 4, housingTop, frameTop)
	lo, hi = prismBox(frame)
	add(lo, hi, housingID, func() {
		solid(frame, ink.metal, nil)
		if cam.Sees(iso.Vec{0, 1, 0}) {
			face := func(u, v float32) iso.Vec { return iso.Vec{u, frameY1 + 0.01, v} }
			shape(iso.On(iso.RoundedRect(off+8, 82, 284, 6, 1, 2), face), ink.paper, ink.metal.Edge)
			var ticks, majors [][]iso.Vec
			for i := 0; i <= 56; i++ {
				x := off + 10 + float32(i)*5
				if i%5 == 0 {
					majors = append(majors, []iso.Vec{face(x, 88), face(x, 84.4)})
				} else {
					ticks = append(ticks, []iso.Vec{face(x, 88), face(x, 86.2)})
				}
			}
			stroke(ticks, width*0.7, ink.metal.Crease)
			stroke(majors, width*0.9, ink.metal.Edge)
		}
	})
	turned := typed.feed * lead / platenR
	knurl := func(x0, x1, r float32) func() {
		return func() {
			var lines [][]iso.Vec
			for i := range 18 {
				a := float64(turned) + float64(i)/18*2*math.Pi
				n := iso.Vec{0, float32(math.Cos(a)), float32(math.Sin(a))}
				if n.Dot(toward) < 0.08 {
					continue
				}
				p := iso.Vec{0, platenY + r*n[1], platenZ + r*n[2]}
				lines = append(lines, []iso.Vec{{x0, p[1], p[2]}, {x1, p[1], p[2]}})
			}
			stroke(lines, width*0.8, ink.rubber.Crease)
		}
	}
	axis := func(x float32) iso.Vec { return iso.Vec{x, platenY, platenZ} }
	for _, end := range [][2]float32{{6, 20}, {280, 294}} {
		knob := rod(axis(off+end[0]), axis(off+end[1]), knobR, 24)
		lines := knurl(off+end[0]+1.5, off+end[1]-1.5, knobR)
		lo, hi := prismBox(knob)
		add(lo, hi, -1, func() { solid(knob, ink.rubber, nil); lines() })
	}
	for _, end := range [][2]float32{{20, 22}, {278, 280}} {
		flange := rod(axis(off+end[0]), axis(off+end[1]), platenR+1.5, 28)
		addSolid(flange, ink.metal, -1, nil)
	}
	platen := rod(axis(off+22), axis(off+278), platenR, 32)
	addSolid(platen, ink.rubber, -1, nil)
	lever := []iso.Vec{{off + 2, 24, frameTop}, {off + 2, 24, 126}, {off - 1, 34, 136}, {off - 3, 68, 142}}
	lo, hi = boxOf(lever...)
	lo, hi = lo.Sub(iso.Vec{3, 3, 0}), hi.Add(iso.Vec{3, 3, 4})
	add(lo, hi, -1, func() {
		wire(lever, 2.6, ink.metal)
		end := lever[len(lever)-1]
		x, y := cam.Project(end)
		ball := iso.Shape{iso.Circle(x, y, 3.6*unit, 20)}
		k.Fill(ball, ink.rubber.Lit)
		k.Stroke(iso.Shape{append(iso.Circle(x, y, 3.6*unit, 20), [2]float32{x + 3.6*unit, y})}, width, ink.rubber.Edge)
	})
	bail := rod(iso.Vec{off + 24, bailY, bailZ}, iso.Vec{off + 276, bailY, bailZ}, 1.4, 10)
	addSolid(bail, ink.metal, -1, nil)
	for _, x := range []float32{100, 190} {
		roller := rod(iso.Vec{off + x, bailY, bailZ}, iso.Vec{off + x + 10, bailY, bailZ}, 3.4, 18)
		addSolid(roller, ink.rubber, -1, nil)
	}

	// The paper, and on it the story so far.
	place := onPaper(off)
	corners := iso.On([][2]float32{{0, 0}, {paperW, 0}, {paperW, paperH}, {0, paperH}}, place)
	lo, hi = boxOf(corners...)
	add(lo, hi, -1, func() {
		front := cam.Sees(iso.Vec{0, cosTilt, sinTilt})
		if !front {
			shape(corners, ink.back, ink.body.Edge)
			return
		}
		shape(corners, ink.paper, ink.body.Edge)
		var tones [3]iso.Shape
		for back := 0; float32(back)*lead < paperH+lead; back++ {
			g := typed.line - back
			if g < 0 {
				break
			}
			l := text.lines[g%len(text.lines)]
			count := len(l.strokes)
			if back == 0 {
				count = typed.count
			}
			v := baseline + (typed.feed-float32(g))*lead
			if v > paperH {
				break
			}
			for i, s := range l.strokes[:count] {
				h := hash(uint32(g*131 + i))
				jitter := (float32(h%5) - 2) * 0.05
				gl := serif.glyph(s.r)
				for _, c := range gl.contours {
					plan := make([][2]float32, len(c))
					for j, q := range c {
						plan[j] = [2]float32{margin + (s.x+q[0])*em, v + jitter + q[1]*em}
					}
					if v+em > paperH {
						plan = iso.Clip(plan, 0, 0, paperW, paperH)
						if len(plan) < 3 {
							continue
						}
					}
					tones[h/5%3] = append(tones[h/5%3], iso.Poly(cam, iso.On(plan, place), true)...)
				}
			}
		}
		for i, s := range tones {
			k.Fill(s, ink.ink[i])
		}
	})

	// The ribbon from spool to spool through the vibrator, which lifts
	// it into the type bar's path on each strike.
	lift := float32(0)
	if struck >= 0 {
		lift = 6 * float32(math.Sin(math.Pi*float64(typed.since)/float64(strike)))
	}
	spools := [2][2]float32{{50, 96}, {250, 96}}
	band := func(a, b iso.Vec, za, zb float32) []iso.Vec {
		return []iso.Vec{{a[0], a[1], za}, {b[0], b[1], zb}, {b[0], b[1], zb + ribbH}, {a[0], a[1], za + ribbH}}
	}
	ribbonZ0 := float32(ribbonZ)
	add(iso.Vec{64, ribbonY, ribbonZ}, iso.Vec{236, 90, ribbonZ + ribbH + 6}, -1, func() {
		for _, side := range []float32{-1, 1} {
			spool := spools[(int(side)+1)/2]
			dir := iso.Vec{typeX + side*10 - spool[0], ribbonY - spool[1], 0}.Unit()
			from := iso.Vec{spool[0] + 15*dir[0], spool[1] + 15*dir[1], 0}
			shape(band(from, iso.Vec{typeX + side*10, ribbonY, 0}, ribbonZ0, ribbonZ0+lift), ink.ribbon, ink.rubber.Edge)
		}
		shape(band(iso.Vec{typeX - 10, ribbonY, 0}, iso.Vec{typeX + 10, ribbonY, 0}, ribbonZ0+lift, ribbonZ0+lift), ink.ribbon, ink.rubber.Edge)
	})
	for _, s := range spools {
		pillar := post(s[0], s[1], 2.4, housingTop, spoolZ)
		addSolid(pillar, ink.metal, housingID, nil)
		low := post(s[0], s[1], 19, spoolZ, spoolZ+2)
		wind := post(s[0], s[1], 15, spoolZ+2, spoolZ+8)
		up := post(s[0], s[1], 18, spoolZ+8, spoolZ+10)
		hub := post(s[0], s[1], 3.5, spoolZ+10, spoolZ+13)
		addSolid(low, ink.metal, -1, nil)
		addSolid(wind, iso.Palette{Top: ink.ribbon, Shadow: ink.rubber.Shadow, Lit: ink.rubber.Lit, Edge: ink.rubber.Edge, Crease: ink.rubber.Crease}, -1, nil)
		x, y := s[0], s[1]
		addSolid(up, ink.metal, -1, func() {
			var spokes [][]iso.Vec
			for i := range 3 {
				a := float64(i)/3*2*math.Pi + 0.4
				c, sn := float32(math.Cos(a)), float32(math.Sin(a))
				spokes = append(spokes, []iso.Vec{{x + 3.5*c, y + 3.5*sn, spoolZ + 10}, {x + 16*c, y + 16*sn, spoolZ + 10}})
			}
			stroke(spokes, width, ink.metal.Edge)
			k.Stroke(iso.Outline(cam, iso.On(iso.Circle(x, y, 16, 32), iso.Floor(spoolZ+10))), width, ink.metal.Edge)
		})
		addSolid(hub, ink.metal, -1, nil)
	}

	// The type bar in flight: up from the basket to the type point and
	// back, while its key is down.
	if struck >= 0 {
		pivot := barAt(struck, 56)
		restTip := barAt(struck, 24)
		strikePoint := place(typeX-paperX-off, baseline+1.5).Add(iso.Vec{0, 1.6, 0})
		u := float32(math.Sin(math.Pi * float64(typed.since) / float64(strike)))
		a, b := restTip.Sub(pivot), strikePoint.Sub(pivot)
		length := float32(math.Sqrt(float64(a.Dot(a))))*(1-u) + float32(math.Sqrt(float64(b.Dot(b))))*u
		tip := pivot.Add(a.Unit().Lerp(b.Unit(), u).Unit().Scale(length))
		lo, hi := boxOf(pivot, tip)
		add(lo, hi, housingID, func() { wire([]iso.Vec{pivot, tip}, 1.6, ink.metal) })
	}

	// The microphone: a weighted base, a stem with its height collar, the
	// body with a grille (the one lit part), and the cable to the jack.
	mx, my := micAt[0], micAt[1]
	stack := []struct {
		r, z0, z1 float32
		pal       iso.Palette
	}{
		{micBase, 0, 5, ink.metal}, {2.6, 5, 60, ink.metal}, {5, 60, 67, ink.body}, {2.6, 67, 98, ink.metal},
		{6, 98, 102, ink.metal}, {11, 102, 110, ink.body}, {16.5, 110, 116, ink.rubber}, {16, 116, 152, ink.lit},
		{16.5, 152, 156, ink.rubber}, {15, 156, 158, ink.body}, {12, 158, 159.5, ink.body},
	}
	for i, s := range stack {
		pr := post(mx, my, s.r, s.z0, s.z1)
		lo, hi := prismBox(pr)
		var details func()
		switch i {
		case 0:
			details = func() {
				k.Stroke(iso.Outline(cam, iso.On(iso.Circle(mx, my, micBase-6, 36), iso.Floor(5))), width, ink.metal.Crease)
				for j := range 3 {
					a := float64(j)/3*2*math.Pi + 0.5
					k.Fill(iso.Poly(cam, iso.On(iso.Circle(mx+(micBase-3)*float32(math.Cos(a)), my+(micBase-3)*float32(math.Sin(a)), 0.7, 8), iso.Floor(5)), true), ink.metal.Edge)
				}
			}
		case 7:
			details = func() {
				var rings [][]iso.Vec
				for z := float32(119); z < 150; z += 3 {
					rings = append(rings, rim(mx, my, 16, z))
				}
				stroke(rings, width*0.7, ink.lit.Crease)
				facing := math.Atan2(float64(toward[1]), float64(toward[0]))
				var mesh [][]iso.Vec
				for j := -6; j <= 6; j++ {
					a := facing + float64(j)*math.Pi/14
					c, sn := float32(math.Cos(a)), float32(math.Sin(a))
					mesh = append(mesh, []iso.Vec{{mx + 16*c, my + 16*sn, 116}, {mx + 16*c, my + 16*sn, 152}})
				}
				stroke(mesh, width*0.6, ink.lit.Crease)
			}
		}
		add(lo, hi, -1, func() {
			solid(pr, s.pal, nil)
			if details != nil {
				details()
			}
		})
	}
	route := smooth([]iso.Vec{{mx + micBase, my, 2.5}, {-60, 234, 2.5}, {-46, 216, 2.5}, {-32, 203, 2.8}, {-24, 200, 6}, {-19, 200, 12}}, 8)
	lo, hi = boxOf(route...)
	add(lo.Sub(iso.Vec{3, 3, 3}), hi.Add(iso.Vec{0, 3, 3}), -1, func() { wire(route, 4.6, ink.rubber) })
	relief := rod(iso.Vec{-19, 200, 12}, iso.Vec{-13, 200, 12}, 3, 16)
	shell := rod(iso.Vec{-13, 200, 12}, iso.Vec{0, 200, 12}, 4, 20)
	addSolid(relief, ink.rubber, -1, nil)
	addSolid(shell, ink.metal, -1, nil)

	for _, i := range order(cam, parts) {
		parts[i].draw()
	}
}

// smooth runs a Catmull-Rom curve through points, steps segments apart.
func smooth(points []iso.Vec, steps int) []iso.Vec {
	out := []iso.Vec{points[0]}
	for i := 0; i+1 < len(points); i++ {
		p0, p1, p2, p3 := points[max(0, i-1)], points[i], points[i+1], points[min(len(points)-1, i+2)]
		for s := 1; s <= steps; s++ {
			t := float32(s) / float32(steps)
			t2, t3 := t*t, t*t*t
			var q iso.Vec
			for a := range 3 {
				q[a] = 0.5 * (2*p1[a] + (-p0[a]+p2[a])*t + (2*p0[a]-5*p1[a]+4*p2[a]-p3[a])*t2 + (-p0[a]+3*p1[a]-3*p2[a]+p3[a])*t3)
			}
			out = append(out, q)
		}
	}
	return out
}
