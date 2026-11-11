package ui

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"runtime"
	"testing"
	"weak"

	"golang.org/x/image/bmp"
)

// stripedImage returns a w×h image of vertical stripes, black and white, each
// n pixels wide, shifted by off pixels.
func stripedImage(w, h, n, off int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := uint8(0)
			if (x+off)/n%2 == 1 {
				c = 255
			}
			img.Set(x, y, color.RGBA{c, c, c, 255})
		}
	}
	return img
}

func TestBitmapShowsSmallerSmoothly(t *testing.T) {
	// Stripes of 4 pixels, whose middles a GPU's samples, one in 8 pixels,
	// fall on: all white.
	bm := NewBitmap(stripedImage(512, 512, 4, 2))
	tt := coreNewTester(func(c *context) {
		coreImage(c, bm).Size(64, 64).Fit(FillBox)
	}, 100, 100)
	// Every pixel shows as many black stripes as white: gray, not stripes
	// of its own.
	img := tt.Image()
	lo, hi := uint8(255), uint8(0)
	for y := 4; y < 60; y++ {
		for x := 4; x < 60; x++ {
			v := img.RGBAAt(x, y).R
			lo, hi = min(lo, v), max(hi, v)
		}
	}
	if lo < 100 || hi > 160 {
		t.Errorf("the stripes, 8 times smaller, range from %d to %d", lo, hi)
	}
	if len(bm.levels) != 3 {
		t.Errorf("%d levels made", len(bm.levels))
	}
	// At its size, it is itself.
	if got := bm.smaller(1); got != bm.img {
		t.Error("a level for the bitmap at its size")
	}
}

func TestBitmapPixelated(t *testing.T) {
	// Stripes of one pixel, 8 times as large: a pixelated bitmap shows
	// stripes of 8 pixels, black or white, a smooth one ramps between.
	for _, pixelated := range []bool{false, true} {
		bm := NewBitmap(stripedImage(4, 4, 1, 0))
		if pixelated {
			bm.Pixelated()
		}
		tt := coreNewTester(func(c *context) {
			coreImage(c, bm).Size(32, 32).Fit(FillBox)
		}, 40, 40)
		img := tt.Image()
		grays := 0
		for x := range 32 {
			if v := img.RGBAAt(x, 16).R; v > 10 && v < 245 {
				grays++
			}
		}
		if pixelated && grays != 0 {
			t.Errorf("a pixelated bitmap shows %d grays between its pixels", grays)
		}
		if !pixelated && grays < 8 {
			t.Errorf("a smooth bitmap shows %d grays between its pixels", grays)
		}
		if pixelated && (img.RGBAAt(4, 16).R != 0 || img.RGBAAt(12, 16).R != 255) {
			t.Errorf("pixels %d and %d", img.RGBAAt(4, 16).R, img.RGBAAt(12, 16).R)
		}
	}
}

func TestImageRegion(t *testing.T) {
	// A sheet of four 2×2 parts: red, green, blue and white.
	sheet := image.NewRGBA(image.Rect(0, 0, 4, 4))
	colors := [4]color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 255, 255}}
	for y := range 4 {
		for x := range 4 {
			sheet.Set(x, y, colors[y/2*2+x/2])
		}
	}
	// The left half of the green part red, to see it flip.
	sheet.Set(2, 0, colors[0])
	sheet.Set(2, 1, colors[0])
	bm := NewBitmap(sheet).Pixelated()
	green, white := Rect{X: 2, W: 2, H: 2}, Rect{X: 2, Y: 2, W: 2, H: 2}
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(80, 20).Draw(func(p *Painter, r Rect) {
			p.ImageRegion(bm, green, Rect{X: r.X, Y: r.Y, W: 20, H: 20}, ImageOptions{})
			p.ImageRegion(bm, green, Rect{X: r.X + 30, Y: r.Y, W: 20, H: 20}, ImageOptions{FlipX: true})
			p.ImageRegion(bm, white, Rect{X: r.X + 60, Y: r.Y, W: 20, H: 20}, ImageOptions{Tint: RGB(255, 0, 0)})
		})
	}, 80, 20)
	img := tt.Image()
	px := func(x, y int) color.RGBA { return img.RGBAAt(x, y) }
	for _, c := range []struct {
		x    int
		want color.RGBA
	}{
		{5, colors[0]}, {15, colors[1]}, // red then green
		{35, colors[1]}, {45, colors[0]}, // flipped: green then red
		{65, colors[0]}, {75, colors[0]}, // white tinted red
	} {
		if got := px(c.x, 10); got != c.want {
			t.Errorf("pixel %d = %v, want %v", c.x, got, c.want)
		}
	}
	// Nothing of the blue part below the green one shows at its edge.
	if got := px(15, 19); got != colors[1] {
		t.Errorf("the green part's bottom edge = %v", got)
	}
}

