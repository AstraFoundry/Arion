package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo"
)

// menuIcon makes a menu icon of w by h pixels in one color.
func menuIcon(t *testing.T, w, h int, c color.NRGBA) *mygo.MenuIcon {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	ic, err := mygo.NewMenuIcon(b.Bytes(), false)
	if err != nil {
		t.Fatal(err)
	}
	return ic
}
