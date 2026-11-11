// Launcher is an app launcher in the manner of Spotlight and Raycast, in
// native UI: Alt+Shift+Space shows a panel over every app, full-screen ones
// included, without taking the active app away; type to find an installed
// app, Up and Down to choose, Enter to open it, Escape to clear or close.
// It lives in the menu bar (the notification area on Windows), without a
// Dock icon or a taskbar button.
//
//	go run ./examples/launcher
package main

import (
	"bytes"
	"cmp"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// shortcuts are tried in turn: the first one free is the app's. Not
// Alt+Space, which Raycast, PowerToys Run and others take: macOS gives a
// shortcut to every app that registers it, so both would open.
var shortcuts = []string{"Alt+Shift+Space", "CmdOrCtrl+Alt+Shift+Space"}

const (
	width, height = 680, 420
	iconSize      = 28 // DIPs
)

var searchIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`))

// launcher holds what the panel shows. It is read and changed on the main
// thread, by the view and the callbacks of the app, the tray and the
// shortcut; the apps come from a goroutine through Window.Update.
type launcher struct {
	panel    *mygo.Window
	tray     *mygo.Tray
	shortcut string

	apps    []mygo.InstalledApp
	results []mygo.InstalledApp
	query   string
	chosen  int
	list    ui.ListState
	input   ui.Handle
	icons   map[string]*ui.Bitmap // by app path; nil where the app has none
	loading bool
}

func (l *launcher) view(c *ui.Context) {
	t := c.Theme()
	root := c.Root()
	if c.Vibrancy() {
		root.Background(ui.Transparent) // the material shows through
	} else {
		root.Background(t.Background)
	}
	// An Escape left to the panel closes it, which hides it (ready): with
	// a query, Escape clears it first.
	if l.query != "" && c.Shortcut(0, ui.KeyEscape) {
		l.query = ""
		l.filter()
	}
	if c.Shortcut(0, ui.KeyDown) {
		l.choose(l.chosen + 1)
	}
	if c.Shortcut(0, ui.KeyUp) {
		l.choose(l.chosen - 1)
	}
	if c.Shortcut(0, ui.KeyEnter) {
		l.open()
	}
	c.OnShortcut(ui.Cmd, ui.KeyQ, mygo.App.Quit)

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Padding(16, 18).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, searchIcon).FontSize(20).TextColor(t.TextMuted)
			if ui.TextInputBase(c, &l.query).Bind(&l.input).Grow(1).FontSize(22).
				Placeholder("Search apps").Label("Search apps").Changed() {
				l.filter()
			}
		})
		ui.Divider(c)
		l.list.Selected = &l.chosen
		l.list.Key = func(i int) any { return l.results[i].Path }
		l.list.Label = func(i int) string { return l.results[i].Name }
		list := ui.List(c, &l.list, len(l.results), func(i int) { l.row(c, i) }).Grow(1).Padding(6).Children(func() {
			if len(l.results) == 0 {
				msg := "No apps found"
				if l.loading {
					msg = "Finding apps…"
				}
				ui.Text(c, msg).TextColor(t.TextMuted).Padding(16)
			}
		})
		if list.Submitted() {
			l.open()
		}
		ui.Divider(c)
		ui.Row(c).Padding(8, 16).Gap(16).Children(func() {
			ui.Text(c, keys(l.shortcut)+" to show or hide").FontSize(11).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, "↑↓ Choose   ↵ Open   esc Close").FontSize(11).TextColor(t.TextMuted)
		})
	})
}

// row shows an app with its icon, its name, and where it is.
func (l *launcher) row(c *ui.Context, i int) {
	app := l.results[i]
	t := c.Theme()
	bg := ui.Transparent
	if i == l.chosen {
		bg = t.Accent.Alpha(0.85)
	}
	text, muted := t.Text, t.TextMuted
	if i == l.chosen {
		text, muted = t.AccentText, t.AccentText.Alpha(0.75)
	}
	ui.Row(c).Padding(6, 10).Gap(12).AlignItems(ui.Center).Radius(8).Background(bg).Children(func() {
		if icon := l.icon(app); icon != nil {
			ui.Image(c, icon).Size(iconSize, iconSize)
		} else {
			ui.Avatar(c, app.Name, nil).Size(iconSize, iconSize) // its initial
		}
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, app.Name).TextColor(text).SingleLine()
			ui.Text(c, where(app)).FontSize(11).TextColor(muted).SingleLine()
		})
	})
}

// where says where an app is, shortly: its folder or its ID.
func where(app mygo.InstalledApp) string {
	switch runtime.GOOS {
	case "darwin":
		dir := filepath.Dir(app.Path)
		if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(dir, home) {
			return "~" + dir[len(home):]
		}
		for _, system := range []string{"/System/Applications", "/System/Library/CoreServices/Applications", "/System/Library/CoreServices"} {
			if rest, ok := strings.CutPrefix(dir, system); ok {
				return "System" + rest
			}
		}
		return strings.TrimPrefix(dir, "/")
	case "windows":
		// A desktop app's program, or a packaged app's ID.
		if i := strings.LastIndex(app.ID, `\`); i >= 0 {
			return app.ID[i+1:]
		}
		return "App"
	}
	return strings.TrimSuffix(app.ID, ".desktop")
}

// keys spells a shortcut as the platform does: ⌥Space on macOS.
func keys(shortcut string) string {
	if runtime.GOOS != "darwin" {
		return shortcut
	}
	return strings.NewReplacer("CmdOrCtrl+", "⌘", "Cmd+", "⌘", "Ctrl+", "⌃", "Alt+", "⌥", "Shift+", "⇧").Replace(shortcut)
}

// icon returns an app's icon, drawn for the sharpest display the first time
// it shows (a few milliseconds), and kept.
func (l *launcher) icon(app mygo.InstalledApp) *ui.Bitmap {
	if b, ok := l.icons[app.Path]; ok {
		return b
	}
	scale := 1.0
	for _, d := range mygo.Screen.Displays() {
		scale = max(scale, d.ScaleFactor)
	}
	var bitmap *ui.Bitmap
	if data, err := mygo.Shell.FileIcon(app.Path, int(iconSize*scale)); err == nil {
		bitmap, _ = ui.DecodeBitmap(data)
	}
	l.icons[app.Path] = bitmap
	return bitmap
}

// filter ranks the apps matching the query: names starting with it first,
// then those with a word starting with it, those containing it, and those
// with its letters in order, as "vsc" finds Visual Studio Code.
func (l *launcher) filter() {
	q := strings.ToLower(strings.TrimSpace(l.query))
	type ranked struct {
		app  mygo.InstalledApp
		rank int
	}
	var found []ranked
	for _, app := range l.apps {
		if r := rank(strings.ToLower(app.Name), q); r >= 0 {
			found = append(found, ranked{app, r})
		}
	}
	slices.SortStableFunc(found, func(a, b ranked) int { return cmp.Compare(a.rank, b.rank) })
	l.results = l.results[:0]
	for _, f := range found {
		l.results = append(l.results, f.app)
	}
	l.choose(0)
}

func rank(name, q string) int {
	switch {
	case q == "" || strings.HasPrefix(name, q):
		return 0
	case strings.Contains(name, " "+q) || strings.Contains(name, "-"+q):
		return 1
	case strings.Contains(name, q):
		return 2
	}
	rest := name
	for _, r := range q {
		i := strings.IndexRune(rest, r)
		if i < 0 {
			return -1
		}
		rest = rest[i+len(string(r)):]
	}
	return 3
}

func (l *launcher) choose(i int) {
	l.chosen = max(0, min(i, len(l.results)-1))
	if len(l.results) > 0 {
		l.list.ScrollIntoView(l.chosen)
	}
}

// open launches the chosen app and hides the panel, which gives the
// keyboard back to the app the user was in, before the launched app takes
// it.
func (l *launcher) open() {
	if l.chosen >= len(l.results) {
		return
	}
	app := l.results[l.chosen]
	l.panel.Hide()
	if err := mygo.Shell.LaunchApp(app); err != nil {
		log.Printf("launching %s: %v", app.Name, err)
	}
}

// load lists the installed apps on a goroutine, as at each show, where
// apps may have come or gone since.
func (l *launcher) load() {
	l.loading = true
	go func() {
		apps, err := mygo.Shell.InstalledApps()
		if err != nil {
			log.Print(err)
		}
		l.panel.Update(func() {
			l.apps, l.loading = apps, false
			l.filter()
		})
	}()
}

// show shows the panel on the display with the pointer, a third of the way
// down, with an empty query.
func (l *launcher) show() {
	d := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	w, h := l.panel.Size()
	l.panel.SetPosition(d.WorkArea.X+(d.WorkArea.Width-w)/2, d.WorkArea.Y+(d.WorkArea.Height-h)/3)
	l.query = ""
	l.filter()
	l.load()
	l.input.Focus()
	l.panel.Show()
}

func (l *launcher) toggle() {
	if l.panel.IsVisible() {
		l.panel.Hide()
	} else {
		l.show()
	}
}

func (l *launcher) ready() {
	l.icons = map[string]*ui.Bitmap{}
	l.panel = mygo.NewWindow(mygo.WindowOptions{
		Title:         "Launcher",
		Panel:         true,
		Hidden:        true,
		Frameless:     true,
		Transparent:   true,
		Vibrancy:      mygo.VibrancyGlass,
		CornerRadius:  16,
		Width:         width,
		Height:        height,
		DisableResize: true,
		Content:       ui.View(l.view),
	})
	// Hide rather than close, on Escape, Cmd-W or Alt-F4, and when the user
	// goes elsewhere: the panel shows again at once, as it was. Quitting
	// destroys it.
	l.panel.OnClose(func(e *mygo.CloseEvent) {
		e.PreventDefault()
		l.panel.Hide()
	})
	l.panel.OnBlur(l.panel.Hide)

	for _, s := range shortcuts {
		if err := mygo.GlobalShortcut.Register(s, l.toggle); err == nil {
			l.shortcut = s
			break
		} else {
			log.Printf("shortcut %s: %v", s, err)
		}
	}
	l.setUpTray()
	if !mygo.App.WasOpenedAtLogin() {
		l.show() // started by the user: show what it does
	}
}

func (l *launcher) setUpTray() {
	var err error
	l.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: "Launcher"})
	if err != nil {
		log.Print("no tray icon: ", err)
		return
	}
	var menu *mygo.Menu
	menu = mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open Launcher", Accelerator: l.shortcut, Click: func(*mygo.MenuItem, *mygo.Window) { l.show() }},
		mygo.Separator(),
		{Label: "Open at Login", Type: mygo.MenuItemCheckbox, ID: "login", Checked: mygo.App.OpenAtLogin(),
			Click: func(it *mygo.MenuItem, _ *mygo.Window) {
				if err := mygo.App.SetOpenAtLogin(it.Checked); err != nil {
					log.Print(err)
				}
			}},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
	l.tray.SetMenu(menu)
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { l.tray.Destroy() })
}

// trayIcon draws a magnifying glass, black on transparent for macOS to
// tint, blue elsewhere to show on light and dark taskbars alike.
func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{R: 79, G: 124, B: 255, A: 255}
	if runtime.GOOS == "darwin" {
		ink = color.NRGBA{A: 255}
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx, dy := float64(x)-13, float64(y)-13
			if r := dx*dx + dy*dy; r >= 7*7 && r <= 10*10 {
				img.Set(x, y, ink)
			}
		}
	}
	for i := 0; i < 8; i++ { // the handle
		draw.Draw(img, image.Rect(20+i, 20+i, 23+i, 23+i), image.NewUniform(ink), image.Point{}, draw.Src)
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return // the running launcher shows itself (OnSecondInstance)
	}
	l := &launcher{}
	mygo.App.SetName("Launcher")
	// No Dock icon or app switcher entry (macOS): a bundle says so with
	// LSUIElement in mygo.config.ts' macos.infoPlist, from the start.
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {}) // keep running without windows
	mygo.App.OnSecondInstance(func([]string, string) { l.show() })
	mygo.App.WhenReady(l.ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
