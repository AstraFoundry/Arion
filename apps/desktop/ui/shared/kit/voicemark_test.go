package kit

import (
	"math"
	"testing"
)

func TestMarkLayoutStandsAsTheMark(t *testing.T) {
	width, live, bars := markLayout(20, markPose{Levels: [5]float32{1, 1, 1, 1, 1}, Live: 1})
	if width != 45 || live.Alpha != 1 || live.Rect.W != 8 {
		t.Fatalf("width %v, live %+v", width, live)
	}
	for i, b := range bars {
		if want := 20 * markBars[i]; math.Abs(float64(b.Rect.H-want)) > 1e-3 || b.Alpha != 1 || b.Rect.W != 3 {
			t.Errorf("bar %d = %+v, want height %v", i, b, want)
		}
	}
	// At rest, with the microphone closed, the full mark stands alone.
	if width, live, _ := markLayout(20, markPose{Levels: [5]float32{1, 1, 1, 1, 1}}); width != 27 || live.Alpha != 0 {
		t.Fatalf("resting width %v, live %+v", width, live)
	}
}

func TestMarkLayoutFoldsIntoThreeDots(t *testing.T) {
	for _, orbit := range []float32{0, 1} {
		width, live, bars := markLayout(20, markPose{Levels: [5]float32{1, 0, 0.5, 1, 0}, Fold: 1, Orbit: orbit, Wave: 0.5, Spin: [3]float32{0.3, 0.6, 0.9}})
		if width != 27 || live.Alpha != 0 {
			t.Fatalf("width %v, live %+v", width, live)
		}
		for i, b := range bars {
			if b.Rect.W != b.Rect.H {
				t.Errorf("orbit %v: bar %d is not round: %+v", orbit, i, b.Rect)
			}
			if i%2 == 1 {
				if b.Alpha != 0 {
					t.Errorf("orbit %v: merged bar %d still shows: %+v", orbit, i, b)
				}
				// It rests exactly on the dot it merged into.
				if into := bars[2*markDots[i]]; b.Rect != into.Rect {
					t.Errorf("orbit %v: bar %d at %+v, its dot at %+v", orbit, i, b.Rect, into.Rect)
				}
			}
		}
	}
}

// On the circle the dots sit a third of a turn apart around the center
// dot's row position, inside the mark's height, and a whole turn returns
// them to where they started.
func TestMarkLayoutOrbitsOnACircle(t *testing.T) {
	centerOf := func(s markShape) (float64, float64) {
		return float64(s.Rect.X + s.Rect.W/2), float64(s.Rect.Y + s.Rect.H/2)
	}
	_, _, row := markLayout(20, markPose{Fold: 1, Wave: 0.5})
	cx, _ := centerOf(row[2])
	_, _, start := markLayout(20, markPose{Fold: 1, Orbit: 1})
	_, _, turned := markLayout(20, markPose{Fold: 1, Orbit: 1, Spin: [3]float32{1, 1, 1}})
	for _, i := range []int{0, 2, 4} {
		x, y := centerOf(start[i])
		if r := math.Hypot(x-cx, y-10); math.Abs(r-6) > 1e-3 {
			t.Errorf("dot %d at radius %v", i, r)
		}
		if start[i].Rect.Y < 0 || start[i].Rect.Y+start[i].Rect.H > 20 {
			t.Errorf("dot %d leaves the mark: %+v", i, start[i].Rect)
		}
		tx, ty := centerOf(turned[i])
		if math.Abs(tx-x) > 1e-3 || math.Abs(ty-y) > 1e-3 {
			t.Errorf("dot %d does not come back after a turn: %v,%v vs %v,%v", i, tx, ty, x, y)
		}
	}
	if _, top := centerOf(start[2]); top >= 10 {
		t.Errorf("center dot not at the top: %v", top)
	}
}

// Each orbit cycle is one eased turn then a pause, the dots chasing each
// other: they start in order and all rest at a whole turn.
func TestMarkSpinTurnsThenPauses(t *testing.T) {
	if s := markSpinAt(0); s != [3]float32{} {
		t.Fatalf("cycle starts at %v", s)
	}
	mid := markSpinAt(0.3)
	if !(mid[0] > mid[1] && mid[1] > mid[2] && mid[2] > 0) {
		t.Fatalf("dots do not chase: %v", mid)
	}
	for _, u := range []float32{0.82, 0.9, 0.999} {
		if s := markSpinAt(u); s != [3]float32{1, 1, 1} {
			t.Fatalf("not paused at %v: %v", u, s)
		}
	}
}

func TestMarkLayoutMovesContinuously(t *testing.T) {
	levels := [5]float32{0.3, 0.8, 1, 0.6, 0.2}
	for _, sweep := range []struct {
		name string
		pose func(t float32) markPose
	}{
		{"fold", func(t float32) markPose { return markPose{Levels: levels, Live: 1 - t, Fold: t, Wave: 0.2} }},
		{"orbit", func(t float32) markPose { return markPose{Fold: 1, Orbit: t, Wave: 0.2, Spin: [3]float32{0.4, 0.3, 0.2}} }},
		{"rest", func(t float32) markPose {
			return markPose{Levels: [5]float32{1, 1, 1, 1, 1}, Fold: 1 - t, Orbit: 1 - t, Spin: [3]float32{0.4, 0.3, 0.2}}
		}},
		{"spin", func(t float32) markPose { return markPose{Fold: 1, Orbit: 1, Spin: markSpinAt(t)} }},
	} {
		_, _, prev := markLayout(20, sweep.pose(0))
		for step := 1; step <= 400; step++ {
			_, _, bars := markLayout(20, sweep.pose(float32(step)/400))
			for i := range bars {
				a, b := prev[i].Rect, bars[i].Rect
				if d := max(abs32(a.X-b.X), abs32(a.Y-b.Y), abs32(a.W-b.W), abs32(a.H-b.H)); d > 1 {
					t.Fatalf("%s: bar %d jumps %v DIPs at %v", sweep.name, i, d, float32(step)/400)
				}
			}
			prev = bars
		}
	}
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }
