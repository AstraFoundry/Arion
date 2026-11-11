package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// clockTester returns a tester whose frames happen at the time it sets.
func clockTester(view func(c *context), w, h int) (*Tester, *time.Time) {
	now := time.Unix(1_000_000, 0)
	tt := coreNewTester(view, w, h)
	tt.rt.clock = func() time.Time { return now }
	tt.Frame()
	return tt, &now
}

// at returns where the text s shows, failing when it does not.
func at(t *testing.T, tt *Tester, s string) Rect {
	t.Helper()
	r, ok := tt.Find(s)
	if !ok {
		t.Fatalf("%q does not show", s)
	}
	return r
}

func TestTransitionMovesElements(t *testing.T) {
	order := []string{"A", "B"}
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			for _, s := range order {
				coreRow(c).Key(s).Height(20).Transition(ElementTransition{Duration: 200 * time.Millisecond, Ease: Linear}).Children(func() {
					coreText(c, s)
				})
			}
		})
	}, 200, 100)
	// Where the texts are in their rows.
	off := at(t, tt, "A").Y
	if y := at(t, tt, "B").Y - off; y != 20 {
		t.Fatalf("B starts at %v", y)
	}
	order = []string{"B", "A"}
	tt.Frame()
	if y := at(t, tt, "B").Y - off; y != 20 {
		t.Errorf("B jumped to %v as the change came", y)
	}
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	b, a := at(t, tt, "B").Y-off, at(t, tt, "A").Y-off
	if b != 10 || a != 10 {
		t.Errorf("halfway, B is at %v and A at %v, not both at 10", b, a)
	}
	*now = now.Add(150 * time.Millisecond)
	tt.Frame()
	if b, a := at(t, tt, "B").Y-off, at(t, tt, "A").Y-off; b != 0 || a != 20 {
		t.Errorf("once over, B is at %v and A at %v", b, a)
	}
	if n := len(tt.rt.trans); n != 2 {
		t.Errorf("%d transitions kept for 2 elements", n)
	}
}

func TestTransitionMovesElementsInPopovers(t *testing.T) {
	order := []string{"A", "B", "C"}
	open := true
	tt, now := clockTester(func(c *context) {
		anchor := coreButtonBase(c).Size(40, 20).Label("anchor")
		corePopoverBase(c, anchor, &open, func(*node) {
			coreRow(c).Children(func() {
				for _, s := range order {
					coreBox(c).Key(s).Size(20, 20).Transition(ElementTransition{Position: true, Duration: time.Second, Ease: Linear}).Children(func() {
						coreText(c, s)
					})
				}
			})
		})
	}, 300, 200)
	a := at(t, tt, "A").X
	order = []string{"C", "B", "A"}
	tt.Frame()
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	if x := at(t, tt, "A").X - a; x != 20 {
		t.Errorf("halfway, A moved %v, not 20", x)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if x := at(t, tt, "A").X - a; x != 40 {
		t.Errorf("once over, A moved %v, not 40", x)
	}
}

func TestPopoversFollowMovingAnchors(t *testing.T) {
	top := float32(0)
	open := true
	var anchor *node
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreBox(c).Size(10, top)
			anchor = coreButtonBase(c).Size(40, 20).Label("anchor").Transition(ElementTransition{Position: true, Duration: time.Second, Ease: Linear})
			corePopoverBase(c, anchor, &open, func(*node) { coreText(c, "P") })
		})
	}, 300, 300)
	top = 100
	tt.Frame()
	for _, d := range []time.Duration{500 * time.Millisecond, time.Second} {
		*now = now.Add(d)
		tt.Frame()
		b := anchor.Bounds()
		if y := at(t, tt, "P").Y; y != b.Y+b.H {
			t.Errorf("the panel is at %v, below an anchor at %v", y, b)
		}
	}
	if y := anchor.Bounds().Y; y != 100 {
		t.Errorf("the anchor ends at %v", y)
	}
}

