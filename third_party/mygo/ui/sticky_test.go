package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A day's heading sticks to the top of the timeline while its day is in
// view, and the next day pushes it out; the pointer, the paint and
// assistive technology find it where it shows, and the focus and reading
// order keep it before its day.
func TestStickyTimeline(t *testing.T) {
	var s ScrollState
	var heads, bodies [3]*node
	var headClicks, bodyClicks int
	red, blue := RGB(255, 0, 0), RGB(0, 0, 255)
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			for i := range 3 {
				coreColumn(c).Shrink(0).Children(func() {
					heads[i] = coreButtonBase(c).Height(20).Shrink(0).Background(red).Label([]string{"Mon", "Tue", "Wed"}[i]).Sticky().Top(0)
					bodies[i] = coreButtonBase(c).Height(300).Shrink(0).Background(blue).Label("events")
					if heads[i].Clicked() {
						headClicks++
					}
					if bodies[i].Clicked() {
						bodyClicks++
					}
				})
			}
		})
	}, 200, 200)
	at := func(y float32, want ...float32) {
		t.Helper()
		s.Y = y
		tt.Frame()
		for i, w := range want {
			if heads[i].y != w {
				t.Errorf("scrolled to %v: heading %d at %v, want %v", y, i, heads[i].y, w)
			}
		}
	}
	at(0, 0, 320)
	at(100, 0, 220)
	at(310, -10, 10) // Tuesday's pushes Monday's out with Monday
	at(330, -30, 0)
	at(700, -400, -80, 0)

	// Scrolled into Monday, its heading is on top of its events.
	at(100)
	if c := tt.Image().RGBAAt(100, 5); c.R < 200 || c.B > 50 {
		t.Errorf("the heading stuck at the top paints under its day: %v", c)
	}
	tt.ClickAt(100, 5)
	if headClicks != 1 || bodyClicks != 0 {
		t.Errorf("a click on the heading stuck at the top: %d on headings, %d on events", headClicks, bodyClicks)
	}
	tt.ClickAt(100, 50)
	if headClicks != 1 || bodyClicks != 1 {
		t.Errorf("a click below the heading: %d on headings, %d on events", headClicks, bodyClicks)
	}

	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	head := accessNode(t, tree, platform.RoleButton, "Mon")
	if head.Bounds.Y != 0 || head.Bounds.H != 20 {
		t.Errorf("assistive technology finds the heading at %+v", head.Bounds)
	}
	order := []uint64{}
	for _, n := range tree.Nodes {
		order = append(order, n.ID)
	}
	if h, b := slices.Index(order, heads[0].id), slices.Index(order, bodies[0].id); h < 0 || h > b {
		t.Errorf("assistive technology reads the heading at %d, its events at %d", h, b)
	}
	if h, b := slices.Index(tt.rt.focusOrder, heads[0].id), slices.Index(tt.rt.focusOrder, bodies[0].id); h < 0 || h > b {
		t.Errorf("Tab goes to the heading at %d, its events at %d", h, b)
	}
}

// The check box of a row scrolling sideways stays at its start, at the
// left, or at the right right to left; percentages are of the view.
func TestStickySideways(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		var s ScrollState
		var box, cell *node
		tt := coreNewTester(func(c *context) {
			coreScrollHorizontal(c).Fill().Padding(0, 10).TrackScroll(&s).Children(func() {
				coreRow(c).Width(600).Shrink(0).Children(func() {
					box = coreBox(c).Size(30, 30).Shrink(0).Sticky().Start(0)
					cell = coreBox(c).Size(60, 30).Shrink(0).Sticky().StartPercent(25)
				})
			})
		}, 200, 100)
		if rtl {
			tt.SetDirection(RightToLeft)
		}
		s.X = 100
		tt.Frame()
		wantBox, wantCell := float32(0), float32(50)
		if rtl {
			wantBox, wantCell = 170, 90
		}
		if box.x != wantBox || cell.x != wantCell {
			t.Errorf("rtl %v, scrolled 100 along: the box at %v, the cell at %v, want %v, %v", rtl, box.x, cell.x, wantBox, wantCell)
		}
	}
}

