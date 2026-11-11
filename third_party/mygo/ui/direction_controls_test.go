package ui

import (
	"runtime"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

// fillsOf returns the rectangles the last frame filled with c.
func fillsOf(tt *Tester, c Color) []scene.Rect {
	var out []scene.Rect
	for _, op := range tt.h.last.Ops {
		if op.Kind == scene.OpFill && op.Color == c.scene() {
			out = append(out, op.Rect)
		}
	}
	return out
}

// rtlTester is a Tester whose window is laid out right to left.
func rtlTester(view func(c *context), w, h int) *Tester {
	tt := coreNewTester(view, w, h)
	tt.SetDirection(RightToLeft)
	return tt
}

// TestRTLParagraphs reads paragraphs without a strong character right to
// left in an element laid out so, and the others by their first strong
// character whatever the element.
func TestRTLParagraphs(t *testing.T) {
	for _, tc := range []struct {
		text string
		rtl  bool // the element's direction
		want bool // the paragraph's
	}{
		{"123", false, false},
		{"123", true, true},
		{"", true, true},
		{"Hello", true, false},
		{"שלום", false, true},
		{"(1) مرحبا", false, true},
	} {
		l := text.Shared().Layout(text.Params{Text: tc.text, RTL: tc.rtl})
		if got := l.Lines[0].RTL; got != tc.want {
			t.Errorf("%q in an element laid out right to left %v reads right to left %v, want %v", tc.text, tc.rtl, got, tc.want)
		}
	}
}

// TestRTLTextAlign starts the lines of an element laid out right to left
// at its right, whatever they read, and its End at the left; left to
// right, a paragraph reading right to left starts at the right, as before.
func TestRTLTextAlign(t *testing.T) {
	glyphsX := func(tt *Tester) (lo, hi float32) {
		lo, hi = 1e9, -1e9
		for _, op := range tt.h.last.Ops {
			if op.Kind == scene.OpGlyphs {
				for _, g := range tt.h.last.Glyphs[op.Start:op.End] {
					lo, hi = min(lo, g.X), max(hi, g.X)
				}
			}
		}
		return lo, hi
	}
	align := Start
	s := "Hello"
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() { coreText(c, s).TextAlign(align) })
	}
	tt := rtlTester(view, 300, 100)
	if lo, _ := glyphsX(tt); lo < 200 {
		t.Errorf("right to left, English starts at %v, want at the right", lo)
	}
	align = End
	tt.Frame()
	if _, hi := glyphsX(tt); hi > 100 {
		t.Errorf("right to left, English aligned at the end reaches %v, want at the left", hi)
	}
	s, align = "שלום", Start
	ltr := coreNewTester(view, 300, 100)
	if lo, _ := glyphsX(ltr); lo < 200 {
		t.Errorf("left to right, Hebrew starts at %v, want at the right as it reads", lo)
	}
	// Custom widgets shaping text in the element's direction align so too.
	l := ShapeRichTextWith([]Span{{Text: "Hi"}}, Font{}, TextShape{Width: 300, Direction: RightToLeft})
	if x, _, _ := l.l.CaretAt(text.CaretPosition{}); x < 250 {
		t.Errorf("shaped right to left, the text starts at %v, want at the right", x)
	}
}

