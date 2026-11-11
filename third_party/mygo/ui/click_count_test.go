package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// A handler gets the count of clicks with a press and its release, and
// each button counts its own.
func TestHandlerClickCount(t *testing.T) {
	var clicks []int
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputPointerDown || ev.Kind == InputPointerUp {
				clicks = append(clicks, ev.Clicks)
			}
			return true
		})
	}, 100, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.ClickAt(50, 50)
	tt.ClickAt(50, 50)
	tt.RightClickAt(50, 50)
	tt.ClickAt(50, 50)
	if want := []int{1, 1, 2, 2, 1, 1, 1, 1}; !slices.Equal(clicks, want) {
		t.Fatalf("presses and releases saw clicks %v, want %v", clicks, want)
	}
}

// A release carries its press's count, though another button went down
// meanwhile.
func TestReleaseCarriesPressCount(t *testing.T) {
	var ups []int
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputPointerUp && ev.Button == 0 {
				ups = append(ups, ev.Clicks)
			}
			return ev.Button == 0
		})
	}, 100, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.ClickAt(50, 50)
	tt.Press(50, 50)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: 50, Y: 50, Button: 1})
	tt.Release(50, 50)
	if want := []int{1, 2}; !slices.Equal(ups, want) {
		t.Fatalf("releases saw clicks %v, want %v", ups, want)
	}
}

// A click that puts another element under the pointer does not make the
// next click on it a double click, inside a handler that leaves presses.
func TestClickOnNewElementIsNotDouble(t *testing.T) {
	opened := false
	doubles := 0
	tt := NewTester(func(c *Context) {
		Box(c).Size(200, 100).HandleInput(func(InputEvent) bool { return false }).Children(func() {
			if !opened {
				opened = Button(c.Key("open"), "Open").Size(100, 40).Clicked()
			} else if Button(c.Key("row"), "Row").Size(100, 40).DoubleClicked() {
				doubles++
			}
		})
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.ClickAt(50, 20)
	tt.Frame()
	tt.Frame()
	if !opened {
		t.Fatal("the click did not open")
	}
	tt.ClickAt(50, 20)
	tt.Frame()
	if doubles != 0 {
		t.Fatal("the first click on the new element was a double click")
	}
}

// A backend's count, as macOS's, starts over on another element than the
// last press's: a click then a click on its neighbor is no double click,
// and the release carries the count its press got.
func TestBackendCountPerElement(t *testing.T) {
	var a, b []int
	record := func(to *[]int) func(InputEvent) bool {
		return func(ev InputEvent) bool {
			if ev.Kind == InputPointerDown {
				*to = append(*to, ev.Clicks)
			} else if ev.Kind == InputPointerUp && ev.Clicks != (*to)[len(*to)-1] {
				t.Errorf("a release carried clicks %d, its press %d", ev.Clicks, (*to)[len(*to)-1])
			}
			return true
		}
	}
	tt := NewTester(func(c *Context) {
		Row(c).Children(func() {
			Box(c).Size(50, 50).HandleInput(record(&a))
			Box(c).Size(50, 50).HandleInput(record(&b))
		})
	}, 100, 50)
	defer tt.rt.close()
	tt.Frame()
	for i, x := range []float64{25, 75, 75, 75, 25} {
		tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: x, Y: 25, Clicks: i + 1})
		tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: x, Y: 25, Clicks: i + 1})
	}
	if !slices.Equal(a, []int{1, 1}) || !slices.Equal(b, []int{1, 2, 3}) {
		t.Fatalf("the left box saw clicks %v and the right %v, want [1 1] and [1 2 3]", a, b)
	}
}

// Two handlers in one focusable element count apart: the element pressed
// is the innermost of the target and the handler.
func TestBackendCountPerHandlerInFocusable(t *testing.T) {
	var a, b []int
	record := func(to *[]int) func(InputEvent) bool {
		return func(ev InputEvent) bool {
			if ev.Kind == InputPointerDown {
				*to = append(*to, ev.Clicks)
			}
			return true
		}
	}
	tt := NewTester(func(c *Context) {
		Row(c).Focusable().Label("Grid").Children(func() {
			Box(c).Size(50, 50).HandleInput(record(&a))
			Box(c).Size(50, 50).HandleInput(record(&b))
		})
	}, 100, 50)
	defer tt.rt.close()
	tt.Frame()
	for i, x := range []float64{25, 75, 75} {
		tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: x, Y: 25, Clicks: i + 1})
		tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: x, Y: 25, Clicks: i + 1})
	}
	if !slices.Equal(a, []int{1}) || !slices.Equal(b, []int{1, 2}) {
		t.Fatalf("the left box saw clicks %v and the right %v, want [1] and [1 2]", a, b)
	}
}

