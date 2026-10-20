package ui

import (
	"testing"
	"time"
)

// throw presses at (x, y) and moves the pointer by (dx, dy) in steps of
// 10 ms, without letting go.
func throw(tt *Tester, now *time.Time, x, y, dx, dy float32, steps int) {
	tt.Move(x, y)
	tt.Press(x, y)
	for i := 1; i <= steps; i++ {
		*now = now.Add(10 * time.Millisecond)
		k := float32(i) / float32(steps)
		tt.Move(x+dx*k, y+dy*k)
	}
}

func TestDraggableFollowsThePointer(t *testing.T) {
	var off Point
	var card *node
	var seen Point
	dragging, clicks := false, 0
	tt, now := motionTester(func(c *context) {
		// Read before Draggable changes it: the frame builds again.
		seen = off
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{})
		dragging = card.Dragging()
		if card.Clicked() {
			clicks++
		}
	})
	tt.Move(20, 20)
	tt.Press(20, 20)
	*now = now.Add(10 * time.Millisecond)
	tt.Move(22, 21)
	if off != (Point{}) || dragging {
		t.Fatalf("a move of a few DIPs dragged the card to %v (dragging %v)", off, dragging)
	}
	*now = now.Add(10 * time.Millisecond)
	tt.Move(50, 30)
	if off != (Point{30, 10}) || !dragging || seen != off {
		t.Fatalf("dragged by 30, 10: offset %v, dragging %v, the view saw %v", off, dragging, seen)
	}
	if w := card.st.world; w.X != 30 || w.Y != 10 {
		t.Fatalf("dragged by 30, 10, the card shows at %+v", w)
	}
	*now = now.Add(5 * time.Millisecond)
	tt.Release(50, 30)
	if clicks != 0 || dragging {
		t.Fatalf("let go: %d clicks, dragging %v", clicks, dragging)
	}
	// Without Snap, it rests where it was let go, however fast it went.
	*now = now.Add(time.Second)
	tt.Frame()
	if off != (Point{30, 10}) || card.st.world.X != 30 || card.Transitioning() {
		t.Fatalf("at rest, the offset is %v, the card at %+v", off, card.st.world)
	}
	// A press that does not move is a click.
	tt.ClickAt(50, 30)
	if clicks != 1 || off != (Point{30, 10}) {
		t.Fatalf("a click: %d clicks, offset %v", clicks, off)
	}
}

func TestDraggableThrowsToSnapPoints(t *testing.T) {
	var off Point
	var card *node
	tt, now := motionTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{
			Axis:   Horizontal,
			Bounds: &Rect{W: 200},
			Snap:   Nearest(Point{}, Point{X: 200}),
		})
	})
	// Let go slowly at 60: the nearest is 0.
	throw(tt, now, 20, 20, 60, 0, 6)
	*now = now.Add(200 * time.Millisecond)
	tt.Release(80, 20)
	if off != (Point{}) {
		t.Fatalf("let go slowly at 60, it rests at %v", off)
	}
	if !card.Transitioning() {
		t.Fatal("let go at 60, it does not go back")
	}
	*now = now.Add(2 * time.Second)
	tt.Frame()
	if card.st.world.X != 0 || card.Transitioning() {
		t.Fatalf("back at rest, the card shows at %+v", card.st.world)
	}
	// Thrown at 1000 DIPs per second from 60, it would go past 200.
	throw(tt, now, 20, 20, 60, 0, 6)
	*now = now.Add(5 * time.Millisecond)
	tt.Release(80, 20)
	if off != (Point{X: 200}) {
		t.Fatalf("thrown from 60, it rests at %v", off)
	}
	// The spring starts at the pointer's velocity.
	if v := card.st.dragRec.ch[0].velocity; v < 990 || v > 1010 {
		t.Fatalf("thrown at 1000, it goes at %v", v)
	}
	// Vertical moves do nothing.
	*now = now.Add(2 * time.Second)
	tt.Frame()
	throw(tt, now, 220, 20, 0, 50, 5)
	tt.Release(220, 70)
	if off != (Point{X: 200}) || card.st.world.Y != 0 {
		t.Fatalf("dragged down, it rests at %v", off)
	}
}

func TestDraggableBounds(t *testing.T) {
	var off Point
	rigid := false
	var card *node
	tt, now := motionTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{Bounds: &Rect{W: 100, H: 50}, Rigid: rigid})
	})
	// Past its bounds, it goes less and less far, as long as its parent
	// (the window, 300 by 150) is.
	throw(tt, now, 20, 20, 200, -40, 4)
	want := Point{Rubberband(200, 0, 100, 300), Rubberband(-40, 0, 50, 150)}
	closeFloat(t, off.X, want.X)
	closeFloat(t, off.Y, want.Y)
	*now = now.Add(200 * time.Millisecond)
	tt.Release(220, -20)
	if off != (Point{100, 0}) {
		t.Fatalf("let go past its bounds, it rests at %v", off)
	}
	*now = now.Add(2 * time.Second)
	tt.Frame()
	if w := card.st.world; w.X != 100 || w.Y != 0 {
		t.Fatalf("back within its bounds, it shows at %+v", w)
	}
	// Rigid bounds stop it.
	rigid = true
	throw(tt, now, 120, 20, 100, 100, 4)
	if off != (Point{100, 50}) {
		t.Fatalf("dragged past rigid bounds, it is at %v", off)
	}
	tt.Release(220, 120)
	// An offset the app sets goes within the bounds.
	off = Point{-50, 20}
	tt.Frame()
	if off != (Point{0, 20}) {
		t.Fatalf("set to -50, 20, the offset is %v", off)
	}
}

