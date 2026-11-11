package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The gallery shows the names it is given, and its options change them.
func TestGallery(t *testing.T) {
	g := newGallery(nil)
	tt := ui.NewTester(g.view, 1000, 3000)
	for _, name := range fewNames {
		if _, ok := tt.Find(name); !ok {
			t.Errorf("%s: not shown", name)
		}
	}
	for _, click := range []string{"48", "Natural size", "Black", "Fill"} {
		if err := tt.Click(click); err != nil {
			t.Fatal(err)
		}
	}
	if g.size != 3 || !g.natural || g.weight != 3 || !g.fill {
		t.Errorf("size %d, natural %v, weight %d, fill %v: want 3, true, 3, true", g.size, g.natural, g.weight, g.fill)
	}
}

// The search keeps the names holding every word of it.
func TestSearch(t *testing.T) {
	g := newGallery([]string{"trash", "trash.fill", "trash.circle", "gearshape", "xmark.circle"})
	g.query = "circle tr"
	g.filter()
	if len(g.shown) != 1 || g.shown[0] != "trash.circle" {
		t.Errorf("shown %v, want [trash.circle]", g.shown)
	}
	g.query = ""
	g.filter()
	if len(g.shown) != 5 {
		t.Errorf("%d shown with no query, want 5", len(g.shown))
	}
}

// A long name ends with an ellipsis within its cell rather than running
// into the gap and the next cell: the gap between the first two cells
// stays the window's background below their symbols, where the names are.
func TestLongNameFits(t *testing.T) {
	long := "zl.button.roundedtop.horizontal.fill"
	g := newGallery([]string{long, long, long})
	tt := ui.NewTester(g.view, 1000, 400)
	tt.Frame()
	sym, ok := tt.Find(long) // the first cell's symbol, labeled with its name
	if !ok {
		t.Fatal("the long name's symbol is not shown")
	}
	img := tt.Image()
	s := img.Rect.Dx() / 1000
	// The first cell's right edge: the first column right of the symbol
	// whose pixel at its middle is the background.
	bg := img.RGBAAt(990*s, 390*s)
	y := int(sym.Y+sym.H/2) * s
	gap := int(sym.X+sym.W) * s
	for gap < img.Rect.Dx() && img.RGBAAt(gap, y) != bg {
		gap++
	}
	if gap >= 500*s {
		t.Fatal("no gap found after the first cell")
	}
	for y := int(sym.Y+sym.H) * s; y < int(sym.Y+sym.H+50)*s; y++ {
		for x := gap + s; x < gap+5*s; x++ {
			if px := img.RGBAAt(x, y); px != bg {
				t.Fatalf("pixel %d,%d in the gap is %v, not the background %v: the name overflows", x/s, y/s, px, bg)
			}
		}
	}
}
