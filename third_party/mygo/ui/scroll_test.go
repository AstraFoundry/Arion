package ui

import (
	"math"
	"testing"
	"time"
)

func TestTrackScroll(t *testing.T) {
	var s ScrollState
	var top *node
	seen := float32(-1)
	tt := coreNewTester(func(c *context) {
		seen = s.Y
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			top = coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	if s.Y != 0 || s.MaxY != 800 || s.MaxX != 0 {
		t.Fatalf("at the start: %+v", s)
	}
	// The user scrolls: the state follows, before the frame builds.
	tt.Move(100, 100)
	tt.Scroll(100, 100, 0, 120)
	if s.Y != 120 || seen != 120 || top.y != -120 {
		t.Errorf("scrolled 120 down: %+v, the view saw %v, the content is at %v", s, seen, top.y)
	}
	// The app scrolls.
	s.Y = 300
	tt.Frame()
	if top.y != -300 {
		t.Errorf("set to 300: the content is at %v", top.y)
	}
	// Past the end is the end, before the start the start, and the axis it
	// does not scroll stays put.
	s.Y = math.MaxFloat32
	tt.Frame()
	if s.Y != 800 || top.y != -800 {
		t.Errorf("set past the end: %+v, the content is at %v", s, top.y)
	}
	s.X, s.Y = 50, -10
	tt.Frame()
	if s.X != 0 || s.Y != 0 || top.x != 0 || top.y != 0 {
		t.Errorf("set to (50, -10): %+v, the content is at %v, %v", s, top.x, top.y)
	}
}

func TestTrackScrollSettles(t *testing.T) {
	var s ScrollState
	long := true
	tt := coreNewTester(func(c *context) {
		// Built from the state before the layout keeps the offset within
		// the content: the next frame builds from the kept one.
		if s.Y > 0 {
			coreText(c, "Back to top")
		}
		coreScroll(c).Grow(1).TrackScroll(&s).Children(func() {
			if long {
				coreBox(c).Height(1000).Shrink(0)
			}
		})
	}, 200, 200)
	s.Y = math.MaxFloat32
	tt.Frame()
	if !tt.HasText("Back to top") {
		t.Fatalf("scrolled to the end, %+v", s)
	}
	long = false
	tt.Frame()
	if tt.HasText("Back to top") {
		t.Errorf("the content shrank to nothing, yet %+v", s)
	}
}

func TestTrackScrollFollowsTheEnd(t *testing.T) {
	var s ScrollState
	lines, builds := 10, 0
	tt := coreNewTester(func(c *context) {
		builds++
		if s.Y >= s.MaxY {
			s.Y = math.MaxFloat32
		}
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			for range lines {
				coreBox(c).Height(50).Shrink(0)
			}
		})
	}, 200, 200)
	if s.Y != 300 || s.MaxY != 300 {
		t.Fatalf("ten lines of 50 in 200: %+v", s)
	}
	lines, builds = 20, 0
	tt.Frame()
	if s.Y != 800 || s.MaxY != 800 {
		t.Errorf("ten lines more: %+v", s)
	}
	if builds > 3 {
		t.Errorf("following the end took %d builds", builds)
	}
	// Scrolled up from the end, it stays where the user left it.
	tt.Move(100, 100)
	tt.Scroll(100, 100, 0, -100)
	lines = 30
	tt.Frame()
	if s.Y != 700 || s.MaxY != 1300 {
		t.Errorf("scrolled up, then ten lines more: %+v", s)
	}
}

func TestTrackScrollPerPage(t *testing.T) {
	pages := map[string]*ScrollState{"a": {}, "b": {}}
	page := "a"
	var top *node
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(pages[page]).Children(func() {
			top = coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Move(100, 100)
	tt.Scroll(100, 100, 0, 400)
	page = "b"
	tt.Frame()
	if top.y != 0 || pages["a"].Y != 400 {
		t.Errorf("page b shows from %v; page a kept %v", -top.y, pages["a"].Y)
	}
	page = "a"
	tt.Frame()
	if top.y != -400 {
		t.Errorf("back on page a, it shows from %v", -top.y)
	}
}

func TestTrackScrollList(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreList(c, nil, 1000, func(i int) { coreTextf(c, "Row %d", i).Height(20) }).Fill().TrackScroll(&s)
	}, 200, 200)
	// The frame that scrolls builds the rows it shows.
	s.Y = 500 * 20
	tt.rt.runFrame()
	if r, ok := tt.Find("Row 500"); !ok || r.Y != 0 {
		t.Errorf("scrolled to row 500, it is at %v (%v)", r.Y, ok)
	}
	s.Y = math.MaxFloat32
	tt.rt.runFrame()
	if r, ok := tt.Find("Row 999"); !ok || r.Y+r.H > 200 {
		t.Errorf("scrolled to the end, the last row is at %v (%v)", r, ok)
	}
}

