package ui

import (
	"slices"
	"testing"
	"time"
)

// built keeps the elements the view built last, by key.
var built = map[any]*node{}

// keep keeps e, built with key k, for keyed.
func keep(k any, e *node) *node {
	built[k] = e
	return e
}

// keyed returns the element of the last frame with key k.
func keyed(_ *Tester, k any) *node { return built[k] }

func TestTransitionSpringKeepsGoing(t *testing.T) {
	x := float32(0)
	a := Snappy
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(300, 100).Children(func() {
			keep("box", coreBox(c).Key("box")).Size(20, 20).Absolute().Left(x).Transition(ElementTransition{Animation: &a})
		})
	}, 300, 100)
	x = 200
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	r := tt.rt.trans[keyed(tt, "box").id]
	before, v := keyed(tt, "box").x, r.ch[0].velocity
	if before <= 0 || before >= 200 || v <= 0 {
		t.Fatalf("on its way, the box is at %v going %v/s", before, v)
	}
	// Sent back, it keeps going forward for a moment, then turns.
	x = 0
	tt.Frame()
	*now = now.Add(16 * time.Millisecond)
	tt.Frame()
	if e := keyed(tt, "box"); e.x <= before {
		t.Errorf("sent back, the box turned at once: %v, from %v", e.x, before)
	}
	if !keyed(tt, "box").Transitioning() {
		t.Error("the box moving is not transitioning")
	}
	*now = now.Add(2 * time.Second)
	tt.Frame()
	if e := keyed(tt, "box"); e.x != 0 || len(tt.rt.trans) != 1 || tt.rt.trans[e.id].moving != 0 {
		t.Errorf("once over, the box is at %v", e.x)
	}
	tt.Frame()
	if keyed(tt, "box").Transitioning() {
		t.Error("the box at rest is transitioning")
	}
}

func TestTransitionDelaysChanges(t *testing.T) {
	y := float32(0)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(300, 100).Children(func() {
			for i := range 3 {
				a := Animation{Duration: 100 * time.Millisecond, Ease: Linear, Delay: time.Duration(i) * 50 * time.Millisecond}
				keep(i, coreBox(c).Key(i)).Size(20, 20).Absolute().Left(float32(i) * 30).Top(y).Transition(ElementTransition{Animation: &a})
			}
		})
	}, 300, 100)
	y = 60
	tt.Frame()
	*now = now.Add(75 * time.Millisecond)
	tt.Frame()
	for i, want := range []float32{45, 15, 0} {
		if e := keyed(tt, i); e.y != want {
			t.Errorf("staggered, box %d is at %v, not %v", i, e.y, want)
		}
	}
}

func TestTransitionEntersScaledAndTurned(t *testing.T) {
	show := false
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(200, 200).Children(func() {
			if show {
				keep("pop", coreBox(c).Key("pop")).Size(40, 40).Transition(ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear,
					Enter: &Motion{Scale: 0.5, Rotate: 90, Opacity: 1},
					Exit:  &Motion{Scale: 2}})
			}
		})
	}, 200, 200)
	show = true
	tt.Frame()
	e := keyed(tt, "pop")
	if w := e.st.world; !w.Set || w.A > 0.01 || w.B < 0.49 || w.B > 0.51 {
		t.Fatalf("entering, the box shows as %+v", w)
	}
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	b := keyed(tt, "pop").Bounds()
	if b.W <= 30 || b.W >= 60 {
		t.Errorf("halfway in, the box is %v wide", b.W)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if keyed(tt, "pop").st.world.Set {
		t.Error("in, the box is still transformed")
	}
	show = false
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	var ghost *node
	for g := tt.rt.c.root.first.first; g != nil; g = g.next {
		if g.leaving != 0 {
			ghost = g
		}
	}
	if ghost == nil || !ghost.world.Set || ghost.world.A < 1.4 || ghost.world.A > 1.6 {
		t.Fatalf("halfway out, the copy is %+v", ghost)
	}
}

