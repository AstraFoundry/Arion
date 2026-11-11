package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"maps"
	"runtime"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestContextMenu(t *testing.T) {
	var (
		chosen       []string
		pinned       bool
		locked       = true
		rightClicked int
	)
	view := func(c *context) {
		coreColumn(c).Fill().Padding(20).Gap(10).Children(func() {
			row := coreRow(c).Padding(8).Children(func() {
				coreText(c, "Report.pdf")
			})
			row.ContextMenu(func(m *Menu) {
				if m.Item("Open").Shortcut(Cmd, KeyO).Chosen() {
					chosen = append(chosen, "Open")
				}
				if m.Item("Pinned").Checked(pinned).Chosen() {
					pinned = !pinned
				}
				m.Separator()
				m.Submenu("Move to", func(m *Menu) {
					for _, folder := range []string{"Desktop", "Documents"} {
						if m.Item(folder).Chosen() {
							chosen = append(chosen, "Move to "+folder)
						}
					}
				})
				if m.Item("Delete").Disabled(locked).Chosen() {
					chosen = append(chosen, "Delete")
				}
			})
			// Without a context menu, a right-click is the box's.
			if coreBox(c).Size(60, 30).Label("plain").RightClicked() {
				rightClicked++
			}
			coreText(c, map[bool]string{false: "loose", true: "pinned"}[pinned])
		})
	}
	tt := coreNewTester(view, 400, 300)
	if tt.Menu() != nil {
		t.Fatalf("a menu shows before any click: %q", tt.Menu())
	}
	if err := tt.RightClick("Report.pdf"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"Open", "Pinned", "-", "Move to", "Delete"}; !slices.Equal(got, want) {
		t.Fatalf("menu %q, want %q", got, want)
	}
	items := tt.h.menu.Items
	if items[0].Accelerator != accelerator(Cmd, KeyO) || items[0].Accelerator == "" {
		t.Errorf("Open shows the shortcut %q", items[0].Accelerator)
	}
	if items[1].Type != platform.MenuItemCheckbox || items[1].Checked {
		t.Errorf("Pinned is %+v, an unchecked check box item", items[1])
	}
	if items[4].Enabled {
		t.Error("Delete is enabled while locked")
	}
	if err := tt.ChooseMenuItem("Delete"); err == nil {
		t.Error("chose a disabled item")
	}
	if err := tt.ChooseMenuItem("Pinned"); err != nil {
		t.Fatal(err)
	}
	if tt.Menu() != nil {
		t.Error("the menu stays after a choice")
	}
	if !pinned || !tt.HasText("pinned") {
		t.Errorf("choosing Pinned left pinned %v, texts %q", pinned, tt.Texts())
	}

	tt.RightClick("Report.pdf")
	if !tt.h.menu.Items[1].Checked {
		t.Error("Pinned has no check mark once pinned")
	}
	if err := tt.ChooseMenuItem("Move to", "Documents"); err != nil {
		t.Fatal(err)
	}
	locked = false
	tt.Frame()
	tt.RightClick("Report.pdf")
	if err := tt.ChooseMenuItem("Delete"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Move to Documents", "Delete"}; !slices.Equal(chosen, want) {
		t.Errorf("chosen %q, want %q", chosen, want)
	}

	// A menu closed without a choice reports nothing.
	tt.RightClick("Report.pdf")
	tt.CloseMenu()
	tt.Frame()
	if len(chosen) != 2 {
		t.Errorf("closing the menu chose %q", chosen[2:])
	}

	tt.RightClick("plain")
	if tt.Menu() != nil || rightClicked != 1 {
		t.Errorf("a right-click on an element without a menu: menu %q, RightClicked %d times", tt.Menu(), rightClicked)
	}
}

func TestInnermostContextMenu(t *testing.T) {
	var got string
	view := func(c *context) {
		card := coreColumn(c).Padding(20).Children(func() {
			coreText(c, "Card")
			coreButton(c, "Share").ContextMenu(func(m *Menu) {
				if m.Item("Copy Link").Chosen() {
					got = "Copy Link"
				}
			})
		})
		card.ContextMenu(func(m *Menu) {
			if m.Item("Remove Card").Chosen() {
				got = "Remove Card"
			}
		})
	}
	tt := coreNewTester(view, 400, 300)
	tt.RightClick("Share")
	if m := tt.Menu(); !slices.Equal(m, []string{"Copy Link"}) {
		t.Fatalf("the button's menu is %q", m)
	}
	tt.CloseMenu()
	tt.RightClick("Card")
	if err := tt.ChooseMenuItem("Remove Card"); err != nil {
		t.Fatal(err)
	}
	if got != "Remove Card" {
		t.Errorf("chose %q", got)
	}
}

