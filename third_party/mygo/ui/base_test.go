package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestBasesHaveNoLook builds every base on a white window: they draw
// nothing of their own.
func TestBasesHaveNoLook(t *testing.T) {
	var on, check bool
	var radio, tab int
	var value = 50.0
	var size, text = "M", "hello"
	open, menu, section := true, true, true
	mixed := CheckMixed
	tt := coreNewTester(func(c *context) {
		th := *c.Theme()
		th.Background = RGB(255, 255, 255)
		c.SetTheme(&th)
		coreColumn(c).Fill().Padding(10).Gap(10).AlignItems(Start).Children(func() {
			coreButtonBase(c).Size(60, 20)
			coreCheckboxBase(c, &check).Size(60, 20)
			coreCheckboxStateBase(c, &mixed).Size(60, 20)
			coreSwitchBase(c, &on).Size(60, 20)
			coreRadioBase(c, &radio, 1).Size(60, 20)
			coreSliderBase(c, &value, 0, 100).Size(60, 20)
			tabs := coreTabsBase(c, &tab, 2)
			tabs.List.Children(func() {
				tabs.Tab(0).Size(30, 20)
				tabs.Tab(1).Size(30, 20)
			})
			sel := coreSelectBase(c, &size)
			sel.Trigger.Size(60, 20)
			sel.Popup(func(panel *node) {
				sel.Item("S").Size(60, 20)
				sel.Item("M").Size(60, 20)
			})
			coreTextInputBase(c, &text).Width(60).TextColor(RGB(255, 255, 255))
			corePopoverBase(c, sel.Trigger, &open, func(panel *node) { coreBox(c).Size(30, 30) })
			acc := coreAccordionBase(c)
			acc.Root.Children(func() {
				item := acc.Item(&section)
				item.Trigger.Size(60, 20)
				item.Panel(func() { coreBox(c).Size(60, 20) })
			})
			m := coreMenuBase(c, &menu)
			m.Trigger.Size(60, 20)
			m.Popup(func(panel *node) {
				m.Item().Size(60, 20)
				m.CheckItem(true).Size(60, 20)
				sub := m.SubmenuItem().Size(60, 20)
				m.Submenu(sub, func(panel *node) { m.RadioItem(true).Size(60, 20) })
			})
		})
	}, 300, 400)
	img := tt.Image()
	white := color.RGBA{255, 255, 255, 255}
	for y := range img.Rect.Dy() {
		for x := range img.Rect.Dx() {
			if c := img.RGBAAt(x, y); c != white {
				t.Fatalf("a base drew %v at (%d, %d)", c, x, y)
			}
		}
	}
}

// TestDimDisabled: a disabled element shows at half its opacity, bases
// and the themed widgets alike, unless DimDisabled(false) leaves its look
// to the view, as a card that keeps the reason it is unavailable.
func TestDimDisabled(t *testing.T) {
	plan := "pro"
	var para *node
	img := coreRender(func(c *context) {
		th := *c.Theme()
		th.Background = RGB(255, 255, 255)
		c.SetTheme(&th)
		coreColumn(c).Fill().AlignItems(Start).Children(func() {
			coreRadioBase(c, &plan, "team").Size(20, 20).Background(RGB(0, 0, 0)).Disabled(true)
			coreRadioBase(c, &plan, "team").Size(20, 20).Background(RGB(0, 0, 0)).Disabled(true).DimDisabled(false)
			coreBox(c).Disabled(true).DimDisabled(false).DimDisabled(true).Children(func() {
				coreBox(c).Size(20, 20).Background(RGB(0, 0, 0))
			})
			para = coreRichText(c).Children(func() {
				coreText(c, "dim").TextColor(RGB(0, 0, 0)).Disabled(true)
				coreText(c, "kept").TextColor(RGB(0, 0, 0)).Disabled(true).DimDisabled(false)
			})
		})
	}, 100, 100, 1)
	for _, tc := range []struct {
		y, want int
	}{{10, 128}, {30, 0}, {50, 128}} {
		if got := int(img.RGBAAt(10, tc.y).R); got < tc.want-2 || got > tc.want+2 {
			t.Errorf("at y %d: %d, want %d", tc.y, got, tc.want)
		}
	}
	for _, s := range para.inlineSpans() {
		if dim := s.Color.A < 255; dim != (s.Text == "dim") {
			t.Errorf("the span %q has alpha %d", s.Text, s.Color.A)
		}
	}
}

