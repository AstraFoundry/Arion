package ui

import (
	"image"
	"image/color"
	"testing"
)

// BenchmarkPaintTiles paints the frames of a 2D game's view at
// 1280×720 DIPs at twice the density, as it animates with
// Painter.AnimationFrame: a map of tiles 16 pixels drawn 3 times as large
// in three layers, through a transform that scrolls, under 300 sprites
// walking, flipped as they face, all parts of one pixelated sheet drawn
// with ImageRegion. It measures what a frame costs the CPU before a
// renderer draws it.
func BenchmarkPaintTiles(b *testing.B) {
	sheet := image.NewRGBA(image.Rect(0, 0, 128, 128))
	for y := range 128 {
		for x := range 128 {
			sheet.Set(x, y, color.RGBA{uint8(x * 2), uint8(y * 2), 120, 255})
		}
	}
	bm := NewBitmap(sheet).Pixelated()
	var tiles [64]Rect
	for i := range tiles {
		tiles[i] = Rect{X: float32(i%8) * 16, Y: float32(i/8) * 16, W: 16, H: 16}
	}
	const cols, rows, size = 60, 40, 48 // a map larger than the view
	frame := 0
	view := func(c *context) {
		coreBox(c).Size(1280, 720).Draw(func(p *Painter, r Rect) {
			camX := float32(frame%600) * 2
			p.Transform(Translation(-camX, -100), r.X, r.Y, func() {
				for l := range 3 {
					for y := range rows {
						for x := range cols {
							if l > 0 && (x+y+l)%3 != 0 {
								continue
							}
							p.ImageRegion(bm, tiles[(x*7+y*13+l)%64], Rect{X: r.X + float32(x)*size, Y: r.Y + float32(y)*size, W: size, H: size}, ImageOptions{})
						}
					}
				}
				for i := range 300 {
					src := tiles[i%8]
					src.H = 32
					x := float32((i*97+frame)%(cols*size)) + 0.5
					y := float32(i*61%(rows*size-96)) + 0.25
					p.ImageRegion(bm, src, Rect{X: r.X + x, Y: r.Y + y, W: size, H: 2 * size}, ImageOptions{FlipX: (i+frame/30)%2 == 0})
				}
			})
		})
	}
	tt := coreNewTester(view, 1280, 720)
	tt.SetScale(2)
	rt := tt.rt
	images := 0
	for b.Loop() {
		frame++
		rt.paint(rt.c.root, 1280, 720, 2)
	}
	for _, op := range rt.scene.Ops {
		if op.Image != nil {
			images++
		}
	}
	b.ReportMetric(float64(images), "sprites")
}
