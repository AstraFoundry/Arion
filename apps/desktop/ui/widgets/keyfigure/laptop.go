package keyfigure

import (
	"math"

	"arion.local/desktop/ui/shared/iso"
	"github.com/egoist/mygo/ui"
)

// The laptop, in world units of about a millimetre: one key pitch is U.
// The keyboard's back left corner is the origin; y runs toward the front.
const (
	U         = 18
	keyGap    = 1.6
	baseH     = 10  // the base's thickness
	capLift   = 0.3 // a cap's underside above the deck
	capH      = 1.6 // a cap's thickness
	travel    = 0.9 // how far a pressed cap sinks
	lidGap    = 2.1 // between the deck and the closed lid, over the caps
	lidH      = 4   // the lid's thickness
	lidOpen   = 112 // degrees the lid opens
	sideRim   = 14  // deck beside the keyboard
	backRim   = 14  // deck behind the keyboard
	padGap    = 12  // between the keyboard and the trackpad
	padD      = 4 * U
	padW      = 6.4 * U
	frontRim  = 14
	hingeR    = 3
	hingeEdge = 34 // the hinge stops this far from each side
)

var rowDepths = [...]float32{0.62, 1, 1, 1, 1, 1}

// keysDepth is how deep the keyboard is.
var keysDepth = func() float32 {
	var d float32
	for _, r := range rowDepths {
		d += r * U
	}
	return d
}()

// base is the laptop's plan.
var base = struct{ x, y, w, d float32 }{-sideRim, -backRim, 15*U + 2*sideRim, backRim + keysDepth + padGap + padD + frontRim}

// key is one cap: its plan and what is printed on it.
type key struct {
	id         string
	x, y, w, d float32
	mark       string
	homing     bool
	touch      bool
}

// A row lists keys as id, width in U and an optional legend. Only the
// bottom row's modifiers carry legends; the arrows are half height.
type keySpec struct {
	id    string
	width float32
	mark  string
}

func numbered(prefix string, n int) []keySpec {
	out := make([]keySpec, n)
	for i := range out {
		out[i] = keySpec{id: prefix + string(rune('a'+i)), width: 1}
	}
	return out
}

func row(specs ...[]keySpec) []keySpec {
	var out []keySpec
	for _, s := range specs {
		out = append(out, s...)
	}
	return out
}

var upperRows = [][]keySpec{
	row([]keySpec{{"esc", 1.5, ""}}, numbered("f", 12), []keySpec{{"power", 1.5, ""}}),
	row([]keySpec{{"grave", 1, ""}}, numbered("n", 12), []keySpec{{"delete", 2, ""}}),
	row([]keySpec{{"tab", 1.5, ""}}, numbered("q", 12), []keySpec{{"backslash", 1.5, ""}}),
	row([]keySpec{{"caps", 1.75, ""}}, numbered("a", 11), []keySpec{{"return", 2.25, ""}}),
	row([]keySpec{{"shift", 2.25, ""}}, numbered("z", 10), []keySpec{{"rshift", 2.75, ""}}),
}

// bottomRows are each platform's modifier row; the ids of the dictation
// keys are their runtime.HoldKey values.
var bottomRows = map[string][]keySpec{
	"darwin": {{"fn", 1, "fn"}, {"lctrl", 1, "control"}, {"lopt", 1, "option"}, {"lcmd", 1.25, "command"}, {"space", 5.5, ""},
		{"right_command", 1.25, "command"}, {"right_option", 1, "option"}, {"arrows", 3, ""}},
	"windows": {{"lctrl", 1.25, "Ctrl"}, {"lwin", 1.25, "win"}, {"lalt", 1.25, "Alt"}, {"space", 5.25, ""},
		{"right_alt", 1, "Alt"}, {"menu", 1, "menu"}, {"right_ctrl", 1, "Ctrl"}, {"arrows", 3, ""}},
}

