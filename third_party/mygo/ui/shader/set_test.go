package shader

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/shaderir"
	"github.com/egoist/mygo/ui"
)

// Uniforms take numbers of any of Go's kinds, alone, in slices and in
// arrays, and integers keep their low 32 bits.
func TestSet(t *testing.T) {
	f := math.Float32bits
	for _, c := range []struct {
		kind shaderir.BasicType
		n    int
		v    any
		want []uint32
	}{
		{shaderir.Float, 9, [9]float32{1, 2, 3, 4, 5, 6, 7, 8, 9}, []uint32{f(1), f(2), f(3), f(4), f(5), f(6), f(7), f(8), f(9)}},
		{shaderir.Float, 4, [4]float64{0.5, 1, 2, 3}, []uint32{f(0.5), f(1), f(2), f(3)}},
		{shaderir.Int, 2, [2]int{-1, 7}, []uint32{0xFFFFFFFF, 7}},
		{shaderir.Uint, 1, int64(-1), []uint32{0xFFFFFFFF}},
		{shaderir.Uint, 1, uint32(0x9E3779B9), []uint32{0x9E3779B9}},
		{shaderir.Int, 1, uint32(0xFFFFFFFF), []uint32{0xFFFFFFFF}},
		{shaderir.Bool, 3, []bool{true, false, true}, []uint32{1, 0, 1}},
		{shaderir.Uint, 2, []uint8{3, 255}, []uint32{3, 255}},
		{shaderir.Float, 1, 3, []uint32{f(3)}},
		{shaderir.Float, 4, ui.Color{R: 255, A: 255}, []uint32{f(1), 0, 0, f(1)}},
	} {
		lanes := make([]uint32, c.n)
		set(lanes, c.kind, "X", c.v)
		for i := range c.want {
			if lanes[i] != c.want[i] {
				t.Errorf("%T %v: lanes %#x, want %#x", c.v, c.v, lanes, c.want)
				break
			}
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("a string set a uniform")
		}
	}()
	set(make([]uint32, 1), shaderir.Float, "X", "1")
}