func TestTransitionResizesAndLaysOutContent(t *testing.T) {
	wide := false
	tt, now := clockTester(func(c *context) {
		w := float32(100)
		if wide {
			w = 300
		}
		coreBox(c).Width(w).Height(40).Transition(ElementTransition{Size: true, Ease: Linear}).Children(func() {
			coreText(c, "Right").AlignSelf(End)
		})
	}, 400, 100)
	if x := at(t, tt, "Right"); x.X+x.W > 101 {
		t.Fatalf("Right ends at %v", x.X+x.W)
	}
	wide = true
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	// The content lays out at the size shown: halfway, 200 wide.
	if r := at(t, tt, "Right"); r.X+r.W < 195 || r.X+r.W > 205 {
		t.Errorf("halfway, Right ends at %v", r.X+r.W)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if r := at(t, tt, "Right"); r.X+r.W < 299 {
		t.Errorf("once over, Right ends at %v", r.X+r.W)
	}
}

func TestTransitionAnimatesOneAxis(t *testing.T) {
	var tr ElementTransition
	var paneW, panelH float32 = 100, 40
	var panel *node
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreColumn(c).Width(paneW).Children(func() {
				panel = coreColumn(c).Key("details").Width(paneW).Height(panelH).Transition(tr)
			})
		})
	}, 400, 400)
	size := func() (float32, float32) { return panel.w, panel.h }
	// start shows the panel at rest at 100×40, with tr.
	start := func(with ElementTransition) {
		tr, paneW, panelH = with, 100, 40
		tt.Frame()
		*now = now.Add(10 * time.Second)
		tt.Frame()
		if w, h := size(); w != 100 || h != 40 {
			t.Fatalf("%+v: the panel rests at %v×%v", with, w, h)
		}
	}
	// Both axes change: the chosen ones animate, the others jump.
	for _, tc := range []struct {
		name         string
		tr           ElementTransition
		halfW, halfH float32
	}{
		{"Height", ElementTransition{Height: true, Ease: Linear}, 300, 80},
		{"Width", ElementTransition{Width: true, Ease: Linear}, 200, 120},
		{"Size", ElementTransition{Size: true, Ease: Linear}, 200, 80},
		{"Width and Height", ElementTransition{Width: true, Height: true, Ease: Linear}, 200, 80},
		{"Size with Height", ElementTransition{Size: true, Height: true, Ease: Linear}, 200, 80},
		{"none chosen", ElementTransition{Ease: Linear}, 200, 80},
		{"Colors alone", ElementTransition{Colors: true, Ease: Linear}, 300, 120},
	} {
		start(tc.tr)
		paneW, panelH = 300, 120
		tt.Frame()
		*now = now.Add(100 * time.Millisecond)
		tt.Frame()
		if w, h := size(); w != tc.halfW || h != tc.halfH {
			t.Errorf("%s: halfway, the panel is %v×%v, want %v×%v", tc.name, w, h, tc.halfW, tc.halfH)
		}
		*now = now.Add(time.Second)
		tt.Frame()
		if w, h := size(); w != 300 || h != 120 {
			t.Errorf("%s: once over, the panel is %v×%v", tc.name, w, h)
		}
		if panel.st.trec.moving != 0 {
			t.Errorf("%s: the transition still moves once over", tc.name)
		}
	}
	// The width follows a drag 1:1 while the height opens: every frame of
	// the drag shows the pane's width, and the height goes on as it went.
	start(ElementTransition{Height: true, Ease: Linear})
	panelH = 120
	tt.Frame()
	for i := 1; i <= 4; i++ {
		paneW = 100 + float32(i)*30
		*now = now.Add(25 * time.Millisecond)
		tt.Frame()
		want := 40 + float32(i)*10
		if w, h := size(); w != paneW || h < want-0.01 || h > want+0.01 {
			t.Errorf("dragged for %d frames, the panel is %v×%v, want %v×%v", i, w, h, paneW, want)
		}
	}
	// A drag alone, the height at rest, animates nothing.
	*now = now.Add(time.Second)
	tt.Frame()
	paneW = 150
	tt.Frame()
	if w, h := size(); w != 150 || h != 120 || panel.st.trec.moving != 0 || panel.Transitioning() {
		t.Errorf("dragged at rest, the panel is %v×%v, moving %b", w, h, panel.st.trec.moving)
	}
}

