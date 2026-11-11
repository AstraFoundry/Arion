package ui

import (
	"errors"
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/vec"
)

// These tests cover what widgets of your own are made of: the states and
// roles they give assistive technology, overlays and their placement, and
// the settings of bases.

func TestAccessStatesOfYourOwn(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreText(c, "Settings").Role(RoleHeading).Level(2)
			coreBox(c).Size(20, 20).Focusable().Role(RoleCheckBox).Label("All").Mixed()
			coreBox(c).Size(20, 20).Focusable().Role(RolePopUpButton).Label("Date").Expanded(true).Value("May 4")
			coreBox(c).Size(80, 20).Focusable().Role(RoleSlider).Label("Rating").Range(0, 5, 3)
			menu := coreColumn(c).Focusable().Role(RoleMenu).Label("Edit").AutoFocus()
			menu.Children(func() {
				coreText(c, "Cut").Role(RoleMenuItem)
				bold := coreText(c, "Bold").Role(RoleMenuItemCheckBox).Checked(true)
				coreText(c, "Left").Role(RoleMenuItemRadio).Checked(false)
				menu.ActiveDescendant(bold)
			})
		})
	}, 300, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	if h := accessNode(t, tree, platform.RoleHeading, "Settings"); h.Level != 2 {
		t.Errorf("heading level %d", h.Level)
	}
	if all := accessNode(t, tree, platform.RoleCheckBox, "All"); all.States&platform.AccessMixed == 0 {
		t.Errorf("the check box is not mixed: %b", all.States)
	}
	if date := accessNode(t, tree, platform.RolePopUpButton, "Date"); date.States&platform.AccessExpanded == 0 || date.Value != "May 4" {
		t.Errorf("pop-up button: %+v", date)
	}
	if r := accessNode(t, tree, platform.RoleSlider, "Rating"); r.Min != 0 || r.Max != 5 || r.Now != 3 {
		t.Errorf("slider range %v..%v at %v", r.Min, r.Max, r.Now)
	}
	menu := accessNode(t, tree, platform.RoleMenu, "Edit")
	accessNode(t, tree, platform.RoleMenuItem, "Cut")
	bold := accessNode(t, tree, platform.RoleMenuItemCheckBox, "Bold")
	left := accessNode(t, tree, platform.RoleMenuItemRadio, "Left")
	if bold.States&platform.AccessChecked == 0 || left.States&platform.AccessChecked != 0 {
		t.Errorf("menu items checked: bold %b, left %b", bold.States, left.States)
	}
	// A menu holds its items, which are not folded into its name.
	if bold.Parent < 0 || tree.Nodes[bold.Parent].ID != menu.ID {
		t.Error("the item is not inside the menu")
	}
	if tree.Focus != bold.ID {
		t.Errorf("the focus is on %d, not on the active descendant %d", tree.Focus, bold.ID)
	}
}

// TestBusyAndCurrent checks aria-busy and aria-current: a region marked
// busy, seen without a role or a Label of its own, until it is no longer,
// and the current items of a navigation landmark.
func TestBusyAndCurrent(t *testing.T) {
	loading := true
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreColumn(c).Busy(loading).Children(func() {
				coreText(c, "Inbox")
			})
			coreRow(c).Role(RoleNavigation).Label("Steps").Children(func() {
				coreText(c, "Cart")
				coreText(c, "Payment").Current(CurrentStep)
				coreText(c, "Today").Current(CurrentDate)
			})
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	inbox := accessNode(t, tree, platform.RoleText, "Inbox")
	if inbox.Parent < 0 || tree.Nodes[inbox.Parent].Role != platform.RoleGroup || tree.Nodes[inbox.Parent].States&platform.AccessBusy == 0 {
		t.Fatalf("the text is not in a busy group: %+v", inbox)
	}
	region := tree.Nodes[inbox.Parent].ID
	steps := accessNode(t, tree, platform.RoleNavigation, "Steps")
	for _, c := range []struct {
		label string
		want  platform.AccessCurrent
	}{{"Cart", platform.CurrentNone}, {"Payment", platform.CurrentStep}, {"Today", platform.CurrentDate}} {
		n := accessNode(t, tree, platform.RoleText, c.label)
		if n.Current != c.want {
			t.Errorf("%s: current %v, want %v", c.label, n.Current, c.want)
		}
		if n.Parent < 0 || tree.Nodes[n.Parent].ID != steps.ID {
			t.Errorf("%s is not in the landmark", c.label)
		}
	}
	// Loaded, the region is no longer busy, nor seen: it has nothing of
	// its own to say.
	loading = false
	tt.Frame()
	tree = tt.h.access
	for _, n := range tree.Nodes {
		if n.ID == region || n.States&platform.AccessBusy != 0 {
			t.Errorf("still busy: %+v", n)
		}
	}
	if inbox := accessNode(t, tree, platform.RoleText, "Inbox"); inbox.Parent >= 0 && tree.Nodes[inbox.Parent].Role == platform.RoleGroup {
		t.Errorf("the text is still in a group: %+v", tree.Nodes[inbox.Parent])
	}
}

