package ui

import (
	"testing"
	"time"
)

// foldList is a chat of n messages of 30 DIPs following its end, whose
// last message opens a fold 300 DIPs tall when clicked: clicked runs in
// the frame of the click, and the fold opens in it, or in the next when
// later is set.
type foldList struct {
	s       ListState
	n       int
	open    bool
	later   bool
	pending uint64
	clicked func(s *ListState)
}

func (l *foldList) view(c *context) {
	if l.pending != 0 && c.rt.frame > l.pending {
		l.pending, l.open = 0, true
	}
	coreList(c, &l.s, l.n, func(i int) {
		h := float32(30)
		if i == l.n-1 && l.open {
			h = 300
		}
		b := coreBox(c).Height(h).Children(func() { coreTextf(c, "Message %d", i) })
		if i == l.n-1 && b.Clicked() {
			if l.clicked != nil {
				l.clicked(&l.s)
			}
			if l.later {
				l.pending = c.rt.frame
				c.rt.animating = true
			} else {
				l.open = true
			}
		}
	}).Grow(1)
}

// TestListFollowEndOffInTheClicksFrame is issue 1389: FollowEnd turned
// off in the frame of a click that opens a fold in the last row keeps the
// line clicked where it was, rather than following the end as the row
// grows.
func TestListFollowEndOffInTheClicksFrame(t *testing.T) {
	for _, later := range []bool{false, true} {
		l := &foldList{s: ListState{FollowEnd: true}, n: 40, later: later}
		l.clicked = func(s *ListState) { s.FollowEnd = false }
		tt := coreNewTester(l.view, 300, 400)
		before, _ := rowBox(tt, &l.s, l.n-1)
		if !l.s.AtEnd() || before.Y+before.H != 400 {
			t.Fatalf("at the start: at the end %v, the last row %v", l.s.AtEnd(), before)
		}
		if err := tt.Click("Message 39"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		r, _ := rowBox(tt, &l.s, l.n-1)
		if !l.open || r.Y != before.Y || r.H != 300 {
			t.Errorf("later %v: the fold opened (%v): the last row went from %v to %v", later, l.open, before, r)
		}
		if l.s.AtEnd() {
			t.Errorf("later %v: the end shows below the fold", later)
		}
	}
}

// TestListFollowEndOffBetweenLayouts turns FollowEnd off with no frame
// laid out after it: the next layout, as a row grows, reads it.
func TestListFollowEndOffBetweenLayouts(t *testing.T) {
	l := &foldList{s: ListState{FollowEnd: true}, n: 40}
	tt := coreNewTester(l.view, 300, 400)
	before, _ := rowBox(tt, &l.s, l.n-1)
	l.s.FollowEnd, l.open = false, true
	tt.Frame()
	if r, _ := rowBox(tt, &l.s, l.n-1); r.Y != before.Y {
		t.Errorf("the last row grew: its top moved from %v to %v", before.Y, r.Y)
	}
}

// TestListFollowEndOnAtTheEnd turns FollowEnd on in a list showing its
// end: it follows the end from the next layout.
func TestListFollowEndOnAtTheEnd(t *testing.T) {
	l := &foldList{n: 40}
	tt := coreNewTester(l.view, 300, 400)
	l.s.ScrollToEnd()
	tt.Frame()
	if !l.s.AtEnd() {
		t.Fatal("ScrollToEnd: not at the end")
	}
	l.s.FollowEnd, l.open = true, true
	tt.Frame()
	if r, _ := rowBox(tt, &l.s, l.n-1); r.Y+r.H != 400 || !l.s.AtEnd() {
		t.Errorf("FollowEnd on at the end, the last row grew: it ends at %v", r.Y+r.H)
	}
	// Away from the end, it does not follow it.
	l2 := &foldList{n: 40}
	tt2 := coreNewTester(l2.view, 300, 400)
	l2.s.FollowEnd = true
	l2.n++
	tt2.Frame()
	if first, _ := l2.s.Visible(); first != 0 || l2.s.AtEnd() {
		t.Errorf("FollowEnd on at the top: row %d first, at the end %v", first, l2.s.AtEnd())
	}
}

// TestListHoldPlace keeps a list following its end where it is as the
// last row grows in the frame of the click, and FollowEnd on: scrolling
// back to the end follows it again.
func TestListHoldPlace(t *testing.T) {
	for _, anim := range []*Animation{nil, &Smooth} {
		l := &foldList{s: ListState{FollowEnd: true, FollowAnimation: anim}, n: 40}
		l.clicked = (*ListState).HoldPlace
		tt := coreNewTester(l.view, 300, 400)
		before, _ := rowBox(tt, &l.s, l.n-1)
		if err := tt.Click("Message 39"); err != nil {
			t.Fatal(err)
		}
		tt.Advance(2 * time.Second)
		r, _ := rowBox(tt, &l.s, l.n-1)
		if !l.open || r.Y != before.Y || r.H != 300 {
			t.Errorf("animation %v: the fold opened (%v): the last row went from %v to %v", anim != nil, l.open, before, r)
		}
		if !l.s.FollowEnd || l.s.AtEnd() {
			t.Errorf("animation %v: FollowEnd %v, at the end %v", anim != nil, l.s.FollowEnd, l.s.AtEnd())
		}
		// Scrolled back to the end, the list follows it again.
		l.s.ScrollToEnd()
		tt.Frame()
		l.n++
		tt.Frame()
		tt.Advance(2 * time.Second)
		if r, _ := rowBox(tt, &l.s, l.n-1); r.Y+r.H != 400 || !l.s.AtEnd() {
			t.Errorf("animation %v: back at the end, a row came: it ends at %v", anim != nil, r.Y+r.H)
		}
	}
	// Without a row that changes, the hold changes nothing.
	l := &foldList{s: ListState{FollowEnd: true}, n: 40}
	tt := coreNewTester(l.view, 300, 400)
	l.s.HoldPlace()
	tt.Frame()
	l.n++
	tt.Frame()
	if r, _ := rowBox(tt, &l.s, l.n-1); r.Y+r.H != 400 || !l.s.AtEnd() {
		t.Errorf("held in a frame where nothing changed, then a row came: it ends at %v", r.Y+r.H)
	}
}