// The issue's panel: a column whose height follows its content, in a pane
// the user drags wider.
func TestTransitionHeightFollowsContentWhileWidthFollowsPane(t *testing.T) {
	paneW, open := float32(200), false
	var panel *node
	tt, now := clockTester(func(c *context) {
		coreRow(c).Width(paneW).Children(func() {
			panel = coreColumn(c).Key("details").Grow(1).Transition(ElementTransition{Height: true, Ease: Linear}).Children(func() {
				coreText(c, "Details").Height(20)
				if open {
					coreText(c, "Body").Height(60)
				}
			})
		})
	}, 600, 400)
	if panel.w != 200 || panel.h != 20 {
		t.Fatalf("the closed panel is %v×%v", panel.w, panel.h)
	}
	open, paneW = true, 300
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if panel.w != 300 || panel.h != 50 {
		t.Errorf("halfway open while dragged, the panel is %v×%v, want 300×50", panel.w, panel.h)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if panel.w != 300 || panel.h != 80 {
		t.Errorf("open, the panel is %v×%v", panel.w, panel.h)
	}
}

func TestTransitionEntersAndExits(t *testing.T) {
	items := []string{"One", "Two"}
	motion := ElementTransition{Ease: Linear, Enter: &Motion{Collapse: true}, Exit: &Motion{Collapse: true}}
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			for _, s := range items {
				coreRow(c).Key(s).Height(30).Transition(motion).Children(func() { coreText(c, s) })
			}
		})
	}, 200, 200)
	if _, ok := tt.Find("One"); !ok {
		t.Fatal("the first frame shows nothing")
	}
	// The first frame's elements appear at once: their parent is new too.
	if n := len(tt.rt.trans); n != 2 {
		t.Fatalf("%d transitions", n)
	}
	for _, r := range tt.rt.trans {
		if r.moving != 0 {
			t.Error("an element of the first frame came in")
		}
	}

	items = []string{"New", "One", "Two"}
	tt.Frame()
	row := func(s string) *node {
		for e := tt.rt.c.root.first.first; e != nil; e = e.next {
			if e.first != nil && e.first.text == s {
				return e
			}
		}
		return nil
	}
	if e := row("New"); e == nil || e.h != 0 || e.flags&flagClip == 0 {
		t.Fatalf("the new row starts at %+v", e)
	}
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if h := row("New").h; h < 14 || h > 16 {
		t.Errorf("halfway, the new row is %v high", h)
	}
	*now = now.Add(150 * time.Millisecond)
	tt.Frame()
	if e := row("New"); e.h != 30 {
		t.Errorf("once in, the new row is %v high", e.h)
	}

	// Gone, a row leaves as an inert copy, which collapses under its
	// siblings and takes no input.
	items = []string{"New", "Two"}
	tt.Frame()
	var ghost *node
	for e := tt.rt.c.root.first.first; e != nil; e = e.next {
		if e.leaving != 0 {
			ghost = e
		}
	}
	if ghost == nil || ghost.first == nil || ghost.first.text != "One" {
		t.Fatalf("no copy of the row gone: %+v", ghost)
	}
	if ghost.flags&flagInert == 0 || ghost.h != 30 {
		t.Errorf("the copy is not inert, or %v high", ghost.h)
	}
	if _, ok := tt.Find("One"); ok {
		t.Error("the row gone is still found")
	}
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	for e := tt.rt.c.root.first.first; e != nil; e = e.next {
		if e.leaving != 0 && (e.h < 14 || e.h > 16) {
			t.Errorf("halfway, the copy is %v high", e.h)
		}
	}
	*now = now.Add(150 * time.Millisecond)
	tt.Frame()
	for e := tt.rt.c.root.first.first; e != nil; e = e.next {
		if e.leaving != 0 {
			t.Error("the copy is still there once its exit ended")
		}
	}
	if n := len(tt.rt.trans); n != 2 {
		t.Errorf("%d transitions kept for 2 elements", n)
	}

	// Back while its copy was leaving, a row comes back from there.
	items = []string{"New", "Two", "Three"}
	tt.Frame()
	*now = now.Add(time.Second)
	tt.Frame()
	items = []string{"New", "Two"}
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	items = []string{"New", "Two", "Three"}
	tt.Frame()
	leaving := 0
	for e := tt.rt.c.root.first.first; e != nil; e = e.next {
		if e.leaving != 0 {
			leaving++
		}
	}
	if leaving != 0 {
		t.Errorf("%d copies left as the row came back", leaving)
	}
	if e := row("Three"); e == nil || e.h < 14 || e.h > 16 {
		t.Errorf("the row back does not start from where its copy was: %+v", e)
	}
}

