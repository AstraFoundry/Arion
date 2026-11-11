//go:build darwin

package sfsymbols

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestImage checks that a symbol the app names draws, at the
// size asked and with the color asked, and that one this macOS has not
// says so rather than drawing nothing.
func TestImage(t *testing.T) {
	for _, name := range []string{"xmark", "gearshape", "sidebar.leading", "terminal"} {
		src, ok := Image(name, 48, ui.Hex("#ffffff"), Options{})
		if !ok {
			t.Fatalf("%s: the system draws no such symbol", name)
		}
		if w, h := src.(*ui.Bitmap).Size(); w != 48 || h != 48 {
			t.Fatalf("%s: %dx%d pixels, want 48x48", name, w, h)
		}
	}
	if _, ok := Image("no.symbol.of.that.name", 48, ui.Hex("#ffffff"), Options{}); ok {
		t.Fatal("a name the system has not drew as a symbol")
	}
}

// TestCoverage checks that a symbol covers some of its box and not
// all of it, and that tint keeps the premultiplied alpha of the shape.
func TestCoverage(t *testing.T) {
	coverage, ok, _ := rasterize("xmark", 64, false, Regular)
	if !ok {
		t.Fatal("xmark drew nothing")
	}
	lit := 0
	for i := range 64 * 64 {
		if coverage.Pix[i*4+3] != 0 {
			lit++
		}
	}
	if lit == 0 || lit == 64*64 {
		t.Fatalf("%d of %d pixels lit", lit, 64*64)
	}
	col := ui.RGBA(0xff, 0x40, 0x20, 0.5)
	tinted := tint(coverage, col)
	for i := range 64 * 64 {
		a, r := tinted.Pix[i*4+3], tinted.Pix[i*4+0]
		if r == 0 && a != 0 && coverage.Pix[i*4+3] != 0 {
			t.Fatal("a lit pixel lost its color")
		}
		if r > a {
			t.Fatalf("pixel %d: %d red over %d alpha, not premultiplied", i, r, a)
		}
	}
}

// TestFillsItsBox checks that a symbol's ink fills the box rather
// than the alignment margins of the image it comes in, and that it sits
// centered in it.
func TestFillsItsBox(t *testing.T) {
	for _, name := range []string{"xmark", "gearshape", "rectangle.split.2x1", "chevron.down", "square.and.arrow.up"} {
		coverage, ok, _ := rasterize(name, 48, false, Regular)
		if !ok {
			t.Fatalf("%s drew nothing", name)
		}
		b := inkOf(coverage)
		if big := max(b.Dx(), b.Dy()); big < 40 || big > 44 {
			t.Fatalf("%s: ink is %dx%d, want about 41x41 in a 48 box", name, b.Dx(), b.Dy())
		}
		cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
		// Margins on either side within a pixel: centered, not shifted by
		// the image's uneven alignment margins.
		if l, r, tp, bt := b.Min.X, 48-b.Max.X, b.Min.Y, 48-b.Max.Y; max(l-r, r-l) > 1 || max(tp-bt, bt-tp) > 1 {
			t.Fatalf("%s: margins %d left, %d right, %d top, %d bottom: want even", name, l, r, tp, bt)
		}
		if max(cx-24, 24-cx) > 1 || max(cy-24, 24-cy) > 1 {
			t.Fatalf("%s: ink centered at %d,%d, want 24,24", name, cx, cy)
		}
	}
}