// keysOf lays out a platform's keyboard.
func keysOf(platform string) []key {
	rows := append(append([][]keySpec{}, upperRows...), bottomRows[platform])
	var keys []key
	var y float32
	for r, specs := range rows {
		d := rowDepths[r] * U
		var x float32
		for _, s := range specs {
			w := s.width * U
			if s.id == "arrows" {
				// ← ↑↓ →: the inverted T in half-height caps.
				half, unit := d/2, w/3
				keys = append(keys,
					key{id: "left", x: x, y: y + half, w: unit, d: half, mark: "left"},
					key{id: "up", x: x + unit, y: y, w: unit, d: half, mark: "up"},
					key{id: "down", x: x + unit, y: y + half, w: unit, d: half, mark: "down"},
					key{id: "right", x: x + 2*unit, y: y + half, w: unit, d: half, mark: "right"})
			} else {
				// Homing bumps sit on F and J, the home row's fourth and seventh letters.
				keys = append(keys, key{id: s.id, x: x, y: y, w: w, d: d, mark: s.mark,
					homing: s.id == "ad" || s.id == "ag", touch: s.id == "power" && platform == "darwin"})
			}
			x += w
		}
		y += d
	}
	for i := range keys {
		k := &keys[i]
		k.x, k.y, k.w, k.d = k.x+keyGap/2, k.y+keyGap/2, k.w-keyGap, k.d-keyGap
	}
	return keys
}

// center is the middle of a cap's top.
func (k key) center(sink float32) iso.Vec {
	return iso.Vec{k.x + k.w/2, k.y + k.d/2, baseH + capLift + capH - sink}
}