func TestHalve(t *testing.T) {
	// 3×3 of values 0, 40, 80 by column: an odd column counts twice.
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	for y := range 3 {
		for x := range 3 {
			img.Set(x, y, color.RGBA{uint8(40 * x), 0, 0, 255})
		}
	}
	h := halve(NewBitmap(img).img)
	if h.W != 2 || h.H != 2 {
		t.Fatalf("%d×%d", h.W, h.H)
	}
	if r0, r1 := h.Pix[0], h.Pix[4]; r0 != 20 || r1 != 80 {
		t.Errorf("columns %d and %d", r0, r1)
	}
}

// withOrientation returns a JPEG with an EXIF segment giving orientation
// o, in byte order bo.
func withOrientation(t *testing.T, img image.Image, o uint16, bo binary.ByteOrder) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	tiff := make([]byte, 8+2+12+4)
	if bo == binary.LittleEndian {
		copy(tiff, "II")
	} else {
		copy(tiff, "MM")
	}
	bo.PutUint16(tiff[2:], 42)
	bo.PutUint32(tiff[4:], 8)
	bo.PutUint16(tiff[8:], 1)       // one entry
	bo.PutUint16(tiff[10:], 0x0112) // Orientation
	bo.PutUint16(tiff[12:], 3)      // SHORT
	bo.PutUint32(tiff[14:], 1)
	bo.PutUint16(tiff[18:], o)
	app1 := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(2+len(app1)))
	data := buf.Bytes()
	return append(append(append([]byte{}, data[:2]...), append(seg, app1...)...), data[2:]...)
}

