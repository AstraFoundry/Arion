// Flyout opens flyouts from a small window of native UI: a list of colors,
// a note to edit and an About box with the look of the system's popovers
// on macOS. They take the keyboard, and close as the user clicks elsewhere
// or presses Escape. They extend beyond the window, and flip or slide to
// stay on the screen.
//
//	go run ./examples/flyout
package main

import (
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var colors = []struct {
	name  string
	color ui.Color
}{
	{"Red", ui.RGB(0xff, 0x3b, 0x30)},
	{"Orange", ui.RGB(0xff, 0x95, 0x00)},
	{"Yellow", ui.RGB(0xff, 0xcc, 0x00)},
	{"Green", ui.RGB(0x34, 0xc7, 0x59)},
	{"Blue", ui.RGB(0x00, 0x7a, 0xff)},
	{"Purple", ui.RGB(0xaf, 0x52, 0xde)},
}

type app struct {
	win    *mygo.Window
	list   *mygo.Window // the open list of colors, or nil
	editor *mygo.Window // the open note editor, or nil
	about  *mygo.Window // the open About box, or nil
	color  int          // the chosen color
	hover  int          // the color the keys move to in the list
	note   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(12).Padding(16).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(14).Height(14).Radius(7).Background(colors[a.color].color)
			b := ui.Button(c.Key("color"), colors[a.color].name+" ▾")
			if b.Clicked() {
				a.toggleList(b.Bounds())
			}
		})
		b := ui.Button(c.Key("note"), "Edit note…")
		if b.Clicked() {
			a.openEditor(b.Bounds())
		}
		b = ui.Button(c.Key("about"), "About")
		if b.Clicked() {
			a.openAbout(b.Bounds())
		}
		note := a.note
		if note == "" {
			note = "No note"
		}
		ui.Text(c, note).FontSize(12).TextColor(t.TextMuted)
	})
}

// toggleList opens the list of colors below the button, or closes it.
func (a *app) toggleList(button ui.Rect) {
	if a.list != nil {
		a.list.Close()
		return
	}
	a.hover = a.color
	a.list = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent:    a.win,
		Anchor:    anchor(button),
		Gap:       4,
		Width:     180,
		Height:    len(colors)*28 + 12,
		Focusable: true,
		Shadow:    true,
		Content:   ui.View(a.listView),
	})
	a.list.OnClosed(func() {
		a.list = nil
		a.win.Invalidate()
	})
	a.win.Invalidate()
}

func (a *app) listView(c *ui.Context) {
	t := c.Theme()
	if c.Shortcut(0, ui.KeyDown) {
		a.hover = (a.hover + 1) % len(colors)
	}
	if c.Shortcut(0, ui.KeyUp) {
		a.hover = (a.hover + len(colors) - 1) % len(colors)
	}
	if c.Shortcut(0, ui.KeyEnter) {
		a.choose(a.hover)
	}
	c.Root().Background(ui.Transparent)
	panel(c).Padding(6).Children(func() {
		for i, col := range colors {
			row := ui.Row(c.Key(i)).Height(28).Gap(8).Padding(0, 8).AlignItems(ui.Center).Radius(6)
			if i == a.hover {
				row.Background(t.Accent)
			}
			if row.Hovered() && a.hover != i {
				a.hover = i
				c.Invalidate()
			}
			if row.Clicked() {
				a.choose(i)
			}
			row.Children(func() {
				ui.Box(c).Width(12).Height(12).Radius(6).Background(col.color)
				text := ui.Text(c, col.name)
				if i == a.hover {
					text.TextColor(t.AccentText)
				}
			})
		}
	})
}

func (a *app) choose(i int) {
	a.color = i
	if a.list != nil {
		a.list.Close()
	}
	a.win.Invalidate()
}

// openEditor opens the note editor to the right of the button, which
// takes the keyboard.
func (a *app) openEditor(button ui.Rect) {
	if a.editor != nil {
		return
	}
	a.editor = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent:    a.win,
		Anchor:    anchor(button),
		Placement: mygo.PlacementRightStart,
		Gap:       8,
		Width:     260,
		Height:    96,
		Focusable: true,
		Shadow:    true,
		Content:   ui.View(a.editorView),
	})
	a.editor.OnClosed(func() {
		a.editor = nil
		a.win.Invalidate()
	})
}

func (a *app) editorView(c *ui.Context) {
	c.Root().Background(ui.Transparent)
	panel(c).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "Note").Bold()
		ui.TextInput(c.Key("note"), &a.note).Placeholder("Write something").AutoFocus()
		if c.Shortcut(0, ui.KeyEnter) {
			a.editor.Close()
		}
	})
}

// openAbout opens the About box above the button, as a popover on macOS.
func (a *app) openAbout(button ui.Rect) {
	if a.about != nil {
		return
	}
	a.about = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent:    a.win,
		Anchor:    anchor(button),
		Placement: mygo.PlacementTop,
		Gap:       6,
		Width:     240,
		Height:    84,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(a.aboutView),
	})
	a.about.OnClosed(func() { a.about = nil })
}

func (a *app) aboutView(c *ui.Context) {
	c.Root().Background(ui.Transparent)
	box := ui.Column(c).Fill() // over the popover's material
	if !c.Vibrancy() {
		box = panel(c) // no popover here
	}
	box.Padding(12).Gap(4).Children(func() {
		ui.Text(c, "Flyout").Bold()
		ui.Text(c, "Windows owned by their parent, placed next to an anchor.").FontSize(12).TextColor(c.Theme().TextMuted)
	})
}

// panel is the box a flyout draws: the window itself draws nothing. The
// shadow of a window outlines it on macOS, and Windows rounds and outlines
// the window too: elsewhere, a border does.
func panel(c *ui.Context) ui.Element {
	t := c.Theme()
	box := ui.Column(c).Fill().Background(t.Surface)
	switch runtime.GOOS {
	case "darwin":
		box.Radius(10)
	case "windows":
	default:
		box.Radius(10).Border(1, t.Border)
	}
	return box
}

func anchor(r ui.Rect) mygo.Rectangle {
	return mygo.Rectangle{X: int(r.X), Y: int(r.Y), Width: int(r.W), Height: int(r.H)}
}

func main() {
	a := &app{note: ""}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Flyout",
			Width:   300,
			Height:  240,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
