package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// repaintTester runs view as a window does, with the frames the surface
// gives (engine.surfaceFrame), at a clock the test sets.
type repaintTester struct {
	*Tester
	clock  time.Time
	builds int
}

func newRepaintTester(view func(c *context)) *repaintTester {
	rt := &repaintTester{clock: time.Unix(1000, 0)}
	rt.Tester = coreNewTester(func(c *context) {
		rt.builds++
		view(c)
	}, 200, 100)
	rt.rt.clock = func() time.Time { return rt.clock }
	rt.Frame()
	return rt
}

// frame draws the frame the surface asks for after d, and reports whether
// the view built it and whether it asked for another.
func (t *repaintTester) frame(d time.Duration) (built, more bool) {
	t.clock = t.clock.Add(d)
	n := t.builds
	t.h.requested.Store(false)
	t.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceFrame})
	return t.builds > n, t.h.requested.Load()
}

func TestRepaintFrames(t *testing.T) {
	blue := RGB(0, 0, 255)
	draws := 0
	var drawn time.Time
	tt := newRepaintTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) {
			// A bar moving a DIP a millisecond.
			draws++
			drawn = p.Now()
			p.AnimationFrame()
			p.Fill(Rect{r.X + float32(p.Now().UnixMilli()%50), r.Y, 2, r.H}, blue, 0)
		})
	})
	at := func(x int) bool {
		r, g, b, _ := tt.Image().At(x, 10).RGBA()
		return r == 0 && g == 0 && b == 0xffff
	}
	// Frames paint the bar again, where the time puts it, without building
	// the view.
	for i, x := range []int{8, 16} {
		n := draws
		built, more := tt.frame(8 * time.Millisecond)
		if built || !more || draws != n+1 || !drawn.Equal(tt.clock) || !at(x) || at(x-8) {
			t.Fatalf("frame %d: built %v, asked for another %v, painted %d times at %v, bar at %d: %v, at %d: %v",
				i, built, more, draws-n, drawn.Sub(tt.clock), x, at(x), x-8, at(x-8))
		}
	}
	// Anything else that may change what the view builds builds it anew.
	for _, change := range []struct {
		what string
		do   func()
	}{
		{"an event", func() { tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 150, Y: 50}) }},
		{"a frame asked for", func() { tt.rt.requestFrame() }},
		{"a change of the app's state", func() { tt.rt.changed() }},
		{"a change of the appearance", func() { tt.rt.themeChanged() }},
		{"the text system forgetting its layouts", func() { tt.rt.gen-- }},
		{"another size", func() { tt.h.w = 210 }},
	} {
		change.do()
		if built, _ := tt.frame(8 * time.Millisecond); !built {
			t.Errorf("after %s, the frame did not build the view", change.what)
		}
		if built, _ := tt.frame(8 * time.Millisecond); built {
			t.Errorf("after %s, the next frame built the view", change.what)
		}
	}
}

func TestRepaintOnlyInView(t *testing.T) {
	draws := 0
	tt := newRepaintTester(func(c *context) {
		coreScroll(c).Height(100).Children(func() {
			coreBox(c).Height(300).Shrink(0)
			coreBox(c).Size(100, 20).Shrink(0).Draw(func(p *Painter, r Rect) {
				draws++
				p.AnimationFrame()
			})
		})
	})
	if draws != 0 || tt.rt.redraw || tt.h.requested.Load() {
		t.Errorf("out of view: painted %d times, asks to be painted again %v, asks for a frame %v", draws, tt.rt.redraw, tt.h.requested.Load())
	}
}

