package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// cardStrip is a strip of n cards w DIPs wide, that snaps as snap says, in
// a window 400 DIPs wide, with s tracking its offset.
func cardStrip(s *ScrollState, snap Snap, n int, w float32, align Align) func(c *context) {
	return func(c *context) {
		coreScrollHorizontal(c).Fill().ScrollSnap(snap).TrackScroll(s).Children(func() {
			for i := range n {
				coreBox(c).Key(i).Width(w).Height(100).Shrink(0).SnapAlign(align)
			}
		})
	}
}

func TestSnapTouchpadSettles(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Start), 400, 200)
	tt.Move(200, 50)
	// The fingers move it freely, and it rests once they settle.
	tt.Scroll(200, 50, 100, 0)
	if s.X != 100 {
		t.Fatalf("a touchpad scrolled to %v, want 100 at once", s.X)
	}
	tt.Advance(snapSettle / 2)
	if s.X != 100 {
		t.Errorf("snapped before the touchpad settled: at %v", s.X)
	}
	// The momentum of a fling keeps it going.
	tt.send(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 200, Y: 50, DX: 20, Precise: true, Momentum: true})
	tt.Advance(snapSettle / 2)
	if s.X != 120 {
		t.Errorf("snapped during the momentum: at %v", s.X)
	}
	tt.Advance(snapSettle)
	if s.X <= 120 || s.X >= 150 {
		t.Errorf("settled, on the way to 150: at %v", s.X)
	}
	tt.Advance(2 * time.Second)
	if s.X != 150 || tt.FramePending() {
		t.Errorf("at rest at %v, frames pending %v; want 150, none", s.X, tt.FramePending())
	}
	// Nearer the card it came from, it goes back to it.
	tt.Scroll(200, 50, 60, 0)
	tt.Advance(2 * time.Second)
	if s.X != 150 {
		t.Errorf("60 past a card: at %v, want 150", s.X)
	}
	// The end, where the last cards' starts can't come, is a place too.
	tt.Scroll(200, 50, 2000, 0)
	tt.Advance(2 * time.Second)
	if s.X != 1100 || s.MaxX != 1100 {
		t.Errorf("at the end: at %v of %v, want 1100", s.X, s.MaxX)
	}
}

func TestSnapWheelAndKeysGoToNext(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Start), 400, 200)
	tt.Move(200, 50)
	// A notch goes to the next card, however short it is.
	tt.Wheel(200, 50, 40, 0)
	tt.Advance(50 * time.Millisecond)
	if s.X <= 0 || s.X >= 150 {
		t.Errorf("a notch on its way: at %v", s.X)
	}
	tt.Advance(2 * time.Second)
	if s.X != 150 || tt.FramePending() {
		t.Errorf("after a notch: at %v, frames pending %v; want 150", s.X, tt.FramePending())
	}
	// Notches on the way go a card further each.
	tt.Wheel(200, 50, 40, 0)
	tt.Advance(20 * time.Millisecond)
	tt.Wheel(200, 50, 40, 0)
	tt.Advance(2 * time.Second)
	if s.X != 450 {
		t.Errorf("after two notches: at %v, want 450", s.X)
	}
	// The arrows go a card, back and on.
	tt.Key(0, KeyLeft)
	tt.Advance(2 * time.Second)
	if s.X != 300 {
		t.Errorf("after Left: at %v, want 300", s.X)
	}
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	tt.Advance(2 * time.Second)
	if s.X != 600 {
		t.Errorf("after Right twice: at %v, want 600", s.X)
	}
	// A notch far past the end stops there; at the end, it moves nothing.
	for range 10 {
		tt.Wheel(200, 50, 40, 0)
		tt.Advance(20 * time.Millisecond)
	}
	tt.Advance(2 * time.Second)
	if s.X != 1100 {
		t.Errorf("past the end: at %v, want 1100", s.X)
	}
	tt.Wheel(200, 50, 40, 0)
	if tt.FramePending() {
		t.Error("a notch at the end asked for a frame")
	}
}

