package kit

import (
	"math"
	"testing"
)

func TestMarkLayoutStandsAsTheMark(t *testing.T) {
	width, live, bars := markLayout(20, [5]float32{1, 1, 1, 1, 1}, 0, 0)
	if width != 45 || live.Alpha != 1 || live.Rect.W != 8 {
		t.Fatalf("width %v, live %+v", width, live)
	}
	for i, b := range bars {
		if want := 20 * markBars[i]; math.Abs(float64(b.Rect.H-want)) > 1e-3 || b.Alpha != 1 || b.Rect.W != 3 {
			t.Errorf("bar %d = %+v, want height %v", i, b, want)
		}
	}
}

func TestMarkLayoutFoldsIntoThreeDots(t *testing.T) {
	width, live, bars := markLayout(20, [5]float32{1, 0, 0.5, 1, 0}, 1, 0.5)
	if width != 27 || live.Alpha != 0 {
		t.Fatalf("width %v, live %+v", width, live)
	}
	for i, b := range bars {
		if b.Rect.W != b.Rect.H {
			t.Errorf("bar %d is not round: %+v", i, b.Rect)
		}
		if i%2 == 1 {
			if b.Alpha != 0 {
				t.Errorf("merged bar %d still shows: %+v", i, b)
			}
			// It rests exactly on the dot it merged into.
			if into := bars[2*markDots[i]]; b.Rect != into.Rect {
				t.Errorf("bar %d at %+v, its dot at %+v", i, b.Rect, into.Rect)
			}
		}
	}
	if !(bars[0].Rect.X < bars[2].Rect.X && bars[2].Rect.X < bars[4].Rect.X) {
		t.Errorf("dots out of order: %+v", bars)
	}
}

func TestMarkLayoutMovesContinuously(t *testing.T) {
	levels := [5]float32{0.3, 0.8, 1, 0.6, 0.2}
	_, _, prev := markLayout(20, levels, 0, 0.2)
	for step := 1; step <= 100; step++ {
		_, _, bars := markLayout(20, levels, float32(step)/100, 0.2)
		for i := range bars {
			a, b := prev[i].Rect, bars[i].Rect
			if d := max(abs32(a.X-b.X), abs32(a.Y-b.Y), abs32(a.W-b.W), abs32(a.H-b.H)); d > 1 {
				t.Fatalf("bar %d jumps %v DIPs at fold %v", i, d, float32(step)/100)
			}
		}
		prev = bars
	}
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }
