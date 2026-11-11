package menuicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// A PNG's straight alpha comes out premultiplied, in BGRA's order.
func TestDecodePremultiplies(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 128, B: 0, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 128})
	img, err := Decode(encode(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if img.W != 2 || img.H != 1 {
		t.Fatalf("size = %dx%d, want 2x1", img.W, img.H)
	}
	if got := img.Px[0]; got != 0xFFFF8000 {
		t.Errorf("opaque orange = %#08x, want 0xffff8000", got)
	}
	if got := img.Px[1]; got != 0x80808080 {
		t.Errorf("half white = %#08x, want 0x80808080", got)
	}
}

func TestDecodeRejectsOtherData(t *testing.T) {
	if _, err := Decode([]byte("not a png")); err == nil {
		t.Error("Decode of text succeeded")
	}
}

// Shrinking averages the pixels each covers, in premultiplied space: a
// transparent pixel does not darken an opaque one.
func TestScaleAverages(t *testing.T) {
	img := &Image{W: 4, H: 2, Px: []uint32{
		0xFFFF0000, 0x00000000, 0xFF0000FF, 0xFF0000FF,
		0xFFFF0000, 0x00000000, 0xFF0000FF, 0xFF0000FF,
	}}
	got := img.Scale(2, 1)
	if got.W != 2 || got.H != 1 {
		t.Fatalf("size = %dx%d", got.W, got.H)
	}
	if got.Px[0] != 0x80800000 {
		t.Errorf("red over transparent = %#08x, want half red, half covered (0x80800000)", got.Px[0])
	}
	if got.Px[1] != 0xFF0000FF {
		t.Errorf("blue = %#08x, want 0xff0000ff", got.Px[1])
	}
}

// A size that does not divide the image's weighs the pixels split between
// two by how much of each is covered.
func TestScaleFractional(t *testing.T) {
	img := &Image{W: 3, H: 1, Px: []uint32{0xFF000000, 0xFFFFFFFF, 0xFF000000}}
	got := img.Scale(2, 1)
	for i, p := range got.Px {
		if p>>24 != 0xFF || p&0xFF < 0x50 || p&0xFF > 0x56 {
			t.Errorf("pixel %d = %#08x, want opaque, a third white", i, p)
		}
	}
	if same := img.Scale(3, 1); same != img {
		t.Error("Scale to the image's own size copied it")
	}
	if up := img.Scale(6, 2); up.Px[0] != 0xFF000000 || up.Px[2] != 0xFFFFFFFF || up.Px[6+5] != 0xFF000000 {
		t.Errorf("growing = %#08x", up.Px)
	}
}

func TestTintKeepsAlpha(t *testing.T) {
	img := &Image{W: 2, H: 1, Px: []uint32{0xFF000000, 0x80000000}}
	got := img.Tint(0xFFFFFF)
	if got.Px[0] != 0xFFFFFFFF || got.Px[1] != 0x80808080 {
		t.Errorf("white template = %#08x, want 0xffffffff and 0x80808080", got.Px)
	}
	if img.Px[0] != 0xFF000000 {
		t.Error("Tint changed the image")
	}
}

func TestOver(t *testing.T) {
	for _, c := range []struct {
		src, dst uint32
		opacity  float64
		want     uint32
	}{
		{0xFFFF0000, 0x00FFFFFF, 1, 0xFF0000},   // opaque covers
		{0x00000000, 0x00123456, 1, 0x123456},   // transparent leaves
		{0x80800000, 0x00000000, 1, 0x800000},   // half red over black
		{0x80800000, 0x00FFFFFF, 1, 0xFF7F7F},   // half red over white
		{0xFF0000FF, 0x00FF0000, 0.5, 0x800080}, // half opacity
	} {
		if got := Over(c.src, c.dst, c.opacity); got != c.want {
			t.Errorf("Over(%#08x, %#06x, %v) = %#06x, want %#06x", c.src, c.dst, c.opacity, got, c.want)
		}
	}
}