func TestToggleBases(t *testing.T) {
	var check, on bool
	radio := "a"
	var changed []string
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			if coreCheckboxBase(c, &check).Size(20, 20).Label("check").Changed() {
				changed = append(changed, "check")
			}
			coreSwitchBase(c, &on).Size(20, 20).Label("switch")
			coreRadioBase(c, &radio, "a").Size(20, 20).Label("a")
			if coreRadioBase(c, &radio, "b").Size(20, 20).Label("b").Changed() {
				changed = append(changed, "b")
			}
		})
	}, 200, 200)
	tt.Click("check")
	tt.Click("switch")
	tt.Click("b")
	if !check || !on || radio != "b" {
		t.Fatalf("clicks: check %v, switch %v, radio %q", check, on, radio)
	}
	if len(changed) != 2 {
		t.Errorf("Changed reported %q", changed)
	}
	// Space toggles the one with the focus.
	tt.Key(0, KeyTab)
	if !tt.Focused("check") {
		t.Fatal("Tab does not focus the check box")
	}
	tt.Key(0, KeySpace)
	if check {
		t.Error("Space does not toggle the check box")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for label, role := range map[string]platform.AccessRole{"check": platform.RoleCheckBox, "switch": platform.RoleSwitch, "b": platform.RoleRadio} {
		n := accessNode(t, tt.h.access, role, label)
		on := n.States&platform.AccessChecked != 0
		if want := label != "check"; on != want {
			t.Errorf("%s is checked %v to assistive technology, want %v", label, on, want)
		}
	}
}

func TestSliderBaseMapsItsContentBox(t *testing.T) {
	value := 0.0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Padding(10).Children(func() {
			coreSliderBase(c, &value, 0, 100).Size(120, 20).PaddingX(10).Label("volume")
		})
	}, 200, 100)
	r, _ := tt.Find("volume")
	// The content box spans 20 DIPs to 120 of the window: 100 DIPs.
	for _, tc := range []struct{ x, want float32 }{{r.X + 10, 0}, {r.X + 60, 50}, {r.X + 110, 100}, {r.X + 2, 0}, {r.X + 118, 100}} {
		tt.Press(tc.x, r.Y+10)
		tt.Release(tc.x, r.Y+10)
		if float32(value) != tc.want {
			t.Errorf("pressing at %v sets %v, want %v", tc.x, value, tc.want)
		}
	}
	tt.Key(0, KeyHome)
	if value != 0 {
		t.Errorf("Home sets %v", value)
	}
	tt.Key(0, KeyRight)
	if value != 1 {
		t.Errorf("Right sets %v", value)
	}
	tt.Key(0, KeyEnd)
	if value != 100 {
		t.Errorf("End sets %v", value)
	}
}

func TestTabsBase(t *testing.T) {
	tab, changes := 0, 0
	tt := coreNewTester(func(c *context) {
		tabs := coreTabsBase(c, &tab, 3)
		tabs.List.Children(func() {
			for i, name := range []string{"One", "Two", "Three"} {
				tabs.Tab(i).Padding(4).Children(func() { coreText(c, name) })
			}
		})
		if tabs.List.Changed() {
			changes++
		}
	}, 300, 100)
	tt.Click("Three")
	if tab != 2 || changes != 1 {
		t.Fatalf("a click chooses tab %d, %d changes", tab, changes)
	}
	tt.Key(0, KeyRight)
	if tab != 0 || !tt.Focused("One") {
		t.Errorf("Right from the last tab chooses %d, focused %v", tab, tt.Focused("One"))
	}
	tt.Key(0, KeyEnd)
	if tab != 2 || !tt.Focused("Three") {
		t.Errorf("End chooses %d", tab)
	}
}