func TestSnapWheelPassesOnAtTheEnd(t *testing.T) {
	var outer, inner ScrollState
	tt := coreNewTester(func(c *context) {
		coreScrollHorizontal(c).Fill().TrackScroll(&outer).Children(func() {
			coreScrollHorizontal(c).Width(300).Height(100).Shrink(0).ScrollSnap(SnapMandatory).TrackScroll(&inner).Children(func() {
				for i := range 2 {
					coreBox(c).Key(i).Width(300).Height(100).Shrink(0).SnapAlign(Start)
				}
			})
			coreBox(c).Width(1000).Height(100).Shrink(0)
		})
	}, 400, 200)
	tt.Move(100, 50)
	tt.Wheel(100, 50, 40, 0)
	tt.Advance(2 * time.Second)
	if inner.X != 300 || outer.X != 0 {
		t.Fatalf("a notch: inner at %v, outer at %v; want 300, 0", inner.X, outer.X)
	}
	// At the last place, notches scroll the container around it.
	tt.Wheel(100, 50, 40, 0)
	tt.Advance(2 * time.Second)
	if inner.X != 300 || outer.X != 40 {
		t.Errorf("a notch at the end: inner at %v, outer at %v; want 300, 40", inner.X, outer.X)
	}
}

func TestSnapProximity(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScrollHorizontal(c).Fill().ScrollSnap(SnapProximity).Gap(300).TrackScroll(&s).Children(func() {
			for i := range 5 {
				coreBox(c).Key(i).Width(150).Height(100).Shrink(0).SnapAlign(Start)
			}
		})
	}, 400, 200)
	tt.Move(200, 50)
	// Far from a place, within a third of the view of none, it rests
	// where it stops.
	tt.Scroll(200, 50, 200, 0)
	tt.Advance(2 * time.Second)
	if s.X != 200 {
		t.Errorf("between places: at %v, want 200", s.X)
	}
	// Near one, it snaps.
	tt.Scroll(200, 50, 200, 0)
	tt.Advance(2 * time.Second)
	if s.X != 450 {
		t.Errorf("near a place: at %v, want 450", s.X)
	}
	// A notch goes to the next place only near it, and otherwise as far
	// as it scrolls.
	tt.Wheel(200, 50, 40, 0)
	tt.Advance(2 * time.Second)
	if s.X != 490 {
		t.Errorf("a notch from a place: at %v, want 490", s.X)
	}
	tt.Wheel(200, 50, 340, 0)
	tt.Advance(2 * time.Second)
	if s.X != 900 {
		t.Errorf("a notch near the next place: at %v, want 900", s.X)
	}
}

func TestSnapAlignments(t *testing.T) {
	for _, tc := range []struct {
		align Align
		rtl   bool
		// The third card, 350 to 500 DIPs from the start of the content,
		// after a gap of 50, rests at want.
		want float32
	}{
		{Start, false, 350},
		{Center, false, 225},
		{End, false, 100},
		// Right to left, the offset and Start count from the right, where
		// the content starts: the same.
		{Start, true, 350},
		{Center, true, 225},
		{End, true, 100},
	} {
		t.Run(fmt.Sprint(tc.align, tc.rtl), func(t *testing.T) {
			var s ScrollState
			tt := coreNewTester(func(c *context) {
				coreScrollHorizontal(c).Fill().ScrollSnap(SnapMandatory).TrackScroll(&s).Children(func() {
					coreBox(c).Width(50).Shrink(0)
					for i := range 10 {
						coreBox(c).Key(i).Width(150).Height(100).Shrink(0).SnapAlign(tc.align)
					}
				})
			}, 400, 200)
			if tc.rtl {
				tt.SetDirection(RightToLeft)
			}
			tt.Move(200, 50)
			s.X = tc.want + 30
			tt.Frame()
			tt.Scroll(200, 50, 0.1, 0) // the user scrolls a bit, and lets go
			tt.Advance(2 * time.Second)
			if s.X != tc.want {
				t.Errorf("at %v, want %v", s.X, tc.want)
			}
		})
	}
}