func TestDecodeBitmapTurnsPhotosUpright(t *testing.T) {
	// Stored on its side, 32×16: red on the left, blue on the right.
	img := image.NewRGBA(image.Rect(0, 0, 32, 16))
	for y := range 16 {
		for x := range 32 {
			c := color.RGBA{255, 0, 0, 255}
			if x >= 16 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	red := func(b *Bitmap, x, y int) bool {
		p := b.img.Pix[4*(y*b.w+x):]
		return p[0] > 200 && p[2] < 60
	}
	for _, c := range []struct {
		o          uint16
		bo         binary.ByteOrder
		w, h       int
		redX, redY int // a pixel that is red
	}{
		{1, binary.BigEndian, 32, 16, 4, 8},
		{3, binary.LittleEndian, 32, 16, 28, 8}, // upside down
		{6, binary.BigEndian, 16, 32, 8, 4},     // turned right: red on top
		{8, binary.LittleEndian, 16, 32, 8, 28}, // turned left: red at the bottom
	} {
		b, err := DecodeBitmap(withOrientation(t, img, c.o, c.bo))
		if err != nil {
			t.Fatal(err)
		}
		if b.w != c.w || b.h != c.h || !red(b, c.redX, c.redY) {
			t.Errorf("orientation %d: %d×%d, red at (%d, %d): %v", c.o, b.w, b.h, c.redX, c.redY, red(b, c.redX, c.redY))
		}
	}
	if exifOrientation([]byte("not a jpeg")) != 0 {
		t.Error("an orientation in what is no JPEG")
	}
}

func TestDecodeBitmapOfWebPAndBMP(t *testing.T) {
	webp, _ := base64.StdEncoding.DecodeString("UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAwA0JaQAA3AA/vuUAAA=")
	b, err := DecodeBitmap(webp)
	if err != nil || b.w != 1 || b.h != 1 {
		t.Errorf("WebP: %v", err)
	}
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, stripedImage(6, 4, 1, 0)); err != nil {
		t.Fatal(err)
	}
	if b, err = DecodeBitmap(buf.Bytes()); err != nil || b.w != 6 || b.h != 4 || b.img.Pix[4] != 255 {
		t.Errorf("BMP: %v", err)
	}
}

func TestDecodeImageOfBitmapsAndSVGs(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, stripedImage(6, 4, 1, 0)); err != nil {
		t.Fatal(err)
	}
	src, err := DecodeImage(buf.Bytes())
	if b, ok := src.(*Bitmap); err != nil || !ok || b.w != 6 || b.h != 4 {
		t.Errorf("PNG: %T, %v", src, err)
	}
	for _, data := range []string{
		`<svg width="30" height="20"/>`,
		"\xef\xbb\xbf \n\t<?xml version=\"1.0\"?>\n<svg viewBox=\"0 0 30 20\"/>", // a BOM and spaces
		`<!DOCTYPE svg><!-- a logo --><svg width="30" height="20"/>`,
	} {
		src, err := DecodeImage([]byte(data))
		if s, ok := src.(*SVG); err != nil || !ok || s.w != 30 || s.h != 20 {
			t.Errorf("%q: %T, %v", data, src, err)
		}
	}
	for _, data := range []string{"", "not a picture \x00", `<html><body/></html>`, `<svg width="0" height="0"/>`} {
		if src, err := DecodeImage([]byte(data)); err == nil || src != nil {
			t.Errorf("%q: %T, no error", data, src)
		}
	}
}

func TestImageSize(t *testing.T) {
	if w, h := ImageSize(NewBitmap(image.NewRGBA(image.Rect(0, 0, 6, 4)))); w != 6 || h != 4 {
		t.Errorf("a bitmap is %v×%v", w, h)
	}
	if w, h := ImageSize(MustParseSVG([]byte(`<svg viewBox="0 0 30 20"/>`))); w != 30 || h != 20 {
		t.Errorf("an SVG is %v×%v", w, h)
	}
	if w, h := ImageSize(nil); w != 0 || h != 0 {
		t.Errorf("nil is %v×%v", w, h)
	}
	if w, h := ImageSize((*Bitmap)(nil)); w != 0 || h != 0 {
		t.Errorf("a nil bitmap is %v×%v", w, h)
	}
}

// BenchmarkHalvePhoto halves a photo of 4000×3000 pixels, a step of
// drawing an image smaller than it is.
func BenchmarkHalvePhoto(b *testing.B) {
	img := NewBitmap(image.NewRGBA(image.Rect(0, 0, 4000, 3000))).img
	b.ResetTimer()
	for range b.N {
		halve(img)
	}
}

// heapInuse returns the bytes the heap has in use, after collecting.
func heapInuse() uint64 {
	runtime.GC()
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return s.HeapInuse
}

