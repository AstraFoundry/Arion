//go:build !js

package mygo

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/egoist/mygo/ui"
)

// svgPixels decodes an icon's PNG image.
func svgPixels(t *testing.T, ic *MenuIcon) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(ic.p.PNG))
	if err != nil {
		t.Fatalf("the icon's image is no PNG image: %v", err)
	}
	return img
}

func TestNewMenuIconSVG(t *testing.T) {
	// A red square in the left half of a viewBox twice as wide as high.
	wide := ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 16"><rect width="16" height="16" fill="#ff0000"/></svg>`))
	ic, err := NewMenuIconSVG(wide, false)
	if err != nil {
		t.Fatal(err)
	}
	if ic.p.Template {
		t.Error("an icon in its own colors is a template")
	}
	img := svgPixels(t, ic)
	if b := img.Bounds(); b.Dx() != menuIconSVGPixels || b.Dy() != menuIconSVGPixels/2 {
		t.Fatalf("the image is %v, want %d by %d, keeping the SVG's aspect", b, menuIconSVGPixels, menuIconSVGPixels/2)
	}
	if r, g, b, a := img.At(menuIconSVGPixels/4, menuIconSVGPixels/4).RGBA(); r>>8 < 250 || g>>8 > 5 || b>>8 > 5 || a>>8 < 250 {
		t.Errorf("the square is (%d, %d, %d, %d), want red", r>>8, g>>8, b>>8, a>>8)
	}
	if _, _, _, a := img.At(3*menuIconSVGPixels/4, menuIconSVGPixels/4).RGBA(); a != 0 {
		t.Errorf("the right half has alpha %d, want none", a>>8)
	}

	// A tall one, as a template: the shapes' coverage.
	tall := ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="16"><circle cx="4" cy="8" r="4" fill="currentColor"/></svg>`))
	tmpl, err := NewMenuIconSVG(tall, true)
	if err != nil {
		t.Fatal(err)
	}
	if !tmpl.p.Template {
		t.Error("a template icon is not a template")
	}
	img = svgPixels(t, tmpl)
	if b := img.Bounds(); b.Dx() != menuIconSVGPixels/2 || b.Dy() != menuIconSVGPixels {
		t.Errorf("the tall image is %v, want %d by %d", b, menuIconSVGPixels/2, menuIconSVGPixels)
	}
	if _, _, _, a := img.At(menuIconSVGPixels/4, menuIconSVGPixels/2).RGBA(); a>>8 < 250 {
		t.Errorf("the circle's middle has alpha %d, want opaque", a>>8)
	}

	// Icons of the same SVG share their image, which backends decode once,
	// as those of SVGs parsed again.
	again, err := NewMenuIconSVG(wide, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.p != ic.p {
		t.Error("two icons of the same SVG do not share their image")
	}
	again, err = NewMenuIconSVG(ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 16"><rect width="16" height="16" fill="#ff0000"/></svg>`)), false)
	if err != nil {
		t.Fatal(err)
	}
	if again.p != ic.p {
		t.Error("two icons of an SVG parsed again do not share their image")
	}
	if snap := NewMenu([]*MenuItem{{Label: "Flag", Icon: ic}}).snapshot(); snap.Items[0].Icon != ic.p {
		t.Error("the item's snapshot does not carry the SVG's icon")
	}

	for _, s := range []SVGImage{nil, (*ui.SVG)(nil), ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0"/>`))} {
		if ic, err := NewMenuIconSVG(s, false); err == nil || ic != nil {
			t.Errorf("NewMenuIconSVG(%v) = %v, %v, want an error", s, ic, err)
		}
	}
}

// TestMenuIconSVGInADrawnMenu shows an SVG's template icon in a menu that
// native UI draws, in its item's IconColor.
func TestMenuIconSVGInADrawnMenu(t *testing.T) {
	square := ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16" fill="#333"/></svg>`))
	ic, err := NewMenuIconSVG(square, true)
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			ui.Text(c, "Row").PopoverMenu(func(m *ui.Menu) {
				m.Item("Trash").Icon(ic).IconColor(ui.RGB(220, 20, 20))
			})
		})
	}, 300, 200)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	row, ok := tt.Find("Trash")
	if !ok {
		t.Fatal("no menu")
	}
	// The icon is after the row's padding of 8, 16 wide.
	r, g, b, _ := tt.Image().At(int(row.X)+8+8, int(row.Y+row.H/2)).RGBA()
	if r>>8 < 180 || g>>8 > 70 || b>>8 > 70 {
		t.Errorf("the icon is (%d, %d, %d), want red", r>>8, g>>8, b>>8)
	}
}