// TestRTLToggles mirrors a switch, whose knob is on the left when on, and
// puts a check box after its label's right.
func TestRTLToggles(t *testing.T) {
	on, checked := true, false
	view := func(c *context) {
		coreColumn(c).Padding(10).Children(func() {
			coreSwitch(c, &on).Label("Wi-Fi")
			coreCheckbox(c, &checked, "Remember me")
		})
	}
	sw := func(tt *Tester) Rect { return at(t, tt, "Wi-Fi") }
	knob := func(tt *Tester) scene.Rect {
		t.Helper()
		// The white disc inside the switch.
		s := sw(tt)
		for _, k := range fillsOf(tt, RGB(255, 255, 255)) {
			if k.X >= s.X && k.X+k.W <= s.X+s.W && k.Y >= s.Y && k.Y+k.H <= s.Y+s.H && k.W < s.W/2 {
				return k
			}
		}
		t.Fatal("no knob")
		return scene.Rect{}
	}
	ltr := coreNewTester(view, 300, 100)
	if k, s := knob(ltr), sw(ltr); k.X < s.X+s.W/2 {
		t.Errorf("on, left to right, the knob is at %v of a switch over %v+%v, want on the right", k.X, s.X, s.W)
	}
	rtl := rtlTester(view, 300, 100)
	if k, s := knob(rtl), sw(rtl); k.X > s.X+s.W/2 {
		t.Errorf("on, right to left, the knob is at %v of a switch over %v+%v, want on the left", k.X, s.X, s.W)
	}
	if s := sw(rtl); s.X+s.W != 290 {
		t.Errorf("right to left, the switch ends at %v, want 290", s.X+s.W)
	}
	if label := at(t, rtl, "Remember me"); label.X+label.W > 290-16 {
		t.Errorf("right to left, the check box's label ends at %v, want left of its box", label.X+label.W)
	}
}

// TestRTLSlider puts a slider's lo at the right: the fill goes from
// there, a press near the right sets a low value, and Left raises it.
func TestRTLSlider(t *testing.T) {
	v := 25.0
	tt := rtlTester(func(c *context) {
		coreColumn(c).Padding(0, 20).Children(func() {
			coreSlider(c, &v, 0, 100).Width(220).Label("Volume")
		})
	}, 260, 80)
	track := at(t, tt, "Volume")
	fills := fillsOf(tt, tt.rt.c.theme.Accent)
	if len(fills) == 0 {
		t.Fatal("no fill")
	}
	if f := fills[0]; f.X+f.W < float32(track.X+track.W)-1 || f.X < track.X+track.W/2 {
		t.Errorf("at a quarter, the fill covers %v+%v of a track over %v+%v, want its right quarter", f.X, f.W, track.X, track.W)
	}
	tt.Click("Volume") // the middle
	if v != 50 {
		t.Fatalf("a click in the middle set %v, want 50", v)
	}
	tt.Key(0, KeyLeft)
	if v != 51 {
		t.Errorf("Left right to left set %v, want 51", v)
	}
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	if v != 49 {
		t.Errorf("Right twice set %v, want 49", v)
	}
	tt.Press(track.X+track.W-4, track.Y+track.H/2)
	tt.Release(track.X+track.W-4, track.Y+track.H/2)
	if v != 0 {
		t.Errorf("a press at the right set %v, want 0", v)
	}
}

// TestRTLProgress fills progress bars and meters from the right, and a
// reversed progress bar from the left.
func TestRTLProgress(t *testing.T) {
	reverse := false
	tt := rtlTester(func(c *context) {
		coreColumn(c).Gap(10).Children(func() {
			p := coreProgress(c, 0.25).Width(200)
			if reverse {
				p.Reverse()
			}
			coreMeter(c, 0.25, 0, 1, nil).Width(200)
		})
	}, 200, 100)
	accent, success := tt.rt.c.theme.Accent, tt.rt.c.theme.Success
	for _, f := range [][]scene.Rect{fillsOf(tt, accent), fillsOf(tt, success)} {
		if len(f) != 1 || f[0].X != 150 || f[0].W != 50 {
			t.Errorf("right to left, a quarter fills %v, want 150+50", f)
		}
	}
	reverse = true
	tt.Frame()
	if f := fillsOf(tt, accent); len(f) != 1 || f[0].X != 0 {
		t.Errorf("reversed right to left, a quarter fills %v, want from 0", f)
	}
}

// TestRTLRangeSlider puts the knobs of a range slider from the right.
func TestRTLRangeSlider(t *testing.T) {
	low, high := 20.0, 60.0
	tt := rtlTester(func(c *context) {
		coreRangeSlider(c, &low, &high, 0, 100, 1).Width(220).Label("Price")
	}, 220, 80)
	fill := fillsOf(tt, tt.rt.c.theme.Accent)
	if len(fill) != 1 {
		t.Fatalf("fills %v", fill)
	}
	// Between 60 and 20 from the right of the track, which the knobs'
	// centers travel 200 DIPs of, from 10.
	lowX, highX := float32(10+200*0.8), float32(10+200*0.4)
	if f := fill[0]; abs(f.X-highX) > 0.5 || abs(f.X+f.W-lowX) > 0.5 {
		t.Errorf("right to left, the range fills %v+%v, want %v to %v", f.X, f.W, highX, lowX)
	}
	tt.Key(0, KeyTab)
	tt.Key(0, KeyLeft)
	if low != 21 {
		t.Errorf("Left on the minimum set it to %v, want 21", low)
	}
}