func TestReadOnlyInput(t *testing.T) {
	text, readOnly := "hello", true
	tt := coreNewTester(func(c *context) {
		coreTextInputBase(c, &text).Width(100).Label("name").ReadOnly(readOnly)
	}, 200, 50)
	tt.Click("name")
	tt.Key(0, KeyEnd)
	tt.Type("x")
	tt.Key(0, KeyBackspace)
	tt.Command("paste")
	if text != "hello" {
		t.Fatalf("a read-only input became %q", text)
	}
	if _, ok := tt.TextCaret(); ok {
		t.Error("a read-only input turned the input method on")
	}
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "hello" {
		t.Errorf("copied %q", got)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "name")
	if field.States&platform.AccessReadOnly == 0 || field.Actions&platform.ActionSetValue != 0 {
		t.Errorf("assistive technology sees %+v", field)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "bye"})
	if text != "hello" {
		t.Fatalf("assistive technology set a read-only input to %q", text)
	}

	// Editable again once the view no longer says it is read-only.
	readOnly = false
	tt.Frame()
	tt.Key(0, KeyEnd)
	tt.Type("!")
	if text != "hello!" {
		t.Errorf("typed into %q", text)
	}
}

func TestComposing(t *testing.T) {
	text, composing := "", false
	submitted := 0
	tt := coreNewTester(func(c *context) {
		in := coreTextInputBase(c, &text).Width(100).Label("name")
		composing = in.Composing()
		if in.Submitted() {
			submitted++
		}
	}, 200, 50)
	tt.Click("name")
	tt.Compose("ni", 2)
	if !composing || text != "" {
		t.Fatalf("composing %v, text %q", composing, text)
	}
	tt.Type("你")
	if composing || text != "你" {
		t.Errorf("after the commit: composing %v, text %q", composing, text)
	}
	if submitted != 0 {
		t.Errorf("submitted %d times", submitted)
	}
}

func TestOpenURLOutcome(t *testing.T) {
	var outcome error
	calls := 0
	tt := coreNewTester(func(c *context) {
		if coreButtonBase(c).Size(40, 20).Label("go").Clicked() {
			c.OpenURLThen("x-unknown://a", func(err error) {
				outcome = err
				calls++
			})
		}
		if coreButtonBase(c).Size(40, 20).Label("plain").Clicked() {
			c.OpenURL("https://example.com")
		}
		if outcome != nil {
			coreText(c, "Couldn't open the link")
		}
	}, 200, 100)
	tt.Click("go")
	if calls != 1 || outcome != nil {
		t.Fatalf("opened: %d calls, error %v", calls, outcome)
	}
	tt.FailOpenURL(errors.New("no app"))
	tt.Click("go")
	if calls != 2 || outcome == nil || !tt.HasText("Couldn't open the link") {
		t.Errorf("failing: %d calls, error %v, texts %q", calls, outcome, tt.Texts())
	}
	tt.Click("plain")
	if got := tt.OpenedURLs(); len(got) != 3 || got[2] != "https://example.com" || calls != 2 {
		t.Errorf("opened %q, %d calls", got, calls)
	}
}

func TestPopoverLightDismiss(t *testing.T) {
	open, inner := false, true
	behind := 0
	tt := coreNewTester(func(c *context) {
		coreRow(c).Padding(10).Gap(100).AlignItems(Start).Children(func() {
			more := coreButtonBase(c).Size(60, 20).Label("more")
			if more.Clicked() {
				open = !open
			}
			if coreButtonBase(c).Size(60, 20).Label("behind").Clicked() {
				behind++
			}
			corePopoverBase(c, more, &open, func(panel *node) {
				coreButtonBase(c).Size(80, 20).Label("rename")
				sub := coreButtonBase(c).Size(80, 20).Label("submenu")
				corePopoverBase(c, sub, &inner, func(*node) { coreButtonBase(c).Size(40, 20).Label("deep") })
			})
		})
	}, 400, 300)
	tt.Click("more")
	r, ok := tt.Find("rename")
	if !open || !ok || r.X != 10 || r.Y != 30 {
		t.Fatalf("the popover is at %v (%v), open %v", r, ok, open)
	}
	// Pressing in a popover of an element in the panel is not outside it,
	// and pressing in the panel is outside that popover.
	tt.Click("deep")
	if !open || !inner {
		t.Fatalf("a press in the inner popover: open %v, inner %v", open, inner)
	}
	tt.Click("rename")
	if !open || inner {
		t.Fatalf("a press in the panel: open %v, inner %v", open, inner)
	}
	// Outside, the press closes it and goes on to what is under it.
	tt.Click("behind")
	if open || behind != 1 {
		t.Fatalf("pressing outside: open %v, behind clicked %d times", open, behind)
	}
	// The anchor toggles it, as its press is not outside.
	tt.Click("more")
	tt.Click("more")
	if open {
		t.Error("pressing the anchor again left the popover open")
	}
	tt.Click("more")
	tt.Key(0, KeyEscape)
	if open {
		t.Error("Escape left the popover open")
	}
}