// A press the scroll bar takes is of no element: a click on the content,
// one on the bar beside it, and one on the content again make no double
// click.
func TestScrollbarPressStartsOver(t *testing.T) {
	doubles := 0
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Size(300, 200).Children(func() {
			row := coreBox(c).Key("row").Size(300, 2000)
			row.flags |= flagClickable
			if row.DoubleClicked() {
				doubles++
			}
		})
	}, 300, 200)
	defer tt.rt.close()
	tt.Frame()
	tt.Move(150, 100)
	var st *state
	for id := tt.rt.hitChain(150, 100)[0]; id != 0; id = st.parent {
		if st = tt.rt.states[id]; st.flags&flagScrollY != 0 {
			break
		}
	}
	bars := scrollBars(Rect{st.x, st.y, st.w, st.h}, st.barInset, float32(st.contentW), float32(st.contentH), 0, float32(st.scrollY), st.flags, tt.rt.c.theme.scrollbarWidth(), false)
	content, bar, y := bars.vTrack.X-1, bars.vTrack.X+4, bars.v.Y+2
	tt.Move(content, y)
	tt.ClickAt(content, y)
	tt.ClickAt(bar, y)
	tt.Frame()
	tt.ClickAt(content, y)
	tt.Frame()
	if doubles != 0 || tt.rt.pressClicks != 1 {
		t.Fatalf("%d double clicks, the last press counted %d, want none and 1", doubles, tt.rt.pressClicks)
	}
}

// A hidden scroll bar takes no press: one on its track before the
// pointer showed it, as the first click into a window, goes to the row
// under it. Once the bar shows, a press there pages, and once the
// pointer leaves and it hides again, a press goes to the row again.
func TestHiddenScrollbarPassesPress(t *testing.T) {
	var sc *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		sc = coreScroll(c).Size(100, 200).Children(func() {
			row := coreBox(c).Key("row").Size(100, 800)
			row.flags |= flagClickable
			if row.Clicked() {
				clicks++
			}
		})
	}, 200, 200)
	defer tt.rt.close()
	tt.Frame()
	st := sc.st
	g := scrollBars(Rect{st.x, st.y, st.w, st.h}, st.barInset, float32(st.contentW), float32(st.contentH), 0, 0, st.flags, tt.rt.c.theme.scrollbarWidth(), false)
	// On the track, below the thumb.
	x, y := g.vTrack.X+g.vTrack.W/2, g.v.Y+g.v.H+20
	press := func() {
		tt.Press(x, y)
		tt.Release(x, y)
		tt.Frame()
	}
	press()
	if clicks != 1 || st.scrollY != 0 {
		t.Fatalf("a press on the hidden track made %d clicks and scrolled to %v, want 1 and 0", clicks, st.scrollY)
	}
	if !st.barsShown {
		t.Fatal("the bar does not show with the pointer over the container")
	}
	press()
	if clicks != 1 || st.scrollY == 0 {
		t.Fatalf("a press on the track showing made %d clicks and scrolled to %v, want 1 and down a page", clicks, st.scrollY)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.PointerLeave})
	tt.Frame()
	scrolled := st.scrollY
	press()
	if clicks != 2 || st.scrollY != scrolled {
		t.Fatalf("a press on the track hidden again made %d clicks and scrolled from %v to %v, want 2 and no scrolling", clicks, scrolled, st.scrollY)
	}
}

// A release that comes while a handler handles its press, through a main
// loop it runs nested, as GTK's clipboard waits do, ends the press once it
// is recorded: the handler hears of it after the press, and the element a
// handler left the press to is clicked.
func TestReleaseDuringPress(t *testing.T) {
	for _, takes := range []bool{true, false} {
		var events []InputKind
		clicked := 0
		var tt *Tester
		tt = NewTester(func(c *Context) {
			Box(c).Size(100, 100).HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputPointerDown || ev.Kind == InputPointerUp {
					events = append(events, ev.Kind)
				}
				if ev.Kind == InputPointerDown {
					tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: 50, Y: 50})
				}
				return takes
			}).Children(func() {
				if Button(c, "OK").Size(100, 100).Clicked() {
					clicked++
				}
			})
		}, 100, 100)
		tt.Frame()
		tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: 50, Y: 50})
		tt.Frame()
		if tt.rt.pressed != nil {
			t.Errorf("taken %v: the press outlived its release", takes)
		}
		want, clicks := []InputKind{InputPointerDown, InputPointerUp}, 0
		if !takes {
			want, clicks = []InputKind{InputPointerDown}, 1
		}
		if !slices.Equal(events, want) {
			t.Errorf("taken %v: the handler heard %v, want %v", takes, events, want)
		}
		if clicked != clicks {
			t.Errorf("taken %v: the button was clicked %d times, want %d", takes, clicked, clicks)
		}
		tt.rt.close()
	}
}