// decodedStripes returns a bitmap DecodeBitmap made of stripedImage's
// stripes as a PNG, which can release its pixels.
func decodedStripes(t *testing.T, w, h, n, off int) *Bitmap {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, stripedImage(w, h, n, off)); err != nil {
		t.Fatal(err)
	}
	b, err := DecodeBitmap(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// cloneImage returns a copy of img's pixels, which later frames do not
// overwrite.
func cloneImage(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	copy(out.Pix, img.Pix)
	return out
}

// TestReleasePixelsDropsThePixels checks issue #664: a bitmap whose pixels
// a GPU renderer uploaded can drop its CPU copies, base and levels, and
// keeps the source to re-make them from, which the CPU renderer's draw
// brings back, and a renderer's upload drops again.
func TestReleasePixelsDropsThePixels(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, stripedImage(2048, 2048, 64, 0)); err != nil {
		t.Fatal(err)
	}
	b, err := DecodeBitmap(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	b.smaller(8) // levels for drawing it smaller
	if b.img.Pix == nil || len(b.levels) != 3 {
		t.Fatal("the bitmap has no pixels and levels to drop")
	}
	before := heapInuse()
	b.ReleasePixels()
	if got := heapInuse(); got > before-12<<20 {
		t.Fatalf("releasing the pixels of 20 MiB left %d MiB of %d in use", got>>20, before>>20)
	}
	if b.img.Pix != nil {
		t.Error("the base kept its pixels")
	}
	for i, lvl := range b.levels {
		if lvl.Pix != nil {
			t.Errorf("level %d kept its pixels", i)
		}
	}
	if w, h := b.Size(); w != 2048 || h != 2048 {
		t.Errorf("size %d×%d after releasing", w, h)
	}
	if pix := b.img.Pixels(); pix == nil {
		t.Error("the pixels did not come back for the CPU renderer")
	}
	b.img.Uploaded()
	if b.img.Pix != nil {
		t.Error("the pixels did not go again after the upload")
	}
	runtime.KeepAlive(b)
}

// TestReleasedBitmapDrawsTheSame checks that a released bitmap still
// draws as itself: the CPU renderer re-creates its pixels from what the
// bitmap was made of.
func TestReleasedBitmapDrawsTheSame(t *testing.T) {
	bm := decodedStripes(t, 300, 200, 30, 0)
	tt := coreNewTester(func(c *context) {
		coreImage(c, bm).Size(150, 100).Fit(FillBox)
	}, 200, 150)
	want := cloneImage(tt.Image())
	bm.ReleasePixels()
	// The CPU renderer redraws what changed since; a size away and back
	// draws everything, released pixels included.
	tt.SetSize(220, 170)
	tt.SetSize(200, 150)
	got := tt.Image()
	if !bytes.Equal(want.Pix, got.Pix) {
		t.Fatal("a released bitmap draws other pixels")
	}
}

// TestReleasedBitmapDrawsSmallerFromRebuiltLevels checks that a released
// bitmap re-makes the level a smaller drawing needs from its source,
// while the base's pixels stay dropped.
func TestReleasedBitmapDrawsSmallerFromRebuiltLevels(t *testing.T) {
	bm := decodedStripes(t, 512, 512, 4, 2)
	tt := coreNewTester(func(c *context) {
		coreImage(c, bm).Size(64, 64).Fit(FillBox)
	}, 100, 100)
	if len(bm.levels) != 3 {
		t.Fatalf("%d levels made before releasing", len(bm.levels))
	}
	want := cloneImage(tt.Image())
	bm.ReleasePixels()
	// A size away and back draws everything again.
	tt.SetSize(120, 90)
	tt.SetSize(100, 100)
	if got := tt.Image(); !bytes.Equal(want.Pix, got.Pix) {
		t.Fatal("a released bitmap draws other pixels smaller")
	}
	if bm.img.Pix != nil {
		t.Error("the base's pixels came back with the level")
	}
	if len(bm.levels) != 3 {
		t.Errorf("%d levels after releasing", len(bm.levels))
	}
}

func TestBitmapUpdate(t *testing.T) {
	solid := func(w, h int, c color.RGBA) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
		}
		return img
	}
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	bm := NewBitmap(solid(64, 64, red))
	tt := coreNewTester(func(c *context) {
		coreImage(c, bm).Size(16, 16).Fit(FillBox)
	}, 40, 40)
	if got := tt.Image().RGBAAt(8, 8); got != red {
		t.Fatalf("before: %v", got)
	}
	level := bm.smaller(4) // a level, which the new picture redoes
	img, ver, levelVer := bm.img, bm.img.Version(), level.Version()
	next := solid(64, 64, blue)
	bm.Update(next)
	// The same size keeps the image, so the GPU keeps its texture.
	if bm.img != img || bm.img.Version() == ver || &bm.img.Pix[0] != &next.Pix[0] {
		t.Error("an update of the same size made another image, or copied")
	}
	// Its levels too, made again from the new picture as they show.
	if got := bm.smaller(4); got != level || got.Version() == levelVer || got.Pix[2] != 255 || got.Pix[0] != 0 {
		t.Errorf("level after the update: same %v, version %d → %d, pixel %v", got == level, levelVer, got.Version(), got.Pix[:4])
	}
	if v := level.Version(); bm.smaller(4) != level || level.Version() != v {
		t.Error("an unchanged picture's level was made again")
	}
	tt.Frame()
	if got := tt.Image().RGBAAt(8, 8); got != blue {
		t.Errorf("after: %v", got)
	}
	bm.Update(solid(32, 48, red))
	if w, h := bm.Size(); w != 32 || h != 48 || bm.img == img {
		t.Errorf("resized to %dx%d", w, h)
	}
}

