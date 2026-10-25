package iso

import (
	"encoding/binary"
	"image"
	"image/draw"
	"math"
	"runtime"
	"slices"
	"sync"
	"unsafe"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/vector"
)

// strip is how wide, in pixels, the columns are that shapes rasterize in.
// A rasterizer pass costs its whole box, and isometric edges run
// diagonally: in columns, a long thin line or the rim of a large face
// costs about its own area. Columns start on pixel edges, so a pixel is
// only ever covered within one of them and no seam shows.
const strip = 64

// Canvas is a picture of a view at a pixel scale, drawn on the CPU and
// shown as one bitmap. Fills and strokes are recorded, then End paints
// them in order, in horizontal bands at once, one per processor: a band's
// pixels are its own, so the bands never wait on each other.
type Canvas struct {
	img *image.RGBA
	// pixels32 is img's pixels, one little-endian RGBA word each.
	pixels32 []uint32
	view     ui.Rect
	scale    float32
	ops      []op
	workers  []worker
}

// op is one recorded fill: polygons in pixels, their bounds and their
// color.
type op struct {
	polys  [][][2]float32
	bounds [][4]float32
	color  ui.Color
}

func newOp(polys [][][2]float32, c ui.Color) op {
	bounds := make([][4]float32, len(polys))
	for i, poly := range polys {
		b := [4]float32{math.MaxFloat32, math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
		for _, p := range poly {
			b = [4]float32{min(b[0], p[0]), min(b[1], p[1]), max(b[2], p[0]), max(b[3], p[1])}
		}
		bounds[i] = b
	}
	return op{polys, bounds, c}
}

// worker paints one band with its own rasterizer.
type worker struct {
	z      vector.Rasterizer
	mask   image.Alpha
	pieces []piece
}

// Begin clears img, of the view at scale pixels to a DIP, to draw a new
// picture on it.
func (k *Canvas) Begin(img *image.RGBA, view ui.Rect, scale float32) {
	k.img, k.view, k.scale = img, view, scale
	k.pixels32 = unsafe.Slice((*uint32)(unsafe.Pointer(unsafe.SliceData(img.Pix))), len(img.Pix)/4)
	k.ops = k.ops[:0]
	clear(img.Pix)
}

// End paints what was recorded since Begin.
func (k *Canvas) End() {
	bands := min(runtime.GOMAXPROCS(0), 8)
	if len(k.workers) < bands {
		k.workers = make([]worker, bands)
	}
	h := k.img.Bounds().Dy()
	var wg sync.WaitGroup
	for i := 0; i < bands; i++ {
		y0, y1 := h*i/bands, h*(i+1)/bands
		w := &k.workers[i]
		wg.Go(func() {
			for _, o := range k.ops {
				k.rasterize(w, o, y0, y1)
			}
		})
	}
	wg.Wait()
}

// Fill fills each polygon of s with c.
func (k *Canvas) Fill(s Shape, c ui.Color) {
	if len(s) == 0 || c.A == 0 {
		return
	}
	polys := make([][][2]float32, len(s))
	for i, poly := range s {
		polys[i] = k.pixels(poly)
	}
	k.ops = append(k.ops, newOp(polys, c))
}

// Stroke draws each line of s width DIPs wide, with round joins and caps.
func (k *Canvas) Stroke(s Shape, width float32, c ui.Color) {
	if len(s) == 0 || c.A == 0 {
		return
	}
	hw := width * k.scale / 2
	var polys [][][2]float32
	for _, line := range s {
		points := k.pixels(line)
		for i, p := range points {
			polys = append(polys, disc(p, hw))
			if i == 0 {
				continue
			}
			q := points[i-1]
			dx, dy := p[0]-q[0], p[1]-q[1]
			l := float32(math.Hypot(float64(dx), float64(dy)))
			if l == 0 {
				continue
			}
			nx, ny := -dy/l*hw, dx/l*hw
			polys = append(polys, [][2]float32{{q[0] + nx, q[1] + ny}, {p[0] + nx, p[1] + ny}, {p[0] - nx, p[1] - ny}, {q[0] - nx, q[1] - ny}})
		}
	}
	k.ops = append(k.ops, newOp(polys, c))
}

// disc is a round cap or join, wound as the segments' quads are, so that
// where they overlap their coverage adds up rather than cancelling.
func disc(p [2]float32, r float32) [][2]float32 {
	const sides = 10
	out := make([][2]float32, sides)
	for i := range out {
		a := -float64(i) / sides * 2 * math.Pi
		out[i] = [2]float32{p[0] + r*float32(math.Cos(a)), p[1] + r*float32(math.Sin(a))}
	}
	return out
}

func (k *Canvas) pixels(points [][2]float32) [][2]float32 {
	out := make([][2]float32, len(points))
	for i, p := range points {
		out[i] = [2]float32{(p[0] - k.view.X) * k.scale, (p[1] - k.view.Y) * k.scale}
	}
	return out
}

// rasterize paints o within rows y0 to y1, column by column.
func (k *Canvas) rasterize(w *worker, o op, y0, y1 int) {
	c := o.color
	bounds := k.img.Bounds()
	minX, maxX := float32(math.MaxFloat32), float32(-math.MaxFloat32)
	for i, b := range o.bounds {
		// Only what falls in the band.
		if b[3] <= float32(y0) || b[1] >= float32(y1) {
			continue
		}
		minX, maxX = min(minX, o.bounds[i][0]), max(maxX, o.bounds[i][2])
	}
	for x0 := max(int(math.Floor(float64(minX)))/strip*strip, 0); x0 < min(int(math.Ceil(float64(maxX))), bounds.Max.X); x0 += strip {
		x1 := min(x0+strip, bounds.Max.X)
		box := [4]float32{float32(x0), float32(y0), float32(x1), float32(y1)}
		pieces := w.pieces[:0]
		for i, poly := range o.polys {
			if p := pieceOf(poly, o.bounds[i], box); p.points != nil {
				pieces = append(pieces, p)
			}
		}
		w.pieces = pieces
		// Pieces far apart down the column rasterize apart: a ring or a
		// long outline has one near the top and one near the bottom. Each
		// group starts on a row past the last, so no pixel is shared.
		slices.SortFunc(pieces, func(a, b piece) int { return a.y0 - b.y0 })
		for start := 0; start < len(pieces); {
			end, top, bottom := start+1, pieces[start].y0, pieces[start].y1
			for end < len(pieces) && pieces[end].y0 < bottom+gap {
				bottom = max(bottom, pieces[end].y1)
				end++
			}
			top, bottom = max(top, y0), min(bottom, y1)
			if bottom > top {
				k.draw(w, pieces[start:end], x0, top, x1, bottom, c)
			}
			start = end
		}
	}
}

// gap is how many empty rows split a column's pieces into two passes.
const gap = 8

// piece is a polygon cut to a column, with the rows it covers.
type piece struct {
	points [][2]float32
	y0, y1 int
}

func pieceOf(poly [][2]float32, b, box [4]float32) piece {
	if b[2] <= box[0] || b[0] >= box[2] || b[3] <= box[1] || b[1] >= box[3] {
		return piece{}
	}
	if b[0] >= box[0] && b[2] <= box[2] && b[1] >= box[1] && b[3] <= box[3] {
		// Wholly inside: no need to cut.
		return piece{poly, int(math.Floor(float64(b[1]))), int(math.Ceil(float64(b[3])))}
	}
	points := clipPolygon(poly, box)
	if len(points) < 3 {
		return piece{}
	}
	minY, maxY := float32(math.MaxFloat32), float32(-math.MaxFloat32)
	for _, p := range points {
		minY, maxY = min(minY, p[1]), max(maxY, p[1])
	}
	return piece{points, int(math.Floor(float64(minY))), int(math.Ceil(float64(maxY)))}
}

// draw rasterizes pieces in the box x0, y0, x1, y1 and paints them.
func (k *Canvas) draw(wk *worker, pieces []piece, x0, y0, x1, y1 int, c ui.Color) {
	w, h := x1-x0, y1-y0
	wk.z.Reset(w, h)
	for _, p := range pieces {
		wk.z.MoveTo(p.points[0][0]-float32(x0), p.points[0][1]-float32(y0))
		for _, q := range p.points[1:] {
			wk.z.LineTo(q[0]-float32(x0), q[1]-float32(y0))
		}
		wk.z.ClosePath()
	}
	if cap(wk.mask.Pix) < w*h {
		wk.mask.Pix = make([]uint8, w*h)
	}
	wk.mask.Pix, wk.mask.Stride, wk.mask.Rect = wk.mask.Pix[:w*h], w, image.Rect(0, 0, w, h)
	wk.z.DrawOp = draw.Src
	wk.z.Draw(&wk.mask, wk.mask.Rect, image.Opaque, image.Point{})
	k.blend(&wk.mask, x0, y0, c)
}

// blend paints c over the picture at (x0, y0) through the coverage mask:
// most pixels of a shape are either outside it or wholly inside, which
// cost a skip or a store.
func (k *Canvas) blend(mask *image.Alpha, x0, y0 int, c ui.Color) {
	a := uint32(c.A)
	// The color premultiplied, as the picture is, and as one pixel.
	r, g, b := uint32(c.R)*a/255, uint32(c.G)*a/255, uint32(c.B)*a/255
	solid := r | g<<8 | b<<16 | a<<24
	opaque := a == 255
	w, h := mask.Rect.Dx(), mask.Rect.Dy()
	stride := k.img.Stride / 4
	for y := 0; y < h; y++ {
		row := mask.Pix[y*w : y*w+w]
		pix := k.pixels32[(y0+y)*stride+x0 : (y0+y)*stride+x0+w]
		for x := 0; x < w; {
			// Eight at a time where the mask is empty, or full and the
			// color opaque.
			if x+8 <= w {
				word := binary.LittleEndian.Uint64(row[x:])
				if word == 0 {
					x += 8
					continue
				}
				if word == ^uint64(0) && opaque {
					run := pix[x : x+8 : x+8]
					for i := range run {
						run[i] = solid
					}
					x += 8
					continue
				}
			}
			switch m := uint32(row[x]); {
			case m == 0:
			case m == 255 && opaque:
				pix[x] = solid
			default:
				d := pix[x]
				keep := 255 - m*a/255
				pix[x] = (r*m+(d&0xff)*keep)/255 | ((g*m+(d>>8&0xff)*keep)/255)<<8 |
					((b*m+(d>>16&0xff)*keep)/255)<<16 | ((a*m+(d>>24)*keep)/255)<<24
			}
			x++
		}
	}
}
