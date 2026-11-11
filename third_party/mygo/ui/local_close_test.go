package ui

import (
	"io"
	"slices"
	"testing"
	"time"
)

// feed stands for a component's background work, which Close cancels.
type feed struct {
	name   string
	closed *[]string
	// inFrame notes whether Close ran during a frame of *rt.
	rt      **engine
	inFrame *bool
}

func (f *feed) Close() error {
	*f.closed = append(*f.closed, f.name)
	if f.rt != nil && (*f.rt).inFrame {
		*f.inFrame = true
	}
	return nil
}

// A Local that is an io.Closer stays open while its element is built and
// closes once, after the frame that no longer builds it.
func TestLocalClosesWhenElementGoes(t *testing.T) {
	var closed []string
	var inFrame bool
	shown, made := true, 0
	var rt *engine
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			if shown {
				e := Box(c).Key("feed").Size(10, 10)
				Local(e, "feed", func() *feed {
					made++
					return &feed{name: "feed", closed: &closed, rt: &rt, inFrame: &inFrame}
				})
			}
			Box(c).Key("other").Size(10, 10)
		})
	}, 100, 100)
	rt = tt.rt
	for range 3 {
		tt.Frame()
	}
	if made != 1 || len(closed) != 0 {
		t.Fatalf("built three times: made %d, closed %v", made, closed)
	}
	shown = false
	tt.Frame()
	if !slices.Equal(closed, []string{"feed"}) {
		t.Fatalf("the element went: closed %v", closed)
	}
	if inFrame {
		t.Error("Close ran during the frame")
	}
	tt.Frame()
	tt.Frame()
	if len(closed) != 1 {
		t.Fatalf("closed %d times", len(closed))
	}
	// Built again, it starts anew.
	shown = true
	tt.Frame()
	if made != 2 || len(closed) != 1 {
		t.Fatalf("back: made %d, closed %v", made, closed)
	}
	tt.rt.close()
	if len(closed) != 2 {
		t.Fatalf("the window closed: closed %v", closed)
	}
	tt.rt.close()
	if len(closed) != 2 {
		t.Fatalf("closed twice: %v", closed)
	}
}

// valueFeed closes through a pointer to it, kept by value.
type valueFeed struct{ closed *[]string }

func (f *valueFeed) Close() error {
	*f.closed = append(*f.closed, "value")
	return nil
}

// Locals close by T or by *T, and through an interface; a nil pointer or
// interface does not close, and one set after init does; others cost
// nothing.
func TestLocalCloserKinds(t *testing.T) {
	var closed []string
	shown := true
	var st *state
	tt := NewTester(func(c *Context) {
		if !shown {
			return
		}
		e := Box(c).Size(10, 10)
		st = e.node().st
		Local(e, "value", func() valueFeed { return valueFeed{&closed} })
		Local(e, "iface", func() io.Closer { return &feed{name: "iface", closed: &closed} })
		Local(e, "nil iface", func() io.Closer { return nil })
		Local(e, "nil", func() *feed { return nil })
		late := Local(e, "late", func() *feed { return nil })
		if *late == nil {
			*late = &feed{name: "late", closed: &closed}
		}
		Local(e, "int", func() int { return 1 })
		Local(e, "struct", func() struct{ f *feed } { return struct{ f *feed }{&feed{name: "struct", closed: &closed}} })
	}, 100, 100)
	if n := len(st.closes); n != 5 {
		t.Fatalf("%d Locals may close, want 5", n)
	}
	shown = false
	tt.Frame()
	// The last made first, as deferred calls run.
	if want := []string{"late", "iface", "value"}; !slices.Equal(closed, want) {
		t.Fatalf("closed %v, want %v", closed, want)
	}
}

