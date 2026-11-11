package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

// pcExtras style the random trees' elements further for painting: moved
// far, out of their clips and back, turned, scaled, blurred, with wide
// shadows, drawings beyond their boxes, dividers and the debug overlay.
var pcExtras = []func(e *node){
	nil,
	func(e *node) { e.Translate(0, -260) },
	func(e *node) { e.Translate(0, 340) },
	func(e *node) { e.Translate(-420, 15) },
	func(e *node) { e.Rotate(30) },
	func(e *node) { e.Scale(0.5, 3) },
	func(e *node) { e.Blur(2) },
	func(e *node) { e.Shadow(0, 60, 30, 10, RGBA(0, 0, 0, 0.4)) },
	func(e *node) {
		e.Draw(func(p *Painter, r Rect) { p.Fill(Rect{r.X - 30, r.Y - 30, r.W + 60, 10}, RGB(0, 120, 0), 0) })
	},
	func(e *node) {
		e.DrawOver(func(p *Painter, r Rect) { p.Fill(Rect{r.X, r.Y + r.H, 20, 40}, RGB(0, 0, 160), 2) })
	},
	func(e *node) { e.Dividers(1, RGB(90, 90, 90)) },
	func(e *node) { e.Debug() },
	func(e *node) { e.Focusable().Background(RGB(250, 250, 200)) },
	func(e *node) { e.Disabled(true).Background(RGB(30, 30, 30)) },
}

// Painting skips what its ink tells it shows nowhere (Painter.paintable),
// and paints what painting everything does: the same operations, but
// for clips pushed and popped around nothing, as random trees change and
// move frame after frame. Each frame is painted again with and without
// skipping, over the same images (effects' layers), which then hold the
// same pixels.
func TestPaintCullMatches(t *testing.T) {
	seeds, frames := 20, 60
	if testing.Short() {
		seeds = 6
	}
	var culled uint64
	for seed := range uint64(seeds) {
		m := newLCModel(seed + 1000)
		extras := map[*lcNode]int{}
		m.extra = func(n *lcNode, e *node) {
			if f := pcExtras[extras[n]]; f != nil {
				f(e)
			}
		}
		restyle := func() {
			nodes, _ := m.all()
			for _, n := range nodes {
				if m.rng.IntN(5) == 0 {
					extras[n] = m.rng.IntN(len(pcExtras))
				}
			}
		}
		restyle()
		tt := lcTester(func(c *context) {
			coreBox(c).Fill().Children(func() { m.build(c, 0) })
		}, m.w, m.h, time.Unix(1e9, 0))
		var log []string
		for f := range frames {
			if m.rng.IntN(3) > 0 {
				for range 1 + m.rng.IntN(3) {
					log = append(log, m.mutate())
				}
				restyle()
			} else {
				log = append(log, "unchanged")
			}
			tt.h.w, tt.h.h = float32(m.w), float32(m.h)
			before := tt.rt.culled
			if m.rng.IntN(4) == 0 {
				tt.Advance(30 * time.Millisecond)
			} else {
				tt.Frame()
			}
			culled += tt.rt.culled - before
			// Painted again, as what the frame's painting did first, as
			// images taking their size, is done.
			w, h, scale := tt.h.size()
			tt.rt.paint(tt.rt.c.root, w, h, scale)
			got := withoutEmptyClips(tt.rt.scene.Ops)
			tt.rt.noCull = true
			tt.rt.paint(tt.rt.c.root, w, h, scale)
			tt.rt.noCull = false
			if d := sameOps(got, withoutEmptyClips(tt.rt.scene.Ops)); d != "" {
				if len(log) > 6 {
					log = log[len(log)-6:]
				}
				t.Fatalf("seed %d, frame %d, after %s: %s", seed, f, strings.Join(log, ", "), d)
			}
		}
	}
	if culled == 0 {
		t.Error("painting skipped no element")
	}
}

// withoutEmptyClips returns ops without the clips pushed and popped
// around nothing.
func withoutEmptyClips(ops []scene.Op) []scene.Op {
	var out []scene.Op
	var open []int // where the clips pushed start in out
	for _, op := range ops {
		switch op.Kind {
		case scene.OpPushClip:
			open = append(open, len(out))
		case scene.OpPopClip:
			if at := open[len(open)-1]; at == len(out)-1 && out[at].Kind == scene.OpPushClip {
				out, open = out[:at], open[:len(open)-1]
				continue
			}
			open = open[:len(open)-1]
		}
		out = append(out, op)
	}
	return out
}

func sameOps(got, want []scene.Op) string {
	for i := range min(len(got), len(want)) {
		if !reflect.DeepEqual(got[i], want[i]) {
			return fmt.Sprintf("op %d of %d: %+v, want %+v", i, len(want), got[i], want[i])
		}
	}
	if len(got) != len(want) {
		return fmt.Sprintf("%d ops, want %d", len(got), len(want))
	}
	return ""
}

// The rows of a long column scrolled by a translation in a clip are
// painted only where they show.
func TestPaintCullsScrolledRows(t *testing.T) {
	var dy float32
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Clip().Children(func() {
			coreColumn(c).Translate(0, dy).Children(func() {
				for i := range 200 {
					coreRow(c).Height(28).Gap(8).Children(func() {
						coreText(c, "Row "+itoa(i))
						coreSpacer(c)
						coreText(c, itoa(i*7%100))
					})
				}
			})
		})
	}, 400, 300)
	for _, d := range []float32{0, -1000, -4000} {
		dy = d
		before := tt.rt.culled
		tt.Frame()
		// 11 rows show at most; the others are skipped whole.
		if n := tt.rt.culled - before; n < 188 || n > 190 {
			t.Errorf("at %v, %d rows skipped", d, n)
		}
	}
}
