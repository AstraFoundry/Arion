package ui

import (
	"testing"
	"time"
)

// A press the system cancels (Pointer Events' pointercancel) ends with
// no click: the button is pressed no more, and the next press clicks as
// usual.
func TestPointerCancelClicksNothing(t *testing.T) {
	clicks, rightClicks, canceled := 0, 0, 0
	var pressed bool
	tt := coreNewTester(func(c *context) {
		b := coreButton(c, "Send")
		if b.Clicked() {
			clicks++
		}
		if b.RightClicked() {
			rightClicks++
		}
		if b.PressCanceled() {
			canceled++
		}
		pressed = b.Pressed()
	}, 200, 100)
	r, _ := tt.Find("Send")
	x, y := r.X+r.W/2, r.Y+r.H/2

	tt.Press(x, y)
	tt.Frame()
	if !pressed {
		t.Fatal("the button pressed is not Pressed")
	}
	tt.CancelPress()
	tt.Frame()
	if clicks != 0 || pressed || canceled != 1 {
		t.Fatalf("canceled: %d clicks, pressed %v, PressCanceled %d times; want 0, false, 1", clicks, pressed, canceled)
	}
	tt.Frame()
	if canceled != 1 {
		t.Fatalf("PressCanceled reported in %d frames, want 1", canceled)
	}
	// The release the system kept never comes; one coming anyway, with
	// nothing pressed, clicks nothing either.
	tt.Release(x, y)
	if clicks != 0 {
		t.Fatalf("a stray release after a cancel clicked %d times", clicks)
	}
	// The next press starts anew.
	tt.ClickAt(x, y)
	if clicks != 1 || canceled != 1 {
		t.Fatalf("after the cancel, a click: %d clicks, %d cancels; want 1, 1", clicks, canceled)
	}

	// The secondary button's press canceled right-clicks nothing.
	tt.PressButton(1, x, y)
	tt.CancelPress()
	tt.Frame()
	if rightClicks != 0 {
		t.Fatalf("a canceled right press right-clicked %d times", rightClicks)
	}
	tt.RightClickAt(x, y)
	if rightClicks != 1 {
		t.Fatalf("a right click after a cancel: %d right clicks, want 1", rightClicks)
	}
}

// HandleInput gets the release of a press the system cancels, as
// Canceled, of the button pressed, so that it lets go of what it holds;
// observers see it too. A release the pointer makes is not Canceled,
// and a cancel with nothing pressed sends nothing.
func TestPointerCancelReachesHandleInput(t *testing.T) {
	var got, seen []InputEvent
	fired := 0
	armed := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(200, 200).ObserveInput(func(ev InputEvent) {
			if ev.Kind == InputPointerUp {
				seen = append(seen, ev)
			}
		}).Children(func() {
			coreBox(c).Size(120, 40).Label("pad").HandleInput(func(ev InputEvent) bool {
				switch ev.Kind {
				case InputPointerDown:
					armed = true
				case InputPointerUp:
					got = append(got, ev)
					if armed && !ev.Canceled {
						fired++
					}
					armed = false
				}
				return true
			})
		})
	}, 200, 200)
	tt.Frame()

	tt.ClickAt(20, 20)
	if len(got) != 1 || got[0].Canceled || fired != 1 {
		t.Fatalf("a click: releases %+v, fired %d", got, fired)
	}
	got, seen = nil, nil
	tt.PressButton(2, 20, 20)
	tt.Move(60, 30)
	tt.CancelPress()
	if len(got) != 1 || !got[0].Canceled || got[0].Button != 2 || fired != 1 || armed {
		t.Fatalf("a middle press canceled: releases %+v, fired %d, armed %v", got, fired, armed)
	}
	if len(seen) != 1 || !seen[0].Canceled || seen[0].Button != 2 {
		t.Fatalf("an observer saw %+v of a canceled press", seen)
	}
	// Nothing is pressed now: a second cancel sends nothing.
	got, seen = nil, nil
	tt.CancelPress()
	if len(got) != 0 || len(seen) != 0 {
		t.Fatalf("a cancel with nothing pressed sent %+v, observed %+v", got, seen)
	}
	// A press still takes the release that follows it.
	tt.Press(20, 20)
	tt.Release(20, 20)
	if len(got) != 1 || got[0].Canceled || fired != 2 {
		t.Fatalf("a click after the cancel: releases %+v, fired %d", got, fired)
	}
}

