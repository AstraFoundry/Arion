package chart

import (
	"math"
	"slices"
	"testing"
)

func TestMinMax(t *testing.T) {
	nan := float32(math.NaN())
	for _, tc := range []struct {
		name    string
		xs, ys  []float32
		width   int
		indices []int
	}{
		{"extrema order", []float32{0, .01, .02, .03, .04, .05, .06}, []float32{3, 8, 2, 9, 0, 4, 5}, 1, []int{0, 3, 4, 6}},
		{"two per bucket", []float32{0, .1, .2, .3, .4, .5, .6, .7, .8, .9}, []float32{2, 0, 4, 3, 2, 2, 0, 4, 3, 2}, 2, []int{0, 1, 2, 6, 7, 9}},
		{"sparse", []float32{0, .3, .6, 1}, []float32{1, 2, 3, 4}, 100, []int{0, 1, 2, 3}},
		{"duplicate x", []float32{.5, .5, .5, .5, .5}, []float32{2, 0, 3, 1, 2}, 10, []int{0, 1, 2, 4}},
		{"edges", []float32{-2, -1, 0, .5, 1, 2, 3}, []float32{0, 1, 2, 3, 4, 5, 6}, 10, []int{1, 2, 3, 4, 5}},
		{"crossing viewport", []float32{-2, -1, 2, 3}, []float32{0, 1, 2, 3}, 10, []int{1, 2}},
		{"outside left", []float32{-2, -1}, []float32{0, 1}, 10, nil},
		{"outside right", []float32{2, 3}, []float32{0, 1}, 10, nil},
		{"empty", nil, nil, 10, nil},
		{"zero width", []float32{0}, []float32{0}, 0, nil},
		{"length mismatch", []float32{0, .5, 1}, []float32{1, 2}, 10, []int{0, 1}},
		{"constant", []float32{0, .01, .02, .03}, []float32{1, 1, 1, 1}, 1, []int{0, 3}},
		{"gap", []float32{0, .1, .2, .3, .4}, []float32{1, 2, nan, 3, 4}, 1, []int{0, 1, -1, 3, 4}},
		{"infinity", []float32{0, .1, .2}, []float32{1, float32(math.Inf(1)), 2}, 1, []int{0, -1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x, y := MinMax(nil, nil, tc.xs, tc.ys, tc.width)
			if len(x) != len(tc.indices) || len(y) != len(x) {
				t.Fatalf("got %v %v", x, y)
			}
			for j, i := range tc.indices {
				if i < 0 {
					if !math.IsNaN(float64(x[j])) || !math.IsNaN(float64(y[j])) {
						t.Fatal("missing break")
					}
					continue
				}
				if x[j] != tc.xs[i] || y[j] != tc.ys[i] {
					t.Fatalf("point %d: %v %v", j, x, y)
				}
			}
		})
	}
}

func TestMinMaxReuseAndDensity(t *testing.T) {
	xs, ys := make([]float32, 15000), make([]float32, 15000)
	for i := range xs {
		xs[i] = float32(i) / 14999
		ys[i] = float32(math.Sin(float64(i)))
	}
	dx, dy := make([]float32, 0, 15000), make([]float32, 0, 15000)
	if a := testing.AllocsPerRun(20, func() { dx, dy = MinMax(dx, dy, xs, ys, 1280) }); a != 0 {
		t.Fatalf("%v allocations", a)
	}
	first := slices.Clone(dx)
	dx, dy = MinMax(dx, dy, xs, ys, 2560)
	if len(dx) <= len(first) || len(dx) > 2*2560+3 {
		t.Fatal("wrong density", len(dx), len(first))
	}
}

// BenchmarkMinMax reduces 45,000 samples into physical pixel buckets, reusing output.
func BenchmarkMinMax(b *testing.B) {
	xs, ys := make([]float32, 15000), make([]float32, 15000)
	for i := range xs {
		xs[i] = float32(i) / 14999
		ys[i] = float32(math.Sin(float64(i)))
	}
	var dx, dy []float32
	dx, dy = MinMax(dx, dy, xs, ys, 1280)
	b.ReportAllocs()
	for b.Loop() {
		for range 3 {
			dx, dy = MinMax(dx, dy, xs, ys, 1280)
		}
	}
}