func TestTransitionFadesColors(t *testing.T) {
	bg := RGB(0, 0, 255)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(50, 50).Background(bg).Transition(ElementTransition{Colors: true, Ease: Linear})
	}, 100, 100)
	pixel := func() color.RGBA { return tt.Image().RGBAAt(25, 25) }
	if p := pixel(); p.B != 255 {
		t.Fatalf("the box is %v", p)
	}
	bg = RGB(255, 0, 0)
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if p := pixel(); p.R < 110 || p.R > 145 || p.B < 110 || p.B > 145 {
		t.Errorf("halfway, the box is %v", p)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if p := pixel(); p.R != 255 || p.B != 0 {
		t.Errorf("once over, the box is %v", p)
	}
}

func TestTransitionFadesColorsInOklab(t *testing.T) {
	bg := RGB(255, 255, 0)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(50, 50).Background(bg).Transition(ElementTransition{Colors: true,
			Animation: &Animation{Duration: 200 * time.Millisecond, Ease: Linear, Oklab: true}})
	}, 100, 100)
	bg = RGB(0, 0, 255)
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	// Blue, as Oklab mixes them, not sRGB's gray.
	want := RGB(255, 255, 0).MixOklab(bg, 0.5)
	d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	if p := tt.Image().RGBAAt(25, 25); d(p.R, want.R) > 1 || d(p.G, want.G) > 1 || d(p.B, want.B) > 1 {
		t.Errorf("halfway, the box is %v, want %+v", p, want)
	}
}

func TestTransitionWithoutMotion(t *testing.T) {
	x := float32(0)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(100, 100).Children(func() {
			coreText(c, "Moving").Absolute().Left(x).Transition(ElementTransition{})
		})
	}, 300, 200)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	x = 50
	tt.Frame()
	if r := at(t, tt, "Moving"); r.X != 50 {
		t.Errorf("without motion, it shows at %v", r.X)
	}
	tt.SetPreferences(Preferences{})
	x = 0
	tt.Frame()
	if r := at(t, tt, "Moving"); r.X == 0 {
		t.Error("with motion, it jumped")
	}
	// Resizing the window moves nothing smoothly.
	*now = now.Add(time.Second)
	tt.Frame()
	x = 20
	tt.SetSize(400, 200)
	if r := at(t, tt, "Moving"); r.X != 20 {
		t.Errorf("as the window resized, it shows at %v", r.X)
	}
}