func TestTransitionMatchesElements(t *testing.T) {
	detail := false
	type photo struct{ id int }
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(400, 300).Children(func() {
			tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear, Match: photo{7}}
			if !detail {
				coreBox(c).Key("grid").Absolute().Left(20).Top(30).Size(100, 100).Children(func() {
					keep("thumb", coreBox(c).Key("thumb")).Size(40, 20).Background(RGB(255, 0, 0)).Transition(tr)
				})
			} else {
				coreBox(c).Key("detail").Absolute().Left(100).Top(100).Size(200, 100).Children(func() {
					keep("hero", coreBox(c).Key("hero")).Size(200, 100).Background(RGB(0, 0, 255)).Transition(tr)
				})
			}
		})
	}, 400, 300)
	detail = true
	tt.Frame()
	if b := keyed(tt, "hero").Bounds(); !nearRect(b, Rect{20, 30, 40, 20}) {
		t.Fatalf("appearing, the hero shows at %v, not where the thumbnail was", b)
	}
	if p := tt.Image().RGBAAt(30, 35); p.R < 200 || p.B > 50 {
		t.Errorf("appearing, the hero is %v, not the thumbnail's red", p)
	}
	if c := keyed(tt, "hero").bg; c != RGB(255, 0, 0) {
		t.Errorf("appearing, the hero's background is %v, not the thumbnail's red", c)
	}
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if b := keyed(tt, "hero").Bounds(); !nearRect(b, Rect{60, 65, 120, 60}) {
		t.Errorf("halfway, the hero shows at %v", b)
	}
	if c := keyed(tt, "hero").bg; c.R < 100 || c.B < 100 {
		t.Errorf("halfway, the hero's background is %v, not between red and blue", c)
	}
	if keyed(tt, "hero").st.world.Set {
		t.Error("halfway, the hero is scaled rather than laid out at its size")
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if b := keyed(tt, "hero").Bounds(); !nearRect(b, Rect{100, 100, 200, 100}) {
		t.Errorf("once over, the hero shows at %v", b)
	}
	if p := tt.Image().RGBAAt(150, 150); p.B < 250 {
		t.Errorf("once over, the hero is %v", p)
	}
	// And back.
	detail = false
	tt.Frame()
	if b := keyed(tt, "thumb").Bounds(); !nearRect(b, Rect{100, 100, 200, 100}) {
		t.Errorf("back, the thumbnail shows at %v", b)
	}
	// Without motion, it shows where it is.
	*now = now.Add(time.Second)
	tt.Frame()
	tt.SetPreferences(Preferences{ReduceMotion: true})
	detail = true
	tt.Frame()
	if b := keyed(tt, "hero").Bounds(); !nearRect(b, Rect{100, 100, 200, 100}) {
		t.Errorf("without motion, the hero shows at %v", b)
	}
}

func TestTransitionWithoutMotionFades(t *testing.T) {
	show := false
	bg := RGB(0, 0, 255)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(200, 200).Children(func() {
			keep("color", coreBox(c).Key("color")).Size(50, 50).Background(bg).Transition(ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear})
			if show {
				keep("fade", coreBox(c).Key("fade")).Size(20, 20).Transition(ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear,
					Enter: &Motion{Y: 30}, Exit: &Motion{Y: 30}})
				keep("fold", coreBox(c).Key("fold")).Size(20, 20).Transition(ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear,
					Enter: &Motion{Collapse: true}})
			}
		})
	}, 200, 200)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	show = true
	bg = RGB(255, 0, 0)
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if e := keyed(tt, "fade"); e.y != 50 || !e.opacitySet || e.opacity < 0.4 || e.opacity > 0.6 {
		t.Errorf("without motion, the box entering is at %v, %v opaque", e.y, e.opacity)
	}
	if e := keyed(tt, "fold"); e.h != 20 || e.opacitySet {
		t.Errorf("without motion, the box collapsing is %v high, %v opaque", e.h, e.opacity)
	}
	if p := tt.Image().RGBAAt(25, 25); p.R < 100 || p.B < 100 {
		t.Errorf("without motion, the colors jumped to %v", p)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	show = false
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	leaving := 0
	for g := tt.rt.c.root.first.first; g != nil; g = g.next {
		if g.leaving != 0 {
			leaving++
			if g.y != 50 || g.opacity < 0.4 || g.opacity > 0.6 {
				t.Errorf("without motion, the copy leaving is at %v, %v opaque", g.y, g.opacity)
			}
		}
	}
	if leaving != 1 {
		t.Errorf("%d copies leaving", leaving)
	}
}