// inView reports whether any of the cap shows in r.
func (k key) inView(cam iso.Camera, r ui.Rect) bool {
	x0, y0, x1, y1 := float32(math.MaxFloat32), float32(math.MaxFloat32), float32(-math.MaxFloat32), float32(-math.MaxFloat32)
	for _, c := range [][3]float32{{k.x, k.y, baseH}, {k.x + k.w, k.y, baseH}, {k.x, k.y + k.d, baseH}, {k.x + k.w, k.y + k.d, baseH}} {
		x, y := cam.Project(iso.Vec(c))
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	return x1 > r.X-4 && x0 < r.X+r.W+4 && y1 > r.Y-8 && y0 < r.Y+r.H+4
}

func (k key) prism(sink float32) iso.Prism {
	return iso.Prism{Ring: iso.On(iso.RoundedRect(k.x, k.y, k.w, k.d, 1.8, 2), iso.Floor(baseH+capLift-sink)), Up: iso.Vec{0, 0, capH}}
}

// lidAt places a point of the lid: u across, v from the hinge to the far
// edge, w out of the inner face, with the lid open by angle degrees.
func lidAt(angle float32) func(u, v, w float32) iso.Vec {
	a := float64(angle) * math.Pi / 180
	along := iso.Vec{0, float32(math.Cos(a)), float32(math.Sin(a))}
	out := iso.Vec{0, -float32(math.Sin(a)), float32(math.Cos(a))}
	hinge := iso.Vec{0, base.y, baseH + lidGap}
	return func(u, v, w float32) iso.Vec {
		return hinge.Add(iso.Vec{u, 0, 0}).Add(along.Scale(v)).Add(out.Scale(w))
	}
}

// Stroke glyphs in an em box, y up, as polylines.
func arc(cx, cy, rx, ry, from, to float32, steps int) [][2]float32 {
	out := make([][2]float32, steps+1)
	for i := range out {
		a := float64(from+(to-from)*float32(i)/float32(steps)) * math.Pi / 180
		out[i] = [2]float32{cx + rx*float32(math.Cos(a)), cy + ry*float32(math.Sin(a))}
	}
	return out
}

func join(parts ...[][2]float32) [][2]float32 {
	var out [][2]float32
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

type glyph struct {
	w     float32
	lines [][][2]float32
}

var letters = map[rune]glyph{
	'f': {0.42, [][][2]float32{join([][2]float32{{0.14, 0}, {0.14, 0.78}}, arc(0.32, 0.78, 0.18, 0.2, 180, 90, 4)[1:]), {{0.02, 0.62}, {0.36, 0.62}}}},
	'n': {0.56, [][][2]float32{{{0.06, 0}, {0.06, 0.62}}, join([][2]float32{{0.06, 0.4}}, arc(0.27, 0.4, 0.21, 0.22, 180, 0, 8), [][2]float32{{0.48, 0}})}},
	'C': {0.7, [][][2]float32{arc(0.38, 0.5, 0.32, 0.5, 50, 310, 14)}},
	't': {0.36, [][][2]float32{join([][2]float32{{0.14, 0.86}, {0.14, 0.16}}, arc(0.28, 0.16, 0.14, 0.16, 180, 270, 3)[1:]), {{0.02, 0.62}, {0.32, 0.62}}}},
	'r': {0.38, [][][2]float32{{{0.06, 0}, {0.06, 0.62}}, join([][2]float32{{0.06, 0.36}}, arc(0.3, 0.36, 0.24, 0.26, 180, 90, 4)[1:])}},
	'l': {0.2, [][][2]float32{{{0.08, 0}, {0.08, 1}}}},
	'A': {0.72, [][][2]float32{{{0, 0}, {0.34, 1}, {0.68, 0}}, {{0.12, 0.34}, {0.56, 0.34}}}},
}

func square(x0, y0, x1, y1 float32) [][2]float32 {
	return [][2]float32{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}, {x0, y0}}
}

// symbols are whole-cap marks, centred in a unit box.
var symbols = map[string][][][2]float32{
	// ⌘: a square with a loop at each corner.
	"command": func() [][][2]float32 {
		out := [][][2]float32{square(0.3, 0.3, 0.7, 0.7)}
		for _, c := range [][3]float32{{0.3, 0.3, 0}, {0.7, 0.3, 1}, {0.7, 0.7, 2}, {0.3, 0.7, 3}} {
			sx, sy := float32(1), float32(1)
			if c[0] > 0.5 {
				sx = -1
			}
			if c[1] > 0.5 {
				sy = -1
			}
			// Each loop leaves the square along one side and comes back along the other.
			start := float32(0)
			if sx < 0 {
				start = 180
			}
			sweep := float32(-270)
			if sx*sy < 0 {
				sweep = 270
			}
			out = append(out, arc(c[0]-sx*0.14, c[1]-sy*0.14, 0.14, 0.14, start, start+sweep, 12))
		}
		return out
	}(),
	// ⌥: a shifted rail and a short bar.
	"option":  {{{0.04, 0.74}, {0.34, 0.74}, {0.68, 0.26}, {0.96, 0.26}}, {{0.6, 0.74}, {0.96, 0.74}}},
	"control": {{{0.22, 0.36}, {0.5, 0.72}, {0.78, 0.36}}},
	// The Windows logo: four panes.
	"win": {square(0.22, 0.22, 0.47, 0.47), square(0.53, 0.22, 0.78, 0.47), square(0.22, 0.53, 0.47, 0.78), square(0.53, 0.53, 0.78, 0.78)},
	// The menu key: a page of three lines.
	"menu":  {square(0.24, 0.16, 0.76, 0.84), {{0.36, 0.64}, {0.64, 0.64}}, {{0.36, 0.5}, {0.64, 0.5}}, {{0.36, 0.36}, {0.64, 0.36}}},
	"up":    {{{0.3, 0.38}, {0.5, 0.62}, {0.7, 0.38}, {0.3, 0.38}}},
	"down":  {{{0.3, 0.62}, {0.5, 0.38}, {0.7, 0.62}, {0.3, 0.62}}},
	"left":  {{{0.62, 0.3}, {0.38, 0.5}, {0.62, 0.7}, {0.62, 0.3}}},
	"right": {{{0.38, 0.3}, {0.62, 0.5}, {0.38, 0.7}, {0.38, 0.3}}},
}

// legend lays a cap's mark on its top, centred on c: em units run along +x
// and up the cap toward the back (−y).
func legend(mark string, c iso.Vec) [][]iso.Vec {
	var lines [][][2]float32
	size := float32(7.5)
	if s, ok := symbols[mark]; ok {
		for _, l := range s {
			moved := make([][2]float32, len(l))
			for i, q := range l {
				moved[i] = [2]float32{q[0] - 0.5, q[1] - 0.5}
			}
			lines = append(lines, moved)
		}
	} else {
		size = 4.6
		var pen float32
		for _, r := range mark {
			g := letters[r]
			for _, l := range g.lines {
				moved := make([][2]float32, len(l))
				for i, q := range l {
					moved[i] = [2]float32{q[0] + pen, q[1]}
				}
				lines = append(lines, moved)
			}
			pen += g.w + 0.08
		}
		width := pen - 0.08
		for _, l := range lines {
			for i := range l {
				l[i] = [2]float32{l[i][0] - width/2, l[i][1] - 0.4}
			}
		}
	}
	out := make([][]iso.Vec, len(lines))
	for i, l := range lines {
		out[i] = iso.On(l, func(u, v float32) iso.Vec { return iso.Vec{c[0] + u*size, c[1] - v*size, c[2]} })
	}
	return out
}