func TestDecodeBitmapThumbnailPreservesOrientationAndBounds(t *testing.T) {
	pixels := stripedImage(128, 64, 16, 0)
	data := withOrientation(t, pixels, 6, binary.BigEndian)
	thumb, err := DecodeBitmapThumbnail(data, 32)
	if err != nil {
		t.Fatal(err)
	}
	if thumb.w != 16 || thumb.h != 32 {
		t.Fatalf("oriented thumbnail = %dx%d", thumb.w, thumb.h)
	}
	small, err := DecodeBitmapThumbnail(data, 256)
	if err != nil || small.w != 64 || small.h != 128 {
		t.Fatal("thumbnail upscaled its source", err)
	}
	for _, edge := range []int{0, -1} {
		if _, err := DecodeBitmapThumbnail(data, edge); err == nil {
			t.Fatal("invalid bound accepted")
		}
	}
	if _, err := DecodeBitmapThumbnail([]byte("corrupt"), 32); err == nil {
		t.Fatal("corrupt image accepted")
	}
}

// TestReleasedBitmapUpdates checks that Update shows another picture in a
// released bitmap, smaller too, from levels whose pixels it dropped, and
// that the bitmap keeps the new picture's pixels, which nothing else
// makes again.
func TestReleasedBitmapUpdates(t *testing.T) {
	bm := decodedStripes(t, 512, 512, 4, 2)
	tt := coreNewTester(func(c *context) {
		coreImage(c, bm).Size(64, 64).Fit(FillBox)
	}, 100, 100)
	bm.ReleasePixels()
	next := stripedImage(512, 512, 4, 3)
	bm.Update(next)
	tt.Frame()
	got := cloneImage(tt.Image())
	ref := NewBitmap(next)
	want := coreNewTester(func(c *context) {
		coreImage(c, ref).Size(64, 64).Fit(FillBox)
	}, 100, 100).Image()
	if !bytes.Equal(want.Pix, got.Pix) {
		t.Fatal("an updated released bitmap draws other pixels")
	}
	if bm.img.Restore != nil || bm.img.Dropped != nil {
		t.Error("the updated bitmap still drops its pixels")
	}
	bm.ReleasePixels() // nothing to make them from: it keeps them
	if bm.img.Pix == nil {
		t.Error("an updated bitmap dropped the pixels nothing makes again")
	}
}