func TestDuplicateKeys(t *testing.T) {
	clicks := 0
	view := func(c *context) {
		coreColumn(c).Children(func() {
			for _, s := range []string{"First", "Second"} {
				coreRow(c).Key("same").Children(func() {
					if coreButton(c, s).Clicked() {
						clicks++
					}
				})
			}
		})
	}
	func() {
		defer func() {
			msg, _ := recover().(string)
			if !strings.Contains(msg, `key "same"`) {
				t.Errorf("tests do not panic on a duplicate key: %q", msg)
			}
		}()
		coreNewTester(view, 200, 100)
	}()

	// Apps log it once, and the inspector lists it.
	var out bytes.Buffer
	log.SetOutput(&out)
	defer log.SetOutput(os.Stderr)
	tt := lenientTester(view, 200, 100)
	tt.Frame()
	tt.Frame()
	if n := strings.Count(out.String(), `key "same"`); n != 1 {
		t.Errorf("logged %d times: %s", n, out.String())
	}
	if len(tt.rt.warnings) != 1 {
		t.Errorf("warnings: %q", tt.rt.warnings)
	}
}

func TestDividers(t *testing.T) {
	red := RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Width(100).Gap(10).Dividers(2, red).Children(func() {
			for range 3 {
				coreBox(c).Height(20)
			}
		})
		coreRow(c).Height(20).Gap(10).Dividers(1, red).Children(func() {
			coreBox(c).Width(30)
			coreBox(c).Width(30)
		})
	}, 200, 200)
	img := tt.Image()
	isRed := func(x, y int) bool { p := img.RGBAAt(x, y); return p.R == 255 && p.G == 0 }
	// Between the boxes of the column, at 25 and 55, across it; not after
	// the last.
	for _, y := range []int{24, 25, 54, 55} {
		if !isRed(0, y) || !isRed(99, y) {
			t.Errorf("no divider at y %d", y)
		}
	}
	for _, y := range []int{10, 23, 26, 40, 85} {
		if isRed(50, y) {
			t.Errorf("a divider at y %d", y)
		}
	}
	// The row's, at x 35, below the column.
	if !isRed(35, 90) || isRed(35, 70) || isRed(65, 90) {
		t.Error("the row's divider is not between its children")
	}
}

func TestListDividers(t *testing.T) {
	red := RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreList(c, nil, 3, func(i int) { coreBox(c).Height(20) }).Height(100).Dividers(1, red)
	}, 200, 100)
	img := tt.Image()
	for _, y := range []int{20, 40} {
		if p := img.RGBAAt(50, y); p != (color.RGBA{255, 0, 0, 255}) {
			t.Errorf("no divider at y %d: %v", y, p)
		}
	}
	if p := img.RGBAAt(50, 60); p.R == 255 && p.G == 0 {
		t.Error("a divider after the last row")
	}
}

func TestAttach(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 60).Margin(20).Padding(5).Children(func() {
			coreText(c, "Badge").Size(20, 10).Attach(AnchorTopRight, AnchorCenter)
			coreText(c, "Corner").Size(30, 10).Attach(AnchorBottomRight, AnchorBottomRight).Right(4).Bottom(4)
			coreText(c, "Middle").Size(40, 10).Attach(AnchorCenter, AnchorCenter)
		})
	}, 300, 200)
	for _, c := range []struct {
		s    string
		x, y float32
	}{
		{"Badge", 20 + 100 - 10, 20 - 5},
		{"Corner", 20 + 100 - 30 - 4, 20 + 60 - 10 - 4},
		{"Middle", 20 + 30, 20 + 25},
	} {
		if r := at(t, tt, c.s); r.X != c.x || r.Y != c.y {
			t.Errorf("%s at %v, %v; want %v, %v", c.s, r.X, r.Y, c.x, c.y)
		}
	}
}

func TestTransitionResizesAList(t *testing.T) {
	open := true
	tt, now := clockTester(func(c *context) {
		coreRow(c).Fill().Children(func() {
			w := float32(0)
			if open {
				w = 200
			}
			coreColumn(c).Width(w).FillHeight().ClipX().Transition(ElementTransition{Size: true, Ease: Linear}).Children(func() {
				coreList(c, nil, 1000, func(i int) {
					coreText(c, fmt.Sprintf("Row %d", i)).Padding(4)
				}).Grow(1)
			})
			coreText(c, "Main").Grow(1)
		})
	}, 600, 300)
	if !tt.HasText("Row 0") {
		t.Fatal("the list shows no rows")
	}
	open = false
	tt.Frame()
	for i := 1; i <= 4; i++ {
		*now = now.Add(50 * time.Millisecond)
		tt.Frame()
		if r, ok := tt.Find("Row 0"); i < 4 && (!ok || r.W <= 0) {
			t.Errorf("after %d ms, the list shows no rows", i*50)
		}
	}
	if r, ok := tt.Find("Row 0"); ok && r.W > 0 {
		t.Errorf("the closed list still shows rows: %v", r)
	}
	open = true
	tt.Frame()
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if r, ok := tt.Find("Row 0"); !ok || r.W <= 0 {
		t.Error("the list opening shows no rows")
	}
}

