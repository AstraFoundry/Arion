// Package iso draws isometric solids with MyGo's painter, following the
// Anatomy skill's kit (.agents/skills/anatomy/kit/iso-kit.ts): the same
// camera, solids filled in paper and shaded by facing, one outline where a
// face turns away, a faint crease where a top meets a side, and painter's
// order back to front. Unlike the kit's static SVG it projects every frame,
// so a figure can turn, open and zoom.
//
// World units are about a millimetre: x and y on the floor, z up.
package iso

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Vec is a point or direction in the world.
type Vec [3]float32

func (a Vec) Add(b Vec) Vec             { return Vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a Vec) Sub(b Vec) Vec             { return Vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a Vec) Scale(s float32) Vec       { return Vec{a[0] * s, a[1] * s, a[2] * s} }
func (a Vec) Dot(b Vec) float32         { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a Vec) Lerp(b Vec, t float32) Vec { return a.Add(b.Sub(a).Scale(t)) }
func (a Vec) Cross(b Vec) Vec {
	return Vec{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

// Unit is a of length 1.
func (a Vec) Unit() Vec {
	l := float32(math.Sqrt(float64(a.Dot(a))))
	if l == 0 {
		return a
	}
	return a.Scale(1 / l)
}

// unit is the kit's scale of one world unit at Scale 1.
var unit = float32(math.Sqrt(1.6))

// ScaleFor is the Scale at which units world units span dips DIPs.
func ScaleFor(dips, units float32) float32 { return dips / (units * unit) }

// Camera looks down at the world from Azimuth around z and Elevation above
// the floor, in degrees, Scale DIPs to a world unit, and draws Focus at
// Center. Turning a figure by θ is drawing it at Azimuth − θ.
type Camera struct {
	// View is the box shapes are clipped to before they are drawn: MyGo
	// rasterizes each path whole, so geometry far outside it would only
	// cost memory.
	View                   ui.Rect
	Center                 [2]float32
	Focus                  Vec
	Scale                  float32
	sinA, cosA, sinE, cosE float32
}

// NewCamera returns a camera drawing focus at the middle of view; the
// kit's default is azimuth 45, elevation 30.
func NewCamera(azimuth, elevation, scale float32, focus Vec, view ui.Rect) Camera {
	cx, cy := view.X+view.W/2, view.Y+view.H/2
	a := float64(azimuth) * math.Pi / 180
	e := float64(elevation) * math.Pi / 180
	return Camera{
		View: view, Center: [2]float32{cx, cy}, Focus: focus, Scale: scale,
		sinA: float32(math.Sin(a)), cosA: float32(math.Cos(a)), sinE: float32(math.Sin(e)), cosE: float32(math.Cos(e)),
	}
}

// Project returns where p shows, in DIPs.
func (c Camera) Project(p Vec) (float32, float32) {
	d := p.Sub(c.Focus)
	k := c.Scale * unit
	return c.Center[0] + (d[0]*c.sinA-d[1]*c.cosA)*k, c.Center[1] + ((d[0]*c.cosA+d[1]*c.sinA)*c.sinE-d[2]*c.cosE)*k
}

// Toward is the direction to the viewer: a face whose normal points along
// it is seen, and a point further along it is nearer.
func (c Camera) Toward() Vec { return Vec{c.cosA * c.cosE, c.sinA * c.cosE, c.sinE} }

// Depth is how near p is: greater is nearer, painted later.
func (c Camera) Depth(p Vec) float32 { return p.Dot(c.Toward()) }

// Sees reports whether a face with normal n faces the viewer.
func (c Camera) Sees(n Vec) bool { return n.Dot(c.Toward()) > 1e-4 }

// light comes from the viewer's left, as the kit's.
func (c Camera) light() Vec { return Vec{-c.sinA, c.cosA, 0} }

// Palette colors a solid: Top for faces looking up, sides
// from Shadow to Lit by how they face the light, Edge for the outline and
// Crease where a top meets a side.
type Palette struct {
	Top, Shadow, Lit, Edge, Crease ui.Color
}

// Face is the color of a face with unit normal n. Tones change smoothly
// with the normal, so a turning solid never jumps between shades.
func (pal Palette) Face(c Camera, n Vec) ui.Color {
	lit := (n.Dot(c.light()) + 1) / 2
	// Steps of a twelfth let sides of one tone share a path and are too
	// fine to see change as a solid turns; the kit's static four would.
	lit = float32(math.Round(float64(lit)*12)) / 12
	return pal.Shadow.Mix(pal.Lit, lit).Mix(pal.Top, max(0, n[2]))
}

// Poly builds the path of points as they show, clipped to the view: a
// closed polygon is cut to it, and an open line keeps the pieces inside.
func Poly(c Camera, points []Vec, closed bool) *ui.Path {
	path := &ui.Path{}
	c.add(path, points, closed)
	return path
}

// Outline builds the outline of a closed shape, clipped to the view as a
// line, so that the view's edge never draws as part of it.
func Outline(c Camera, points []Vec) *ui.Path {
	return Poly(c, append(append([]Vec{}, points...), points[0]), false)
}

func (c Camera) add(path *ui.Path, points []Vec, closed bool) {
	flat := make([][2]float32, len(points))
	for i, p := range points {
		flat[i][0], flat[i][1] = c.Project(p)
	}
	if closed {
		addPolygon(path, clipPolygon(flat, c.bounds()))
		return
	}
	b := c.bounds()
	pen := false
	for i := 1; i < len(flat); i++ {
		a, z, ok := clipSegment(flat[i-1], flat[i], b)
		if !ok {
			pen = false
			continue
		}
		if !pen || a != flat[i-1] {
			path.MoveTo(a[0], a[1])
		}
		path.LineTo(z[0], z[1])
		pen = z == flat[i]
	}
}

// bounds is the view with room for strokes.
func (c Camera) bounds() [4]float32 {
	const margin = 4
	return [4]float32{c.View.X - margin, c.View.Y - margin, c.View.X + c.View.W + margin, c.View.Y + c.View.H + margin}
}

func addPolygon(path *ui.Path, points [][2]float32) {
	if len(points) < 3 {
		return
	}
	path.MoveTo(points[0][0], points[0][1])
	for _, q := range points[1:] {
		path.LineTo(q[0], q[1])
	}
	path.Close()
}

// clipPolygon cuts a polygon to the box b (x0, y0, x1, y1): Sutherland–Hodgman.
func clipPolygon(points [][2]float32, b [4]float32) [][2]float32 {
	inside := []func(q [2]float32) bool{
		func(q [2]float32) bool { return q[0] >= b[0] },
		func(q [2]float32) bool { return q[1] >= b[1] },
		func(q [2]float32) bool { return q[0] <= b[2] },
		func(q [2]float32) bool { return q[1] <= b[3] },
	}
	cross := func(edge int, p, q [2]float32) [2]float32 {
		var t float32
		switch edge {
		case 0, 2:
			x := b[edge]
			t = (x - p[0]) / (q[0] - p[0])
		default:
			y := b[edge]
			t = (y - p[1]) / (q[1] - p[1])
		}
		return [2]float32{p[0] + (q[0]-p[0])*t, p[1] + (q[1]-p[1])*t}
	}
	out := points
	for edge, in := range inside {
		if len(out) == 0 {
			break
		}
		src := out
		out = nil
		prev := src[len(src)-1]
		for _, q := range src {
			switch {
			case in(q):
				if !in(prev) {
					out = append(out, cross(edge, prev, q))
				}
				out = append(out, q)
			case in(prev):
				out = append(out, cross(edge, prev, q))
			}
			prev = q
		}
	}
	return out
}

// clipSegment cuts a segment to the box b: Liang–Barsky.
func clipSegment(p, q [2]float32, b [4]float32) ([2]float32, [2]float32, bool) {
	t0, t1 := float32(0), float32(1)
	dx, dy := q[0]-p[0], q[1]-p[1]
	for _, e := range [4][2]float32{{-dx, p[0] - b[0]}, {dx, b[2] - p[0]}, {-dy, p[1] - b[1]}, {dy, b[3] - p[1]}} {
		if e[0] == 0 {
			if e[1] < 0 {
				return p, q, false
			}
			continue
		}
		t := e[1] / e[0]
		if e[0] < 0 {
			t0 = max(t0, t)
		} else {
			t1 = min(t1, t)
		}
		if t0 > t1 {
			return p, q, false
		}
	}
	return [2]float32{p[0] + dx*t0, p[1] + dy*t0}, [2]float32{p[0] + dx*t1, p[1] + dy*t1}, true
}

// Prism is a convex solid: a flat convex Ring swept along Up. Rings on the
// floor make slabs and keycaps; a ring standing on end makes a barrel.
type Prism struct {
	Ring []Vec
	Up   Vec
}

// Center is the middle of the prism, for sorting.
func (pr Prism) Center() Vec {
	var sum Vec
	for _, p := range pr.Ring {
		sum = sum.Add(p)
	}
	return sum.Scale(1 / float32(len(pr.Ring))).Add(pr.Up.Scale(0.5))
}

// Faces says which of a prism's faces show: its cap at Ring, its cap at
// Ring + Up, and side i between Ring[i] and Ring[i+1].
type Faces struct {
	Bottom, Top bool
	Sides       []bool
	normals     []Vec
	up          Vec
}

// Facing works out which faces of pr the camera sees.
func (pr Prism) Facing(c Camera) Faces {
	n := len(pr.Ring)
	center := pr.Center()
	up := pr.Up.Unit()
	f := Faces{Bottom: c.Sees(up.Scale(-1)), Top: c.Sees(up), Sides: make([]bool, n), normals: make([]Vec, n), up: up}
	for i := range pr.Ring {
		a, b := pr.Ring[i], pr.Ring[(i+1)%n]
		normal := b.Sub(a).Cross(pr.Up).Unit()
		if normal.Dot(a.Add(b).Scale(0.5).Add(pr.Up.Scale(0.5)).Sub(center)) < 0 {
			normal = normal.Scale(-1)
		}
		f.normals[i] = normal
		f.Sides[i] = c.Sees(normal)
	}
	return f
}

// Draw paints the prism: each face it shows in its tone, which together
// cover its silhouette, then its creases and outline, width DIPs wide. top, when
// not nil, paints on the top face once it is filled.
func (pr Prism) Draw(p *ui.Painter, c Camera, pal Palette, width float32, top func()) {
	n := len(pr.Ring)
	f := pr.Facing(c)
	upper := make([]Vec, n)
	for i, v := range pr.Ring {
		upper[i] = v.Add(pr.Up)
	}
	// Sides of one tone go in one path.
	var run *ui.Path
	var tone ui.Color
	flush := func() {
		if run != nil {
			p.FillPath(run, tone)
			run = nil
		}
	}
	for i := range pr.Ring {
		if !f.Sides[i] {
			flush()
			continue
		}
		color := pal.Face(c, f.normals[i])
		if run != nil && color != tone {
			flush()
		}
		if run == nil {
			run, tone = &ui.Path{}, color
		}
		j := (i + 1) % n
		c.add(run, []Vec{pr.Ring[i], pr.Ring[j], upper[j], upper[i]}, true)
	}
	flush()
	if f.Top {
		p.FillPath(Poly(c, upper, true), pal.Face(c, f.up))
		if top != nil {
			top()
		}
	}
	if f.Bottom {
		p.FillPath(Poly(c, pr.Ring, true), pal.Face(c, f.up.Scale(-1)))
	}
	outline, crease := &ui.Path{}, &ui.Path{}
	edge := func(path *ui.Path, a, b Vec) { c.add(path, []Vec{a, b}, false) }
	for i := range pr.Ring {
		j := (i + 1) % n
		side := f.Sides[i]
		switch {
		case side != f.Bottom:
			edge(outline, pr.Ring[i], pr.Ring[j])
		}
		switch {
		case side != f.Top:
			edge(outline, upper[i], upper[j])
		case side && f.Top:
			edge(crease, upper[i], upper[j])
		}
		if side != f.Sides[(i+n-1)%n] {
			edge(outline, pr.Ring[i], upper[i])
		}
	}
	p.StrokePath(crease, width, pal.Crease)
	p.StrokePath(outline, width, pal.Edge)
}

// RoundedRect is the kit's rounded plan: a rectangle x, y, w, d with
// corners of radius r in steps segments, clockwise on screen.
func RoundedRect(x, y, w, d, r float32, steps int) [][2]float32 {
	r = max(0, min(r, w/2, d/2))
	corners := [4][3]float32{{x + w - r, y + r, -90}, {x + w - r, y + d - r, 0}, {x + r, y + d - r, 90}, {x + r, y + r, 180}}
	var out [][2]float32
	for _, k := range corners {
		count := steps
		if r == 0 {
			count = 0
		}
		for s := 0; s <= count; s++ {
			angle := float64(k[2]) * math.Pi / 180
			if count > 0 {
				angle += float64(s) / float64(count) * math.Pi / 2
			}
			out = append(out, [2]float32{k[0] + r*float32(math.Cos(angle)), k[1] + r*float32(math.Sin(angle))})
		}
	}
	return out
}

// Circle is a ring of steps points of radius r around (cx, cy).
func Circle(cx, cy, r float32, steps int) [][2]float32 {
	out := make([][2]float32, steps)
	for i := range out {
		a := float64(i) / float64(steps) * 2 * math.Pi
		out[i] = [2]float32{cx + r*float32(math.Cos(a)), cy + r*float32(math.Sin(a))}
	}
	return out
}

// On maps a plan into the world through place.
func On(plan [][2]float32, place func(u, v float32) Vec) []Vec {
	out := make([]Vec, len(plan))
	for i, q := range plan {
		out[i] = place(q[0], q[1])
	}
	return out
}

// Floor places a plan at height z.
func Floor(z float32) func(u, v float32) Vec {
	return func(u, v float32) Vec { return Vec{u, v, z} }
}