func TestSnapBothAxes(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScrollBoth(c).Fill().ScrollSnap(SnapMandatory).TrackScroll(&s).Children(func() {
			coreColumn(c).Children(func() {
				for r := range 5 {
					coreRow(c).Key(r).Children(func() {
						for i := range 5 {
							coreBox(c).Key(i).Size(200, 200).Shrink(0).SnapAlign(Start)
						}
					})
				}
			})
		})
	}, 300, 300)
	tt.Move(150, 150)
	tt.Scroll(150, 150, 130, 250)
	tt.Advance(2 * time.Second)
	if s.X != 200 || s.Y != 200 {
		t.Errorf("at %v, %v; want 200, 200", s.X, s.Y)
	}
	// SnapAlignX snaps sideways only.
	var s2 ScrollState
	tt = coreNewTester(func(c *context) {
		coreScrollBoth(c).Fill().ScrollSnap(SnapMandatory).TrackScroll(&s2).Children(func() {
			coreRow(c).Children(func() {
				for i := range 5 {
					coreBox(c).Key(i).Size(200, 1000).Shrink(0).SnapAlignX(Start)
				}
			})
		})
	}, 300, 300)
	tt.Move(150, 150)
	tt.Scroll(150, 150, 130, 250)
	tt.Advance(2 * time.Second)
	if s2.X != 200 || s2.Y != 250 {
		t.Errorf("snapping sideways only: at %v, %v; want 200, 250", s2.X, s2.Y)
	}
}

func TestSnapLargeElementRestsAnywhereInIt(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().ScrollSnap(SnapMandatory).TrackScroll(&s).Children(func() {
			coreBox(c).Key(0).Height(200).Shrink(0).SnapAlign(Start)
			coreBox(c).Key(1).Height(1000).Shrink(0).SnapAlign(Start)
			coreBox(c).Key(2).Height(200).Shrink(0).SnapAlign(Start)
		})
	}, 200, 200)
	tt.Move(100, 100)
	// Inside the tall element, it rests where it stops.
	tt.Scroll(100, 100, 0, 500)
	tt.Advance(2 * time.Second)
	if s.Y != 500 {
		t.Errorf("in the tall element: at %v, want 500", s.Y)
	}
	// Notches scroll through it, and stop at its end, before the next.
	tt.Wheel(100, 100, 0, 40)
	tt.Advance(2 * time.Second)
	if s.Y != 540 {
		t.Errorf("a notch in it: at %v, want 540", s.Y)
	}
	tt.Wheel(100, 100, 0, 500)
	tt.Advance(2 * time.Second)
	if s.Y != 1000 {
		t.Errorf("a notch past its end: at %v, want 1000, its end", s.Y)
	}
	tt.Wheel(100, 100, 0, 40)
	tt.Advance(2 * time.Second)
	if s.Y != 1200 {
		t.Errorf("a notch at its end: at %v, want 1200, the next", s.Y)
	}
}

func TestSnapStaysAtItsElementAsTheLayoutMoves(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Center), 400, 200)
	tt.Move(200, 50)
	tt.Wheel(200, 50, 40, 0)
	tt.Wheel(200, 50, 40, 0)
	tt.Advance(2 * time.Second)
	// The third card, from 300 to 450, is centered.
	if s.X != 175 {
		t.Fatalf("after two notches: at %v, want 175", s.X)
	}
	tt.SetSize(500, 200)
	if s.X != 125 {
		t.Errorf("wider: at %v, want 125, still centering the third card", s.X)
	}
	// Where the app scrolls it, it stays.
	s.X = 210
	tt.Frame()
	tt.SetSize(400, 200)
	if s.X != 210 {
		t.Errorf("set by the app: at %v, want 210", s.X)
	}
}

func TestSnapFreshMandatoryRests(t *testing.T) {
	s := ScrollState{X: 160}
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Start), 400, 200)
	tt.Frame()
	if s.X != 150 {
		t.Errorf("first shown at 160: at %v, want 150", s.X)
	}
}