// An element coming in moves with the layout at once where its Transition
// does not animate: only the Enter's own motion plays.
func TestTransitionEnterFollowsTheLayout(t *testing.T) {
	shown, above := false, float32(20)
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Size(300, 200).Children(func() {
			coreBox(c).Size(20, above)
			if shown {
				keep("panel", coreBox(c).Key("panel")).Size(40, 20).Transition(ElementTransition{
					Duration: 100 * time.Millisecond, Colors: true, Enter: &Motion{X: 8},
				})
			}
		})
	}, 300, 200)
	shown = true
	tt.Frame()
	*now = now.Add(30 * time.Millisecond)
	tt.Frame()
	// The box above grows on the way in: the panel goes down at once.
	above = 50
	tt.Frame()
	e := keyed(tt, "panel")
	if e.y != 50 {
		t.Errorf("on its way in, the panel is at y=%v, not at once under the box at 50", e.y)
	}
	if e.x <= 0 || e.x >= 8 {
		t.Errorf("on its way in, the panel is at x=%v, not sliding in", e.x)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if e := keyed(tt, "panel"); e.x != 0 || e.y != 50 {
		t.Errorf("in, the panel is at (%v, %v)", e.x, e.y)
	}
}

// An element built again as its copy goes comes back from where the copy
// was, even where its Transition does not animate, and not past its place
// when the layout moved it meanwhile.
func TestTransitionBackFollowsNoLayoutTwice(t *testing.T) {
	shown, above := true, float32(20)
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Size(300, 200).Children(func() {
			coreBox(c).Size(20, above)
			if shown {
				keep("panel", coreBox(c).Key("panel")).Size(40, 20).Transition(ElementTransition{
					Duration: 100 * time.Millisecond, Ease: Linear, Colors: true, Exit: &Motion{X: 8},
				})
			}
		})
	}, 300, 200)
	shown = false
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	// Back, under a box that grew meanwhile.
	shown, above = true, 50
	tt.Frame()
	if e := keyed(tt, "panel"); e.y != 20 {
		t.Errorf("back, the panel starts at y=%v, not where its copy was at 20", e.y)
	}
	*now = now.Add(25 * time.Millisecond)
	tt.Frame()
	if e := keyed(tt, "panel"); e.y <= 20 || e.y >= 50 {
		t.Errorf("on its way back, the panel is at y=%v, not between 20 and 50", e.y)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if e := keyed(tt, "panel"); e.x != 0 || e.y != 50 {
		t.Errorf("back, the panel is at (%v, %v)", e.x, e.y)
	}
}

// An element collapsed coming in keeps opening when the layout makes it
// smaller on the way, where its Transition does not animate its size.
func TestTransitionEnterCollapsedFollowsTheLayout(t *testing.T) {
	shown, h := false, float32(40)
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Size(300, 200).Children(func() {
			if shown {
				keep("row", coreBox(c).Key("row")).Size(40, h).Transition(ElementTransition{
					Duration: 100 * time.Millisecond, Ease: Linear, Colors: true, Enter: &Motion{Opacity: 1, Collapse: true},
				})
			}
		})
	}, 300, 200)
	shown = true
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if e := keyed(tt, "row"); e.h != 20 {
		t.Fatalf("halfway in, the row is %v high", e.h)
	}
	h = 10
	for _, want := range [][2]float32{{0, 5}, {25, 7.5}} {
		*now = now.Add(time.Duration(want[0]) * time.Millisecond)
		tt.Frame()
		if e := keyed(tt, "row"); e.h != want[1] {
			t.Errorf("%v ms in, the row is %v high, not %v", 50+want[0], e.h, want[1])
		}
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if e := keyed(tt, "row"); e.h != 10 {
		t.Errorf("in, the row is %v high", e.h)
	}
}

