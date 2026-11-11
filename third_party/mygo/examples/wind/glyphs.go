package main

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// charset is the characters an atlas holds, ASCII's printable ones and a
// middle dot, sixteen to a row.
const charset = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~·"

// looks are the sets of characters a visit shows, each a sequence that a
// cell steps through from a place of its own.
var looks = []struct{ name, chars string }{
	{"mixed", "ab0cAde1fEgh2iGjk3lHmn4oKpq5rMst6uRvw7xSyz8nT9eW"},
	{"lettered", "abcd3efgRhijk7lmnAopqr0stuEvwxy5zaeKiou2gcS9"},
	{"figured", "0a1b2C3d4e5F6g7h8i9J0k1m2N3p4q5R6s7t8U9v0w1X2y3z"},
}

// The glyphs: glyphSize DIPs, in a light weight on screens of 1.5 pixels
// a DIP or more and a regular one on others, where a light stroke would
// break up; in the system's monospaced font.
const (
	glyphSize   = 9
	lightWeight = 300
	family      = "ui-monospace, Menlo, Consolas, monospace"
	// inkFrom is the coverage from which a pixel counts in a glyph's box.
	inkFrom = 8
)

// atlas is the charset rasterized for cells of cellW×cellH pixels, white
// on transparent, which glyphs.kage reads; each character's box, where
// its coverage reaches inkFrom (left, top, right and bottom, in pixels of
// its cell); and the characters the font does not draw a cell wide.
type atlas struct {
	cellW, cellH int
	bitmap       *ui.Bitmap
	boxes        [][4]float32
	missing      []bool
}

// newAtlas rasterizes the charset for cells of cellW pixels at scale
// pixels a DIP, as large as glyphSize allows in a cell with a pixel to
// spare either side, its baseline placed to center the font's ascent and
// descent, and each glyph centered as the zero's advance. It draws a frame
// of its own (ui.Render), which must not happen within a window's.
func newAtlas(cellW int, scale float32) *atlas {
	cellH := 2 * cellW
	weight := lightWeight
	if scale < 1.5 {
		weight += 100
	}
	ratio := float32(0.6)
	if g := ui.Shape("0", ui.Font{Family: family, Size: 100, Weight: weight}); len(g) == 1 {
		ratio = g[0].Advance / 100
	}
	px := min(float32(math.Round(float64(glyphSize*scale))), float32(math.Floor(float64(float32(cellW-2)/ratio))))
	f := ui.Font{Family: family, Size: px / scale, Weight: weight, Antialiased: true}
	m := f.Metrics()
	base := float32(math.Round(float64((float32(cellH)-(m.Ascent+m.Descent)*scale)/2 + m.Ascent*scale)))
	var zero float32
	if g := ui.Shape("0", f); len(g) == 1 {
		zero = g[0].Advance * scale
	}
	left := float32(math.Floor(float64((float32(cellW) - zero) / 2)))

	chars := []rune(charset)
	a := &atlas{cellW: cellW, cellH: cellH, boxes: make([][4]float32, len(chars)), missing: make([]bool, len(chars))}
	shaped := make([][]ui.Glyph, len(chars))
	for i, r := range chars {
		g := ui.Shape(string(r), f)
		if len(g) != 1 || abs(g[0].Advance*scale-zero) > 0.5 {
			a.missing[i] = true
			continue
		}
		shaped[i] = g
	}
	w, h := 16*cellW, (len(chars)+15)/16*cellH
	sheet := ui.Render(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, _ ui.Rect) {
			for i, g := range shaped {
				if g == nil {
					continue
				}
				x, y := float32(i%16*cellW), float32(i/16*cellH)
				p.Clip(ui.Rect{X: x / scale, Y: y / scale, W: float32(cellW) / scale, H: float32(cellH) / scale}, 0, func() {
					p.Glyphs(g, (x+left)/scale, (y+base)/scale, ui.Hex("#ffffff"))
				})
			}
		})
	}, int(math.Ceil(float64(float32(w)/scale))), int(math.Ceil(float64(float32(h)/scale))), scale)
	a.bitmap = ui.NewBitmap(sheet)

	for i := range chars {
		ox, oy := i%16*cellW, i/16*cellH
		x0, y0, x1, y1 := cellW, cellH, -1, -1
		for y := range cellH {
			for x := range cellW {
				sx, sy := ox+x, oy+y
				if sx < sheet.Rect.Dx() && sy < sheet.Rect.Dy() && sheet.Pix[sy*sheet.Stride+4*sx+3] >= inkFrom {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
				}
			}
		}
		a.boxes[i] = [4]float32{float32(x0), float32(y0), float32(x1), float32(y1)}
	}
	return a
}

// sequence returns the characters of look that the atlas draws, as
// indexes of charset.
func (a *atlas) sequence(look string) []int {
	chars := looks[0].chars
	for _, l := range looks {
		if l.name == look {
			chars = l.chars
		}
	}
	var seq []int
	for _, r := range chars {
		i := indexOf(r)
		if i > 0 && !a.missing[i] {
			seq = append(seq, i)
		}
	}
	if len(seq) == 0 {
		seq = append(seq, indexOf('.'))
	}
	return seq
}

func indexOf(r rune) int {
	for i, c := range []rune(charset) {
		if c == r {
			return i
		}
	}
	return -1
}