func TestSnapAppTakesOver(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Start), 400, 200)
	tt.Move(200, 50)
	tt.Wheel(200, 50, 40, 0)
	tt.Advance(50 * time.Millisecond)
	s.ScrollTo(620, 0, Animation{Duration: 100 * time.Millisecond})
	tt.Advance(2 * time.Second)
	if s.X != 620 {
		t.Errorf("ScrollTo on the way: at %v, want 620", s.X)
	}
	tt.Scroll(200, 50, 10, 0)
	tt.Advance(snapSettle / 2)
	s.X = 10
	tt.Advance(2 * time.Second)
	if s.X != 10 {
		t.Errorf("set while settling: at %v, want 10", s.X)
	}
}

func TestSnapReducedMotion(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Start), 400, 200)
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	tt.Move(200, 50)
	tt.Wheel(200, 50, 40, 0)
	if s.X != 150 {
		t.Errorf("a notch: at %v, want 150 at once", s.X)
	}
	tt.Scroll(200, 50, 100, 0)
	tt.Advance(snapSettle)
	if s.X != 300 {
		t.Errorf("a touchpad settled: at %v, want 300 at once", s.X)
	}
}

func TestSnapRevealLinesUp(t *testing.T) {
	var s ScrollState
	focus := -1
	tt := coreNewTester(func(c *context) {
		coreScrollHorizontal(c).Fill().ScrollSnap(SnapMandatory).TrackScroll(&s).Children(func() {
			for i := range 10 {
				coreBox(c).Key(i).Width(150).Height(100).Shrink(0).SnapAlign(Center).Children(func() {
					// Inside the card, what comes into view.
					inner := coreBox(c).Size(20, 20)
					if i == focus {
						inner.ScrollIntoView()
					}
				})
			}
		})
	}, 400, 200)
	tt.Frame()
	focus = 4
	tt.Frame()
	// Card 4, from 600 to 750, is centered, not just shown.
	if s.X != 475 {
		t.Errorf("revealed: at %v, want 475", s.X)
	}
}

func TestSnapScrollbarTrack(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapMandatory, 10, 150, Start), 400, 200)
	tt.Move(200, 100)
	tt.Frame()
	// A press on the track past the thumb goes to the place nearest a
	// page on, 360 DIPs, past where it was.
	tt.Press(390, 196)
	tt.Release(390, 196)
	tt.Advance(2 * time.Second)
	if s.X != 300 {
		t.Errorf("a press on the track: at %v, want 300", s.X)
	}
}

func TestSnapOff(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(cardStrip(&s, SnapNone, 10, 150, Start), 400, 200)
	tt.Move(200, 50)
	tt.Scroll(200, 50, 100, 0)
	tt.Advance(2 * time.Second)
	if s.X != 100 {
		t.Errorf("without ScrollSnap: at %v, want 100", s.X)
	}
}

func TestSnapList(t *testing.T) {
	var s ListState
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 1000, func(i int) {
			coreBox(c).Height(varied(i) * 3).SnapAlign(Start)
		}).Grow(1).ScrollSnap(SnapMandatory)
	}, 300, 400)
	tt.Move(100, 100)
	// Through rows not measured before, the touchpad settles at the top
	// of one.
	for range 5 {
		tt.Scroll(100, 100, 0, 333)
	}
	tt.Advance(2 * time.Second)
	first, _ := s.Visible()
	if r, _ := rowBox(tt, &s, first); r.Y != 0 || first == 0 {
		t.Fatalf("at rest, row %d starts at %v, want 0", first, r.Y)
	}
	// A notch goes on to the next row.
	for i := 1; i <= 3; i++ {
		tt.Wheel(100, 100, 0, 40)
		tt.Advance(2 * time.Second)
		if r, _ := rowBox(tt, &s, first+i); r.Y != 0 {
			t.Errorf("after %d notches, row %d starts at %v, want 0", i, first+i, r.Y)
		}
	}
}