// Siblings leaving in one frame go in the order they had: their copies,
// which overlap on the way, paint the same from run to run.
func TestTransitionExitsKeepSiblingOrder(t *testing.T) {
	colors := []Color{RGB(255, 0, 0), RGB(0, 255, 0), RGB(0, 0, 255), RGB(255, 255, 0), RGB(0, 255, 255), RGB(255, 0, 255)}
	run := func() ([]Color, []byte) {
		shown := true
		tt, now := clockTester(func(c *context) {
			coreRow(c).Size(200, 40).Children(func() {
				if shown {
					for i, bg := range colors {
						coreBox(c).Key(i).Size(20, 20).Background(bg).Transition(ElementTransition{
							Duration: 100 * time.Millisecond, Ease: Linear, Exit: &Motion{X: 15},
						})
					}
				}
			})
		}, 200, 40)
		shown = false
		tt.Frame()
		*now = now.Add(50 * time.Millisecond)
		tt.Frame()
		var order []Color
		for g := tt.rt.c.root.first.first; g != nil; g = g.next {
			if g.leaving != 0 {
				order = append(order, g.bg)
			}
		}
		return order, slices.Clone(tt.Image().Pix)
	}
	order, pix := run()
	if !slices.Equal(order, colors) {
		t.Fatalf("the copies leaving go in the order %v, not %v", order, colors)
	}
	for range 20 {
		o, p := run()
		if !slices.Equal(o, order) || !slices.Equal(p, pix) {
			t.Fatalf("another run paints the copies leaving in the order %v, not %v", o, order)
		}
	}
}

// leavingUnder returns the copies leaving under p.
func leavingUnder(p *node) []*node {
	var gs []*node
	for g := p.first; g != nil; g = g.next {
		if g.leaving != 0 {
			gs = append(gs, g)
		}
	}
	return gs
}

func TestTransitionEntersAndExitsBlurred(t *testing.T) {
	show, own := false, float32(0)
	tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear,
		Enter: &Motion{Opacity: 1, Blur: 8},
		Exit:  &Motion{Opacity: 1, Blur: 4}}
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(200, 200).Children(func() {
			if show {
				keep("blur", coreBox(c).Key("blur")).Absolute().Left(80).Top(80).Size(40, 40).
					Background(RGB(255, 0, 0)).Blur(own).Transition(tr)
			}
		})
	}, 200, 200)
	// Just outside the box, what a blur spreads.
	outside := func() uint8 { return 255 - tt.Image().RGBAAt(124, 100).G }
	show = true
	tt.Frame()
	if b := keyed(tt, "blur").blur; b != 8 {
		t.Fatalf("entering, the box is blurred by %v", b)
	}
	if outside() == 0 {
		t.Error("entering, the blur spreads nothing beyond the box")
	}
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if b := keyed(tt, "blur").blur; !within(b, 4, 0.01) {
		t.Errorf("halfway in, the box is blurred by %v", b)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if b := keyed(tt, "blur").blur; b != 0 {
		t.Errorf("in, the box is blurred by %v", b)
	}
	if v := outside(); v != 0 {
		t.Errorf("in, the box spreads %v beyond it", v)
	}

	// Its copy blurs too, from its own blur.
	own = 3
	tt.Frame()
	show = false
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	gs := leavingUnder(tt.rt.c.root.first)
	if len(gs) != 1 {
		t.Fatalf("%d copies leaving", len(gs))
	}
	// Its own 3 and the exit's 2, as Gaussians compose.
	if b := gs[0].blur; !within(b, 3.606, 0.01) {
		t.Errorf("halfway out, the copy is blurred by %v", b)
	}
	if outside() == 0 {
		t.Error("halfway out, the copy's blur spreads nothing beyond it")
	}
}