func TestPainterAfter(t *testing.T) {
	draws := 0
	tt := newRepaintTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) {
			draws++
			p.After(30 * time.Millisecond)
		})
	})
	due := tt.clock.Add(30 * time.Millisecond)
	if !tt.rt.redraw || !tt.rt.repaintDue.Equal(due) || tt.h.requested.Load() {
		t.Fatalf("asks to be painted again %v at %v, for a frame now %v", tt.rt.redraw, tt.rt.repaintDue.Sub(tt.clock), tt.h.requested.Load())
	}
	// The frame comes when it is due.
	tt.clock = due.Add(-10 * time.Millisecond)
	tt.rt.repaintNow()
	if tt.h.requested.Load() {
		t.Fatal("a frame asked for before it is due")
	}
	tt.clock = due
	tt.rt.repaintNow()
	if !tt.h.requested.Load() {
		t.Fatal("no frame asked for once due")
	}
	n := draws
	if built, more := tt.frame(time.Millisecond); built || more || draws != n+1 || !tt.rt.repaintDue.Equal(tt.clock.Add(30*time.Millisecond)) {
		t.Errorf("the frame: built %v, asked for another now %v, painted %d times, next due in %v", built, more, draws-n, tt.rt.repaintDue.Sub(tt.clock))
	}
	// A frame built since replaces the one due.
	due = tt.rt.repaintDue
	tt.rt.changed()
	tt.clock = due
	tt.h.requested.Store(false)
	tt.rt.repaintNow()
	if tt.h.requested.Load() {
		t.Error("a frame asked for though one building the view comes")
	}
	if built, _ := tt.frame(time.Millisecond); !built {
		t.Fatal("the frame asked for did not build the view")
	}
	// An event that asks for no frame, as the pointer moving over an
	// element that does not look at it, leaves the frame due: it comes,
	// and builds the view anew (issue 149).
	due = tt.rt.repaintDue
	tt.h.requested.Store(false)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 50, Y: 10})
	if tt.h.requested.Load() {
		t.Fatal("the pointer moving asked for a frame")
	}
	tt.clock = due
	tt.rt.repaintNow()
	if !tt.h.requested.Load() {
		t.Fatal("no frame asked for once due, after the pointer moved")
	}
	n = draws
	if built, _ := tt.frame(time.Millisecond); !built || draws != n+1 || !tt.rt.repaintDue.Equal(tt.clock.Add(30*time.Millisecond)) {
		t.Errorf("the frame after the pointer moved: built %v, painted %d times, next due in %v", built, draws-n, tt.rt.repaintDue.Sub(tt.clock))
	}
}

func TestIndicatorsRepaint(t *testing.T) {
	var value float64
	tt := newRepaintTester(func(c *context) {
		coreColumn(c).Gap(10).Padding(10).Children(func() {
			coreSpinner(c)
			coreProgress(c, value).Width(100)
		})
	})
	// The spinner is painted again for its next spoke, the progress bar of
	// unknown length as soon as the display can show it.
	if !tt.rt.redraw || tt.rt.animating || tt.rt.repaintDue.IsZero() || tt.rt.repaintDue.Sub(tt.clock) > 75*time.Millisecond {
		t.Errorf("the spinner alone: asks to be painted again %v (at %v), keeps building frames %v", tt.rt.redraw, tt.rt.repaintDue.Sub(tt.clock), tt.rt.animating)
	}
	value = -1
	tt.Frame()
	if !tt.rt.redraw || tt.rt.animating || !tt.h.requested.Load() {
		t.Errorf("with a moving progress bar: asks to be painted again %v, keeps building frames %v, asks for a frame %v", tt.rt.redraw, tt.rt.animating, tt.h.requested.Load())
	}
	before := tt.Image()
	if built, _ := tt.frame(400 * time.Millisecond); built || bytes.Equal(before.Pix, tt.Image().Pix) {
		t.Errorf("400 ms later, the frame built the view %v, or shows the same", built)
	}
}