// An element leaving with an exit transition closes its Locals, and its
// children's, once its copy is gone, not as it starts leaving.
func TestLocalClosesAfterExit(t *testing.T) {
	var closed []string
	shown := true
	tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear, Exit: &Motion{}}
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			if shown {
				e := Box(c).Key("panel").Size(40, 40).Transition(tr)
				Local(e, "feed", func() *feed { return &feed{name: "panel", closed: &closed} })
				e.Children(func() {
					child := Box(c).Size(10, 10)
					Local(child, "feed", func() *feed { return &feed{name: "child", closed: &closed} })
				})
			}
		})
	}, 100, 100)
	tt.Frame()
	shown = false
	tt.Frame()
	if len(tt.rt.exits) == 0 {
		t.Fatal("the element left without an exit")
	}
	if len(closed) != 0 {
		t.Fatalf("closed %v as the exit started", closed)
	}
	tt.Advance(50 * time.Millisecond)
	if len(closed) != 0 {
		t.Fatalf("closed %v halfway through the exit", closed)
	}
	tt.Advance(100 * time.Millisecond)
	slices.Sort(closed)
	if !slices.Equal(closed, []string{"child", "panel"}) {
		t.Fatalf("after the exit: closed %v", closed)
	}
	tt.Advance(100 * time.Millisecond)
	if len(closed) != 2 || len(tt.rt.exits) != 0 {
		t.Fatalf("closed %v, %d exits waiting", closed, len(tt.rt.exits))
	}
}

// An element built again while its copy leaves made its Locals anew: the
// copy's close then.
func TestLocalClosesWhenExitInterrupted(t *testing.T) {
	var closed []string
	shown, made := true, 0
	tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear, Exit: &Motion{}}
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			if shown {
				e := Box(c).Key("panel").Size(40, 40).Transition(tr)
				Local(e, "feed", func() *feed { made++; return &feed{name: "panel", closed: &closed} })
			}
		})
	}, 100, 100)
	shown = false
	tt.Frame()
	tt.Advance(30 * time.Millisecond)
	if len(closed) != 0 {
		t.Fatalf("closed %v while leaving", closed)
	}
	shown = true
	tt.Frame()
	if made != 2 || len(closed) != 1 {
		t.Fatalf("back while leaving: made %d, closed %v", made, closed)
	}
	tt.Advance(time.Second)
	if len(closed) != 1 {
		t.Fatalf("built again: closed %v", closed)
	}
}

// Rows of a list scrolled out of view are not built: their Locals close,
// and a row back in view makes them anew.
func TestLocalClosesListRowsScrolledOut(t *testing.T) {
	open := map[int]int{}
	var s ListState
	tt := NewTester(func(c *Context) {
		List(c, &s, 1000, func(i int) {
			row := Box(c).Height(20)
			Local(row, "row", func() *rowFeed { open[i]++; return &rowFeed{i, open} })
		}).Grow(1)
	}, 100, 100)
	first, last := s.Visible()
	if first != 0 || open[0] != 1 {
		t.Fatalf("rows %d to %d, open %v", first, last, open)
	}
	s.ScrollTo(500, Start)
	tt.Frame()
	if open[0] != 0 || open[500] != 1 {
		t.Fatalf("scrolled to row 500: open %v", open)
	}
	if len(open) > 2*(last-first+1)+2 {
		t.Fatalf("%d rows open", len(open))
	}
	s.ScrollTo(0, Start)
	tt.Frame()
	if open[0] != 1 || open[500] != 0 {
		t.Fatalf("back to row 0: open %v", open)
	}
}

// rowFeed counts how many of a row's Locals are open.
type rowFeed struct {
	i    int
	open map[int]int
}

func (f *rowFeed) Close() error {
	if f.open[f.i]--; f.open[f.i] == 0 {
		delete(f.open, f.i)
	}
	return nil
}

// panicky panics as it closes.
type panicky struct{ closed *int }

func (p *panicky) Close() error {
	*p.closed++
	panic("closing")
}

// A Close that panics panics out of the frame; the Locals left close after
// the next.
func TestLocalClosePanics(t *testing.T) {
	closed, shown := 0, true
	var others []string
	tt := NewTester(func(c *Context) {
		if shown {
			e := Box(c).Size(10, 10)
			Local(e, "a", func() *feed { return &feed{name: "a", closed: &others} })
			Local(e, "b", func() *panicky { return &panicky{&closed} })
		}
	}, 100, 100)
	shown = false
	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not come out of the frame")
			}
		}()
		tt.Frame()
	}()
	if closed != 1 || len(others) != 0 {
		t.Fatalf("closed %d, others %v", closed, others)
	}
	tt.Frame()
	if closed != 1 || len(others) != 1 {
		t.Fatalf("the next frame: closed %d, others %v", closed, others)
	}
}