func TestContextMenuFromTheKeyboard(t *testing.T) {
	var chosen int
	view := func(c *context) {
		coreColumn(c).Padding(20).Gap(8).Children(func() {
			coreButton(c, "First")
			coreButton(c, "Second").ContextMenu(func(m *Menu) {
				if m.Item("Duplicate").Chosen() {
					chosen++
				}
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	tt.Key(0, KeyTab)
	tt.Key(0, KeyContextMenu)
	if tt.Menu() != nil {
		t.Fatalf("the menu key opened %q over a button without a menu", tt.Menu())
	}
	tt.Key(0, KeyTab)
	tt.Key(0, KeyContextMenu)
	if !slices.Equal(tt.Menu(), []string{"Duplicate"}) {
		t.Fatalf("the menu key opened %q", tt.Menu())
	}
	tt.CloseMenu()
	tt.Key(Shift, KeyF10)
	if err := tt.ChooseMenuItem("Duplicate"); err != nil {
		t.Fatalf("Shift+F10: %v", err)
	}
	if chosen != 1 {
		t.Errorf("chosen %d times", chosen)
	}
}

func TestTextInputContextMenu(t *testing.T) {
	value := "hello world"
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			coreTextInput(c, &value).Label("Name")
		})
	}, 400, 200)
	r, ok := tt.Find("Name")
	if !ok {
		t.Fatal("no text input")
	}
	// A right-click focuses the input and puts the caret where it is.
	tt.RightClickAt(r.X+r.W-4, r.Y+r.H/2)
	if !tt.Focused("Name") {
		t.Error("the right-click left the input without the focus")
	}
	menu := tt.Menu()
	for _, label := range []string{"Cut", "Copy", "Paste", "Select All"} {
		if !slices.Contains(menu, label) {
			t.Fatalf("the menu %q has no %s", menu, label)
		}
	}
	if err := tt.ChooseMenuItem("Copy"); err == nil {
		t.Error("copied without a selection")
	}
	if err := tt.ChooseMenuItem("Select All"); err != nil {
		t.Fatal(err)
	}
	// In the selection, a right-click keeps it.
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
	if err := tt.ChooseMenuItem("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "hello world" {
		t.Errorf("copied %q", tt.Clipboard())
	}
	tt.SetClipboard("bye")
	tt.Key(0, KeyContextMenu)
	if err := tt.ChooseMenuItem("Paste"); err != nil {
		t.Fatal(err)
	}
	if value != "bye" {
		t.Errorf("pasting over the selection left %q", value)
	}
}

func TestAccelerator(t *testing.T) {
	for _, c := range []struct {
		mods Modifiers
		key  Key
		want string
	}{
		{Super, KeyC, "Super+c"},
		{Ctrl | Shift, KeyZ, "Ctrl+Shift+z"},
		{Alt, KeyF4, "Alt+F4"},
		{0, KeyDelete, "Delete"},
		{Ctrl, KeyComma, "Ctrl+,"},
		{Ctrl, KeyContextMenu, ""},
	} {
		if got := accelerator(c.mods, c.key); got != c.want {
			t.Errorf("accelerator(%v, %v) = %q, want %q", c.mods, c.key, got, c.want)
		}
	}
}

// sortView is a menu button choosing an order, with a button after it.
type sortView struct {
	sort     string
	disabled bool
}

func (s *sortView) view(c *context) {
	coreColumn(c).Padding(20).Gap(10).AlignItems(Start).Children(func() {
		coreMenuButton(c, "Sort by", func(m *Menu) {
			for _, by := range []string{"Name", "Date", "Size"} {
				if m.Item(by).Checked(s.sort == by).Chosen() {
					s.sort = by
				}
			}
		}).Disabled(s.disabled)
		coreButton(c, "Other")
	})
}

func TestMenuButton(t *testing.T) {
	s := &sortView{sort: "Name"}
	tt := coreNewTester(s.view, 400, 300)
	// The menu opens as the button goes down, below it.
	r, _ := tt.Find("Sort by")
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	if got, want := tt.Menu(), []string{"Name", "Date", "Size"}; !slices.Equal(got, want) {
		t.Fatalf("menu %q, want %q", got, want)
	}
	if at := tt.h.menuAt; at[0] > r.X || at[1] < r.Y+r.H || at[1] > r.Y+r.H+20 {
		t.Errorf("the menu shows at %v, the label being at %v", at, r)
	}
	if !tt.h.menu.Items[0].Checked {
		t.Error("Name is not checked")
	}
	tt.Release(r.X+r.W/2, r.Y+r.H/2)
	if err := tt.ChooseMenuItem("Date"); err != nil {
		t.Fatal(err)
	}
	if s.sort != "Date" || !tt.Focused("Sort by") {
		t.Fatalf("chose %q; focused %v", s.sort, tt.Focused("Sort by"))
	}
	// The keys open it while it has the focus.
	for _, k := range []Key{KeyEnter, KeySpace, KeyDown} {
		tt.CloseMenu()
		tt.Key(0, k)
		if tt.Menu() == nil {
			t.Errorf("%v does not open the menu", k)
		}
	}
	tt.CloseMenu()
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnter)
	if tt.Menu() != nil {
		t.Error("Enter on the next button opens the menu")
	}
	s.disabled = true
	tt.Frame()
	tt.Click("Sort by")
	if tt.Menu() != nil {
		t.Error("a disabled menu button opens its menu")
	}
}