func TestScrollIntoView(t *testing.T) {
	n, show := 2, -1
	rows := map[int]*node{}
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().Children(func() {
			for i := range n {
				h := float32(100)
				if i == 7 {
					h = 400
				}
				rows[i] = coreBox(c).Height(h).Shrink(0)
				if i == show {
					rows[i].ScrollIntoView()
					show = -1
				}
			}
		})
	}, 200, 250)
	// A row added below shows in the frame that adds it, at the bottom.
	n, show = 10, 9
	tt.rt.runFrame()
	if r := rows[9]; r.y != 150 {
		t.Errorf("row 9 shows at %v", r.y)
	}
	// A row above comes down to the top; one in view stays.
	show = 4
	tt.Frame()
	if r := rows[4]; r.y != 0 {
		t.Errorf("row 4 shows at %v", r.y)
	}
	show = 5
	tt.Frame()
	if r := rows[4]; r.y != 0 {
		t.Errorf("showing row 5, in view, moved row 4 to %v", r.y)
	}
	// What is taller than the view shows from its start.
	show = 7
	tt.Frame()
	if r := rows[7]; r.y != 0 {
		t.Errorf("row 7, 400 high, shows from %v", r.y)
	}
}

func TestScrollIntoViewNested(t *testing.T) {
	show := false
	var row *node
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().Children(func() {
			coreBox(c).Height(300).Shrink(0)
			coreScroll(c).Height(150).Shrink(0).Children(func() {
				for i := range 10 {
					r := coreBox(c).Height(50).Shrink(0)
					if i == 8 {
						row = r
						if show {
							r.ScrollIntoView()
						}
					}
				}
			})
		})
	}, 200, 200)
	show = true
	tt.Frame()
	// The inner list shows row 8 at its bottom, and the window the list's
	// bottom at its own.
	if row.y != 150 {
		t.Errorf("row 8 shows at %v", row.y)
	}
}

// revealTester shows 30 boxes of 20 DIPs in a container 100 DIPs long,
// down or sideways, which brings box 15 into view as *o says while ask
// is set; *s tracks the container.
func revealTester(sideways bool, s *ScrollState, ask *bool, o *[]ScrollIntoViewOptions) *Tester {
	return coreNewTester(func(c *context) {
		box := coreScroll(c).Height(100).Width(100)
		if sideways {
			box = coreScrollHorizontal(c).Height(100).Width(100)
		}
		box.TrackScroll(s).Children(func() {
			for i := range 30 {
				r := coreBox(c).Height(20).Width(20).Shrink(0)
				if i == 15 && *ask {
					r.ScrollIntoView(*o...)
				}
			}
		})
	}, 200, 200)
}

// TestScrollIntoViewAlign lines a box up as the web's scrollIntoView's
// block and inline do, down and sideways, both ways.
func TestScrollIntoViewAlign(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    []ScrollIntoViewOptions
		want float32
	}{
		{"none", nil, 220},
		{"zero", []ScrollIntoViewOptions{{}}, 220},
		{"nearest", []ScrollIntoViewOptions{{Block: ScrollNearest, Inline: ScrollNearest}}, 220},
		{"start", []ScrollIntoViewOptions{{Block: ScrollStart, Inline: ScrollStart}}, 300},
		{"center", []ScrollIntoViewOptions{{Block: ScrollCenter, Inline: ScrollCenter}}, 260},
		{"end", []ScrollIntoViewOptions{{Block: ScrollEnd, Inline: ScrollEnd}}, 220},
	} {
		for _, way := range []string{"down", "sideways", "sideways right to left"} {
			var s ScrollState
			ask := false
			tt := revealTester(way != "down", &s, &ask, &tc.o)
			if way == "sideways right to left" {
				// Offsets count from the start, at the right, as does the
				// inline start: the same offsets line the box up.
				tt.SetDirection(RightToLeft)
			}
			ask = true
			tt.Frame()
			ask = false
			got := s.Y
			if way != "down" {
				got = s.X
			}
			if got != tc.want {
				t.Errorf("%s, %s: scrolled to %v, want %v", tc.name, way, got, tc.want)
			}
			// It stays there in the next frame.
			tt.Frame()
			if way == "down" && s.Y != got || way != "down" && s.X != got {
				t.Errorf("%s, %s: moved on to %v, %v in the next frame", tc.name, way, s.X, s.Y)
			}
		}
	}
	// Lined up from below, as from above.
	var s ScrollState
	ask := false
	o := []ScrollIntoViewOptions{{Block: ScrollStart}}
	tt := revealTester(false, &s, &ask, &o)
	s.Y = math.MaxFloat32
	tt.Frame()
	ask = true
	tt.Frame()
	if s.Y != 300 {
		t.Errorf("start from the end: scrolled to %v, want 300", s.Y)
	}
}

// TestScrollIntoViewSmooth glides to the box, as the web's behavior:
// "smooth" does.
func TestScrollIntoViewSmooth(t *testing.T) {
	var s ScrollState
	ask := false
	o := []ScrollIntoViewOptions{{Block: ScrollCenter, Animation: &Animation{Duration: 200 * time.Millisecond, Ease: Linear}}}
	tt := revealTester(false, &s, &ask, &o)
	ask = true
	tt.Frame()
	ask = false
	if s.Y != 0 || !tt.FramePending() {
		t.Fatalf("a glide scrolled at once to %v, or asked for no frame (%v)", s.Y, tt.FramePending())
	}
	tt.Advance(100 * time.Millisecond)
	if s.Y < 100 || s.Y > 160 {
		t.Errorf("half way: at %v, want about 130", s.Y)
	}
	tt.Advance(time.Second)
	if math.Abs(float64(s.Y-260)) > 0.01 || tt.FramePending() {
		t.Errorf("there: at %v, want 260, frames pending %v", s.Y, tt.FramePending())
	}
	// Asked again on the way, it goes on.
	s.Y = 0
	tt.Frame()
	ask = true
	for range 30 {
		tt.Frame()
		tt.Advance(10 * time.Millisecond)
	}
	ask = false
	if math.Abs(float64(s.Y-260)) > 0.01 {
		t.Errorf("asked in every frame: at %v, want 260", s.Y)
	}
	// The user's wheel takes over.
	s.Y = 0
	tt.Frame()
	ask = true
	tt.Frame()
	ask = false
	tt.Advance(50 * time.Millisecond)
	tt.Move(50, 50)
	tt.Wheel(50, 50, 0, -100)
	tt.Advance(time.Second)
	if s.Y != 0 {
		t.Errorf("a notch up on the way: at %v, want 0", s.Y)
	}
	// So does the app.
	ask = true
	tt.Frame()
	ask = false
	tt.Advance(50 * time.Millisecond)
	s.Y = 10
	tt.Advance(time.Second)
	if s.Y != 10 {
		t.Errorf("set to 10 on the way: at %v", s.Y)
	}
	// With less motion, or animations disabled, it scrolls at once.
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	s.Y = 0
	tt.Frame()
	ask = true
	tt.Frame()
	ask = false
	if s.Y != 260 {
		t.Errorf("with less motion: at %v, want 260", s.Y)
	}
}

// TestScrollIntoViewSmoothSideways glides sideways, right to left too.
func TestScrollIntoViewSmoothSideways(t *testing.T) {
	for _, dir := range []Direction{LeftToRight, RightToLeft} {
		var s ScrollState
		ask := false
		o := []ScrollIntoViewOptions{{Inline: ScrollCenter, Animation: &Smooth}}
		tt := revealTester(true, &s, &ask, &o)
		tt.SetDirection(dir)
		ask = true
		tt.Frame()
		ask = false
		tt.Advance(50 * time.Millisecond)
		if s.X <= 0 || s.X >= 260 {
			t.Errorf("%v: on the way at %v", dir, s.X)
		}
		tt.Advance(2 * time.Second)
		if math.Abs(float64(s.X-260)) > 0.01 {
			t.Errorf("%v: at %v, want 260", dir, s.X)
		}
	}
}