// A release that comes with moves no frame read, and the window losing the
// focus before the next frame, click the element once: the release ended
// the press, and the blur ends no other.
func TestBlurAfterReleaseOfDrag(t *testing.T) {
	held, clicks := false, 0
	tt := NewTester(func(c *Context) {
		b := Box(c).Size(100, 100)
		_, _, held = b.Dragged()
		if b.Clicked() {
			clicks++
		}
	}, 100, 100)
	tt.Press(20, 50)
	tt.Move(30, 50)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: 40, Y: 50})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceBlur})
	tt.Frame()
	if held || clicks != 1 {
		t.Errorf("let go, then the window lost the focus: still pressed %v, %d clicks, want 1", held, clicks)
	}
}

// Enter on an element whose release still waits for its pass clicks it
// after that release: the element sees both clicks, the last without the
// release's Shift.
func TestEnterAfterReleaseOfDrag(t *testing.T) {
	clicks, mods := 0, Modifiers(0)
	tt := NewTester(func(c *Context) {
		b := Box(c).Size(100, 100).Focusable()
		if n := b.Clicks(); n > 0 {
			clicks += n
			mods = b.ClickModifiers()
		}
	}, 100, 100)
	shift := platform.Modifiers(Shift)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: 20, Y: 50, Mods: shift})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 30, Y: 50, Mods: shift})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: 30, Y: 50, Mods: shift})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyEnter)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(KeyEnter)})
	tt.Frame()
	if clicks != 2 || mods != 0 {
		t.Errorf("Shift-clicked, then Enter before a frame: %d clicks, the last with %v, want 2 and none", clicks, mods)
	}
}

// A press that moves the element taking it into another container, as a
// Command-click on a link in a terminal opening a side panel beside it,
// gets its release, though the element pressed is gone: one built over by
// a pane of the split, or, keyed, not built again. The next click on the
// element moved is a press and a release of its own, not that release.
// Built over, the release comes in the axes of the element pressed, not
// of the pane built in its place.
func TestPressOfElementGoneReleases(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		split, held, presses, releases := false, false, 0, 0
		var down, up [2]float32
		size := float32(100)
		tt := NewTester(func(c *Context) {
			element := func() {
				b := c
				if keyed {
					b = c.Key("element")
				}
				Box(b).Size(100, 100).HandleInput(func(ev InputEvent) bool {
					if held {
						// The press is the handler's: its moves and its
						// release are no one else's.
						held = ev.Kind != InputPointerUp
						if !held {
							releases++
							if releases == 1 {
								up = [2]float32{ev.X, ev.Y}
							}
						}
						return true
					}
					if ev.Kind == InputPointerDown && ev.Button == 0 {
						presses++
						if presses == 1 {
							down = [2]float32{ev.X, ev.Y}
						}
						held, split = true, true
						return true
					}
					return false
				})
			}
			if !split {
				Row(c).Size(200, 100).Padding(20).Children(element)
			} else {
				Split(c, &size, element, func() { Box(c).Size(10, 10) }).Size(200, 100)
			}
		}, 200, 100)
		tt.Frame()
		tt.ClickAt(50, 50)
		if !split || held {
			t.Errorf("keyed %v: the press moved the element %v, and its handler still held it %v; want moved, not held", keyed, split, held)
		}
		if !keyed && up != down {
			t.Errorf("the release came at %v, want %v, where the press came in the axes of the element pressed", up, down)
		}
		tt.Advance(time.Second)
		tt.ClickAt(50, 50)
		if presses != 2 || releases != 2 {
			t.Errorf("keyed %v: the handler saw %d presses and %d releases, want 2 and 2", keyed, presses, releases)
		}
		tt.rt.close()
	}
}