// TestRTLArrows moves along rows with the arrows the way they point: tabs,
// ratings, steppers and toolbars go to their next with Left right to
// left.
func TestRTLArrows(t *testing.T) {
	tab, stars := 0, 2
	step := 5.0
	tt := rtlTester(func(c *context) {
		coreColumn(c).Gap(10).Children(func() {
			coreTabs(c, &tab, "One", "Two", "Three")
			coreRating(c, &stars, 5).Label("Stars")
			coreStepper(c, &step, 0, 10, 1).Label("Step")
			coreToolbar(c, func() {
				coreButton(c, "Cut")
				coreButton(c, "Copy")
			}).Label("Edit")
		})
	}, 400, 300)
	one, two := at(t, tt, "One"), at(t, tt, "Two")
	if two.X >= one.X {
		t.Errorf("right to left, the second tab is at %v, right of the first at %v", two.X, one.X)
	}
	tt.Click("One")
	tt.Key(0, KeyLeft)
	if tab != 1 {
		t.Errorf("Left chose tab %d, want 1", tab)
	}
	tt.Key(0, KeyRight)
	if tab != 0 {
		t.Errorf("Right chose tab %d, want 0", tab)
	}
	tt.Key(0, KeyTab)
	tt.Key(0, KeyLeft)
	if stars != 3 {
		t.Errorf("Left set %d stars, want 3", stars)
	}
	tt.Key(0, KeyTab)
	tt.Key(0, KeyLeft)
	if step != 6 {
		t.Errorf("Left stepped to %v, want 6", step)
	}
	tt.Key(0, KeyTab)
	if !tt.Focused("Cut") {
		t.Fatal("Tab did not go into the toolbar")
	}
	tt.Key(0, KeyLeft)
	if !tt.Focused("Copy") {
		t.Errorf("Left in the toolbar did not go to Copy")
	}
	if cut, copy := at(t, tt, "Cut"), at(t, tt, "Copy"); copy.X >= cut.X {
		t.Errorf("right to left, Copy is at %v, right of Cut at %v", copy.X, cut.X)
	}
}

// TestRTLTree opens an item of a tree with Left right to left and closes
// it with Right, its arrow before its label at the right.
func TestRTLTree(t *testing.T) {
	open := false
	tt := rtlTester(func(c *context) {
		coreTree(c, func() {
			coreTreeItem(c, "Folder", &open, func() {
				coreTreeItem(c, "File", nil, nil)
			})
		}).Width(200).Label("Files")
	}, 200, 200)
	tt.Click("Folder")
	tt.Key(0, KeyLeft)
	if !open {
		t.Fatal("Left did not open the item")
	}
	folder, file := at(t, tt, "Folder"), at(t, tt, "File")
	if file.X+file.W >= folder.X+folder.W {
		t.Errorf("right to left, the child ends at %v, not indented left of its parent ending at %v", file.X+file.W, folder.X+folder.W)
	}
	tt.Key(0, KeyRight)
	if open {
		t.Error("Right did not close the item")
	}
}

// TestRTLSplit puts the first pane of a split on the right: dragging the
// divider left and Left widen it.
func TestRTLSplit(t *testing.T) {
	size := float32(100)
	tt := rtlTester(func(c *context) {
		coreSplit(c, &size, func() { coreText(c, "First") }, func() { coreText(c, "Second") }).Fill()
	}, 400, 200)
	first, second := at(t, tt, "First"), at(t, tt, "Second")
	if first.X < 300 || second.X >= 300 {
		t.Fatalf("right to left, the first pane's text is at %v and the second's at %v", first.X, second.X)
	}
	// The divider's handle, at the left of the first pane.
	x := float32(400 - 100)
	tt.Press(x, 100)
	tt.Move(x-30, 100)
	tt.Release(x-30, 100)
	if size != 130 {
		t.Errorf("dragging the divider 30 to the left made the pane %v wide, want 130", size)
	}
	tt.Key(0, KeyLeft)
	if size <= 130 {
		t.Errorf("Left made the pane %v wide, want wider", size)
	}
}