// lenientTester is NewTester as an app's window is: duplicate keys only
// log.
func lenientTester(view func(c *context), width, height int) *Tester {
	h := &headless{w: float32(width), h: float32(height), scale: 1}
	t := &Tester{rt: newRuntime(view, h), h: h}
	t.rt.collect = true
	t.settle()
	return t
}

// A copy of a rich text going with its exit transition keeps the styles
// of its own spans, where another rich text takes its place, and its
// state, in the same frame: the copy painting with the other's styles,
// one span more than its own, went out of their range.
func TestTransitionExitKeepsItsSpans(t *testing.T) {
	red := RGB(200, 0, 0)
	first := true
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			if first {
				coreRichText(c, Span{Text: "one "}, Span{Text: "two", Color: red, Background: red}).
					Transition(ElementTransition{Exit: &Motion{Opacity: 0}})
			} else {
				coreRichText(c, Span{Text: "a "}, Span{Text: "b "}, Span{Text: "c", Color: red, Background: red})
			}
		})
	}, 200, 100)
	first = false
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	var ghost *node
	for e := tt.rt.c.root.first.first; e != nil; e = e.next {
		if e.leaving != 0 {
			ghost = e
		}
	}
	if ghost == nil {
		t.Fatal("the rich text gone has no copy going")
	}
	if sc := ghost.st.spans; sc == nil || len(sc.styles) != len(ghost.spans) {
		t.Errorf("the copy paints %d spans with the styles of another's", len(ghost.spans))
	}
}