func TestAttachTo(t *testing.T) {
	var at, self Anchor
	right := false
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 20).Label("anchor").Absolute().Top(100)
		if right {
			b.Left(350)
		} else {
			b.Left(100)
		}
		coreOverlay(c, func() {
			coreBox(c).Size(60, 30).Label("panel").AttachTo(b, at, self).Margin(4, 0, 0, 0)
		})
	}, 400, 300)
	find := func() Rect {
		t.Helper()
		r, ok := tt.Find("panel")
		if !ok {
			t.Fatal("no panel")
		}
		return r
	}
	at, self = AnchorBottomLeft, AnchorTopLeft
	tt.Frame()
	if r := find(); r.X != 100 || r.Y != 124 {
		t.Errorf("below: %v", r)
	}
	at, self = AnchorRight, AnchorLeft
	tt.Frame()
	if r := find(); r.X != 140 || r.Y != 99 {
		t.Errorf("to the right: %v", r)
	}
	// Near the right edge it goes to the left, and the left-aligned panel
	// below aligns its right edge instead.
	right = true
	tt.Frame()
	if r := find(); r.X != 290 {
		t.Errorf("to the right near the edge: %v", r)
	}
	at, self = AnchorBottomLeft, AnchorTopLeft
	tt.Frame()
	if r := find(); r.X != 330 || r.Y != 124 {
		t.Errorf("below near the right edge: %v", r)
	}
	// Above, where there is no room below, apart by its top margin.
	tt.SetSize(400, 140)
	tt.Frame()
	if r := find(); r.Y != 66 {
		t.Errorf("above: %v", r)
	}
}

// TestAttachEdgeMargin keeps an attached panel further from the window's
// edges than the default 4.
func TestAttachEdgeMargin(t *testing.T) {
	var edge float32 = -1
	var tall bool
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 20).Label("anchor").Absolute().Left(2).Top(10)
		coreOverlay(c, func() {
			p := coreBox(c).Size(288, 30).Label("panel").AttachTo(b, AnchorBottomLeft, AnchorTopLeft)
			if tall {
				p.Height(400).flags |= flagScrollY
			}
			if edge >= 0 {
				p.AttachEdgeMargin(edge)
			}
		})
	}, 320, 300)
	find := func() Rect {
		t.Helper()
		r, ok := tt.Find("panel")
		if !ok {
			t.Fatal("no panel")
		}
		return r
	}
	// Its left edge on the anchor's, 2 from the window's, it moves along
	// the anchor into the window: 4 from the edge by default.
	if r := find(); r.X != 4 {
		t.Errorf("default: x %v, want 4", r.X)
	}
	edge = 12
	tt.Frame()
	if r := find(); r.X != 12 {
		t.Errorf("edge margin 12: x %v, want 12", r.X)
	}
	edge = 0
	tt.Frame()
	if r := find(); r.X != 2 {
		t.Errorf("edge margin 0: x %v, want 2, on the anchor", r.X)
	}
	// One that scrolls is no taller than the room below the anchor, which
	// ends 30 from the top, less the margin.
	edge, tall = 12, true
	tt.Frame()
	if r := find(); r.H != 300-30-12 {
		t.Errorf("scrolling panel: h %v, want %v", r.H, 300-30-12)
	}
}

func TestAttachedAnchors(t *testing.T) {
	var viewAt, viewSelf, drawAt, drawSelf Anchor
	var panel *node
	left, top := float32(100), float32(100)
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 20).Label("anchor").Absolute().Left(left).Top(top)
		coreOverlay(c, func() {
			p := coreBox(c).Size(60, 30).Label("panel").AttachTo(b, AnchorBottomStart, AnchorTopStart).AttachedOrigin()
			viewAt, viewSelf = p.AttachedAnchors()
			p.Draw(func(*Painter, Rect) { drawAt, drawSelf = p.AttachedAnchors() })
			panel = p
		})
	}, 400, 300)
	check := func(what string, at, self, wantAt, wantSelf Anchor) {
		t.Helper()
		if at != wantAt || self != wantSelf {
			t.Errorf("%s: %v %v, want %v %v", what, at, self, wantAt, wantSelf)
		}
	}
	origin := func(x, y float32) {
		t.Helper()
		if panel.originX != x || panel.originY != y {
			t.Errorf("origin %v %v, want %v %v", panel.originX, panel.originY, x, y)
		}
	}
	check("below, in the view", viewAt, viewSelf, AnchorBottomStart, AnchorTopStart)
	check("below, drawn", drawAt, drawSelf, AnchorBottomStart, AnchorTopStart)
	origin(0, 0)

	// Without room below, it goes above: drawn and its origin in this
	// frame, read by the view in the next.
	top = 260
	tt.Frame()
	check("above, in the view of the frame it moved", viewAt, viewSelf, AnchorBottomStart, AnchorTopStart)
	check("above, drawn", drawAt, drawSelf, AnchorTopStart, AnchorBottomStart)
	origin(0, 1)
	tt.Frame()
	check("above, in the view", viewAt, viewSelf, AnchorTopStart, AnchorBottomStart)

	// Near the right edge too, it aligns its end with the anchor's.
	left = 350
	tt.Frame()
	tt.Frame()
	check("above, near the right edge", viewAt, viewSelf, AnchorTopEnd, AnchorBottomEnd)
	origin(1, 1)
}