func TestTransitionBlurWithoutMotion(t *testing.T) {
	show := false
	tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear, Enter: &Motion{Opacity: 1, Blur: 6, Y: 20}}
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(200, 200).Children(func() {
			if show {
				keep("blur", coreBox(c).Key("blur")).Size(40, 40).Transition(tr)
			}
		})
	}, 200, 200)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	show = true
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	// It blurs in place, as it would fade: a blur moves nothing.
	e := keyed(tt, "blur")
	if !within(e.blur, 3, 0.01) || e.y != 0 {
		t.Errorf("without motion, halfway in, the box is at %v blurred by %v", e.y, e.blur)
	}
}

func TestTransitionChildExitDecidesAsItGoes(t *testing.T) {
	page, dir := 0, float32(1)
	parentExit := true
	tt, now := clockTester(func(c *context) {
		slot := coreBox(c).Size(200, 100)
		if parentExit {
			slot.ChildExit(Motion{X: -40 * dir})
		}
		slot.Children(func() {
			coreBox(c).Key(page).Absolute().Size(100, 100).Transition(ElementTransition{
				Duration: 100 * time.Millisecond, Ease: Linear,
				Enter: &Motion{Opacity: 1, X: 40 * dir},
				Exit:  &Motion{Opacity: 1, X: -40 * dir},
			})
		})
	}, 200, 100)
	slot := func() *node { return tt.rt.c.root.first }
	halfway := func() float32 {
		t.Helper()
		*now = now.Add(50 * time.Millisecond)
		tt.Frame()
		gs := leavingUnder(slot())
		if len(gs) != 1 {
			t.Fatalf("%d copies leaving", len(gs))
		}
		return gs[0].x - slot().x
	}
	// Forward, then back: the page that goes as the step turns around,
	// built going forward, goes as its parent says now, back.
	page = 1
	tt.Frame()
	*now = now.Add(time.Second)
	tt.Frame()
	page, dir = 0, -1
	tt.Frame()
	if x := halfway(); !within(x, 20, 0.01) {
		t.Errorf("going back, halfway, the page gone is at %v, not 20", x)
	}
	// Without its parent's, it goes as it was built to.
	*now = now.Add(time.Second)
	tt.Frame()
	parentExit = false
	page, dir = 1, 1
	tt.Frame()
	if x := halfway(); !within(x, 20, 0.01) {
		t.Errorf("without ChildExit, halfway, the page gone is at %v, not 20", x)
	}
}

func TestTransitionAfterExits(t *testing.T) {
	page := 0
	tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear, Enter: &Motion{}, Exit: &Motion{}, AfterExits: true}
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(200, 100).Children(func() {
			keep(page, coreBox(c).Key(page)).Absolute().Size(100, 100).Background(RGB(0, 0, 255)).Transition(tr)
		})
	}, 200, 100)
	slot := func() *node { return tt.rt.c.root.first }
	page = 1
	tt.Frame()
	if len(leavingUnder(slot())) != 1 {
		t.Fatal("the page gone does not leave")
	}
	if e := keyed(tt, 1); e.flags&flagInvisible == 0 {
		t.Error("the page coming shows while the other leaves")
	}
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if e := keyed(tt, 1); e.flags&flagInvisible == 0 || len(leavingUnder(slot())) != 1 {
		t.Error("halfway, the page coming shows, or the other no longer leaves")
	}
	// The page gone gone, the other comes in.
	*now = now.Add(60 * time.Millisecond)
	tt.Frame()
	if len(leavingUnder(slot())) != 0 {
		t.Fatal("the page gone is still leaving")
	}
	if e := keyed(tt, 1); e.flags&flagInvisible != 0 || !e.opacitySet || e.opacity != 0 {
		t.Errorf("the page coming does not start coming in: %v", e.opacity)
	}
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if e := keyed(tt, 1); !within(e.opacity, 0.5, 0.01) {
		t.Errorf("halfway in, the page coming is %v opaque", e.opacity)
	}
	// Gone before it showed, it leaves nothing.
	*now = now.Add(time.Second)
	tt.Frame()
	shown := keyed(tt, 1).id
	page = 2
	tt.Frame()
	page = 3
	tt.Frame()
	if gs := leavingUnder(slot()); len(gs) != 1 || gs[0].id != shown {
		t.Errorf("%d copies leave for the page shown and one that never showed", len(gs))
	}
	if e := keyed(tt, 3); e.flags&flagInvisible == 0 {
		t.Error("the page coming shows while the other leaves")
	}
}

// TestTransitionAfterExitsFocus checks that an element held back by
// AfterExits, given the focus as it replaces another (by a Handle or
// AutoFocus), takes it once it shows, unless the focus went elsewhere
// meanwhile.
func TestTransitionAfterExitsFocus(t *testing.T) {
	for _, how := range []string{"handle", "autofocus", "elsewhere"} {
		t.Run(how, func(t *testing.T) {
			confirm := false
			var h Handle
			tr := ElementTransition{Duration: 100 * time.Millisecond, Ease: Linear, Enter: &Motion{}, Exit: &Motion{}}
			tt := NewTester(func(c *Context) {
				Column(c).Children(func() {
					ButtonBase(c).Size(80, 20).Label("other")
					Box(c).Size(200, 40).Children(func() {
						if !confirm {
							ButtonBase(c).Key("actions").Size(80, 20).Label("delete").Transition(tr)
							return
						}
						held := tr
						held.AfterExits = true
						b := ButtonBase(c).Key("confirm").Size(80, 20).Label("confirm").Transition(held).Bind(&h)
						if how == "autofocus" {
							b.AutoFocus()
						}
					})
				})
			}, 200, 100)
			if err := tt.Click("delete"); err != nil {
				t.Fatal(err)
			}
			confirm = true
			if how != "autofocus" {
				h.Focus()
			}
			tt.Frame()
			if tt.Focused("confirm") {
				t.Error("the element held back has the focus while unseen")
			}
			if how == "elsewhere" {
				if err := tt.Click("other"); err != nil {
					t.Fatal(err)
				}
			}
			tt.Advance(50 * time.Millisecond)
			if tt.Focused("confirm") {
				t.Error("halfway through the exit, the element held back has the focus")
			}
			tt.Advance(time.Second)
			if want := how != "elsewhere"; tt.Focused("confirm") != want {
				t.Errorf("once it shows, the element held back has the focus: %v, want %v", !want, want)
			}
			if how == "elsewhere" && !tt.Focused("other") {
				t.Error("the focus moved elsewhere meanwhile was taken back")
			}
		})
	}
}

func TestTransitionMotionPercent(t *testing.T) {
	show := false
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(200, 200).Children(func() {
			if show {
				keep("p", coreBox(c).Key("p")).Absolute().Left(100).Top(100).Size(40, 20).Transition(ElementTransition{
					Duration: 100 * time.Millisecond, Ease: Linear,
					Enter: &Motion{Opacity: 1, XPercent: -100},
					Exit:  &Motion{Opacity: 1, X: 10, YPercent: 50},
				})
			}
		})
	}, 200, 200)
	show = true
	tt.Frame()
	if x := keyed(tt, "p").x; x != 60 {
		t.Errorf("entering, it is at %v, its width to the left", x)
	}
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if x := keyed(tt, "p").x; !within(x, 80, 0.01) {
		t.Errorf("halfway in, it is at %v", x)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	show = false
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	gs := leavingUnder(tt.rt.c.root.first)
	if len(gs) != 1 || !within(gs[0].x, 105, 0.01) || !within(gs[0].y, 105, 0.01) {
		t.Fatalf("halfway out, the copies are %v", gs)
	}
}