func TestDraggableCatchesItMoving(t *testing.T) {
	var off Point
	var card *node
	clicks := 0
	tt, now := motionTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{Animation: Animation{Duration: 100 * time.Millisecond, Ease: Linear}})
		if card.Clicked() {
			clicks++
		}
	})
	off = Point{X: 200}
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if w := card.st.world; w.X != 100 || !card.Transitioning() {
		t.Fatalf("halfway, the card shows at %+v", w)
	}
	// A press catches it where it shows, and the pointer takes it from
	// there.
	tt.Move(120, 20)
	tt.Press(120, 20)
	if off != (Point{X: 100}) || card.Transitioning() {
		t.Fatalf("caught, it rests at %v", off)
	}
	*now = now.Add(10 * time.Millisecond)
	tt.Move(130, 20)
	if off != (Point{X: 110}) {
		t.Fatalf("caught and dragged by 10, it is at %v", off)
	}
	tt.Release(130, 20)
	if clicks != 0 {
		t.Fatal("a drag clicked")
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if card.st.world.X != 110 {
		t.Fatalf("let go, it shows at %+v", card.st.world)
	}
}

func TestDraggableGoesToRestInFramesThatOnlyPaint(t *testing.T) {
	var off Point
	var card *node
	tt := newRepaintTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{
			Axis: Horizontal,
			Snap: Nearest(Point{}),
		})
	})
	now := &tt.clock
	throw(tt.Tester, now, 20, 20, 100, 0, 5)
	*now = now.Add(200 * time.Millisecond)
	tt.Release(120, 20)
	if off != (Point{}) {
		t.Fatalf("let go, it rests at %v", off)
	}
	builds := tt.builds
	if built, more := tt.frame(16 * time.Millisecond); built || !more {
		t.Fatalf("going back: built %v, more %v", built, more)
	}
	if x := card.st.world.X; x <= 0 || x >= 100 {
		t.Fatalf("going back, it shows at %v", x)
	}
	// Once at rest, the view builds again, and sees it there.
	for i := 0; i < 200 && tt.builds == builds; i++ {
		tt.frame(16 * time.Millisecond)
	}
	if tt.builds != builds+1 || card.st.world.X != 0 || card.Transitioning() {
		t.Fatalf("at rest: %d builds, at %+v", tt.builds-builds, card.st.world)
	}
	if _, more := tt.frame(16 * time.Millisecond); more {
		t.Fatal("at rest, it asks for frames")
	}
}

func TestDraggableWithoutMotion(t *testing.T) {
	var card *node
	var shown Point
	tt, now := motionTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(nil, DraggableOptions{Snap: Nearest(Point{})})
		shown = card.DragOffset()
	})
	throw(tt, now, 20, 20, 50, 30, 5)
	if shown != (Point{50, 30}) {
		t.Fatalf("dragged, it shows at %v", shown)
	}
	tt.SetPreferences(Preferences{ReduceMotion: true})
	tt.Release(70, 50)
	if shown != (Point{}) || card.st.world.X != 0 || card.Transitioning() {
		t.Fatalf("let go without motion, it shows at %v", shown)
	}
}

func TestDraggableInATransformedParent(t *testing.T) {
	var off Point
	tt, now := motionTester(func(c *context) {
		coreBox(c).Size(100, 60).TransformOrigin(0, 0).Scale(2, 2).Children(func() {
			coreBox(c).Size(20, 20).Draggable(&off, DraggableOptions{})
		})
	})
	// 40 DIPs of the window are 20 of the parent.
	throw(tt, now, 10, 10, 40, 20, 4)
	if off != (Point{20, 10}) {
		t.Fatalf("dragged by 40, 20 in a parent twice as large, it is at %v", off)
	}
}

func TestNearest(t *testing.T) {
	snap := Nearest(Point{}, Point{X: 100}, Point{X: 100, Y: 100})
	for _, c := range []struct{ p, want Point }{
		{Point{X: 40}, Point{}},
		{Point{X: 60}, Point{X: 100}},
		{Point{X: 90, Y: 70}, Point{X: 100, Y: 100}},
	} {
		if got := snap(c.p); got != c.want {
			t.Errorf("nearest to %v: %v, want %v", c.p, got, c.want)
		}
	}
}

// A card lifting as the pointer drags it (AnimatePose) stays under the
// pointer: the offset moves it in its parent's axes, not its own.
func TestDraggableWithAPose(t *testing.T) {
	var off Point
	var card *node
	tt, now := motionTester(func(c *context) {
		card = coreBox(c).Size(40, 40).Draggable(&off, DraggableOptions{})
		lift := Pose{}
		if card.Dragging() {
			lift = Pose{Scale: 2}
		}
		card.AnimatePose(lift, Animation{Duration: 50 * time.Millisecond, Ease: Linear})
	})
	throw(tt, now, 20, 20, 60, 30, 6)
	if off != (Point{60, 30}) {
		t.Fatalf("lifted and dragged by 60, 30, it is at %v", off)
	}
	// Scaled twice around its center (20, 20), moved by 60, 30.
	if w := card.st.world; w.A != 2 || w.X != 40 || w.Y != 10 {
		t.Fatalf("lifted and dragged, it shows as %+v", w)
	}
}

func TestFlingVelocity(t *testing.T) {
	if v := flingVelocity(Point{300, 400}); v != (Point{300, 400}) {
		t.Errorf("500 DIPs per second went at %v", v)
	}
	v := flingVelocity(Point{60000, 80000})
	closeFloat(t, v.X, 4800)
	closeFloat(t, v.Y, 6400)
}