// TestHeldWhileOccluded checks that what moves draws no frames while
// nothing of the window shows, nor do changes of the state, and that both
// go on once some of it shows again: as browsers draw nothing for windows
// out of sight, and as Apple asks of Mac apps.
func TestHeldWhileOccluded(t *testing.T) {
	count := 0
	tt := newRepaintTester(func(c *context) {
		coreTextf(c, "%d", count)
		coreProgress(c, -1).Width(100)
		coreBox(c).Size(10, 10).Draw(func(p *Painter, r Rect) { p.After(50 * time.Millisecond) })
		spin := coreBox(c).Size(10, 10)
		spin.Rotate(spin.Loop("spin", time.Second, Linear) * 360)
	})
	// The frame the view animating asked for comes, and asks for no more.
	tt.h.hidden = true
	if _, more := tt.frame(8 * time.Millisecond); more || !tt.rt.held || !tt.rt.repaintDue.IsZero() {
		t.Fatalf("out of sight: asked for another %v, held %v, repaint due %v", more, tt.rt.held, tt.rt.repaintDue)
	}
	// The app's state changes: the frame waits for the window to show,
	// and asks for no more.
	count++
	tt.rt.changed()
	if built, more := tt.frame(8 * time.Millisecond); built || more || !tt.rt.held {
		t.Fatalf("a change out of sight: built %v, asked for another %v, held %v", built, more, tt.rt.held)
	}
	// Some of the window shows again: the frame builds what changed, and
	// what moves goes on.
	tt.h.hidden = false
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	if tt.rt.held || !tt.h.requested.Load() {
		t.Fatalf("shown again: held %v, asked for a frame %v", tt.rt.held, tt.h.requested.Load())
	}
	if built, more := tt.frame(8 * time.Millisecond); !built || !more || !tt.HasText("1") {
		t.Errorf("the frame after showing: built %v, asked for another %v, shows %q", built, more, tt.Texts())
	}
	// Shown while nothing waited, it asks for nothing.
	tt.h.requested.Store(false)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	if tt.h.requested.Load() {
		t.Error("shown again while nothing waited, it asked for a frame")
	}
}

// TestFocusChangeFrames checks that the window gaining or losing the
// keyboard focus draws a frame only when the last frame read it, as a
// focused text input does for its caret.
// TestHeldFrameKeepsNoDisplayTime checks that a frame the display asked
// for, drawn late as the GPU renderer could take it (windowHost.
// waitDisplay), which then waited for the window to show, is built at the
// time it draws rather than at the time the display asked for it.
func TestHeldFrameKeepsNoDisplayTime(t *testing.T) {
	var built time.Time
	tt := newRepaintTester(func(c *context) {
		built = c.now
		coreText(c, "hello")
	})
	tt.h.hidden = true
	asked := tt.clock
	tt.rt.frameAt = asked
	tt.rt.changed()
	if b, _ := tt.frame(8 * time.Millisecond); b || !tt.rt.held {
		t.Fatalf("out of sight: built %v, held %v", b, tt.rt.held)
	}
	tt.clock = tt.clock.Add(10 * time.Minute)
	tt.h.hidden = false
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	if b, _ := tt.frame(8 * time.Millisecond); !b {
		t.Fatal("shown again, the frame did not build")
	}
	if !built.Equal(tt.clock) {
		t.Errorf("the frame after showing was built at %v, %v after the time it drew at", built, tt.clock.Sub(built))
	}
}

func TestFocusChangeFrames(t *testing.T) {
	text, input := "", false
	tt := newRepaintTester(func(c *context) {
		coreText(c, "Hello")
		if input {
			coreTextInputBase(c, &text).Width(100).Label("name").AutoFocus()
		}
	})
	for _, focused := range []bool{false, true} {
		tt.h.requested.Store(false)
		tt.rt.event(platform.SurfaceEvent{Kind: map[bool]platform.SurfaceEventKind{false: platform.SurfaceBlur, true: platform.SurfaceFocus}[focused]})
		if tt.h.requested.Load() {
			t.Errorf("focused %v, nothing reading it: asked for a frame", focused)
		}
	}
	input = true
	tt.rt.changed()
	tt.frame(time.Millisecond)
	tt.frame(time.Millisecond)
	for _, focused := range []bool{false, true} {
		tt.h.requested.Store(false)
		tt.rt.event(platform.SurfaceEvent{Kind: map[bool]platform.SurfaceEventKind{false: platform.SurfaceBlur, true: platform.SurfaceFocus}[focused]})
		if !tt.h.requested.Load() {
			t.Errorf("focused %v, with a focused text input: asked for no frame", focused)
		}
		tt.frame(time.Millisecond)
	}
}

// TestOccludedFramesAfterInput checks that a window out of sight draws the
// frame after input, as keys typed into a key window another app's
// covers, and holds the next one again.
func TestOccludedFramesAfterInput(t *testing.T) {
	count := 0
	rt := newRepaintTester(func(c *context) { coreTextf(c, "%d", count) })
	rt.h.hidden = true
	count = 1
	rt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyA)})
	if built, _ := rt.frame(8 * time.Millisecond); !built || !rt.HasText("1") {
		t.Errorf("a key out of sight: built %v, shows %q", built, rt.Texts())
	}
	// The input was the last frame's: a change since waits.
	count = 2
	rt.rt.changed()
	if built, _ := rt.frame(8 * time.Millisecond); built || !rt.rt.held {
		t.Errorf("a change out of sight, after the input's frame: built %v, held %v", built, rt.rt.held)
	}
}

func TestAnimationFrameRate(t *testing.T) {
	var fps float32
	full := false
	draws := 0
	tt := newRepaintTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) {
			draws++
			if fps > 0 {
				p.AnimationFrameRate(fps)
			}
		})
		coreBox(c).Size(10, 10).Draw(func(p *Painter, r Rect) {
			if full {
				p.AnimationFrame()
			}
		})
	})
	// paints returns how many of n refreshes of a display of hz paint the
	// drawing again, each asking for the next.
	paints := func(hz float32, n int) int {
		tt.h.hz = hz
		// Built anew, at the display's rate.
		tt.rt.changed()
		tt.frame(0)
		start := draws
		for i := range n {
			if _, more := tt.frame(time.Duration(float32(time.Second) / hz)); !more {
				t.Fatalf("at %v Hz asking for %v, refresh %d asked for no other", hz, fps, i)
			}
		}
		return draws - start
	}
	for _, c := range []struct {
		hz, fps float32
		every   int
		told    float32
	}{
		{120, 60, 2, 60},
		{120, 50, 2, 60}, // what the display shows evenly, at least as many
		{120, 30, 4, 30},
		{120, 100, 1, 0},
		{120, 120, 1, 0},
		{60, 60, 1, 0},
		{60, 30, 2, 30},
		{59.94, 30, 2, 29.97},
	} {
		fps = c.fps
		if got, want := paints(c.hz, 24), 24/c.every; got != want {
			t.Errorf("at %v Hz asking for %v: painted %d of 24 refreshes, want %d", c.hz, c.fps, got, want)
		}
		if tt.h.frameRate != c.told {
			t.Errorf("at %v Hz asking for %v: told the surface %v frames a second, want %v", c.hz, c.fps, tt.h.frameRate, c.told)
		}
	}
	// Another drawing asking for the display's rate has each refresh paint.
	fps, full = 30, true
	if got := paints(120, 24); got != 24 || tt.h.frameRate != 0 {
		t.Errorf("with AnimationFrame too: painted %d of 24 refreshes, told %v, want 24 and 0", got, tt.h.frameRate)
	}
	// An event between the frames due builds at once.
	full = false
	paints(120, 4)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 150, Y: 50})
	if built, _ := tt.frame(time.Second / 120); !built {
		t.Error("the frame after an event did not build the view")
	}
	// Once nothing moves, the surface is told of the display's rate again.
	fps = 0
	tt.rt.changed()
	tt.frame(time.Second / 120)
	if tt.h.frameRate != 0 {
		t.Errorf("with nothing moving, told the surface %v frames a second, want 0", tt.h.frameRate)
	}
}

// TestAnimationFrameRateAtOnce checks the frames of AnimationFrameRate on a
// surface whose frames asked for come at once, not at the display's next
// refresh, as Windows': the refreshes between wait on the repaint timer,
// rather than ask for frames that would come again and again.
func TestAnimationFrameRateAtOnce(t *testing.T) {
	draws := 0
	tt := newRepaintTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) {
			draws++
			p.AnimationFrameRate(30)
		})
	})
	tt.h.hz, tt.h.atOnce = 60, true
	tt.rt.changed()
	tt.frame(0)
	start, began := draws, tt.clock
	// The frame asked for comes at once: it draws nothing and asks for no
	// other, but for one when the next is due.
	if _, more := tt.frame(time.Millisecond); more || draws != start {
		t.Fatalf("a frame coming at once: painted %d, asked for another %v, want 0 and false", draws-start, more)
	}
	if due := began.Add(time.Second / 30); !tt.rt.repaintDue.Equal(due) || !tt.h.repaintAt.Equal(due) {
		t.Fatalf("a frame coming at once: repaint due %v, timer at %v, want both at %v", tt.rt.repaintDue.Sub(began), tt.h.repaintAt.Sub(began), due.Sub(began))
	}
	// The timer asks for it at its time, and it paints.
	tt.clock = began.Add(time.Second / 30)
	tt.h.requested.Store(false)
	tt.rt.repaintNow()
	if !tt.h.requested.Load() {
		t.Fatal("the repaint timer did not ask for the frame due")
	}
	if _, more := tt.frame(0); !more || draws != start+1 {
		t.Errorf("the frame due: painted %d, asked for another %v, want 1 and true", draws-start, more)
	}
	// A timer firing late does not make the frames after it late too: the
	// next is due a frame after the last was, not after the timer fired.
	tt.frame(time.Millisecond)
	late := began.Add(2*time.Second/30 + 3*time.Millisecond)
	tt.clock = late
	tt.rt.repaintNow()
	tt.frame(0)
	tt.frame(time.Millisecond)
	if due := began.Add(3 * (time.Second / 30)); !tt.rt.repaintDue.Equal(due) {
		t.Errorf("after a frame 3ms late, the next is due %v, want %v", tt.rt.repaintDue.Sub(began), due.Sub(began))
	}
}

// partsView is a window of elements painting what does not move, and of
// drawings that move, painted again in parts: under a rounded clip that
// scrolls, at half opacity, scaled, with text, one over another painted
// after it, in a Draw and a DrawOver callback. statics counts the calls of
// the callbacks that do not move.
func partsView(statics *int) func(c *context) {
	red, blue, green := RGB(220, 40, 40), RGB(37, 99, 235), RGB(20, 160, 60)
	static := func(p *Painter, r Rect) {
		*statics++
		p.Fill(Rect{r.X + 2, r.Y + 2, r.W - 4, r.H - 4}, green, 4)
	}
	moving := func(p *Painter, r Rect) {
		ms := float32(p.Now().UnixMilli() % 40)
		p.AnimationFrame()
		p.Fill(Rect{r.X + ms, r.Y, 6, r.H}, blue, 3)
		p.Stroke(Rect{r.X + 40 - ms, r.Y + 2, 16, 16}, red, 8, 1)
		p.Text(r.X+ms/2, r.Y+r.H-4, "ab", 10, red)
	}
	return func(c *context) {
		coreColumn(c).Background(RGB(250, 250, 250)).Padding(4).Children(func() {
			coreBox(c).Size(120, 20).Draw(static)
			coreScroll(c).Height(40).Radius(10).Children(func() {
				coreBox(c).Size(120, 24).Shrink(0).Draw(moving)
				coreBox(c).Size(120, 24).Shrink(0).Draw(static)
			})
			coreRow(c).Children(func() {
				coreBox(c).Size(60, 20).Opacity(0.5).Background(red).Draw(moving).DrawOver(static)
				coreBox(c).Size(60, 20).Scale(0.8, 1.2).Draw(static).DrawOver(func(p *Painter, r Rect) {
					p.After(16 * time.Millisecond)
					p.Fill(Rect{r.X + float32(p.Now().UnixMilli()%30), r.Y, 4, 4}, red, 0)
				})
			})
			coreBox(c).Size(120, 20).Margin(-30, 0, 0, 0).Draw(static) // over the row
		})
	}
}

