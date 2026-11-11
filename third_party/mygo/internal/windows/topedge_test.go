//go:build windows && (amd64 || arm64)

package windows

import "testing"

// TestTopEdgeHit checks the codes the top edge of a window without a
// caption answers: the corners near the ends it reaches, the top between,
// and no top-right corner under the window controls of a hidden title bar,
// which end the edge before the window's right.
func TestTopEdgeHit(t *testing.T) {
	const corner = 32 // 16 DIPs at 200%
	for _, c := range []struct {
		name            string
		x, edge, client int32
		want            uintptr
	}{
		{"frameless, left end", 0, 800, 800, htTopLeft},
		{"frameless, left corner", corner - 1, 800, 800, htTopLeft},
		{"frameless, middle", corner, 800, 800, htTop},
		{"frameless, before the right corner", 800 - corner - 1, 800, 800, htTop},
		{"frameless, right corner", 800 - corner, 800, 800, htTopRight},
		{"frameless, right end", 799, 800, 800, htTopRight},
		{"hidden title bar, left corner", 3, 524, 800, htTopLeft},
		{"hidden title bar, before the buttons", 523, 524, 800, htTop},
		{"hidden title bar without buttons, right corner", 790, 800, 800, htTopRight},
		{"narrower than two corners", 20, 30, 30, htTopLeft},
	} {
		if got := topEdgeHit(c.x, c.edge, c.client, corner); got != c.want {
			t.Errorf("%s: topEdgeHit(%d, %d, %d) = %d, want %d", c.name, c.x, c.edge, c.client, got, c.want)
		}
	}
}
