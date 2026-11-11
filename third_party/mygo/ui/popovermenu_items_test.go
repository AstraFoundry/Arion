package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
)

// glyphColor returns the color of the first glyph the last frame painted
// in r, and whether one was.
func glyphColor(tt *Tester, r Rect) (Color, bool) {
	s := tt.h.scale
	for _, op := range tt.h.last.Ops {
		if op.Kind != scene.OpGlyphs {
			continue
		}
		for _, g := range tt.h.last.Glyphs[op.Start:op.End] {
			x, y := (g.X+g.W/2)/s, (g.Y+g.H/2)/s
			if x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H {
				return Color{R: g.Color.R, G: g.Color.G, B: g.Color.B, A: g.Color.A}, true
			}
		}
	}
	return Color{}, false
}

// findText returns the box of the last element showing text s, or
// labeled s: of a menu item's text, rather than of the item labeled with
// it, which comes before.
func findText(tt *Tester, s string) (Rect, bool) {
	var r Rect
	ok := false
	for _, n := range tt.rt.labels {
		if n.text == s {
			r, ok = n.r, true
		}
	}
	return r, ok
}

// textFontSize returns the font size set on the element showing text s,
// 0 when it sets none.
func textFontSize(tt *Tester, s string) float32 {
	var size float32
	var walk func(e *node)
	walk = func(e *node) {
		for ; e != nil; e = e.next {
			if e.kind == kindText && e.text == s && e.ts.set&setSize != 0 {
				size = e.ts.size
			}
			walk(e.first)
		}
	}
	walk(tt.rt.c.root)
	return size
}

// textCount returns how many elements show text s, or are labeled s.
func textCount(tt *Tester, s string) int {
	n := 0
	for _, t := range tt.Texts() {
		if t == s {
			n++
		}
	}
	return n
}

// textColor returns the color the text s was painted in.
func textColor(t *testing.T, tt *Tester, s string) Color {
	t.Helper()
	r, ok := findText(tt, s)
	if !ok {
		t.Fatalf("no text %q among %q", s, tt.Texts())
	}
	c, ok := glyphColor(tt, r)
	if !ok {
		t.Fatalf("no glyphs of %q painted at %+v", s, r)
	}
	return c
}

func sameRGB(a, b Color) bool { return a.R == b.R && a.G == b.G && a.B == b.B }

// itemsApp is a view with a menu button whose drawn menu has items of
// the richer kinds, and what the menu did.
type itemsApp struct {
	theme   *Theme
	chosen  []string
	preview string // the theme highlighted, as the menu built
	content []bool // the highlight Content saw, as it built
	build   func(m *Menu)
}

func (a *itemsApp) view(c *context) {
	if a.theme != nil {
		c.SetTheme(a.theme)
	}
	coreColumn(c).Fill().Padding(20).Gap(10).AlignItems(Start).Children(func() {
		corePopoverMenuButton(c, "File", a.build).Label("File")
	})
}

func (a *itemsApp) choose(s string) { a.chosen = append(a.chosen, s) }

func TestPopoverMenuDangerItem(t *testing.T) {
	a := &itemsApp{}
	a.build = func(m *Menu) {
		if m.Item("Rename").Chosen() {
			a.choose("Rename")
		}
		if m.Item("Delete…").Danger().Chosen() {
			a.choose("Delete")
		}
		m.Item("Erase").Danger().Disabled(true)
	}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("File")
	th := LightTheme()
	if c := textColor(t, tt, "Delete…"); !sameRGB(c, th.Danger) {
		t.Errorf("Delete… in %v, not the theme's Danger %v", c, th.Danger)
	}
	if c := textColor(t, tt, "Rename"); !sameRGB(c, th.Text) {
		t.Errorf("Rename in %v, not the theme's Text %v", c, th.Text)
	}
	if c := textColor(t, tt, "Erase"); !sameRGB(c, th.TextMuted) {
		t.Errorf("Erase, disabled, in %v, not TextMuted %v", c, th.TextMuted)
	}
	// Highlighted, it keeps the theme's highlight, with the highlight's
	// text, as red is not legible on the theme's blue.
	r, _ := tt.Find("Delete…")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if c := textColor(t, tt, "Delete…"); !sameRGB(c, th.AccentText) {
		t.Errorf("Delete…, highlighted, in %v, not AccentText %v", c, th.AccentText)
	}
	// On a highlight it is legible on, it keeps the Danger color.
	pale := LightTheme()
	pale.Accent, pale.AccentText, pale.Danger = Hex("#e4e4e7"), Hex("#18181b"), Hex("#991b1b")
	a.theme = pale
	tt.Frame()
	if c := textColor(t, tt, "Delete…"); !sameRGB(c, pale.Danger) {
		t.Errorf("Delete… on a pale highlight in %v, not Danger %v", c, pale.Danger)
	}
	if err := tt.Click("Delete…"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.chosen, []string{"Delete"}) {
		t.Errorf("chose %q", a.chosen)
	}
}