// TestRTLTable lays a table's columns out from the right, resizing them
// from their left edges.
func TestRTLTable(t *testing.T) {
	var s ListState
	cols := []TableColumn{{Title: "Name", Width: 100}, {Title: "Size", Width: 80}}
	tt := rtlTester(func(c *context) {
		coreTable(c, &s, cols, 3, func(row, col int) {
			coreText(c, []string{"file", "1 KB"}[col])
		}).Fill()
	}, 400, 200)
	name, size := at(t, tt, "Name"), at(t, tt, "Size")
	if name.X < size.X || name.X < 300 {
		t.Fatalf("right to left, Name is at %v and Size at %v", name.X, size.X)
	}
	// Name's column spans 300 to 400: its end edge is at 300.
	tt.Press(301, 10)
	tt.Move(281, 10)
	tt.Release(281, 10)
	if w, ok := s.Columns.width(&cols[0]); !ok || w != 120 {
		t.Errorf("dragging Name's edge 20 to the left made it %v wide (%v), want 120", w, ok)
	}
}

// TestRTLPopups opens popovers, comboboxes and menus at the start of
// what opens them: the right right to left.
func TestRTLPopups(t *testing.T) {
	open := true
	tt := rtlTester(func(c *context) {
		coreColumn(c).Padding(20).AlignItems(Start).Children(func() {
			b := coreButton(c, "Anchor").Label("Anchor button")
			corePopover(c, b, &open, func() { coreText(c, "Panel content").Width(200) })
			coreMenuButton(c, "Sort", func(m *Menu) { m.Item("Name") }).Label("Sort button")
		})
	}, 400, 300)
	anchor := at(t, tt, "Anchor button")
	panel := at(t, tt, "Panel content")
	if anchor.X+anchor.W != 380 || abs(panel.X+panel.W-(anchor.X+anchor.W)) > 6 {
		t.Errorf("right to left, the panel ends at %v, the anchor at %v: want them lined up at the right", panel.X+panel.W, anchor.X+anchor.W)
	}
	open = false
	tt.Frame()
	sort := at(t, tt, "Sort button")
	tt.Click("Sort button")
	if tt.h.menu == nil || !tt.h.menu.RightToLeft {
		t.Fatalf("right to left, the menu %v is not right to left", tt.h.menu)
	}
	if x := tt.h.menuAt[0]; x < sort.X+sort.W-12 {
		t.Errorf("right to left, the menu shows at %v, want at the button's right", x)
	}
}

// TestRTLScrollBars puts the vertical scroll bar on the left right to
// left, and the horizontal thumb at the right at the start.
func TestRTLScrollBars(t *testing.T) {
	g := scrollBars(Rect{0, 0, 200, 100}, [4]float32{}, 400, 300, 0, 0, flagScrollX|flagScrollY, 6, true)
	if g.v.X > 10 {
		t.Errorf("right to left, the vertical thumb is at %v, want at the left", g.v.X)
	}
	if g.h.X+g.h.W < 190 {
		t.Errorf("right to left at the start, the horizontal thumb ends at %v, want at the right", g.h.X+g.h.W)
	}
	if g.hTrack.X < g.v.X+g.v.W {
		t.Errorf("right to left, the horizontal track starts at %v, under the vertical bar ending at %v", g.hTrack.X, g.v.X+g.v.W)
	}
	ltr := scrollBars(Rect{0, 0, 200, 100}, [4]float32{}, 400, 300, 0, 0, flagScrollX|flagScrollY, 6, false)
	if ltr.v.X < 190 || ltr.h.X > 10 {
		t.Errorf("left to right, the thumbs are at %v and %v", ltr.v.X, ltr.h.X)
	}
}

