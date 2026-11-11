//go:build linux && (amd64 || arm64)

package linux

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"testing"
)

// TestImageHint gives a notification's image to the server as the spec's
// "image-data": PNG or JPEG decoded, a large one fit to notifyImageBox,
// with as many bytes as libnotify counts; and no hint for bytes that are
// no image.
func TestImageHint(t *testing.T) {
	if err := load(); err != nil {
		t.Skip(err)
	}
	var getInt32 func(v ptr) int32
	mustBind(libGLib, &getInt32, "g_variant_get_int32")

	// The PNGs are translucent, which png.Encode keeps as RGBA: it writes
	// opaque images as RGB.
	encode := func(w, h int, jpg bool) []byte {
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{200, 40, 40, 128}), image.Point{}, draw.Src)
		var b bytes.Buffer
		var err error
		if jpg {
			err = jpeg.Encode(&b, img, nil)
		} else {
			err = png.Encode(&b, img)
		}
		if err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	for _, tc := range []struct {
		name         string
		image        []byte
		w, h         int32
		alpha        bool
		channels     int32
		bitsOfSample int32
	}{
		{"PNG", encode(64, 48, false), 64, 48, true, 4, 8},
		{"JPEG", encode(64, 48, true), 64, 48, false, 3, 8},
		{"large PNG", encode(1024, 512, false), notifyImageBox, notifyImageBox / 2, true, 4, 8},
	} {
		hint := imageHint(tc.image)
		if hint == 0 {
			t.Errorf("%s: no hint", tc.name)
			continue
		}
		if typ := goStr(gVariantGetTypeString(hint)); typ != "{sv}" {
			t.Errorf("%s: the hint is %s", tc.name, typ)
			gVariantUnref(hint)
			continue
		}
		key := gVariantGetChildValue(hint, 0)
		boxed := gVariantGetChildValue(hint, 1)
		value := gVariantGetVariant(boxed)
		if k := goStr(gVariantGetString(key, nil)); k != "image-data" {
			t.Errorf("%s: the hint is %q", tc.name, k)
		}
		if typ := goStr(gVariantGetTypeString(value)); typ != "(iiibiiay)" {
			t.Errorf("%s: image-data is %s", tc.name, typ)
		} else {
			field := func(i uintptr) ptr { return gVariantGetChildValue(value, i) }
			var ints [6]int32
			for _, i := range []uintptr{0, 1, 2, 4, 5} {
				c := field(i)
				ints[i] = getInt32(c)
				gVariantUnref(c)
			}
			c := field(3)
			alpha := gVariantGetBoolean(c)
			gVariantUnref(c)
			c = field(6)
			n := gVariantNChildren(c)
			gVariantUnref(c)
			w, h, stride, bits, channels := ints[0], ints[1], ints[2], ints[4], ints[5]
			if w != tc.w || h != tc.h || alpha != tc.alpha || bits != tc.bitsOfSample || channels != tc.channels {
				t.Errorf("%s: %dx%d, alpha %v, %d bits, %d channels; want %dx%d, alpha %v, %d bits, %d channels",
					tc.name, w, h, alpha, bits, channels, tc.w, tc.h, tc.alpha, tc.bitsOfSample, tc.channels)
			}
			if want := uintptr((h-1)*stride + w*channels*bits/8); n != want {
				t.Errorf("%s: %d bytes, want %d", tc.name, n, want)
			}
		}
		gVariantUnref(value)
		gVariantUnref(boxed)
		gVariantUnref(key)
		gVariantUnref(hint)
	}
	if hint := imageHint([]byte("not an image")); hint != 0 {
		gVariantUnref(hint)
		t.Error("bytes that are no image made a hint")
	}
}