// Each axis sticks to the container scrolling that way: a table's header
// row scrolling sideways in a page scrolling down sticks to the page's
// top, and a cell of it to the table's left; one following the rows
// sideways (ScrollWith) sticks as theirs do.
func TestStickyAxes(t *testing.T) {
	var page, table ScrollState
	var head, corner, follow *node
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&page).Children(func() {
			coreBox(c).Height(50).Shrink(0)
			rows := coreScrollHorizontal(c).Shrink(0).TrackScroll(&table).Children(func() {
				coreColumn(c).Width(600).Shrink(0).Children(func() {
					head = coreRow(c).Height(20).Shrink(0).Sticky().Top(0).Children(func() {
						corner = coreBox(c).Size(40, 20).Shrink(0).Sticky().Left(0)
					})
					coreBox(c).Height(400).Shrink(0)
				})
			})
			coreRow(c).Clip().ScrollWith(rows).Children(func() {
				follow = coreBox(c).Size(40, 20).Shrink(0).Sticky().Left(0)
				coreBox(c).Size(560, 20).Shrink(0)
			})
			coreBox(c).Height(400).Shrink(0)
		})
	}, 200, 200)
	page.Y, table.X = 100, 120
	tt.Frame()
	if head.y != 0 || corner.y != 0 || corner.x != 0 {
		t.Errorf("the header row at y %v, its corner at %v, %v", head.y, corner.x, corner.y)
	}
	if follow.x != 0 {
		t.Errorf("the cell following the rows sideways at %v", follow.x)
	}
	// Scrolled down past the table, its header goes with it.
	page.Y = 500
	tt.Frame()
	if want := float32(50 + 420 - 20 - 500); head.y != want {
		t.Errorf("past the table, its header row at %v, want %v", head.y, want)
	}
}

// A footer sticks to the bottom until its place in the flow comes in
// view; where both insets do not fit, Top wins.
func TestStickyBottom(t *testing.T) {
	var s ScrollState
	var foot, both *node
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			coreColumn(c).Shrink(0).Children(func() {
				coreBox(c).Height(500).Shrink(0)
				foot = coreBox(c).Height(20).Shrink(0).Sticky().Bottom(0)
			})
			coreColumn(c).Shrink(0).Children(func() {
				both = coreBox(c).Height(150).Shrink(0).Sticky().Top(10).Bottom(60)
				coreBox(c).Height(600).Shrink(0)
			})
		})
	}, 200, 200)
	if foot.y != 180 {
		t.Errorf("at the start, the footer at %v", foot.y)
	}
	s.Y = 400
	tt.Frame()
	if foot.y != 100 {
		t.Errorf("its place in view, the footer at %v", foot.y)
	}
	s.Y = 600
	tt.Frame()
	if both.y != 10 {
		t.Errorf("too tall for both insets, at %v", both.y)
	}
}

// Shown where it sticks, an element coming into view, as the focus does,
// leaves the offset; and what is attached to it goes where it sticks.
func TestStickyReveal(t *testing.T) {
	var s ScrollState
	show := false
	var cell, tip *node
	tt := coreNewTester(func(c *context) {
		coreScrollHorizontal(c).Fill().Margin(0, 20).TrackScroll(&s).Children(func() {
			coreRow(c).Width(600).Shrink(0).Children(func() {
				cell = coreBox(c).Size(30, 30).Shrink(0).Sticky().Left(0)
				if show {
					cell.ScrollIntoView()
				}
				coreOverlay(c, func() {
					tip = coreBox(c).Size(50, 20).AttachTo(cell, AnchorBottomLeft, AnchorTopLeft)
				})
			})
		})
	}, 200, 100)
	s.X = 100
	tt.Frame()
	show = true
	tt.Frame()
	if s.X != 100 || cell.x != 20 {
		t.Errorf("revealing the cell stuck at the left scrolled to %v, the cell at %v", s.X, cell.x)
	}
	if tip.x != cell.x || tip.y != cell.y+30 {
		t.Errorf("attached to the cell at %v, %v, at %v, %v", cell.x, cell.y, tip.x, tip.y)
	}
}

func TestStick(t *testing.T) {
	cases := []struct {
		name                 string
		lo, hi, mlo, mhi     float32
		clo, chi, from, to   float32
		fromOK, toOK, toWins bool
		want                 float32
	}{
		{"in view", 50, 70, 0, 0, 0, 500, 0, 200, true, true, false, 0},
		{"above", -50, -30, 0, 0, -100, 500, 0, 200, true, false, false, 50},
		{"pushed out", -50, -30, 0, 0, -100, -20, 0, 200, true, false, false, 10},
		{"margins count", -50, -30, 5, 5, -100, 500, 0, 200, true, false, false, 55},
		{"margins fit in", -50, -30, 5, 5, -100, -27, 0, 200, true, false, false, 0},
		{"below", 250, 270, 0, 0, 0, 500, 0, 200, false, true, false, -70},
		{"held at its start", 250, 270, 0, 0, 240, 500, 0, 200, false, true, false, -10},
		{"from wins", -50, 250, 0, 0, -100, 900, 0, 200, true, true, false, 50},
		{"to wins", -50, 250, 0, 0, -100, 900, 0, 200, true, true, true, -50},
		{"overflowing", -50, -30, 0, 0, -100, -40, 0, 200, true, false, false, 0},
	}
	for _, c := range cases {
		if got := stick(c.lo, c.hi, c.mlo, c.mhi, c.clo, c.chi, c.from, c.to, c.fromOK, c.toOK, c.toWins); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