func TestTransitionExitTiming(t *testing.T) {
	shown := true
	var tr ElementTransition
	var box *node
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreBox(c).Size(10, 10)
			if shown {
				box = coreBox(c).Key("box").Size(50, 50).Transition(tr)
			}
		})
	}, 100, 100)
	ghost := func() *node {
		for e := tt.rt.c.root.first.first; e != nil; e = e.next {
			if e.leaving != 0 {
				return e
			}
		}
		return nil
	}
	near := func(what string, got, want float32) {
		t.Helper()
		if d := got - want; d < -0.02 || d > 0.02 {
			t.Errorf("%s: %v, want %v", what, got, want)
		}
	}
	// enter shows the box again, and returns its opacity after d.
	enter := func(d time.Duration) float32 {
		t.Helper()
		shown = false
		tt.Frame()
		*now = now.Add(10 * time.Second)
		tt.Frame()
		shown = true
		tt.Frame()
		*now = now.Add(d)
		tt.Frame()
		if !box.opacitySet {
			return 1
		}
		return box.opacity
	}
	// exit removes the box, and returns its copy's opacity after d, -1
	// without a copy.
	exit := func(d time.Duration) float32 {
		t.Helper()
		*now = now.Add(10 * time.Second)
		tt.Frame()
		shown = false
		tt.Frame()
		*now = now.Add(d)
		tt.Frame()
		g := ghost()
		o := float32(-1)
		if g != nil {
			o = g.opacity
		}
		shown = true
		tt.Frame()
		*now = now.Add(10 * time.Second)
		tt.Frame()
		return o
	}

	// The Exit goes over its ExitDuration, along its ExitEase, the Enter
	// over Duration.
	tr = ElementTransition{Duration: time.Second, Ease: Linear, ExitDuration: 100 * time.Millisecond,
		Enter: &Motion{}, Exit: &Motion{}}
	near("entering over a second, after half of it", enter(500*time.Millisecond), 0.5)
	near("going over 100 ms, after 50", exit(50*time.Millisecond), 0.5)
	if o := exit(100 * time.Millisecond); o != -1 {
		t.Errorf("going over 100 ms, after 100 its copy shows at %v", o)
	}
	tr.ExitEase = EaseIn
	near("going along EaseIn, after half of it", exit(50*time.Millisecond), 1-EaseIn(0.5))

	// Without either, it goes as it comes, as before them.
	tr.ExitDuration, tr.ExitEase = 0, nil
	near("going over Duration, after half of it", exit(500*time.Millisecond), 0.5)

	// With a spring, ExitDuration makes the exit a curve, along the
	// spring's Ease, and the Animation's Delay holds it.
	tr = ElementTransition{Animation: &Animation{Spring: SpringOptions{Duration: time.Second}, Delay: 50 * time.Millisecond},
		ExitDuration: 100 * time.Millisecond, ExitEase: Linear, Enter: &Motion{}, Exit: &Motion{}}
	near("going after its delay", exit(100*time.Millisecond), 0.5)

	// A negative ExitDuration goes at once, without a copy, while the
	// Enter still comes in.
	tr = ElementTransition{Animation: &Snappy, ExitDuration: -1, Enter: &Motion{}, Exit: &Motion{}}
	if o := exit(0); o != -1 {
		t.Errorf("going at once, its copy shows at %v", o)
	}
	if o := enter(0); o != 0 {
		t.Errorf("coming in with a spring, it starts at %v", o)
	}

	// A negative Duration shows every change at once: the Enter, the Exit
	// and a move.
	tr = ElementTransition{Duration: -1, Enter: &Motion{Y: 20}, Exit: &Motion{}}
	if o := enter(0); o != 1 || box.y != 10 {
		t.Errorf("coming in at once, it shows at %v, %v opaque", box.y, o)
	}
	if o := exit(0); o != -1 {
		t.Errorf("going at once, its copy shows at %v", o)
	}
	if len(tt.rt.trans) > 1 || tt.rt.animating {
		t.Error("going at once, it keeps animating")
	}
}

func TestTransitionFadesShadows(t *testing.T) {
	var shadows []shadow
	tr := ElementTransition{Ease: Linear}
	var box *node
	tt, now := clockTester(func(c *context) {
		box = coreBox(c).Size(50, 50).Transition(tr)
		for _, sh := range shadows {
			if sh.inset {
				box.InsetShadow(sh.x, sh.y, sh.blur, sh.spread, sh.color)
			} else {
				box.Shadow(sh.x, sh.y, sh.blur, sh.spread, sh.color)
			}
		}
	}, 100, 100)
	black := RGB(0, 0, 0)
	shown := func() []shadow {
		t.Helper()
		*now = now.Add(100 * time.Millisecond)
		tt.Frame()
		return box.shadows
	}
	settle := func() {
		*now = now.Add(time.Second)
		tt.Frame()
	}

	// One appearing comes from a transparent one at no offset.
	shadows = []shadow{{0, 4, 8, 0, black, false}}
	tt.Frame()
	if s := shown(); len(s) != 1 || s[0].y != 2 || s[0].blur != 4 || s[0].color.A < 126 || s[0].color.A > 129 || s[0].color.R != 0 {
		t.Errorf("halfway in, the shadow is %+v", s)
	}
	settle()
	if s := box.shadows; len(s) != 1 || s[0] != shadows[0] {
		t.Errorf("once in, the shadow is %+v", s)
	}
	// One going goes to a transparent one, and the list keeps it while it
	// goes.
	shadows = nil
	tt.Frame()
	if s := shown(); len(s) != 1 || s[0].y != 2 || s[0].color.A < 126 || s[0].color.A > 129 {
		t.Errorf("halfway out, the shadow is %+v", s)
	}
	settle()
	if len(box.shadows) != 0 {
		t.Errorf("once gone, the shadows are %+v", box.shadows)
	}
	// It paints as it shows: halfway, 5 DIPs out, at half its alpha.
	shadows = []shadow{{0, 0, 0, 10, RGB(255, 0, 0), false}}
	tt.Frame()
	shown()
	if p := tt.Image().RGBAAt(52, 25); p.G < 110 || p.G > 145 {
		t.Errorf("halfway in, the shadow paints %v", p)
	}
	settle()
	if p := tt.Image().RGBAAt(52, 25); p.G != 0 || tt.Image().RGBAAt(57, 25).G != 0 {
		t.Errorf("once in, the shadow paints %v", p)
	}

	// An inner shadow where an outer one was changes at once.
	shadows = []shadow{{0, 0, 0, 10, RGB(255, 0, 0), true}}
	tt.Frame()
	if s := shown(); len(s) != 1 || s[0] != shadows[0] {
		t.Errorf("turned inward, the shadow is %+v", s)
	}
	settle()

	// Colors alone leave the shadows changing at once; Shadows alone
	// animates them.
	tr = ElementTransition{Colors: true, Ease: Linear}
	shadows = []shadow{{0, 0, 0, 2, black, true}}
	tt.Frame()
	if s := shown(); s[0] != shadows[0] {
		t.Errorf("with Colors, the shadow is %+v", s)
	}
	tr = ElementTransition{Shadows: true, Ease: Linear}
	tt.Frame()
	shadows = []shadow{{0, 0, 0, 6, black, true}}
	tt.Frame()
	if s := shown(); s[0].spread != 4 {
		t.Errorf("with Shadows, halfway, the shadow is %+v", s)
	}
}