// TestTabsBaseDisabledTabs checks that a disabled tab is out of the keys'
// way and of the pointer's, as Radix's disabled tab triggers, and that
// assistive technology sees it disabled.
func TestTabsBaseDisabledTabs(t *testing.T) {
	tab, changes := 0, 0
	off := map[int]bool{1: true, 3: true}
	names := []string{"One", "Two", "Three", "Four"}
	tt := coreNewTester(func(c *context) {
		tabs := coreTabsBase(c, &tab, len(names))
		tabs.List.Children(func() {
			for i, name := range names {
				tabs.Tab(i).Padding(4).Disabled(off[i]).Children(func() { coreText(c, name) })
			}
		})
		if tabs.List.Changed() {
			changes++
		}
	}, 300, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("One")
	tt.Key(0, KeyRight)
	if tab != 2 || !tt.Focused("Three") {
		t.Errorf("Right from One chooses %d, not Three past the disabled Two", tab)
	}
	tt.Key(0, KeyRight)
	if tab != 0 || !tt.Focused("One") {
		t.Errorf("Right from Three chooses %d, not One round past the disabled Four", tab)
	}
	tt.Key(0, KeyLeft)
	if tab != 2 || !tt.Focused("Three") {
		t.Errorf("Left from One chooses %d, not Three round past the disabled Four", tab)
	}
	tt.Key(0, KeyEnd)
	if tab != 2 {
		t.Errorf("End chooses %d, not Three before the disabled Four", tab)
	}
	tt.Key(0, KeyHome)
	if tab != 0 || !tt.Focused("One") {
		t.Errorf("Home chooses %d", tab)
	}
	tt.Click("Two")
	if tab != 0 || changes != 4 {
		t.Errorf("a click on the disabled Two chooses %d, %d changes", tab, changes)
	}
	if n := accessNode(t, tt.h.access, platform.RoleTab, "Two"); n.States&platform.AccessDisabled == 0 {
		t.Errorf("the disabled tab: %+v", n)
	}
	if n := accessNode(t, tt.h.access, platform.RoleTab, "One"); n.States&platform.AccessDisabled != 0 || n.States&platform.AccessChecked == 0 {
		t.Errorf("the tab chosen: %+v", n)
	}
	// With the others disabled, the keys stay on the tab chosen.
	off = map[int]bool{1: true, 2: true, 3: true}
	tt.Click("One")
	tt.Key(0, KeyRight)
	tt.Key(0, KeyEnd)
	if tab != 0 || !tt.Focused("One") {
		t.Errorf("with the others disabled, the keys choose %d", tab)
	}
}

// TestTabsBaseDisabledTabPublic is the same through the public API, the
// tabs disabled after their constructor returned.
func TestTabsBaseDisabledTabPublic(t *testing.T) {
	tab := 0
	tt := NewTester(func(c *Context) {
		tabs := TabsBase(c, &tab, 3)
		tabs.List.Children(func() {
			for i, name := range []string{"One", "Two", "Three"} {
				tb := tabs.Tab(i).Padding(4)
				if i == 1 {
					tb.Disabled(true)
				}
				tb.Children(func() { Text(c, name) })
			}
		})
	}, 300, 100)
	if err := tt.Click("One"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, KeyRight)
	if tab != 2 {
		t.Errorf("Right from One chooses %d, not Three past the disabled Two", tab)
	}
}

func TestSelectBase(t *testing.T) {
	size := "M"
	sizes := []string{"S", "M", "L", "XL"}
	highlighted := map[string]bool{}
	tt := coreNewTester(func(c *context) {
		sel := coreSelectBase(c, &size)
		sel.Trigger.Label("size").Padding(4).Children(func() { coreText(c, "Size: "+size) })
		clear(highlighted)
		sel.Popup(func(panel *node) {
			panel.Label("popup")
			for _, s := range sizes {
				item := sel.Item(s).Padding(4).Label("item " + s)
				if item.Highlighted() {
					highlighted[s] = true
				}
				item.Children(func() { coreText(c, s) })
			}
		})
	}, 300, 300)
	if tt.HasText("XL") {
		t.Fatal("the popup shows before the trigger is clicked")
	}
	tt.Click("size")
	if !tt.HasText("XL") || !highlighted["M"] || len(highlighted) != 1 {
		t.Fatalf("the open popup highlights %v, not the choice", highlighted)
	}
	// The arrows move the highlight, Enter chooses it.
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	if !highlighted["XL"] || len(highlighted) != 1 {
		t.Errorf("Down twice highlights %v", highlighted)
	}
	tt.Key(0, KeyUp)
	tt.Key(0, KeyEnter)
	if size != "L" || tt.HasText("XL") {
		t.Errorf("Enter chooses %q and leaves the popup open: %v", size, tt.HasText("XL"))
	}
	// Down opens the popup from the focused trigger, Escape closes it.
	tt.Key(0, KeyDown)
	if !tt.HasText("XL") {
		t.Fatal("Down does not open the popup")
	}
	tt.Key(0, KeyEscape)
	if tt.HasText("XL") {
		t.Error("Escape does not close the popup")
	}
	// The pointer highlights what it moves onto, and a click chooses.
	tt.Click("size")
	r, _ := tt.Find("item S")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if !highlighted["S"] || len(highlighted) != 1 {
		t.Errorf("the pointer highlights %v", highlighted)
	}
	// While it rests there, the arrows still move the highlight.
	tt.Key(0, KeyDown)
	if !highlighted["M"] {
		t.Errorf("Down under a resting pointer highlights %v", highlighted)
	}
	tt.Click("item XL")
	if size != "XL" || tt.HasText("item S") {
		t.Errorf("a click chooses %q", size)
	}
}

// TestSelectBaseDisabledItems checks that a disabled option is out of the
// keys' way and of the pointer's, as a listbox's aria-disabled options.
func TestSelectBaseDisabledItems(t *testing.T) {
	sel := "a"
	off := map[string]bool{"b": true, "d": true}
	var highlighted, jump string
	tt := coreNewTester(func(c *context) {
		s := coreSelectBase(c, &sel)
		s.Trigger.Label("trigger").Padding(4).Children(func() { coreText(c, "Choice: "+sel) })
		if jump != "" {
			s.Highlight(jump)
			jump = ""
		}
		highlighted = ""
		s.Popup(func(panel *node) {
			for _, v := range []string{"a", "b", "c", "d"} {
				it := s.Item(v).Padding(4).Label("opt-" + v)
				it.Disabled(off[v])
				if it.Highlighted() {
					highlighted = v
				}
				it.Children(func() { coreText(c, v) })
			}
		})
	}, 300, 300)
	tt.Click("trigger")
	tt.Key(0, KeyDown)
	if highlighted != "c" {
		t.Errorf("Down from a highlights %q, not c past the disabled b", highlighted)
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnd)
	if highlighted != "c" {
		t.Errorf("Down and End onto the disabled d highlight %q", highlighted)
	}
	tt.Key(0, KeyUp)
	if highlighted != "a" {
		t.Errorf("Up from c highlights %q, not a past the disabled b", highlighted)
	}
	tt.Key(0, KeyEnd)
	tt.Key(0, KeyHome)
	if highlighted != "a" {
		t.Errorf("Home highlights %q", highlighted)
	}
	jump = "d"
	tt.Frame()
	if highlighted != "a" {
		t.Errorf("Highlight of the disabled d highlights %q", highlighted)
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if sel != "c" {
		t.Errorf("Down and Enter choose %q, not c", sel)
	}
	// The pointer neither highlights nor chooses a disabled option.
	tt.Click("trigger")
	r, _ := tt.Find("opt-b")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if highlighted != "c" {
		t.Errorf("the pointer on the disabled b highlights %q", highlighted)
	}
	tt.Click("opt-b")
	if sel != "c" || !tt.HasText("d") {
		t.Errorf("a click on the disabled b chooses %q, popup open %v", sel, tt.HasText("d"))
	}
	tt.Key(0, KeyEscape)
	// A disabled choice leaves the highlight on the first option that
	// can be chosen.
	sel = "b"
	tt.Click("trigger")
	if highlighted != "a" {
		t.Errorf("the popup of the disabled choice b highlights %q, not a", highlighted)
	}
	tt.Key(0, KeyEscape)
	// With every option disabled, nothing is highlighted or chosen.
	off = map[string]bool{"a": true, "b": true, "c": true, "d": true}
	tt.Click("trigger")
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if highlighted != "" || sel != "b" {
		t.Errorf("with every option disabled, %q is highlighted and %q chosen", highlighted, sel)
	}
}

// TestSelectBaseDisabledItemPublic is the same through the public API,
// where the item is disabled after its constructor returned.
func TestSelectBaseDisabledItemPublic(t *testing.T) {
	sel := "a"
	tt := NewTester(func(c *Context) {
		s := SelectBase(c, &sel)
		s.Trigger.Label("trigger").Children(func() { Text(c, sel) })
		s.Popup(func(panel Element) {
			for _, v := range []string{"a", "b", "c"} {
				it := s.Item(v).Label("opt-" + v)
				if v == "b" {
					it.Disabled(true)
				}
				it.Children(func() { Text(c, v) })
			}
		})
	}, 300, 300)
	if err := tt.Click("trigger"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if sel != "c" {
		t.Errorf("Down and Enter choose %q, not c past the disabled b", sel)
	}
}

func TestDialogAndPopoverBases(t *testing.T) {
	dialog, popover := true, false
	var back *node
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 20).Label("anchor")
		if b.Clicked() {
			popover = true
		}
		corePopoverBase(c, b, &popover, func(panel *node) {
			panel.Label("popover").Size(50, 50)
		})
		coreDialogBase(c, &dialog, func(backdrop, panel *node) {
			back = backdrop
			backdrop.Background(RGBA(0, 0, 0, 0.5))
			panel.Label("dialog").Size(100, 60)
		})
	}, 400, 300)
	if back == nil {
		t.Fatal("DialogBase does not pass its backdrop")
	}
	r, ok := tt.Find("dialog")
	if !ok || r.X != 150 || r.Y != 120 {
		t.Errorf("the dialog is at %v, not centered", r)
	}
	tt.ClickAt(10, 290)
	if dialog {
		t.Error("a click on the backdrop does not close the dialog")
	}
	tt.Click("anchor")
	p, ok := tt.Find("popover")
	if !ok || p.Y != 20 {
		t.Errorf("the popover is at %v, not under its anchor", p)
	}
	tt.Key(0, KeyEscape)
	if popover {
		t.Error("Escape does not close the popover")
	}
}

