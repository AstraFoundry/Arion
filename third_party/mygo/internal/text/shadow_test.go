package text

import "testing"

// TestBlurBitmap blurs a square of coverage: it keeps its sum, spreads
// three standard deviations past it, symmetric, and shades a sharp
// shadow of a color glyph by its alpha.
func TestBlurBitmap(t *testing.T) {
	b := bitmap{left: 2, top: -8, w: 4, h: 4, pix: make([]byte, 16)}
	for i := range b.pix {
		b.pix[i] = 255
	}
	sharp := blurBitmap(b, 0)
	if sharp.w != 4 || sharp.h != 4 || sharp.left != 2 || sharp.top != -8 || sharp.pix[5] != 255 {
		t.Fatalf("unblurred: %dx%d at (%d, %d)", sharp.w, sharp.h, sharp.left, sharp.top)
	}
	soft := blurBitmap(b, 2)
	pad := (soft.w - 4) / 2
	if pad != 7 || soft.h != soft.w || soft.left != 2-pad || soft.top != -8-pad {
		t.Fatalf("blurred: %dx%d at (%d, %d)", soft.w, soft.h, soft.left, soft.top)
	}
	sum := 0
	for _, v := range soft.pix {
		sum += int(v)
	}
	if want := 16 * 255; sum < want*98/100 || sum > want*102/100 {
		t.Errorf("the blur sums %d, the square %d", sum, want)
	}
	center := soft.pix[(soft.h/2)*soft.w+soft.w/2]
	edge := soft.pix[(soft.h/2)*soft.w]
	if center <= edge || center == 255 || soft.pix[(soft.h/2)*soft.w+1] != soft.pix[(soft.h/2)*soft.w+soft.w-2] {
		t.Errorf("the blur shades the center %d, the edge %d, unevenly", center, edge)
	}
	emoji := bitmap{w: 1, h: 1, color: true, pix: []byte{200, 10, 10, 128}}
	if s := blurBitmap(emoji, 0); s.color || s.pix[0] != 128 {
		t.Errorf("a color glyph's shadow is %v", s.pix)
	}
}
