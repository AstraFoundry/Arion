package ui_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestNoShadow checks that NoShadow takes off the outer and inner shadows
// added before it, as box-shadow: none, and that a shadow added after it
// shows alone.
func TestNoShadow(t *testing.T) {
	red, blue := ui.RGB(255, 0, 0), ui.RGB(0, 0, 255)
	white := color.RGBA{255, 255, 255, 255}
	// A box at (50, 50) to (150, 150): (45, 100) is 5 DIPs outside it, and
	// (55, 100) 5 DIPs inside.
	render := func(style func(b ui.Element)) *image.RGBA {
		return ui.Render(func(c *ui.Context) {
			style(ui.Box(c).Margin(50).Size(100, 100).Background(ui.RGB(255, 255, 255)))
		}, 200, 200, 1)
	}
	plain := render(func(ui.Element) {}).RGBAAt(45, 100)

	img := render(func(b ui.Element) { b.Shadow(0, 0, 0, 10, red).InsetShadow(0, 0, 0, 10, red) })
	if c := img.RGBAAt(45, 100); c != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("outside, with a shadow: %v, want red", c)
	}
	if c := img.RGBAAt(55, 100); c != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("inside, with an inner shadow: %v, want red", c)
	}

	img = render(func(b ui.Element) { b.Shadow(0, 0, 0, 10, red).InsetShadow(0, 0, 0, 10, red).NoShadow() })
	if c := img.RGBAAt(45, 100); c != plain {
		t.Errorf("outside, with the shadows taken off: %v, want the background's %v", c, plain)
	}
	if c := img.RGBAAt(55, 100); c != white {
		t.Errorf("inside, with the shadows taken off: %v, want white", c)
	}

	// A narrower shadow after it shows, and the wider one before it not
	// beyond the new one's reach.
	img = render(func(b ui.Element) { b.Shadow(0, 0, 0, 10, red).NoShadow().Shadow(0, 0, 0, 5, blue) })
	if c := img.RGBAAt(47, 100); c != (color.RGBA{0, 0, 255, 255}) {
		t.Errorf("outside, with a shadow after: %v, want blue", c)
	}
	if c := img.RGBAAt(42, 100); c != plain {
		t.Errorf("beyond the shadow after: %v, want the background's %v", c, plain)
	}
}