// Dismissed reports Escape or a click outside closing a dialog or a
// popover, which setting *open back keeps open, and not the dialog's own
// buttons closing it.
func TestDismissed(t *testing.T) {
	dialog, popover, dismissed := true, false, 0
	popoverDismissed := false
	// Elements stand for nil while closed.
	dismissedNow := func(n *node) bool { return n != nil && n.Dismissed() }
	tt := coreNewTester(func(c *context) {
		b := coreButton(c, "anchor")
		if b.Clicked() {
			popover = true
		}
		if dismissedNow(corePopover(c, b, &popover, func() { coreText(c, "popover") })) {
			popoverDismissed = true
		}
		if dismissedNow(coreModal(c, &dialog, func() {
			if coreButton(c, "Close").Clicked() {
				dialog = false
			}
		})) {
			dismissed++
			dialog = true
		}
	}, 400, 300)
	tt.Key(0, KeyEscape)
	tt.Frame()
	if !dialog || dismissed != 1 {
		t.Fatalf("Escape: open %v, dismissed %d times", dialog, dismissed)
	}
	tt.ClickAt(5, 295)
	tt.Frame()
	if !dialog || dismissed != 2 {
		t.Fatalf("a click outside: open %v, dismissed %d times", dialog, dismissed)
	}
	tt.Click("Close")
	tt.Frame()
	if dialog || dismissed != 2 {
		t.Fatalf("Close: open %v, dismissed %d times", dialog, dismissed)
	}
	tt.Click("anchor")
	tt.Frame()
	if !popover {
		t.Fatal("the popover does not open")
	}
	tt.Key(0, KeyEscape)
	if popover || !popoverDismissed {
		t.Errorf("Escape: the popover open %v, dismissed %v", popover, popoverDismissed)
	}
}