func TestAnchorsFlipped(t *testing.T) {
	for a := AnchorTopLeft; a <= AnchorBottomEnd; a++ {
		if a.acrossFlipped().acrossFlipped() != a || a.downFlipped().downFlipped() != a {
			t.Errorf("%v flipped twice is not itself", a)
		}
		for _, rtl := range []bool{false, true} {
			fx, fy := a.physical(rtl).fractions()
			if x, y := a.acrossFlipped().physical(rtl).fractions(); x != 1-fx || y != fy {
				t.Errorf("%v across, rtl %v: %v %v from %v %v", a, rtl, x, y, fx, fy)
			}
			if x, y := a.downFlipped().physical(rtl).fractions(); x != fx || y != 1-fy {
				t.Errorf("%v down, rtl %v: %v %v from %v %v", a, rtl, x, y, fx, fy)
			}
		}
	}
}

func TestModalOverlayOfYourOwn(t *testing.T) {
	open := false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			if coreButtonBase(c).Size(60, 20).Label("open").Clicked() {
				open = true
			}
			coreButtonBase(c).Size(60, 20).Label("outside")
		})
		if open {
			coreOverlay(c, func() {
				back := coreBox(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().Modal()
				if back.OverlayShortcut(0, KeyEscape) {
					open = false
				}
				back.Children(func() {
					coreRow(c).Role(RoleDialog).Label("Rename").Children(func() {
						coreButtonBase(c).Size(60, 20).Label("ok")
						coreButtonBase(c).Size(60, 20).Label("cancel")
					})
				})
			})
		}
	}, 300, 200)
	tt.Key(0, KeyTab) // the focus on "open"
	tt.Key(0, KeyEnter)
	if !open || !tt.Focused("ok") {
		t.Fatalf("open %v, the focus on ok %v", open, tt.Focused("ok"))
	}
	tt.Key(0, KeyTab)
	tt.Key(0, KeyTab)
	if !tt.Focused("ok") {
		t.Error("Tab left the dialog")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, n := range tt.h.access.Nodes {
		if n.Label == "outside" {
			t.Error("assistive technology sees what is behind the dialog")
		}
	}
	tt.Key(0, KeyEscape)
	if open || !tt.Focused("open") {
		t.Errorf("Escape: open %v, the focus back on open %v", open, tt.Focused("open"))
	}
}

func TestSliderBaseSettings(t *testing.T) {
	v, h := 0.0, 0.0
	disabled := false
	tt := coreNewTester(func(c *context) {
		coreRow(c).Gap(20).AlignItems(Start).Children(func() {
			coreSliderBase(c, &v, 0, 100).Size(20, 100).Label("v").Vertical().Step(10)
			coreSliderBase(c, &h, 0, 100).Size(100, 20).Label("h").Step(25).Disabled(disabled)
		})
	}, 300, 200)
	r, _ := tt.Find("v")
	tt.Press(r.X+10, r.Y+r.H*0.3)
	tt.Release(r.X+10, r.Y+r.H*0.3)
	if v != 70 {
		t.Errorf("pressing 30%% down a vertical slider: %v", v)
	}
	tt.Key(0, KeyUp)
	if v != 80 {
		t.Errorf("Up: %v", v)
	}
	tt.Click("h")
	if h != 50 {
		t.Errorf("pressing the middle: %v", h)
	}
	tt.Key(0, KeyRight)
	tt.Key(0, KeyPageUp)
	if h != 100 {
		t.Errorf("Right and Page Up by steps of 25 from 50: %v", h)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleSlider, "v"); n.States&platform.AccessVertical == 0 {
		t.Error("the vertical slider is not vertical to assistive technology")
	}
	disabled = true
	tt.Frame()
	hr, _ := tt.Find("h")
	tt.Press(hr.X+1, hr.Y+10)
	tt.Release(hr.X+1, hr.Y+10)
	tt.Key(0, KeyLeft)
	if h != 100 {
		t.Errorf("a disabled slider moved to %v", h)
	}
}

func TestSelectHighlight(t *testing.T) {
	size := "S"
	jump := false
	tt := coreNewTester(func(c *context) {
		sel := coreSelectBase(c, &size)
		sel.Trigger.Size(60, 20).Label("size")
		if jump {
			sel.Highlight("L")
			jump = false
		}
		sel.Popup(func(panel *node) {
			for _, s := range []string{"S", "M", "L"} {
				sel.Item(s).Size(60, 20).Children(func() { coreText(c, s) })
			}
		})
	}, 200, 200)
	tt.Click("size")
	jump = true
	tt.Frame()
	tt.Key(0, KeyEnter)
	if size != "L" {
		t.Errorf("chose %q", size)
	}
}

