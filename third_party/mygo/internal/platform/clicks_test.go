package platform

import (
	"math"
	"testing"
	"time"
)

func TestClickCounter(t *testing.T) {
	var c ClickCounter
	ms := uint32(math.MaxUint32 - 200) // the system's time wraps around between presses
	for i, step := range []struct {
		button int
		x, y   float64
		after  uint32
		want   int
	}{
		{0, 10, 10, 0, 1},
		{0, 12, 13, 500, 2}, // at the edges of time and distance, past the wrap
		{0, 12, 13, 100, 3},
		{0, 12, 13, 501, 1},   // too late
		{0, 14.5, 13, 100, 1}, // too far across
		{0, 14.5, 17, 100, 1}, // too far down
		{1, 14.5, 17, 100, 1}, // another button
		{1, 14.5, 17, 100, 2},
	} {
		ms += step.after
		if got := c.Press(step.button, step.x, step.y, ms, DoubleClick{Time: 500 * time.Millisecond, DX: 2, DY: 3}); got != step.want {
			t.Fatalf("press %d: count %d, want %d", i, got, step.want)
		}
	}
	// A press timed before the last, as from another clock, starts over.
	if got := c.Press(1, 14.5, 17, ms-10, DoubleClick{Time: 500 * time.Millisecond, DX: 2, DY: 3}); got != 1 {
		t.Fatalf("a press before the last: count %d, want 1", got)
	}
}