func TestTextInputBase(t *testing.T) {
	text := ""
	var in *node
	tt := coreNewTester(func(c *context) {
		in = coreTextInputBase(c, &text).Width(100).Label("name").AutoFocus()
	}, 200, 100)
	tt.Type("Ada")
	if text != "Ada" {
		t.Errorf("typed %q", text)
	}
	if in.pad != [4]float32{} || in.border != [4]float32{} || in.bg.A != 0 {
		t.Errorf("the base has a look: padding %v, border %v, background %v", in.pad, in.border, in.bg)
	}
}

func TestFocusRing(t *testing.T) {
	ringed := func(show bool) bool {
		tt := coreNewTester(func(c *context) {
			th := *c.Theme()
			th.Background = RGB(255, 255, 255)
			c.SetTheme(&th)
			coreColumn(c).Padding(20).AlignItems(Start).Children(func() {
				coreButtonBase(c).Size(40, 20).Label("b").FocusRing(show)
			})
		}, 100, 60)
		tt.Key(0, KeyTab)
		img := tt.Image()
		// Just outside the button's box.
		return img.RGBAAt(18, 30) != color.RGBA{255, 255, 255, 255}
	}
	if !ringed(true) {
		t.Error("no ring around the focused button")
	}
	if ringed(false) {
		t.Error("a ring around a button that turned it off")
	}
}