// The window losing the keyboard ends a press as the system cancelling
// it does: HandleInput's release is Canceled, and the press clicks
// nothing.
func TestPointerCancelOnBlur(t *testing.T) {
	var got []InputEvent
	clicks, canceled := 0, 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreBox(c).Size(100, 40).Label("pad").HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputPointerUp {
					got = append(got, ev)
				}
				return true
			})
			b := coreButton(c, "Send")
			if b.Clicked() {
				clicks++
			}
			if b.PressCanceled() {
				canceled++
			}
		})
	}, 200, 200)
	tt.Frame()
	tt.Press(20, 20)
	tt.SetFocused(false)
	if len(got) != 1 || !got[0].Canceled {
		t.Fatalf("the window lost the keyboard while pressed: releases %+v, want one Canceled", got)
	}
	tt.SetFocused(true)
	r, _ := tt.Find("Send")
	tt.Press(r.X+5, r.Y+5)
	tt.SetFocused(false)
	tt.Release(r.X+5, r.Y+5)
	tt.Frame()
	if clicks != 0 || canceled != 1 {
		t.Fatalf("a press ended by the window losing the keyboard: %d clicks, %d cancels; want 0, 1", clicks, canceled)
	}
}

// A value dragged when the system cancels the press drops nowhere and
// is not missed, as one given up with Escape.
func TestPointerCancelDropsNothing(t *testing.T) {
	dropped, missed, clicks := 0, 0, 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(20).Padding(20).Children(func() {
			src := coreBox(c).Size(100, 40).Label("Source").Drag("hello")
			if src.Clicked() {
				clicks++
			}
			if _, ok := src.DragMissed(); ok {
				missed++
			}
			target := coreBox(c).Size(200, 100).Label("Target")
			if _, ok := coreDrop[string](target); ok {
				dropped++
			}
		})
	}, 400, 400)
	src, _ := tt.Find("Source")
	target, _ := tt.Find("Target")
	tt.Press(src.X+10, src.Y+10)
	tt.Move(src.X+30, src.Y+30)
	tt.Move(target.X+20, target.Y+20)
	tt.CancelPress()
	tt.Frame()
	if dropped != 0 || missed != 0 || clicks != 0 {
		t.Fatalf("a drag canceled over its target: %d drops, %d misses, %d clicks", dropped, missed, clicks)
	}
	if tt.rt.drag != nil || tt.rt.pressed != nil {
		t.Fatal("the drag or the press outlived the cancel")
	}
	// The next drag drops.
	drag(tt, src.X+10, src.Y+10, target.X+20, target.Y+20)
	if dropped != 1 {
		t.Fatalf("a drag after the cancel dropped %d times, want 1", dropped)
	}
}

// A drag the system cancels ends no drag (DragEnded) and reports
// PressCanceled, so that a gesture undoes what it previewed.
func TestPointerCancelEndsNoDrag(t *testing.T) {
	width, preview := float32(100), float32(0)
	ended := 0
	tt := coreNewTester(func(c *context) {
		h := coreBox(c).Size(20, 20).Label("handle")
		dx, _, held := h.Dragged()
		if _, _, ok := h.DragEnded(); ok {
			ended++
		}
		switch {
		case held:
			preview += dx
		case h.PressCanceled():
			preview = 0
		case preview != 0:
			width, preview = width+preview, 0
		}
	}, 200, 200)
	tt.Frame()
	tt.Press(10, 10)
	tt.Move(30, 10)
	tt.Move(50, 10)
	tt.Frame()
	if preview != 40 {
		t.Fatalf("dragged by 40, the preview is %v", preview)
	}
	tt.CancelPress()
	tt.Frame()
	tt.Frame()
	if width != 100 || preview != 0 || ended != 0 {
		t.Fatalf("a drag canceled: width %v, preview %v, DragEnded %d times; want 100, 0, 0", width, preview, ended)
	}
	// Let go, a drag commits.
	drag(tt, 10, 10, 50, 10)
	tt.Frame()
	tt.Frame()
	if width != 140 || ended != 1 {
		t.Fatalf("a drag let go: width %v, DragEnded %d times; want 140, 1", width, ended)
	}
}

// A Draggable element the system takes the press of throws nothing: it
// goes back to where the press took it.
func TestPointerCancelDraggableGoesBack(t *testing.T) {
	var off Point
	var card *node
	tt, now := motionTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{
			Axis:   Horizontal,
			Bounds: &Rect{W: 200},
			Snap:   Nearest(Point{}, Point{X: 200}),
		})
	})
	// Fast enough that a release would throw it to 200.
	throw(tt, now, 20, 20, 60, 0, 6)
	if off != (Point{X: 60}) {
		t.Fatalf("dragged by 60, the offset is %v", off)
	}
	*now = now.Add(5 * time.Millisecond)
	tt.CancelPress()
	if off != (Point{}) {
		t.Fatalf("canceled, it rests at %v, want where the press took it", off)
	}
	if v := card.st.dragRec.ch[0].velocity; v != 0 {
		t.Fatalf("canceled, it was thrown at %v", v)
	}
	*now = now.Add(2 * time.Second)
	tt.Frame()
	if card.st.world.X != 0 || card.Transitioning() {
		t.Fatalf("back at rest, the card shows at %+v", card.st.world)
	}
}