// TestRepaintPartsMatchWhole checks that frames painting again only the
// drawings that move, in parts, paint what painting the window whole does,
// without calling the callbacks of what does not move.
func TestRepaintPartsMatchWhole(t *testing.T) {
	var statics, wholeStatics int
	parts := newRepaintTester(partsView(&statics))
	whole := newRepaintTester(partsView(&wholeStatics))
	whole.rt.noParts = true
	n := statics
	for i := range 12 {
		d := time.Duration(5+i*3) * time.Millisecond
		pb, _ := parts.frame(d)
		wb, _ := whole.frame(d)
		if pb || wb {
			t.Fatalf("frame %d built the view: %v %v", i, pb, wb)
		}
		if !bytes.Equal(parts.Image().Pix, whole.Image().Pix) {
			t.Fatalf("frame %d: painted in parts differs from painted whole", i)
		}
	}
	if statics != n {
		t.Errorf("the drawings that do not move were painted %d times again", statics-n)
	}
	if parts.rt.partPaints != 12 || whole.rt.partPaints != 0 {
		t.Errorf("%d frames painted in parts, want 12 (and %d whole)", parts.rt.partPaints, whole.rt.partPaints)
	}
}

// TestRepaintPartsWhole checks that a frame paints the window whole when
// something other than a drawing in a Draw callback moves: a drawing
// asking from a layer's content, and a drawing whose painting outgrows
// what the scene held.
func TestRepaintPartsWhole(t *testing.T) {
	statics := 0
	tt := newRepaintTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) { statics++ })
		coreBox(c).Size(100, 20).Blur(2).Children(func() {
			coreBox(c).Size(50, 10).Draw(func(p *Painter, r Rect) { p.AnimationFrame() })
		})
	})
	n := statics
	tt.frame(8 * time.Millisecond)
	if tt.rt.partPaints != 0 || statics != n+1 {
		t.Errorf("a drawing moving in a layer: %d frames in parts, the rest painted %d times", tt.rt.partPaints, statics-n)
	}
	// A drawing painting more and more glyphs.
	k := 1
	tt = newRepaintTester(func(c *context) {
		coreBox(c).Size(190, 90).Draw(func(p *Painter, r Rect) {
			p.AnimationFrame()
			for i := range k {
				p.Text(r.X+float32(i%40)*4, r.Y+10+float32(i/40), "x", 6, RGB(0, 0, 0))
			}
			k += 600
		})
	})
	whole := 0
	for range 10 {
		p := tt.rt.partPaints
		tt.frame(8 * time.Millisecond)
		if tt.rt.partPaints == p {
			whole++
		}
	}
	if whole == 0 {
		t.Error("a scene growing in parts was never painted whole")
	}
}

// TestRepaintPartsAfterAtlasMoved checks that a window paints itself whole
// when another window made room in the glyph atlases they share: its text
// painted in parts kept the places its glyphs had before, which then held
// the other window's glyphs and masks, as black boxes where text was.
func TestRepaintPartsAfterAtlasMoved(t *testing.T) {
	a := newRepaintTester(func(c *context) {
		coreBox(c).Size(200, 50).Draw(func(p *Painter, r Rect) {
			p.Text(r.X+4, r.Y+30, "Hello, world", 20, RGB(0, 0, 0))
		})
		coreBox(c).Size(20, 20).Draw(func(p *Painter, r Rect) { p.AnimationFrame() })
	})
	a.frame(8 * time.Millisecond)
	want := bytes.Clone(a.Image().Pix)
	// Another window drawing more glyphs than the atlas has room for.
	b := newRepaintTester(func(c *context) {
		coreBox(c).Size(200, 100).Draw(func(p *Painter, r Rect) {
			for i := range 400 {
				p.Text(r.X, r.Y+40, string(rune('一'+i)), 48, RGB(0, 0, 0))
			}
		})
	})
	b.frame(8 * time.Millisecond)
	a.frame(8 * time.Millisecond)
	if !bytes.Equal(a.Image().Pix, want) {
		t.Error("the window's text changed after another window made room in the atlas")
	}
}

