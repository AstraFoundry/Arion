//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func testIcon(t *testing.T, w, h int, c color.NRGBA, template bool) *platform.MenuIcon {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return &platform.MenuIcon{PNG: b.Bytes(), Template: template}
}

// An icon is fit to the box at each DPI, keeping its aspect, made once a
// size, and a template one takes the color asked for.
func TestIconAt(t *testing.T) {
	ic := testIcon(t, 60, 40, color.NRGBA{255, 0, 0, 255}, false)
	for _, c := range []struct {
		dpi  int
		w, h int
	}{{96, 16, 11}, {144, 24, 16}, {192, 32, 21}} {
		img := iconAt(ic, toPx(platform.MenuIconSize, c.dpi), 0xFFFFFF)
		if img == nil || img.W != c.w || img.H != c.h {
			t.Fatalf("at %d DPI = %+v, want %dx%d", c.dpi, img, c.w, c.h)
		}
		if img.Px[0] != 0xFFFF0000 {
			t.Errorf("a color icon was tinted: %#08x", img.Px[0])
		}
	}
	if iconAt(ic, 16, 0) != iconAt(ic, 16, 0x123456) {
		t.Error("an icon in color was made again for another tint")
	}
	tmpl := testIcon(t, 16, 16, color.NRGBA{0, 0, 0, 128}, true)
	if got := iconAt(tmpl, 16, 0xFFFFFF).Px[0]; got != 0x80808080 {
		t.Errorf("white template = %#08x, want 0x80808080", got)
	}
	if got := iconAt(tmpl, 16, 0x000000).Px[0]; got != 0x80000000 {
		t.Errorf("black template = %#08x, want 0x80000000", got)
	}
	bad := &platform.MenuIcon{PNG: []byte("nope")}
	if iconAt(bad, 16, 0) != nil {
		t.Error("an image that is no PNG gave pixels")
	}
}

// drawIcon centers an icon in its box, over the bitmap's pixels.
func TestDrawIcon(t *testing.T) {
	ic := testIcon(t, 4, 2, color.NRGBA{0, 0, 255, 255}, false)
	img := iconAt(ic, 4, 0)
	px := make([]uint32, 8*6)
	for i := range px {
		px[i] = 0xFFFFFF
	}
	drawIcon(px, 8, img, 2, 1, 4, 1)
	for y := range 6 {
		for x := range 8 {
			want := uint32(0xFFFFFF)
			if x >= 2 && x < 6 && y >= 2 && y < 4 {
				want = 0x0000FF
			}
			if px[y*8+x] != want {
				t.Fatalf("pixel (%d, %d) = %#06x, want %#06x", x, y, px[y*8+x], want)
			}
		}
	}
	// Past the bitmap's edges, nothing is drawn.
	drawIcon(px, 8, img, 6, 5, 4, 1)
}