// TestPopupsOpenWhereTheyFit opens popovers where there is no room below
// or to the right of their anchors: the first frame that paints them has
// them where they fit, instead of moving them once it knows their size.
func TestPopupsOpenWhereTheyFit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		justify Align
		align   Align
	}{{"bottom right", End, End}, {"top left", Start, Start}} {
		open := false
		var first *Rect
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Padding(10).Justify(tc.justify).AlignItems(tc.align).Children(func() {
				b := coreButtonBase(c).Size(80, 24).Label("anchor")
				if b.Clicked() {
					open = true
				}
				corePopoverBase(c, b, &open, func(panel *node) {
					panel.Size(160, 120).Margin(4, 0, 0, 0).Draw(func(p *Painter, r Rect) {
						if first == nil {
							first = &r
						}
					})
				})
			})
		}, 300, 300)
		tt.Click("anchor")
		a, _ := tt.Find("anchor")
		if first == nil {
			t.Fatalf("%s: the popover was not painted", tc.name)
		}
		r := *first
		if r.X < 4 || r.X+r.W > 296 {
			t.Errorf("%s: the popover first shows at %v, out of the window", tc.name, r)
		}
		if tc.justify == End && r.Y+r.H != a.Y-4 {
			t.Errorf("%s: the popover first shows at %v, not 4 DIPs above its anchor at %v", tc.name, r, a)
		}
		if tc.justify == Start && (r.Y != a.Y+a.H+4 || r.X != a.X) {
			t.Errorf("%s: the popover first shows at %v, not 4 DIPs below its anchor at %v", tc.name, r, a)
		}
	}
}

func TestEnterWithWindowShortcut(t *testing.T) {
	window, button, checked := 0, 0, false
	tt := coreNewTester(func(c *context) {
		if c.Shortcut(0, KeyEnter) {
			window++
		}
		coreColumn(c).Children(func() {
			if coreButton(c, "Other").Clicked() {
				button++
			}
			coreCheckbox(c, &checked, "Agree")
		})
	}, 300, 200)
	tt.Key(0, KeyEnter)
	if window != 1 || button != 0 {
		t.Fatalf("Enter without a focus: window %d, button %d", window, button)
	}
	// A focused button takes Enter before the window.
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnter)
	if window != 1 || button != 1 {
		t.Fatalf("Enter on a focused button: window %d, button %d", window, button)
	}
	// A focused check box leaves Enter to the window and takes Space.
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnter)
	if window != 2 || checked {
		t.Fatalf("Enter on a focused check box: window %d, checked %v", window, checked)
	}
	tt.Key(0, KeySpace)
	if window != 2 || !checked {
		t.Fatalf("Space on a focused check box: window %d, checked %v", window, checked)
	}
}

// A focused link takes Enter alone, as in a web page (WAI-ARIA's link
// pattern): Space scrolls the container around it, as elsewhere, and a
// button still takes both.
func TestLinkTakesEnterAlone(t *testing.T) {
	links, roleLinks, buttons := 0, 0, 0
	var scroll ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Size(300, 100).TrackScroll(&scroll).Children(func() {
			if coreLink(c, "Docs", "").AutoFocus().Clicked() {
				links++
			}
			b := coreText(c, "Next").Focusable().Role(RoleLink)
			b.flags |= flagClickable
			if b.Clicked() {
				roleLinks++
			}
			if coreButton(c, "Save").Clicked() {
				buttons++
			}
			coreBox(c).Size(300, 1000)
		})
	}, 300, 100)
	tt.Frame()
	if !tt.Focused("Docs") {
		t.Fatal("the link is not focused")
	}
	tt.Key(0, KeySpace)
	tt.Frame()
	tt.Frame()
	if links != 0 {
		t.Fatalf("Space on a focused link made %d clicks", links)
	}
	if scroll.Y == 0 {
		t.Error("Space on a focused link does not scroll its container")
	}
	tt.Key(0, KeyEnter)
	tt.Frame()
	if links != 1 {
		t.Fatalf("Enter on a focused link made %d clicks, want 1", links)
	}

	// The same for an element with the role of a link.
	tt.Key(0, KeyTab)
	if !tt.Focused("Next") {
		t.Fatal("Tab does not focus the element with the role of a link")
	}
	tt.Key(0, KeySpace)
	tt.Key(0, KeyEnter)
	if roleLinks != 1 {
		t.Fatalf("Space and Enter on a focused element with the role of a link made %d clicks, want 1", roleLinks)
	}

	tt.Key(0, KeyTab)
	if !tt.Focused("Save") {
		t.Fatal("Tab does not focus the button")
	}
	tt.Key(0, KeySpace)
	tt.Key(0, KeyEnter)
	if buttons != 2 {
		t.Fatalf("Space and Enter on a focused button made %d clicks, want 2", buttons)
	}
}