// TestRepaintPartsAfterTransientMasks checks that a window paints itself
// whole after a frame that drew masks for the first time: such masks last
// that frame alone, in room of the atlas the next frame of any window
// draws over, and its icons painted in parts showed the other window's
// shapes.
func TestRepaintPartsAfterTransientMasks(t *testing.T) {
	show := false
	a := newRepaintTester(func(c *context) {
		coreBox(c).Size(200, 50).Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
			if !show {
				return
			}
			var tri Path
			tri.MoveTo(r.X+13, r.Y+7).LineTo(r.X+41, r.Y+29).LineTo(r.X+9, r.Y+43).Close()
			p.FillPath(&tri, RGB(0, 0, 0))
		})
		coreBox(c).Size(20, 20).Draw(func(p *Painter, r Rect) { p.AnimationFrame() })
	})
	// The shape shows in one frame painted whole, as after a click.
	show = true
	a.rt.runFrame()
	want := bytes.Clone(a.Image().Pix)
	// Another window drawing a shape of its own for the first time.
	b := newRepaintTester(func(c *context) {
		coreBox(c).Size(200, 100).Draw(func(p *Painter, r Rect) {
			var square Path
			square.MoveTo(r.X+3, r.Y+3).LineTo(r.X+67, r.Y+3).LineTo(r.X+67, r.Y+71).LineTo(r.X+3, r.Y+71).Close()
			p.FillPath(&square, RGB(0, 0, 0))
		})
	})
	b.frame(8 * time.Millisecond)
	for range 3 {
		a.frame(8 * time.Millisecond)
		if !bytes.Equal(a.Image().Pix, want) {
			t.Fatal("the window's shape changed after another window drew its own")
		}
	}
	if a.rt.partPaints == 0 {
		t.Error("the window was never painted in parts once its masks lasted")
	}
}

// TestRepaintPartsContentTransition checks that a frame paints the window
// whole while a text's content changes beside a Draw callback that moves:
// the content transition asks from the element's painting, not a Draw
// callback, and painted in parts it would stay as the last whole frame
// left it, half faded.
func TestRepaintPartsContentTransition(t *testing.T) {
	label := "Connect"
	view := func(c *context) {
		coreColumn(c).Background(RGB(250, 250, 250)).Children(func() {
			coreText(c, label).FontSize(20).ContentTransition(ContentTransition{Animation: &Animation{Duration: 100 * time.Millisecond, Ease: Linear}})
			coreBox(c).Size(120, 20).Draw(func(p *Painter, r Rect) {
				p.AnimationFrame()
				p.Fill(Rect{r.X + float32(p.Now().UnixMilli()%50), r.Y, 6, r.H}, RGB(0, 0, 255), 0)
			})
		})
	}
	parts := newRepaintTester(view)
	whole := newRepaintTester(view)
	whole.rt.noParts = true
	label = "Disconnect"
	parts.Frame()
	whole.Frame()
	for i := range 10 {
		parts.frame(16 * time.Millisecond)
		whole.frame(16 * time.Millisecond)
		if !bytes.Equal(parts.Image().Pix, whole.Image().Pix) {
			t.Fatalf("frame %d, %v into the transition: painted in parts differs from painted whole", i, time.Duration(i+1)*16*time.Millisecond)
		}
	}
}