func TestPopoverMenuSubtitleAndTrailing(t *testing.T) {
	a := &itemsApp{}
	a.build = func(m *Menu) {
		if m.Item("Back Up Now").Subtitle("Last run: 3 minutes ago").Tooltip("Copies new files").Chosen() {
			a.choose("Back Up")
		}
		m.Item("Image.png").Trailing("2.4 MB").Shortcut(Cmd, KeyO)
	}
	tt := coreNewTester(a.view, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("File")
	label, ok1 := findText(tt, "Back Up Now")
	sub, ok2 := findText(tt, "Last run: 3 minutes ago")
	if !ok1 || !ok2 {
		t.Fatalf("no label or subtitle: %q", tt.Texts())
	}
	if sub.Y < label.Y+label.H-1 || sub.X != label.X {
		t.Errorf("the subtitle at %+v is not a line under the label at %+v", sub, label)
	}
	// Smaller by its font size: the line boxes of two sizes can round to
	// the same height, as with some fonts.
	th := LightTheme()
	if size := textFontSize(tt, "Last run: 3 minutes ago"); !(size > 0 && size < th.FontSize) {
		t.Errorf("the subtitle in a font of size %v, not smaller than the label's %v", size, th.FontSize)
	}
	if c := textColor(t, tt, "Last run: 3 minutes ago"); !sameRGB(c, th.TextMuted) {
		t.Errorf("the subtitle in %v, not TextMuted", c)
	}
	name, _ := findText(tt, "Image.png")
	size, ok := findText(tt, "2.4 MB")
	keys, _ := tt.Find(shortcutText(accelerator(Cmd, KeyO)))
	if !ok || size.X <= name.X+name.W || size.X+size.W > keys.X {
		t.Errorf("the trailing text at %+v is not between the label at %+v and the shortcut at %+v", size, name, keys)
	}
	tree := tt.h.access
	backUp := accessNode(t, tree, platform.RoleMenuItem, "Back Up Now")
	if backUp.Label != "Back Up Now" || backUp.Description != "Last run: 3 minutes ago\nCopies new files" {
		t.Errorf("Back Up Now: label %q, description %q", backUp.Label, backUp.Description)
	}
	if img := accessNode(t, tree, platform.RoleMenuItem, "Image.png"); img.Label != "Image.png" || img.Description != "2.4 MB" {
		t.Errorf("Image.png: label %q, description %q", img.Label, img.Description)
	}
	if err := tt.Click("Last run: 3 minutes ago"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.chosen, []string{"Back Up"}) {
		t.Errorf("clicking the subtitle chose %q", a.chosen)
	}
}

// TestMenuItemSubtitleInTheSystemsMenu checks that the system's menus get
// the subtitle, and neither the trailing text nor the content.
func TestMenuItemSubtitleInTheSystemsMenu(t *testing.T) {
	view := func(c *context) {
		coreText(c, "Files").ContextMenu(func(m *Menu) {
			m.Item("Back Up Now").Subtitle("Last run: now").Trailing("2.4 MB").Danger()
			m.Item("Download").Content(func(x MenuItemContent) { coreText(x.Context.build(), "40%") })
		})
	}
	tt := coreNewTester(view, 300, 200)
	tt.RightClick("Files")
	items := tt.h.menu.Items
	if items[0].Label != "Back Up Now" || items[0].Subtitle != "Last run: now" || items[1].Label != "Download" {
		t.Errorf("items %+v, %+v", items[0], items[1])
	}
	if !slices.Equal(tt.Menu(), []string{"Back Up Now", "Download"}) {
		t.Errorf("menu %q", tt.Menu())
	}
}

func TestPopoverMenuItemContent(t *testing.T) {
	a := &itemsApp{}
	progress := float32(0.4)
	a.build = func(m *Menu) {
		if m.Item("Open").Chosen() {
			a.choose("Open")
		}
		item := m.Item("Download").Trailing("40%")
		item.Content(func(x MenuItemContent) {
			c := x.Context.build()
			a.content = append(a.content, x.Highlighted)
			if x.Highlighted != item.Highlighted() {
				t.Errorf("Content sees highlighted %v, the item %v", x.Highlighted, item.Highlighted())
			}
			coreText(c, "Downloading report.pdf")
			coreBox(c).Height(4).FillWidth().Label("progress").Draw(func(p *Painter, r Rect) {
				p.Fill(Rect{X: r.X, Y: r.Y, W: r.W * progress, H: r.H}, x.Text, 2)
			})
		})
		if item.Chosen() {
			a.choose("Download")
		}
		if m.Item("Close").Chosen() {
			a.choose("Close")
		}
	}
	tt := coreNewTester(a.view, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("File")
	tt.Key(0, KeyDown) // the keys take the menu
	// The item is labeled Download, and shows no such text.
	if !tt.HasText("Downloading report.pdf") || textCount(tt, "Download") != 1 {
		t.Fatalf("the content does not replace the label: %q", tt.Texts())
	}
	if !tt.HasText("40%") {
		t.Error("no trailing text beside the content")
	}
	// The keys go through it, and it is the item Download to assistive
	// technology, highlighted.
	tt.Key(0, KeyHome)
	wantFocus(t, tt, "Open")
	a.content = nil
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Download")
	if len(a.content) == 0 || !a.content[len(a.content)-1] {
		t.Errorf("Content saw the highlights %v, not the item highlighted", a.content)
	}
	if c := textColor(t, tt, "Downloading report.pdf"); !sameRGB(c, LightTheme().AccentText) {
		t.Errorf("the content's text, highlighted, in %v", c)
	}
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Close")
	if a.content[len(a.content)-1] {
		t.Error("Content saw the item highlighted after the keys left it")
	}
	// Typing goes to it by its label.
	tt.TypeKey(0, KeyD, "d")
	wantFocus(t, tt, "Download")
	tt.Key(0, KeyEnter)
	if !slices.Equal(a.chosen, []string{"Download"}) || tt.HasText("Open") {
		t.Errorf("Enter chose %q, the menu shows %v", a.chosen, tt.HasText("Open"))
	}
	// A press on the content chooses the item.
	tt.Click("File")
	if err := tt.Click("Downloading report.pdf"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.chosen, []string{"Download", "Download"}) {
		t.Errorf("clicking the content chose %q", a.chosen)
	}
}

// TestPopoverMenuItemContentWithAnInput checks an input in an item's
// Content: it takes the focus as it is clicked, and the text typed, Space
// too, and the menu keeps Up, Down, Enter and Escape.
func TestPopoverMenuItemContentWithAnInput(t *testing.T) {
	a := &itemsApp{}
	filter := ""
	a.build = func(m *Menu) {
		m.Item("Filter").Content(func(x MenuItemContent) {
			coreTextInput(x.Context.build(), &filter).Label("Filter text")
		})
		for _, f := range []string{"Alpha", "Beta", "Gamma ray"} {
			if !strings.Contains(strings.ToLower(f), strings.ToLower(filter)) {
				continue
			}
			if m.Item(f).ID(f).Chosen() {
				a.choose(f)
			}
		}
	}
	tt := coreNewTester(a.view, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("File")
	if err := tt.Click("Filter text"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Alpha") {
		t.Fatalf("a click in the field closed the menu: %q", tt.Texts())
	}
	if s := tt.rt.states[tt.rt.focused]; s == nil || s.editor == nil {
		t.Fatal("the field did not take the focus")
	}
	tt.Type("a r")
	if filter != "a r" {
		t.Fatalf("typed %q into the field", filter)
	}
	if tt.HasText("Alpha") || !tt.HasText("Gamma ray") {
		t.Errorf("the filter left %q", tt.Texts())
	}
	// Enter is the field's, which it submits.
	tt.Key(0, KeyEnter)
	if len(a.chosen) != 0 || !tt.HasText("Gamma ray") {
		t.Fatalf("Enter in the field: chose %q, menu shows %v", a.chosen, tt.HasText("Gamma ray"))
	}
	// Down moves the highlight, and the focus back to the menu.
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Gamma ray")
	tt.Key(0, KeyEnter)
	if !slices.Equal(a.chosen, []string{"Gamma ray"}) || tt.HasText("Gamma ray") {
		t.Errorf("Down, Enter from the field: chose %q, menu shows %v", a.chosen, tt.HasText("Gamma ray"))
	}
}

func TestPopoverMenuHighlightedInBuild(t *testing.T) {
	a := &itemsApp{}
	themes := []string{"Light", "Dark", "Solarized"}
	a.build = func(m *Menu) {
		a.preview = ""
		for _, th := range themes {
			item := m.Item(th).ID(th)
			if item.Highlighted() {
				a.preview = th
			}
			if item.Chosen() {
				a.choose(th)
			}
		}
		if a.preview != "" {
			// What goes with the item highlighted, after the items, keyed
			// so that the highlight stays.
			m.Separator()
			m.Item("Preview of " + a.preview).ID("preview").Disabled(true)
		}
	}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("File")
	if a.preview != "" {
		t.Errorf("preview %q before the pointer or the keys moved", a.preview)
	}
	r, _ := tt.Find("Dark")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if a.preview != "Dark" || !tt.HasText("Preview of Dark") {
		t.Errorf("pointer on Dark: preview %q, texts %q", a.preview, tt.Texts())
	}
	tt.Key(0, KeyDown)
	if a.preview != "Solarized" || !tt.HasText("Preview of Solarized") {
		t.Errorf("Down to Solarized: preview %q, texts %q", a.preview, tt.Texts())
	}
	tt.Frame()
	if a.preview != "Solarized" {
		t.Errorf("a frame after: preview %q", a.preview)
	}
	// Submenus: the item leading to the one highlighted is highlighted.
	a.build = func(m *Menu) {
		a.preview = ""
		sub := m.Submenu("Theme", func(m *Menu) {
			for _, th := range themes {
				if m.Item(th).Highlighted() {
					a.preview = th
				}
			}
		})
		if sub.Highlighted() {
			a.chosen = append(a.chosen[:0], "Theme")
		}
	}
	tt.Key(0, KeyEscape)
	tt.Click("File")
	tt.Key(0, KeyDown)
	tt.Key(0, KeyRight)
	if a.preview != "Light" || !slices.Equal(a.chosen, []string{"Theme"}) {
		t.Errorf("in the submenu: preview %q, highlighted %q", a.preview, a.chosen)
	}
	tt.Key(0, KeyEscape)
	tt.Key(0, KeyEscape)
	if a.preview != "" || tt.HasText("Theme") {
		t.Errorf("closed: preview %q", a.preview)
	}
}

func TestMenuItemIDKeepsTheChoice(t *testing.T) {
	// The system's menu: the label changes before the choice comes back.
	n := 3
	var chosen []string
	view := func(c *context) {
		coreText(c, "Files").ContextMenu(func(m *Menu) {
			if m.Item(fmt.Sprintf("Sync %d Files", n)).ID("sync").Chosen() {
				chosen = append(chosen, "sync")
			}
			if m.Item(fmt.Sprintf("Clear %d Files", n)).Chosen() {
				chosen = append(chosen, "clear")
			}
		})
	}
	tt := coreNewTester(view, 300, 200)
	tt.RightClick("Files")
	n = 4
	if err := tt.ChooseMenuItem("Sync 3 Files"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("Files")
	n = 5
	tt.ChooseMenuItem("Clear 4 Files")
	if !slices.Equal(chosen, []string{"sync"}) {
		t.Errorf("chose %q: want sync by its ID, and not Clear, whose label changed", chosen)
	}

	// A drawn menu: an item comes before it as it shows.
	a := &itemsApp{}
	extra := false
	a.build = func(m *Menu) {
		if extra {
			if m.Item("Undo").ID("undo").Chosen() {
				a.choose("undo")
			}
		}
		if m.Item("Rename").ID("rename").Chosen() {
			a.choose("rename")
		}
		if m.Item("Rename").ID("rename-all").Chosen() {
			a.choose("rename-all")
		}
	}
	tt = coreNewTester(a.view, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("File")
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	extra = true
	tt.Frame()
	// The highlight stays on the second Rename, its row keyed by its ID.
	if f := accessFocus(t, tt); f.Label != "Rename" || f.PosInSet != 3 {
		t.Errorf("the highlight moved to %q at %d", f.Label, f.PosInSet)
	}
	tt.Key(0, KeyEnter)
	if !slices.Equal(a.chosen, []string{"rename-all"}) {
		t.Errorf("Enter chose %q", a.chosen)
	}
	// The Tester's choice too, as the items before it change.
	extra = false
	tt.Click("File")
	extra = true
	if err := tt.ChooseMenuItem("Rename"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.chosen, []string{"rename-all", "rename"}) {
		t.Errorf("ChooseMenuItem chose %q", a.chosen)
	}
}

func TestPopoverMenuMaxWidth(t *testing.T) {
	long := strings.Repeat("A very long name of a file ", 6) + "end.txt"
	a := &itemsApp{}
	width := float32(240)
	a.build = func(m *Menu) {
		m.MaxWidth(width)
		m.Item(long).Subtitle(long).Trailing("12 KB").Shortcut(Cmd, KeyO)
		m.Item("Short")
		m.Submenu("More", func(m *Menu) {
			m.Item(long + " in the submenu")
		})
		m.Submenu("Wide", func(m *Menu) {
			m.MaxWidth(0)
			m.Item(long + " wide")
		})
	}
	tt := coreNewTester(a.view, 1200, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("File")
	panel := accessNode(t, tt.h.access, platform.RoleMenu, "")
	if panel.Bounds.W > float64(width)+0.5 {
		t.Errorf("the menu is %v wide, past its MaxWidth %v", panel.Bounds.W, width)
	}
	label, _ := findText(tt, long)
	size, ok1 := findText(tt, "12 KB")
	keys, ok2 := tt.Find(shortcutText(accelerator(Cmd, KeyO)))
	if !ok1 || !ok2 || label.X+label.W > size.X || size.X+size.W > keys.X || keys.X+keys.W > float32(panel.Bounds.X+panel.Bounds.W) {
		t.Errorf("label %+v, trailing %+v, shortcut %+v in the menu at %+v", label, size, keys, panel.Bounds)
	}
	if n := accessNode(t, tt.h.access, platform.RoleMenuItem, "end.txt"); n.Label != long {
		t.Errorf("assistive technology reads %q", n.Label)
	}
	if !slices.Contains(tt.Menu(), long) {
		t.Errorf("Tester.Menu has %q", tt.Menu())
	}
	// A submenu takes its menu's width, unless it sets its own.
	tt.Key(0, KeyEnd)
	tt.Key(0, KeyUp)
	tt.Key(0, KeyRight)
	if sub := accessNode(t, tt.h.access, platform.RoleMenu, "More"); sub.Bounds.W > float64(width)+0.5 {
		t.Errorf("the submenu is %v wide", sub.Bounds.W)
	}
	tt.Key(0, KeyLeft)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyRight)
	if sub := accessNode(t, tt.h.access, platform.RoleMenu, "Wide"); sub.Bounds.W <= float64(width) {
		t.Errorf("the submenu of no MaxWidth is %v wide", sub.Bounds.W)
	}
	// Without MaxWidth, the menu is as wide as its labels.
	width = 0
	tt.Key(0, KeyEscape)
	tt.Key(0, KeyEscape)
	tt.Click("File")
	if panel := accessNode(t, tt.h.access, platform.RoleMenu, ""); panel.Bounds.W <= 240 {
		t.Errorf("without MaxWidth, the menu is %v wide", panel.Bounds.W)
	}
}
