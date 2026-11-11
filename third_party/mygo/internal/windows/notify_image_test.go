//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"runtime"
	"testing"
	"unsafe"
)

// readIcon reads an icon's color bitmap, top-down, as BGRA.
func readIcon(t *testing.T, icon uintptr) (w, h int, px []byte) {
	t.Helper()
	var info struct {
		Icon        int32
		XHotspot    uint32
		YHotspot    uint32
		Mask, Color uintptr
	}
	if r, _, _ := user32.NewProc("GetIconInfo").Call(icon, uintptr(unsafe.Pointer(&info))); r == 0 {
		t.Fatal("GetIconInfo failed")
	}
	defer procDeleteObject.Call(info.Mask)
	defer procDeleteObject.Call(info.Color)
	var bm struct {
		Type, Width, Height, WidthBytes int32
		Planes, BitsPixel               uint16
		Bits                            uintptr
	}
	procGetObjectW.Call(info.Color, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm)))
	w, h = int(bm.Width), int(bm.Height)
	hdr := bitmapInfoHeader{Width: bm.Width, Height: -bm.Height, Planes: 1, BitCount: 32}
	hdr.Size = uint32(unsafe.Sizeof(hdr))
	px = make([]byte, w*h*4)
	dc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, dc)
	if r, _, _ := gdi32.NewProc("GetDIBits").Call(dc, info.Color, 0, uintptr(h), uintptr(unsafe.Pointer(&px[0])), uintptr(unsafe.Pointer(&hdr)), 0); int(r) != h {
		t.Fatalf("GetDIBits copied %d rows of %d", r, h)
	}
	return w, h, px
}

// TestImageIcon makes the icons a balloon shows of notifications' images:
// PNG or JPEG, fit to the icon and centered on transparent pixels, with
// straight alpha as icons have it; and none of bytes that are no image.
func TestImageIcon(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, coinitApartmentThreaded)

	encode := func(img image.Image, jpg bool) []byte {
		var b bytes.Buffer
		var err error
		if jpg {
			err = jpeg.Encode(&b, img, &jpeg.Options{Quality: 100})
		} else {
			err = png.Encode(&b, img)
		}
		if err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	fill := func(w, h int, c color.Color) image.Image {
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
		return img
	}
	const size = 32
	for _, tc := range []struct {
		name   string
		image  []byte
		inside image.Point // a pixel of the image
		empty  image.Point // a pixel of the padding
		want   color.NRGBA
	}{
		// 64x32 fits as 32x16, rows 8 to 23.
		{"wide PNG", encode(fill(64, 32, color.NRGBA{255, 0, 0, 255}), false), image.Pt(16, 16), image.Pt(16, 2), color.NRGBA{255, 0, 0, 255}},
		// 16x32 fits as 16x32, columns 8 to 23.
		{"tall JPEG", encode(fill(16, 32, color.NRGBA{0, 255, 0, 255}), true), image.Pt(16, 16), image.Pt(2, 16), color.NRGBA{0, 255, 0, 255}},
		// Not premultiplied, which would halve the red.
		{"half transparent", encode(fill(8, 8, color.NRGBA{255, 0, 0, 128}), false), image.Pt(16, 16), image.Pt(-1, -1), color.NRGBA{255, 0, 0, 128}},
	} {
		// Not in subtests, which would run on threads without COM.
		icon := imageIcon(tc.image, size)
		if icon == 0 {
			t.Errorf("%s: no icon", tc.name)
			continue
		}
		w, h, px := readIcon(t, icon)
		procDestroyIcon.Call(icon)
		if w != size || h != size {
			t.Errorf("%s: the icon is %dx%d, want %dx%d", tc.name, w, h, size, size)
			continue
		}
		at := func(p image.Point) color.NRGBA {
			i := (p.Y*w + p.X) * 4
			return color.NRGBA{px[i+2], px[i+1], px[i], px[i+3]}
		}
		near := func(a, b uint8) bool { return a-b < 8 || b-a < 8 }
		if c := at(tc.inside); !near(c.R, tc.want.R) || !near(c.G, tc.want.G) || !near(c.B, tc.want.B) || !near(c.A, tc.want.A) {
			t.Errorf("%s: the image's pixel %v is %v, want %v", tc.name, tc.inside, c, tc.want)
		}
		if tc.empty.X >= 0 {
			if c := at(tc.empty); c.A != 0 {
				t.Errorf("%s: the padding's pixel %v is %v, want transparent", tc.name, tc.empty, c)
			}
		}
	}
	if icon := imageIcon([]byte("not an image"), size); icon != 0 {
		procDestroyIcon.Call(icon)
		t.Error("bytes that are no image made an icon")
	}
}