// A select with more options than the window has room for, opening below
// or above, keeps its popup in the window, where they scroll: the first
// and the last reached, a click chooses the one under it.
func TestSelectLongListScrolls(t *testing.T) {
	options := make([]string, 60)
	for i := range options {
		options[i] = fmt.Sprintf("Option %02d", i)
	}
	for _, top := range []float32{20, 200} {
		value := options[30]
		tt := NewTester(func(c *Context) {
			Column(c).Padding(top, 20).Children(func() {
				Select(c, &value, options).Label("Choice")
			})
		}, 400, 300)
		tt.Click("Choice")
		for _, end := range []struct {
			key    Key
			option string
		}{{KeyEnd, "Option 59"}, {KeyHome, "Option 00"}} {
			tt.Key(0, end.key)
			tt.Frame()
			if r, ok := tt.Find(end.option); !ok || r.H == 0 || r.Y < 0 || r.Y+r.H > 300 {
				t.Fatalf("at %v: %s at %+v, out of the window", top, end.option, r)
			}
		}
		tt.Click("Option 00")
		if value != "Option 00" {
			t.Errorf("at %v: a click on Option 00 chose %q", top, value)
		}
	}
}

func TestDisabledStepper(t *testing.T) {
	v := 5.0
	tt := coreNewTester(func(c *context) {
		coreStepper(c, &v, 0, 10, 1).Label("count").Disabled(true)
	}, 100, 100)
	r, _ := tt.Find("count")
	tt.ClickAt(r.X+r.W/2, r.Y+4)
	tt.Key(0, KeyTab)
	tt.Key(0, KeyUp)
	if v != 5 {
		t.Errorf("a disabled stepper stepped to %v", v)
	}
}

func TestSnapToTheStepsDecimals(t *testing.T) {
	for _, c := range []struct{ v, lo, hi, step, want float64 }{
		{0.30000000000000004, 0, 1, 0.1, 0.3},
		{5.2e-12, 0, 1, 1e-12, 5e-12},
		{1234567.8914, 0, 1e7, 0.001, 1234567.891},
		{3.4, 0, 10, 2.5, 2.5},
	} {
		if got := snap(c.v, c.lo, c.hi, c.step); got != c.want {
			t.Errorf("snap(%v, %v, %v, %v) = %v, want %v", c.v, c.lo, c.hi, c.step, got, c.want)
		}
	}
}

// TestPathsLookTheSameWhateverDrewBefore draws a path a fraction of a pixel
// off the grid, alone and after another path that shares its mask: it
// shows the same pixels.
func TestPathsLookTheSameWhateverDrewBefore(t *testing.T) {
	shape := func(p *Painter, x, y float32) {
		var path Path
		path.MoveTo(x, y).LineTo(x+13.3, y+2.7).LineTo(x+5.9, y+11.1).Close()
		p.FillPath(&path, RGB(0, 0, 0))
	}
	render := func(before bool) *image.RGBA {
		return coreRender(func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
				if before {
					shape(p, 3.1, 3.1)
				}
				shape(p, 30.05, 3.05)
			})
		}, 60, 20, 1)
	}
	alone, after := render(false), render(true)
	for y := range 20 {
		for x := 28; x < 60; x++ {
			if a, b := alone.RGBAAt(x, y), after.RGBAAt(x, y); a != b {
				t.Fatalf("(%d, %d) is %v alone, %v after another path", x, y, a, b)
			}
		}
	}
}

// TestStrokesAreAsWideAsAsked strokes a circle and a square: the ink of
// their masks is the area of the stroke, along curves too, where the
// stroke's pieces meet at every point of the flattened curve. (Windows
// draws masks with the contrast of its text, so the rendered pixels are
// darker there.)
func TestStrokesAreAsWideAsAsked(t *testing.T) {
	ink := func(path *Path, width, scale float32) float64 {
		var f flatPath
		path.flatten(&f, scale)
		var z vec.Rasterizer
		var bufs strokeBufs
		n := int(50 * scale)
		z.Reset(n, n)
		strokeInto(&z, &bufs, &f, width*scale/2, 0, 0)
		pix := make([]byte, n*n)
		z.Mask(pix, n)
		var sum float64
		for _, v := range pix {
			sum += float64(v) / 255
		}
		return sum / float64(scale*scale)
	}
	for _, scale := range []float32{1, 2} {
		for _, w := range []float32{1, 2} {
			var circle, square Path
			circle.Circle(25.2, 25.3, 10)
			square.MoveTo(10.3, 10.3).LineTo(40.3, 10.3).LineTo(40.3, 40.3).LineTo(10.3, 40.3).Close()
			for _, c := range []struct {
				name string
				path *Path
				area float64
			}{
				{"circle", &circle, 2 * math.Pi * 10 * float64(w)},
				{"square", &square, 4*30*float64(w) - (4-math.Pi)*float64(w*w)/4},
			} {
				if got := ink(c.path, w, scale); math.Abs(got/c.area-1) > 0.02 {
					t.Errorf("a %s stroked %v wide at scale %v has ink %.1f, want %.1f", c.name, w, scale, got, c.area)
				}
			}
		}
	}
}