func TestMenuButtonAccessibility(t *testing.T) {
	s := &sortView{sort: "Name"}
	tt := coreNewTester(s.view, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Sort by")
	if n.Actions&platform.ActionPress == 0 || n.States&platform.AccessFocusable == 0 {
		t.Errorf("the menu button: %+v", n)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
	if tt.Menu() == nil {
		t.Error("pressing it does not open the menu")
	}
}

func TestMenuAndContextMenuOfOneElement(t *testing.T) {
	var chosen []string
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 40).Label("Both").Menu(func(m *Menu) {
			if m.Item("From the menu").Chosen() {
				chosen = append(chosen, "menu")
			}
		}).ContextMenu(func(m *Menu) {
			if m.Item("From the context menu").Chosen() {
				chosen = append(chosen, "context")
			}
		})
	}, 300, 200)
	tt.Click("Both")
	if got := tt.Menu(); !slices.Equal(got, []string{"From the menu"}) {
		t.Fatalf("a click shows %q", got)
	}
	tt.ChooseMenuItem("From the menu")
	tt.RightClick("Both")
	if got := tt.Menu(); !slices.Equal(got, []string{"From the context menu"}) {
		t.Fatalf("a right-click shows %q", got)
	}
	tt.ChooseMenuItem("From the context menu")
	if !slices.Equal(chosen, []string{"menu", "context"}) {
		t.Errorf("chose %q", chosen)
	}
}

// TestMenuButtonInAToolbarsOverflow opens the menu of a menu button that
// does not fit in its toolbar, from the toolbar's overflow menu, below
// the overflow button.
func TestMenuButtonInAToolbarsOverflow(t *testing.T) {
	chosen := ""
	tt := coreNewTester(func(c *context) {
		coreToolbar(c, func() {
			for _, n := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
				coreButton(c, n)
			}
			coreMenuButton(c, "Export", func(m *Menu) {
				for _, as := range []string{"PDF", "PNG"} {
					if m.Item("As " + as).Chosen() {
						chosen = as
					}
				}
			})
		}).Fill()
	}, 260, 300)
	if tt.HasText("Export") {
		t.Fatal("the menu button fits")
	}
	tt.Click("More")
	if err := tt.ChooseMenuItem("Export"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"As PDF", "As PNG"}; !slices.Equal(got, want) {
		t.Fatalf("choosing Export shows %q, want %q", got, want)
	}
	more, _ := tt.Find("More")
	if at := tt.h.menuAt; at[0] != more.X || at[1] != more.Y+more.H {
		t.Errorf("the menu shows at %v, not below the overflow button at %+v", at, more)
	}
	if err := tt.ChooseMenuItem("As PNG"); err != nil || chosen != "PNG" {
		t.Errorf("chose As PNG: %v, chosen %q", err, chosen)
	}
}