// TestScrollIntoViewSmoothSnaps rests a container that snaps at the place
// nearest where it glided.
func TestScrollIntoViewSmoothSnaps(t *testing.T) {
	var s ScrollState
	ask := false
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Height(100).ScrollSnap(SnapMandatory).TrackScroll(&s).Children(func() {
			for i := range 30 {
				r := coreBox(c).Height(30).Shrink(0).SnapAlign(Start)
				if i == 15 && ask {
					// Centered at 415, 5 DIPs from box 14's place.
					r.ScrollIntoView(ScrollIntoViewOptions{Block: ScrollCenter, Animation: &Smooth})
				}
			}
		})
	}, 200, 200)
	ask = true
	tt.Frame()
	ask = false
	tt.Advance(3 * time.Second)
	if math.Abs(float64(s.Y-420)) > 0.01 {
		t.Errorf("at %v, want 420", s.Y)
	}
}

// TestScrollIntoViewSmoothNested glides the containers around the box,
// each to where it lines the box up once they all are there.
func TestScrollIntoViewSmoothNested(t *testing.T) {
	ask := false
	var outer, inner ScrollState
	var row *node
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&outer).Children(func() {
			coreBox(c).Height(300).Shrink(0)
			coreScroll(c).Height(150).Shrink(0).TrackScroll(&inner).Children(func() {
				for i := range 10 {
					r := coreBox(c).Height(50).Shrink(0)
					if i == 8 {
						row = r
						if ask {
							r.ScrollIntoView(ScrollIntoViewOptions{Block: ScrollStart, Animation: &Smooth})
						}
					}
				}
			})
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	ask = true
	tt.Frame()
	ask = false
	tt.Advance(2 * time.Second)
	// The inner container scrolls as far as it goes, showing row 8 50
	// DIPs from its top, and the window row 8 at its own.
	if math.Abs(float64(inner.Y-350)) > 0.01 || math.Abs(float64(outer.Y-350)) > 0.01 || math.Abs(float64(row.y)) > 0.01 {
		t.Errorf("inner at %v, outer at %v, row 8 at %v; want 350, 350, 0", inner.Y, outer.Y, row.y)
	}
}

// TestScrollIntoViewSmoothList glides a List to a row it built.
func TestScrollIntoViewSmoothList(t *testing.T) {
	var s ListState
	show := -1
	rows := map[int]*node{}
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 200, func(i int) {
			r := coreBox(c).Height(20)
			rows[i] = r
			if i == show {
				r.ScrollIntoView(ScrollIntoViewOptions{Block: ScrollCenter, Animation: &Smooth})
				show = -1
			}
		}).Fill()
	}, 200, 200)
	show = 9
	tt.Frame()
	tt.Advance(2 * time.Second)
	if first, last := s.Visible(); first != 4 || last != 14 {
		t.Errorf("row 9 centered: rows %v to %v show, want 4 to 14", first, last)
	}
	if r := rows[9]; math.Abs(float64(r.y-90)) > 0.01 {
		t.Errorf("row 9 shows at %v, want 90", r.y)
	}
}

func TestScrollIntoViewOptionsPanic(t *testing.T) {
	for name, o := range map[string][]ScrollIntoViewOptions{
		"two":       {{}, {}},
		"alignment": {{Block: ScrollEnd + 1}},
		"animation": {{Animation: &Animation{Duration: -1}}},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: no panic", name)
				}
			}()
			coreNewTester(func(c *context) { coreBox(c).ScrollIntoView(o...) }, 100, 100)
		}()
	}
}

func TestTabScrollsTheFocusIntoView(t *testing.T) {
	var button *node
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().Children(func() {
			coreBox(c).Height(300).Shrink(0)
			coreScroll(c).Height(150).Shrink(0).Children(func() {
				coreBox(c).Height(400).Shrink(0)
				button = coreButton(c, "Deep")
			})
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Key(0, KeyTab)
	if !tt.Focused("Deep") || button.y < 0 || button.y+button.h > 200 {
		t.Errorf("Tab to a button in a list below the window: it is at %v..%v", button.y, button.y+button.h)
	}
}
