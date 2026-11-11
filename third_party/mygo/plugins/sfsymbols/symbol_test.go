package sfsymbols

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// box is an SVG square filling its view box, for a fallback that shows.
var box = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="currentColor"/></svg>`))

// TestSymbolElement checks that a Symbol is named for assistive technology
// and sized as asked, at the theme's font size by default.
func TestSymbolElement(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Children(func() {
			Symbol(c, "gearshape")
			Symbol(c, "trash").Size(32).Color(ui.RGB(255, 0, 0)).Weight(Bold).Fill()
		})
	}, 200, 100)
	r, ok := tt.Find("gearshape")
	if !ok || r.W != r.H || r.W <= 0 {
		t.Errorf("gearshape: found %v at %v, want a square", ok, r)
	}
	if r, ok := tt.Find("trash"); !ok || r.W != 32 || r.H != 32 {
		t.Errorf("trash: found %v at %v, want 32 square", ok, r)
	}
}

// TestSymbolFallback checks that a name no system has draws the fallback,
// in the symbol's color.
func TestSymbolFallback(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			Symbol(c, "no.such.symbol").Size(20).Color(ui.RGB(255, 0, 0)).Fallback(box)
		})
	}, 40, 40)
	tt.Frame()
	r, ok := tt.Find("no.such.symbol")
	if !ok {
		t.Fatal("no.such.symbol not found")
	}
	img := tt.Image()
	s := img.Rect.Dx() / 40
	px := img.RGBAAt(int(r.X+r.W/2)*s, int(r.Y+r.H/2)*s)
	if px.R < 200 || px.G > 60 || px.B > 60 {
		t.Errorf("the fallback's middle is %v, want red", px)
	}
}