// inkOf returns the box of the lit pixels of coverage.
func inkOf(coverage *image.RGBA) image.Rectangle {
	w, h := coverage.Rect.Dx(), coverage.Rect.Dy()
	minX, minY, maxX, maxY := w, h, -1, -1
	for y := range h {
		for x := range w {
			if coverage.Pix[y*coverage.Stride+x*4+3] != 0 {
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
			}
		}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// TestNatural checks that a symbol drawn at its natural size is smaller
// than one that fills its box.
func TestNatural(t *testing.T) {
	filled, ok, _ := rasterize("xmark", 64, false, Regular)
	if !ok {
		t.Fatal("xmark drew nothing")
	}
	natural, ok, _ := rasterize("xmark", 64, true, Regular)
	if !ok {
		t.Fatal("xmark drew nothing at its natural size")
	}
	if f, n := inkOf(filled), inkOf(natural); n.Dx() >= f.Dx() {
		t.Fatalf("natural ink %d wide, filled %d: want smaller", n.Dx(), f.Dx())
	}
}

// TestNaturalBalances checks that at their natural size a cross stays
// smaller than a wide rectangle, as SF Symbols balance them, while both
// fill their boxes alike otherwise.
func TestNaturalBalances(t *testing.T) {
	width := func(name string, natural bool) int {
		c, ok, _ := rasterize(name, 100, natural, Regular)
		if !ok {
			t.Fatalf("%s drew nothing", name)
		}
		return inkOf(c).Dx()
	}
	if x, r := width("xmark", true), width("rectangle.on.rectangle", true); x >= r*4/5 {
		t.Errorf("natural xmark %d wide, rectangle.on.rectangle %d: want the cross well smaller", x, r)
	}
	if r := width("rectangle.on.rectangle", true); r > 92 || r < 80 {
		t.Errorf("natural rectangle.on.rectangle %d wide in 100: want it to nearly fill", r)
	}
}

// TestRasterizeAlone checks that rasterize loads what it needs, so that a
// test calling it runs alone.
func TestRasterizeAlone(t *testing.T) {
	if _, ok, absent := rasterize("no.symbol.of.that.name", 32, false, Regular); ok || !absent {
		t.Errorf("a missing name: ok %v, absent %v; want false, true", ok, absent)
	}
}

// TestWeightOrder checks that the weights' order is their heaviness.
func TestWeightOrder(t *testing.T) {
	ws := []Weight{UltraLight, Thin, Light, Regular, Medium, Semibold, Bold, Heavy, Black}
	for i := 1; i < len(ws); i++ {
		if ws[i] <= ws[i-1] || ws[i].fontWeight() <= ws[i-1].fontWeight() {
			t.Errorf("%d after %d is not heavier", ws[i], ws[i-1])
		}
	}
	var zero Weight
	if zero != Regular {
		t.Error("the zero Weight is not Regular")
	}
}

// TestAvailable checks that Available agrees with Image.
func TestAvailable(t *testing.T) {
	if !Available() {
		t.Skip("no AppKit here")
	}
	if _, ok := Image("gearshape", 8, ui.Hex("#000000"), Options{}); !ok {
		t.Fatal("gearshape not drawn though symbols are available")
	}
}

// TestCacheBounded checks that a color animated over many frames keeps the
// caches at their limits, and draws the symbol's shape once.
func TestCacheBounded(t *testing.T) {
	first, ok := Image("gearshape", 24, ui.RGB(0, 0, 0), Options{})
	if !ok {
		t.Fatal("gearshape drew nothing")
	}
	if again, _ := Image("gearshape", 24, ui.RGB(0, 0, 0), Options{}); again != first {
		t.Error("the same symbol, size and color drew again")
	}
	for i := range tintLimit + 100 {
		Image("gearshape", 24, ui.RGB(uint8(i), uint8(i>>8), 0), Options{})
	}
	mu.Lock()
	n, s := tinted.len(), shapes.len()
	_, drawn := shapes.get(shapeKey{name: "gearshape", px: 24})
	mu.Unlock()
	if n > tintLimit || s > shapeLimit {
		t.Fatalf("%d tinted and %d shapes cached, past the limits %d and %d", n, s, tintLimit, shapeLimit)
	}
	if !drawn {
		t.Error("gearshape's shape dropped while it was in use")
	}
}

// TestWeight checks that a heavier weight draws more ink.
func TestWeight(t *testing.T) {
	ink := func(w Weight) int {
		coverage, ok, _ := rasterize("circle", 64, true, w)
		if !ok {
			t.Fatalf("circle at %d drew nothing", w)
		}
		n := 0
		for i := range len(coverage.Pix) / 4 {
			n += int(coverage.Pix[i*4+3])
		}
		return n
	}
	if light, black := ink(UltraLight), ink(Black); black <= light {
		t.Errorf("black ink %d, ultralight %d: want more", black, light)
	}
}

// TestFillVariant checks that a filled variant draws, and that a name
// missing once answers from the cache.
func TestFillVariant(t *testing.T) {
	if _, ok := Image("trash.fill", 32, ui.RGB(0, 0, 0), Options{}); !ok {
		t.Fatal("trash.fill drew nothing")
	}
	if _, ok := Image("gearshape.nope", 32, ui.RGB(0, 0, 0), Options{}); ok {
		t.Fatal("a missing name drew")
	}
	mu.Lock()
	_, absent := missing.get("gearshape.nope")
	mu.Unlock()
	if !absent {
		t.Error("a missing name was not remembered")
	}
}

// TestSymbolDraws checks that a Symbol paints the system's symbol in its
// color.
func TestSymbolDraws(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			Symbol(c, "square.fill").Size(30).Color(ui.RGB(0, 0, 255))
		})
	}, 40, 40)
	tt.Frame()
	img := tt.Image()
	blue := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+2] > 200 && img.Pix[i] < 60 {
			blue++
		}
	}
	if blue == 0 {
		t.Error("square.fill painted no blue")
	}
}

// TestNames checks that the system's catalog lists thousands of symbols,
// these among them, and that they draw.
func TestNames(t *testing.T) {
	list := Names()
	if len(list) < 1000 {
		t.Fatalf("%d names, want thousands", len(list))
	}
	has := map[string]bool{}
	for _, n := range list {
		has[n] = true
	}
	for _, n := range []string{"gearshape", "xmark", "trash.fill", "rectangle.on.rectangle"} {
		if !has[n] {
			t.Errorf("%s not listed", n)
		}
	}
	for _, n := range list[:50] {
		if _, ok := Image(n, 16, ui.RGB(0, 0, 0), Options{}); !ok {
			t.Errorf("listed %s does not draw", n)
		}
	}
	var available []string
	withPool(func() { available = readAvailable() })
	if len(available) < len(list) {
		t.Errorf("the availability list has %d names, fewer than the order's %d", len(available), len(list))
	}
}