// TestDrawer builds the drawer of the overlays guide.
func TestDrawer(t *testing.T) {
	drawer := false
	tt := coreNewTester(func(c *context) {
		th := c.Theme()
		if coreButtonBase(c).Size(60, 20).Label("filters").Clicked() {
			drawer = true
		}
		if drawer {
			coreOverlay(c, func() {
				back := coreRow(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Justify(End).
					Background(RGBA(0, 0, 0, 0.3)).Modal()
				back.Children(func() {
					panel := coreColumn(c).Width(120).FillHeight().Padding(16).Background(th.Background).
						Role(RoleDialog).Label("Filters").Children(func() {
						coreButtonBase(c).Size(60, 20).Label("apply")
					})
					if panel.PressedOutside() || back.OverlayShortcut(0, KeyEscape) {
						drawer = false
					}
				})
			})
		}
	}, 300, 200)
	tt.Click("filters")
	if r, _ := tt.Find("Filters"); !drawer || r.X != 180 {
		t.Fatalf("the drawer: open %v, at %v", drawer, r)
	}
	tt.Click("apply")
	tt.ClickAt(250, 150)
	if !drawer {
		t.Fatal("a click in the drawer closed it")
	}
	tt.ClickAt(30, 150)
	if drawer || !tt.Focused("filters") {
		t.Errorf("a click on the backdrop: open %v, the focus back on its button %v", drawer, tt.Focused("filters"))
	}
}

// TestPopoverScrollsWithItsAnchor scrolls the anchor of a popover: the panel
// moves with it in the same frame.
func TestPopoverScrollsWithItsAnchor(t *testing.T) {
	open := true
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().Children(func() {
			coreColumn(c).Height(1000).AlignItems(Start).Children(func() {
				coreBox(c).Height(100)
				b := coreButtonBase(c).Size(60, 20).Label("anchor")
				corePopoverBase(c, b, &open, func(panel *node) { coreBox(c).Size(40, 20).Label("panel") })
			})
		})
	}, 200, 300)
	if r, _ := tt.Find("panel"); r.Y != 120 {
		t.Fatalf("the panel is at %v", r)
	}
	tt.Scroll(100, 150, 0, 50)
	a, _ := tt.Find("anchor")
	if r, _ := tt.Find("panel"); a.Y != 50 || r.Y != 70 {
		t.Errorf("scrolled by 50: the anchor at %v, the panel at %v", a, r)
	}
}

// TestEscapeClosesTheInnerPopover opens a popover in a popover, whose
// Escape registers after the inner one's: Escape closes the inner first.
func TestEscapeClosesTheInnerPopover(t *testing.T) {
	outer, inner := true, true
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(60, 20).Label("more")
		corePopoverBase(c, b, &outer, func(*node) {
			sub := coreButtonBase(c).Size(60, 20).Label("sub")
			corePopoverBase(c, sub, &inner, func(*node) { coreBox(c).Size(40, 20) })
		})
	}, 300, 300)
	tt.Key(0, KeyEscape)
	if !outer || inner {
		t.Fatalf("one Escape: outer open %v, inner open %v", outer, inner)
	}
	tt.Key(0, KeyEscape)
	if outer {
		t.Error("the second Escape left the outer popover open")
	}
}

// TestAttachToInlineText attaches a card below a link inside a paragraph.
func TestAttachToInlineText(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		var link *node
		coreText(c, "").Width(300).Children(func() {
			coreText(c, "Read the ")
			link = coreLink(c, "guide", "https://example.com")
			coreText(c, " first.")
		})
		coreOverlay(c, func() {
			coreBox(c).Size(80, 40).Label("card").AttachTo(link, AnchorBottomLeft, AnchorTopLeft)
		})
	}, 400, 200)
	tt.Frame()
	l, _ := tt.Find("guide")
	card, _ := tt.Find("card")
	if l.W <= 0 || card.X != l.X || card.Y != l.Y+l.H {
		t.Errorf("the link at %v, the card at %v", l, card)
	}
}

// TestRangeSteps checks how far assistive technology learns the keys move
// the value of each kind of range.
func TestRangeSteps(t *testing.T) {
	a, b, s, low, high := 50.0, 50.0, 5.0, 10.0, 90.0
	stars := 3
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreSlider(c, &a, 0, 200).Label("plain")
			coreSliderBase(c, &b, 0, 100).Size(100, 20).Label("stepped").Step(5)
			coreStepper(c, &s, 0, 10, 0.5).Label("stepper")
			coreRangeSlider(c, &low, &high, 0, 100, 10).Label("price")
			coreRating(c, &stars, 5).Label("stars")
			coreBox(c).Size(80, 20).Focusable().Role(RoleSlider).Label("custom").Range(0, 1, 0.5).Step(0.25)
			coreProgress(c, 0.5).Label("progress")
		})
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	for _, c := range []struct {
		role  platform.AccessRole
		label string
		step  float64
	}{
		{platform.RoleSlider, "plain", 2},
		{platform.RoleSlider, "stepped", 5},
		{platform.RoleStepper, "stepper", 0.5},
		{platform.RoleSlider, "price minimum", 10},
		{platform.RoleSlider, "stars", 1},
		{platform.RoleSlider, "custom", 0.25},
		{platform.RoleProgress, "progress", 0},
	} {
		if n := accessNode(t, tree, c.role, c.label); n.Step != c.step {
			t.Errorf("%s: step %v, want %v", c.label, n.Step, c.step)
		}
	}
}