// relay changes the app's state as it closes, and asks for a frame.
type relay struct {
	status *string
	svc    Services
}

func (r *relay) Close() error {
	*r.status = "closed"
	r.svc.Invalidate()
	return nil
}

// A Close that changes what the view shows asks for a frame, which builds
// the change.
func TestLocalCloseRebuilds(t *testing.T) {
	status, shown := "open", true
	tt := NewTester(func(c *Context) {
		Text(c, status)
		if shown {
			e := Box(c).Size(10, 10)
			Local(e, "relay", func() *relay { return &relay{&status, c.Services()} })
		}
	}, 100, 100)
	shown = false
	tt.Frame()
	if status != "closed" {
		t.Fatalf("status %q", status)
	}
	if !tt.HasText("closed") {
		t.Fatalf("the frame Close asked for shows %v", tt.Texts())
	}
}

// A select keeps a copy of the app's value it shows, which it does not
// close as it goes.
func TestSelectShownValueNotClosed(t *testing.T) {
	var closed []string
	value := &feed{name: "app's", closed: &closed}
	shown, open := true, true
	tt := NewTester(func(c *Context) {
		if shown {
			s := SelectBaseOpen(c, &value, &open)
			s.Trigger.Size(20, 20)
			s.Popup(func(panel Element) {
				s.Item(value).Size(20, 20)
			})
		}
	}, 100, 100)
	tt.Frame()
	shown = false
	tt.Frame()
	tt.rt.close()
	if len(closed) != 0 {
		t.Fatalf("closed %v", closed)
	}
}

// Locals made on an element's place before Key gave it its identity are
// its no more: they close after the frame.
func TestLocalClosesBeforeKey(t *testing.T) {
	var closed []string
	NewTester(func(c *Context) {
		e := Box(c).Size(10, 10)
		Local(e, "early", func() *feed { return &feed{name: "early", closed: &closed} })
		e.Key("late")
	}, 100, 100)
	if !slices.Equal(closed, []string{"early"}) {
		t.Fatalf("closed %v", closed)
	}
}

// A window closed during a frame, as by a close button, closes the Locals
// of its elements as the frame ends.
func TestLocalClosesWithWindowClosedInFrame(t *testing.T) {
	var closed []string
	var inFrame bool
	var rt *engine
	closeNow := false
	tt := NewTester(func(c *Context) {
		e := Box(c).Size(10, 10)
		Local(e, "feed", func() *feed { return &feed{name: "feed", closed: &closed, rt: &rt, inFrame: &inFrame} })
		if closeNow {
			rt.close()
			if len(closed) != 0 {
				t.Error("closed while the frame builds")
			}
		}
	}, 100, 100)
	rt = tt.rt
	closeNow = true
	tt.Frame()
	if len(closed) != 1 || inFrame {
		t.Fatalf("closed %v, during the frame: %v", closed, inFrame)
	}
}

// Elements that keep Closers and stay built cost their frames nothing more
// than those keeping other values.
func TestLocalCloserFrameAllocs(t *testing.T) {
	var closed []string
	allocs := func(closer bool) float64 {
		tt := NewTester(func(c *Context) {
			for i := range 50 {
				e := Box(c).Key(i).Size(2, 2)
				if closer {
					Local(e, "feed", func() *feed { return &feed{closed: &closed} })
				} else {
					Local(e, "n", func() *int { return new(int) })
				}
			}
		}, 100, 100)
		tt.Frame()
		return testing.AllocsPerRun(20, tt.Frame)
	}
	if with, without := allocs(true), allocs(false); with > without {
		t.Fatalf("a frame allocates %.0f times with Closers, %.0f without", with, without)
	}
	if len(closed) != 0 {
		t.Fatalf("closed %d while built", len(closed))
	}
}