// TestReleasePixelsKeepsWhatCannotComeBack checks that a bitmap NewBitmap
// made of an image nothing else keeps, or of an image type of the app's,
// which leave nothing to re-make its pixels from, and a canvas's keep
// them.
func TestReleasePixelsKeepsWhatCannotComeBack(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	gone := weak.Make(src)
	bm := NewBitmap(src)
	src = nil
	for gone.Value() != nil {
		runtime.GC()
	}
	bm.ReleasePixels()
	if bm.img.Pix == nil || bm.img.Restore != nil {
		t.Error("a bitmap whose image went dropped its pixels")
	}
	own := ownImage{stripedImage(8, 8, 1, 0)}
	bm = NewBitmap(own)
	bm.ReleasePixels()
	if bm.img.Pix == nil || bm.img.Restore != nil {
		t.Error("a bitmap of an image type of the app's dropped its pixels")
	}
	runtime.KeepAlive(own)
	cv := NewCanvas(8, 8)
	pix := cv.img.Pix
	cv.Bitmap().ReleasePixels()
	if cv.img.Restore != nil || len(cv.img.Pix) != len(pix) {
		t.Error("a canvas's bitmap dropped its pixels")
	}
}

// ownImage is an image of a type of the app's.
type ownImage struct{ *image.RGBA }

// TestReleasePixelsOfAConvertedBitmap checks issue #1281: a bitmap NewBitmap
// made drops its converted pixels, base and levels, while the app keeps
// the image it was made of, which the bitmap converts again when it
// needs them: it draws the same, smaller too, on the CPU, and a
// renderer's upload drops them again.
func TestReleasePixelsOfAConvertedBitmap(t *testing.T) {
	// An NRGBA image, which the bitmap converts into pixels of its own.
	stripes := stripedImage(512, 512, 4, 2)
	src := image.NewNRGBA(stripes.Rect)
	draw.Draw(src, src.Rect, stripes, image.Point{}, draw.Src)
	bm := NewBitmap(src)
	tt := coreNewTester(func(c *context) {
		coreImage(c, bm).Size(64, 64).Fit(FillBox)
		coreImage(c, bm).Size(100, 100).Absolute().Left(100)
	}, 200, 100)
	if len(bm.levels) == 0 {
		t.Fatal("no levels made")
	}
	want := cloneImage(tt.Image())
	pix := weak.Make(&bm.img.Pix[0])
	bm.ReleasePixels()
	if bm.img.Pix != nil {
		t.Error("the base kept its pixels")
	}
	for i, lvl := range bm.levels {
		if lvl.Pix != nil {
			t.Errorf("level %d kept its pixels", i)
		}
	}
	for range 3 {
		runtime.GC()
	}
	if pix.Value() != nil {
		t.Error("the converted pixels are still in memory")
	}
	// A size away and back draws everything again, on the CPU, from
	// pixels converted again.
	tt.SetSize(220, 120)
	tt.SetSize(200, 100)
	if got := tt.Image(); !bytes.Equal(want.Pix, got.Pix) {
		t.Fatal("a released converted bitmap draws other pixels")
	}
	bm.img.Uploaded()
	if bm.img.Pix != nil {
		t.Error("the pixels did not go again after the upload")
	}
	// Released again, it converts the same image again.
	bm.ReleasePixels()
	if pix := bm.img.Pixels(); len(pix) != 4*512*512 || pix[4*2] != 255 || pix[0] != 0 {
		t.Errorf("pixels converted again: %d bytes", len(pix))
	}

	// Update shows another picture, which the bitmap keeps, and draws.
	next := stripedImage(512, 512, 4, 3)
	bm.Update(next)
	tt.Frame()
	got := cloneImage(tt.Image())
	ref := NewBitmap(next)
	if want := coreNewTester(func(c *context) {
		coreImage(c, ref).Size(64, 64).Fit(FillBox)
		coreImage(c, ref).Size(100, 100).Absolute().Left(100)
	}, 200, 100).Image(); !bytes.Equal(want.Pix, got.Pix) {
		t.Fatal("an updated released bitmap draws other pixels")
	}
	bm.ReleasePixels() // nothing to make them from: it keeps them
	if bm.img.Pix == nil || bm.img.Restore != nil {
		t.Error("an updated bitmap dropped the pixels nothing makes again")
	}
	runtime.KeepAlive(src)
}