// GTK reads the clipboard in a nested event loop, which may draw a frame
// while the paste is being applied: the frame must not apply it again.
func TestPasteDuringFrame(t *testing.T) {
	text := ""
	tt := coreNewTester(func(c *context) {
		coreTextInputBase(c, &text).Width(100).Label("name")
	}, 200, 50)
	tt.Click("name")
	tt.SetClipboard("ab")
	tt.h.reading = tt.Frame
	tt.Command("paste")
	if text != "ab" {
		t.Fatalf("pasted %q", text)
	}
}

// A chat composer sends with Enter and breaks the line with Shift+Enter:
// OnTextKey sees the Enter before the editor takes it, and one taken
// changes nothing — a text area inserts no newline, a single-line input
// does not submit.
func TestOnTextKeyTakesEnterByModifier(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		text, submits, takes := "", 0, false
		sent, sends := "", 0
		tt := coreNewTester(func(c *context) {
			var in *node
			if multiline {
				in = coreTextAreaBase(c, &text)
			} else {
				in = coreTextInputBase(c, &text)
			}
			in.Width(100).Label("draft")
			in.OnTextKey(func(k TextKey) bool {
				if takes && k.Key == KeyEnter {
					sends++
					sent = k.Text()
					if !multiline {
						k.SetText("") // a composer clears the draft it sent
					}
					return true
				}
				return false
			})
			if in.Submitted() {
				submits++
			}
		}, 200, 50)
		tt.Click("draft")
		tt.Type("hello")

		// Not taken: the editor does what it always does.
		tt.Key(0, KeyEnter)
		tt.Frame()
		if multiline {
			if text != "hello\n" {
				t.Fatalf("multiline: text %q, want a newline", text)
			}
		} else if submits != 1 {
			t.Fatalf("single-line: submitted %d times, want 1", submits)
		}

		// Taken: a newline is not inserted, and nothing submits.
		takes = true
		tt.Key(Shift, KeyEnter)
		tt.Frame()
		if sends != 1 || sent != "hello" && sent != "hello\n" {
			t.Fatalf("taken Enter: sent %q %d times, want the draft once", sent, sends)
		}
		if multiline {
			if text != "hello\n" {
				t.Fatalf("taken Shift+Enter: text %q, want the newline kept", text)
			}
		} else {
			if submits != 1 {
				t.Fatalf("taken Enter: submitted %d times, want still 1", submits)
			}
			if text != "" {
				t.Fatalf("taken Enter: text %q, want the draft cleared", text)
			}
		}
		// The draft still edits normally afterwards.
		tt.Type("x")
		tt.Frame()
		if multiline {
			if text != "hello\nx" {
				t.Fatalf("multiline after take: text %q, want hello\nx", text)
			}
		} else if text != "x" {
			t.Fatalf("single-line after clear: text %q, want x", text)
		}
	}
}

// OnTextKey holds until the next frame, as ReadOnly does, and reports the
// modifiers held: Shift+Enter can break the line while Enter sends.
func TestOnTextKeySeesModifiersAndHolds(t *testing.T) {
	text := ""
	var seen []Modifiers
	armed := true
	tt := coreNewTester(func(c *context) {
		in := coreTextAreaBase(c, &text).Width(100).Label("draft")
		in.OnTextKey(func(k TextKey) bool {
			if armed && k.Key == KeyEnter {
				seen = append(seen, k.Mods)
				return true
			}
			return false
		})
	}, 200, 80)
	tt.Click("draft")

	tt.Key(0, KeyEnter)
	tt.Key(Shift, KeyEnter)
	tt.Frame()
	if len(seen) != 2 || seen[0] != 0 || seen[1] != Shift {
		t.Fatalf("saw mods %v, want [0 Shift]", seen)
	}
	if text != "" {
		t.Fatalf("taken Enters changed nothing, text %q", text)
	}

	// The next frame without OnTextKey goes back to the editor.
	armed = false
	tt.Key(0, KeyEnter)
	tt.Frame()
	if text != "\n" {
		t.Fatalf("after the hold lapsed: text %q, want a newline", text)
	}
}