// TestRTLTextInput starts the text of a single-line input at the right
// right to left, whatever it reads, its placeholder and its caret too.
func TestRTLTextInput(t *testing.T) {
	value := ""
	tt := rtlTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreTextInput(c, &value).Width(200).Placeholder("Name").Label("Name field")
		})
	}, 200, 100)
	field := at(t, tt, "Name field")
	tt.Click("Name field")
	if caret, _ := tt.TextCaret(); caret.X < field.X+field.W-20 {
		t.Errorf("right to left, an empty input's caret is at %v, want at the right of %v+%v", caret.X, field.X, field.W)
	}
	var glyphX []float32
	for _, op := range tt.h.last.Ops {
		if op.Kind == scene.OpGlyphs {
			for _, g := range tt.h.last.Glyphs[op.Start:op.End] {
				glyphX = append(glyphX, g.X)
			}
		}
	}
	if len(glyphX) == 0 || slices.Min(glyphX) < 100 {
		t.Errorf("right to left, the placeholder's glyphs start at %v, want at the right", glyphX)
	}
	tt.Type("Hello")
	ed := tt.rt.states[tt.rt.focused].editor
	if ed.originX < 100 {
		t.Errorf("right to left, a short English text starts at %v, want at the right", ed.originX)
	}
}

// TestRTLWordArrows moves by words the way the arrows point in a
// right-to-left paragraph, as AppKit does.
func TestRTLWordArrows(t *testing.T) {
	value := "שלום עולם"
	tt := coreNewTester(func(c *context) {
		coreTextInput(c, &value).Width(300).Label("Field")
	}, 300, 100)
	tt.Click("Field")
	ed := tt.rt.states[tt.rt.focused].editor
	ed.move(0, false)
	word := Ctrl
	if runtime.GOOS == "darwin" {
		word = Alt
	}
	tt.Key(word, KeyLeft)
	if ed.caret == 0 {
		t.Errorf("the word arrow pointing left stayed at the start of a right-to-left paragraph")
	}
}

// TestRTLAccessibility gives assistive technology the frames the elements
// show at, which right to left are mirrored, as the coordinate system is
// not, and has its increments raise a slider whose Left does.
func TestRTLAccessibility(t *testing.T) {
	v := 30.0
	tt := rtlTester(func(c *context) {
		coreColumn(c).Padding(10).Gap(10).AlignItems(Start).Children(func() {
			coreRow(c).Gap(8).Children(func() {
				coreButton(c, "OK")
				coreButton(c, "Cancel")
			})
			coreSlider(c, &v, 0, 100).Width(200).Label("Volume")
			coreScrollHorizontal(c).Size(100, 30).Children(func() {
				coreButton(c, "Wide").Width(250)
			})
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	for _, label := range []string{"OK", "Cancel", "Wide"} {
		n := accessNode(t, tree, platform.RoleButton, label)
		r := at(t, tt, label)
		if b := n.Bounds; float32(b.X) > r.X || float32(b.X+b.W) < r.X+r.W {
			t.Errorf("%s: assistive technology sees %+v, the frame shows it over %+v", label, b, r)
		}
	}
	ok, cancel := accessNode(t, tree, platform.RoleButton, "OK"), accessNode(t, tree, platform.RoleButton, "Cancel")
	if ok.Bounds.X+ok.Bounds.W != 390 || cancel.Bounds.X >= ok.Bounds.X {
		t.Errorf("right to left, OK is at %+v and Cancel at %+v", ok.Bounds, cancel.Bounds)
	}
	// The wide button, scrolled to its start at the right, shows its right.
	if wide := accessNode(t, tree, platform.RoleButton, "Wide"); wide.Bounds.X+wide.Bounds.W != 390 {
		t.Errorf("right to left, the scrolled button is at %+v, want ending at 390", wide.Bounds)
	}
	slider := accessNode(t, tree, platform.RoleSlider, "Volume")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: slider.ID, Action: platform.AccessIncrement})
	if v <= 30 {
		t.Errorf("right to left, incrementing set %v", v)
	}
}