func TestEditMenuEnabled(t *testing.T) {
	value, notes, pin := "", "read only", "1234"
	disabled := false
	tt := NewTester(func(c *Context) {
		c.OnShortcut(Cmd, KeyV, func() {})
		TextInput(c, &value).Label("field").Disabled(disabled)
		TextInput(c, &notes).Label("notes").ReadOnly(true)
		TextInput(c, &pin).Label("pin").ReadOnly(true).Password()
		Box(c).Size(100, 20).Focusable().Label("list").OnShortcut(Cmd, KeyC, func() {})
		Box(c).Size(100, 20).Focusable().Label("terminal").HandleInput(func(InputEvent) bool { return false })
	}, 300, 200)
	mac := runtime.GOOS == "darwin"
	check := func(where string, want map[string]bool) {
		t.Helper()
		got := map[string]bool{}
		tt.rt.focusRead = false
		for name := range want {
			got[name] = tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceCommandEnabled, Text: name})
		}
		if !maps.Equal(got, want) {
			t.Errorf("%s: enabled %v, want %v", where, got, want)
		}
		if tt.rt.focusRead {
			t.Errorf("%s: asking read the window's focus, so its next change draws a frame", where)
		}
	}

	check("no focus, with the window's Cmd+V", map[string]bool{"copy": false, "paste": mac, "selectAll": false})

	tt.Click("field")
	check("an empty text input", map[string]bool{"undo": false, "redo": false, "cut": false, "copy": false, "paste": true, "pasteAndMatchStyle": true, "delete": false, "selectAll": false})
	tt.Type("hi")
	tt.Key(Cmd, KeyA)
	check("a text input with its text selected", map[string]bool{"undo": true, "redo": false, "cut": true, "copy": true, "delete": true, "selectAll": true})
	tt.Command("undo")
	check("a text input after undo", map[string]bool{"undo": false, "redo": true})

	tt.Click("notes")
	// A menu asks before the next frame applies the command.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceCommand, Text: "selectAll"})
	check("a read-only text input with Select All queued", map[string]bool{"copy": true, "cut": false})
	tt.Click("pin")
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceCommand, Text: "selectAll"})
	check("a read-only password input with Select All queued", map[string]bool{"copy": false, "cut": false})
	tt.Click("notes")
	tt.Key(Cmd, KeyA)
	check("a read-only text input", map[string]bool{"copy": true, "selectAll": true, "cut": false, "paste": mac, "delete": false})

	tt.Click("list")
	check("a list copying with Cmd+C", map[string]bool{"copy": mac, "paste": mac, "selectAll": false})

	tt.Click("terminal")
	check("an element taking its input", map[string]bool{"copy": true, "paste": true})

	tt.Click("field")
	tt.Type("hi")
	tt.Key(Cmd, KeyA)
	disabled = true
	tt.Frame()
	if !tt.Focused("field") {
		t.Fatal("the text input lost the focus as it was disabled")
	}
	check("a text input disabled with the focus", map[string]bool{"copy": false, "paste": false, "selectAll": false})
}

func TestEditMenuEnabledSelectable(t *testing.T) {
	disabled := false
	tt := NewTester(func(c *Context) {
		Column(c).Selectable().Gap(8).Children(func() {
			Text(c, "First").Disabled(disabled)
			Text(c, "Last")
			Text(c, "").Width(150).Height(25).Label("empty")
		})
		Column(c).Selectable().Children(func() {
			Text(c, "").Width(150).Height(25).Label("blank")
			Text(c, "").Width(150).Height(25)
		})
		Column(c).Selectable().Children(func() {
			Text(c, "").Width(150).Height(25).Label("alone")
		})
	}, 200, 300)
	enabled := func() bool {
		return tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceCommandEnabled, Text: "selectAll"})
	}
	tt.Click("empty")
	if !enabled() {
		t.Error("an empty paragraph: Select All disabled, want enabled for the container's text")
	}
	tt.Click("blank")
	if !enabled() {
		t.Error("empty paragraphs: Select All disabled, want enabled for the newline between them")
	}
	tt.Click("alone")
	if enabled() {
		t.Error("a lone empty paragraph: Select All enabled, want disabled as there is nothing to select")
	}
	selectBetween(t, tt, "First", 0, "Last", 4)
	disabled = true
	tt.Frame()
	if enabled() {
		t.Error("the focused paragraph disabled: Select All enabled, want disabled as it takes no command")
	}
}

