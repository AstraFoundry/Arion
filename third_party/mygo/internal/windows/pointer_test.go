//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// The pointer structs are laid out as Windows's headers lay them out on
// 64-bit Windows.
func TestPointerInfoLayout(t *testing.T) {
	if n := unsafe.Sizeof(pointerInfo{}); n != 96 {
		t.Errorf("POINTER_INFO is %d bytes, want 96", n)
	}
	if n := unsafe.Sizeof(pointerPenInfo{}); n != 120 {
		t.Errorf("POINTER_PEN_INFO is %d bytes, want 120", n)
	}
	if n := unsafe.Sizeof(pointerTouchInfo{}); n != 144 {
		t.Errorf("POINTER_TOUCH_INFO is %d bytes, want 144", n)
	}
	if o := unsafe.Offsetof(pointerInfo{}.PerformanceCount); o != 80 {
		t.Errorf("PerformanceCount at %d, want 80", o)
	}
	if o := unsafe.Offsetof(pointerPenInfo{}.Pressure); o != 104 {
		t.Errorf("POINTER_PEN_INFO's pressure at %d, want 104", o)
	}
	if o := unsafe.Offsetof(pointerTouchInfo{}.Pressure); o != 140 {
		t.Errorf("POINTER_TOUCH_INFO's pressure at %d, want 140", o)
	}
}

// The cursors Windows lacks are drawn: opaque in the middle, transparent
// at the corners, at the cursor sizes of high DPIs too.
func TestDrawnCursors(t *testing.T) {
	for _, c := range []platform.Cursor{platform.CursorZoomIn, platform.CursorZoomOut, platform.CursorCell} {
		for _, size := range []int{32, 48, 64} {
			and, xor, hx, hy := cursorMasks(c, size)
			if len(and) != size*size/8 || len(xor) != len(and) {
				t.Fatalf("cursor %d at %d: masks of %d and %d bytes", c, size, len(and), len(xor))
			}
			bit := func(m []byte, x, y int) bool { return m[y*size/8+x/8]&(0x80>>(x%8)) != 0 }
			if !bit(and, 0, 0) || !bit(and, size-1, 0) || bit(xor, 0, 0) {
				t.Errorf("cursor %d at %d: a corner is not transparent", c, size)
			}
			if bit(and, hx, hy) {
				t.Errorf("cursor %d at %d: its hot spot %d, %d is transparent", c, size, hx, hy)
			}
		}
	}
	// The plus of zooming in crosses the lens, the minus of zooming out
	// does not.
	in, inXor, hx, hy := cursorMasks(platform.CursorZoomIn, 32)
	out, outXor, _, _ := cursorMasks(platform.CursorZoomOut, 32)
	above := (hy-3)*4 + hx/8
	mask := byte(0x80 >> (hx % 8))
	if in[above]&mask != 0 || inXor[above]&mask != 0 || out[above]&mask != 0 || outXor[above]&mask == 0 {
		t.Error("the plus and the minus are not told apart above the lens's center")
	}
}