// The usage OnTextKey documents, through the public API: fn runs as the
// key comes, between frames, where the Element the view kept is no longer
// valid, and sends and clears the draft through its TextKey.
func TestOnTextKeyComposer(t *testing.T) {
	for _, buffered := range []bool{false, true} {
		draft, buffer := "", NewTextBuffer("")
		var sent []string
		var kept TextKey
		tt := NewTester(func(c *Context) {
			var composer Element
			if buffered {
				composer = TextAreaBuffer(c, buffer)
			} else {
				composer = TextArea(c, &draft)
			}
			composer.Label("draft").OnTextKey(func(k TextKey) bool {
				if k.Key != KeyEnter || k.Mods != 0 {
					return false
				}
				sent = append(sent, k.Text())
				k.SetText("")
				kept = k
				return true
			})
		}, 200, 80)
		text := func() string {
			if buffered {
				return buffer.String()
			}
			return draft
		}
		tt.Click("draft")

		tt.Type("hello")
		tt.Key(Shift, KeyEnter) // breaks the line
		tt.Type("world")
		tt.Key(0, KeyEnter)
		if len(sent) != 1 || sent[0] != "hello\nworld" || text() != "" {
			t.Fatalf("buffered %v: sent %q, text %q; want the draft sent and cleared", buffered, sent, text())
		}

		// The text typed with the Enter before a frame, which the bound
		// value does not hold yet, is what fn reads.
		tt.rt.event(platform.SurfaceEvent{Kind: platform.TextInput, Text: "again"})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyEnter)})
		tt.Frame()
		if len(sent) != 2 || sent[1] != "again" || text() != "" {
			t.Fatalf("buffered %v: sent %q, text %q; want the typed text sent", buffered, sent, text())
		}

		// The input edits on after a clear, and a TextKey kept past fn
		// panics rather than change anything.
		tt.Type("next")
		for _, use := range []func(){func() { kept.SetText("stale") }, func() { kept.Text() }} {
			func() {
				defer func() {
					if recover() == nil {
						t.Fatalf("buffered %v: a kept TextKey did not panic", buffered)
					}
				}()
				use()
			}()
		}
		tt.Frame()
		if text() != "next" {
			t.Fatalf("buffered %v: text %q after a kept TextKey", buffered, text())
		}
		tt.rt.close()
	}
}

// OnTextKey sees the keys that type no text, Tab, Escape, the arrows and
// Cmd chords among them, and what it leaves goes on as it would without
// it: Tab moves the focus, Up moves the caret.
func TestOnTextKeySeesKeysThatTypeNoText(t *testing.T) {
	draft, other := "", ""
	var seen []TextKey
	take := map[Key]bool{}
	escapes := 0
	tt := coreNewTester(func(c *context) {
		col := coreColumn(c)
		col.Children(func() {
			in := coreTextInputBase(c, &draft).Width(100).Label("draft")
			in.OnTextKey(func(k TextKey) bool {
				seen = append(seen, TextKey{Key: k.Key, Mods: k.Mods})
				if k.Key == KeyUp {
					if take[KeyUp] && k.Text() == "" {
						k.SetText("last message") // Up edits the last message
						return true
					}
					return false
				}
				return take[k.Key]
			})
			coreTextInputBase(c, &other).Width(100).Label("other")
		})
		if col.Shortcut(0, KeyEscape) {
			escapes++
		}
	}, 200, 100)
	tt.Click("draft")
	tt.TypeKey(0, KeyA, "a")
	tt.TypeKey(Shift, KeyB, "B")
	if len(seen) != 0 {
		t.Fatalf("keys that type text reached fn: %v", seen)
	}

	// Taken, Tab stays in the input; left, it moves the focus.
	take[KeyTab] = true
	tt.Key(0, KeyTab)
	tt.Frame()
	if len(seen) != 1 || seen[0].Key != KeyTab || draft != "aB" {
		t.Fatalf("saw %v, draft %q", seen, draft)
	}
	tt.Type("c")
	tt.Frame()
	if draft != "aBc" {
		t.Fatalf("a taken Tab moved the focus: draft %q, other %q", draft, other)
	}
	take[KeyTab] = false
	tt.Key(0, KeyTab)
	tt.Type("x")
	tt.Frame()
	if other != "x" {
		t.Fatalf("a Tab left did not move the focus: draft %q, other %q", draft, other)
	}

	// A Cmd chord, and Escape, which a shortcut around the focus takes
	// first, as HandleInput's keys.
	tt.Click("draft")
	seen = nil
	tt.Key(Cmd, KeyEnter)
	tt.Key(0, KeyEscape)
	tt.Frame()
	if len(seen) != 1 || seen[0].Key != KeyEnter || seen[0].Mods != Cmd || escapes != 1 {
		t.Fatalf("saw %v and %d escapes, want Cmd+Enter and the shortcut's Escape", seen, escapes)
	}

	// Up in an empty input puts the last message back.
	take[KeyUp] = true
	tt.Key(0, KeyUp)
	tt.Frame()
	if draft != "aBc" {
		t.Fatalf("Up with a draft: %q", draft)
	}
	tt.Key(Cmd, KeyA)
	tt.Key(0, KeyBackspace)
	tt.Key(0, KeyUp)
	tt.Frame()
	if draft != "last message" {
		t.Fatalf("Up in an empty input: %q, want the last message", draft)
	}
}