// An Enter and an Exit that Flip mirror their offsets where AttachTo put
// the element on the other side of its target, from the frame it shows:
// a menu coming down below its button comes up above it.
func TestTransitionFlipsWithItsAttachment(t *testing.T) {
	top := float32(20)
	open := false
	tr := ElementTransition{Duration: time.Second, Ease: Linear,
		Enter: &Motion{Y: -8, Opacity: 1, Flip: true}, Exit: &Motion{Y: -8, Opacity: 1, Flip: true}}
	var panel *node
	tt, now := clockTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 20).Label("anchor").Absolute().Left(20).Top(top)
		if open {
			coreOverlay(c, func() {
				panel = coreBox(c).Size(60, 60).AttachTo(b, AnchorBottom, AnchorTop).Transition(tr)
			})
		}
	}, 200, 200)
	offset := func() float32 {
		t.Helper()
		r := tt.rt.trans[panel.id]
		if r == nil {
			t.Fatal("no transition")
		}
		return r.shown.y - r.target.y
	}
	ghost := func() *node {
		for e := tt.rt.c.overlay.first; e != nil; e = e.next {
			if e.leaving != 0 {
				return e
			}
		}
		return nil
	}
	for _, c := range []struct {
		name string
		top  float32
		from float32
	}{{"below", 20, -8}, {"above", 150, 8}} {
		top = c.top
		tt.Frame()
		open = true
		tt.Frame()
		if d := offset(); d != c.from {
			t.Errorf("%s, it comes from %v", c.name, d)
		}
		*now = now.Add(2 * time.Second)
		tt.Frame()
		rest := panel.y
		open = false
		tt.Frame()
		*now = now.Add(500 * time.Millisecond)
		tt.Frame()
		g := ghost()
		if g == nil {
			t.Fatalf("%s, no copy going", c.name)
		}
		if d := g.y - rest; d != c.from/2 {
			t.Errorf("%s, halfway out, it is %v from its place", c.name, d)
		}
		*now = now.Add(2 * time.Second)
		tt.Frame()
	}
	// Without Flip, it comes from where its Enter says.
	tr.Enter.Flip = false
	open = true
	tt.Frame()
	if d := offset(); d != -8 {
		t.Errorf("above without Flip, it comes from %v", d)
	}
}
