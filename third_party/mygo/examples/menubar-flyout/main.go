// Menubar-flyout is a menu bar app whose icon opens a flyout, the panel of
// the app: below the icon of the macOS menu bar, in a popover of the
// system's, and above an icon of the Windows taskbar. It closes as the
// user clicks elsewhere, presses Escape or clicks the icon again.
//
//	go run ./examples/menubar-flyout
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// All state is read and changed on the main thread, by tray callbacks and
// the view.
type app struct {
	tray    *mygo.Tray
	panel   *mygo.Window // the open panel, or nil
	focus   bool
	sound   bool
	minutes float64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	c.Root().Background(ui.Transparent)
	box := ui.Column(c).Fill()
	switch {
	case c.Vibrancy(): // over the popover's material on macOS
	case runtime.GOOS == "windows": // which rounds and outlines the flyout
		box.Background(t.Surface)
	default:
		box.Background(t.Surface).Radius(10).Border(1, t.Border)
	}
	box.Padding(16).Gap(12).Children(func() {
		ui.Text(c, "Focus").FontSize(17).Bold()
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Focus mode").Grow(1)
			ui.Switch(c.Key("focus"), &a.focus).Label("Focus mode")
		})
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Sound").Grow(1)
			ui.Switch(c.Key("sound"), &a.sound).Label("Sound")
		})
		ui.Textf(c, "Sessions of %d minutes", int(a.minutes)).FontSize(12).TextColor(t.TextMuted)
		ui.Slider(c.Key("minutes"), &a.minutes, 5, 60).Label("Session length")
		ui.Spacer(c)
		ui.Row(c).Children(func() {
			ui.Spacer(c)
			ui.Button(c.Key("quit"), "Quit").OnClick(mygo.App.Quit)
		})
	})
}

// toggle opens the panel next to the icon, or closes it.
func (a *app) toggle() {
	if a.panel != nil {
		a.panel.Close()
		return
	}
	a.panel = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      a.tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     280,
		Height:    230,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(a.view),
	})
	a.panel.OnClosed(func() { a.panel = nil })
}

func (a *app) ready() {
	var err error
	a.tray, err = mygo.NewTray(mygo.TrayOptions{
		Icon:           trayIcon(),
		IconIsTemplate: true,
		ToolTip:        "Focus",
	})
	if err != nil {
		log.Fatal("tray: ", err)
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open Focus", Click: func(*mygo.MenuItem, *mygo.Window) { a.toggle() }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
	if runtime.GOOS == "linux" {
		// AppIndicator only shows menus, and tells nothing of clicks.
		a.tray.SetMenu(menu)
	} else {
		a.tray.OnClick(a.toggle)
		a.tray.OnRightClick(func() {
			if a.panel != nil {
				a.panel.Close()
			}
			// The menu for this popup only, so that clicks keep toggling
			// the panel.
			a.tray.SetMenu(menu)
			defer a.tray.SetMenu(nil)
			a.tray.PopUpMenu()
		})
	}
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.tray.Destroy() })
}

// trayIcon draws a target, black on transparent for macOS to tint, blue
// elsewhere, which shows on light and dark taskbars.
func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{R: 79, G: 124, B: 255, A: 255}
	if runtime.GOOS == "darwin" {
		ink = color.NRGBA{A: 255}
	}
	for y := range 32 {
		for x := range 32 {
			dx, dy := float64(x)-15.5, float64(y)-15.5
			d := dx*dx + dy*dy
			if d < 15*15 && d > 11*11 || d < 7*7 && d > 4*4 || d < 1.5*1.5 {
				img.Set(x, y, ink)
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	a := &app{sound: true, minutes: 25}
	mygo.App.SetName("Focus")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	// The panel is the app's only window: keep running as it closes.
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(a.ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
