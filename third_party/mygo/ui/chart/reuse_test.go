package chart

import (
	"bytes"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestLineInto(t *testing.T) {
	xs := []float32{0, .1, .3, .6, .8, 1}
	ys := []float32{.1, .5, .2, float32(math.NaN()), .4, .8}
	var path ui.Path
	for _, curve := range []Curve{CurveLinear, CurveMonotone, CurveStep, CurveStepBefore, CurveStepAfter} {
		fresh := ui.Render(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { p.StrokePath(Line(r, xs, ys, curve), 2, ui.RGB(30, 100, 240)) })
		}, 100, 100, 2)
		path.MoveTo(0, 0).LineTo(90, 90).Trim(0, .1)
		reused := ui.Render(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				p.StrokePath(LineInto(&path, r, xs, ys, curve), 2, ui.RGB(30, 100, 240))
			})
		}, 100, 100, 2)
		if !bytes.Equal(fresh.Pix, reused.Pix) {
			t.Fatal("reuse changes geometry or retains Trim", curve)
		}
	}
	LineInto(&path, ui.Rect{W: 100, H: 100}, xs, ys, CurveLinear)
	if a := testing.AllocsPerRun(20, func() { LineInto(&path, ui.Rect{W: 100, H: 100}, xs, ys, CurveLinear) }); a != 0 {
		t.Fatal("linear reuse allocates", a)
	}
}
