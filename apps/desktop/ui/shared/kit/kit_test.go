package kit

import "testing"

func TestNextTurnRollsForward(t *testing.T) {
	for _, tc := range []struct {
		target float32
		digit  int
		want   float32
	}{{0, 7, 7}, {7, 7, 7}, {7, 9, 9}, {9, 0, 10}, {9, 3, 13}, {13, 2, 22}} {
		if got := nextTurn(tc.target, tc.digit); got != tc.want {
			t.Errorf("nextTurn(%v, %d) = %v, want %v", tc.target, tc.digit, got, tc.want)
		}
	}
}