// testMenuIcon is what mygo.NewMenuIcon makes, which this package cannot
// import.
type testMenuIcon struct{ p *platform.MenuIcon }

func (i testMenuIcon) PlatformMenuIcon() *platform.MenuIcon { return i.p }

// menuIconPNG returns a PNG image of 32 by 32 pixels in one color.
func menuIconPNG(t *testing.T, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestContextMenuIcon gives the system's menu the icons of its items, in
// submenus too, and leaves the choice to Chosen as for other items.
func TestContextMenuIcon(t *testing.T) {
	flag, inner := &platform.MenuIcon{}, &platform.MenuIcon{}
	var chosen string
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreText(c, "Row").ContextMenu(func(m *Menu) {
				if m.Item("Flag").Icon(testMenuIcon{flag}).Chosen() {
					chosen = "Flag"
				}
				m.Item("Plain").Icon(nil)
				m.Submenu("More", func(m *Menu) { m.Item("Inner").Icon(testMenuIcon{inner}) })
			})
		})
	}
	tt := coreNewTester(view, 200, 200)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	items := tt.h.menu.Items
	if items[0].Icon != flag || items[1].Icon != nil || items[2].Submenu.Items[0].Icon != inner {
		t.Errorf("the icons of the items are %p, %p and %p, want %p, nil and %p",
			items[0].Icon, items[1].Icon, items[2].Submenu.Items[0].Icon, flag, inner)
	}
	if err := tt.ChooseMenuItem("Flag"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if chosen != "Flag" {
		t.Errorf("chosen = %q, want Flag", chosen)
	}
}

// TestPopoverMenuIcon draws the icon of an item before its label, and
// lines up the label of the item without one.
func TestPopoverMenuIcon(t *testing.T) {
	red := &platform.MenuIcon{PNG: menuIconPNG(t, color.NRGBA{R: 255, A: 255})}
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreText(c, "Row").PopoverMenu(func(m *Menu) {
				m.Item("Flag").Icon(testMenuIcon{red})
				m.Item("Plain")
				m.Separator()
				m.Submenu("More", func(m *Menu) { m.Item("Inner") })
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	// Find gives the item's row, and text the text inside it.
	text := func(s string) Rect {
		var r Rect
		for _, n := range tt.rt.labels {
			if n.text == s {
				r = n.r
			}
		}
		return r
	}
	if f, p := text("Flag"), text("Plain"); f.X != p.X {
		t.Errorf("the labels start at %v and %v", f.X, p.X)
	}
	// at returns the color at the middle of the box of an item's icon.
	at := func(s string) (uint32, uint32, uint32) {
		r := text(s)
		x := int(r.X) - 8 - platform.MenuIconSize/2
		cr, cg, cb, _ := tt.Image().At(x, int(r.Y+r.H/2)).RGBA()
		return cr >> 8, cg >> 8, cb >> 8
	}
	if r, g, b := at("Flag"); r < 200 || g > 50 || b > 50 {
		t.Errorf("the icon is (%d, %d, %d), want red", r, g, b)
	}
	if r, g, b := at("Plain"); r > 200 && g < 50 && b < 50 {
		t.Errorf("the item without an icon shows red: (%d, %d, %d)", r, g, b)
	}
	d := tt.rt.menu.drawn
	tt.Frame()
	if len(d.icons) != 1 {
		t.Errorf("%d bitmaps of one icon over two frames", len(d.icons))
	}
}

// TestMenuIconBitmapTemplate colors a template image with the text's color
// and keeps its alpha.
func TestMenuIconBitmapTemplate(t *testing.T) {
	ic := &platform.MenuIcon{PNG: menuIconPNG(t, color.NRGBA{A: 128}), Template: true}
	b := menuIconBitmap(ic, Color{R: 255, A: 255})
	pix := b.img.Pixels()
	if pix[0] != 128 || pix[1] != 0 || pix[2] != 0 || pix[3] != 128 {
		t.Errorf("premultiplied pixel = %v, want [128 0 0 128]", pix[:4])
	}
	if menuIconBitmap(&platform.MenuIcon{PNG: []byte("not a png")}, Color{}) != nil {
		t.Error("a bitmap of what is not an image")
	}
}
