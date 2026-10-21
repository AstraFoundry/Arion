package bootstrap

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"runtime"

	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/features/update"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

func contextBackground() context.Context { return context.Background() }

// trayIcon draws Arion's mark, five rounded bars, black on transparent so
// macOS tints it; live bars stand taller while the microphone is open.
func trayIcon(live bool) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 36, 36))
	ink := color.NRGBA{A: 255}
	if runtime.GOOS != "darwin" {
		ink = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	heights := []int{10, 18, 26, 18, 10}
	if live {
		heights = []int{16, 26, 32, 22, 14}
	}
	for i, h := range heights {
		x0 := 5 + i*6
		y0 := (36 - h) / 2
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+3; x++ {
				// Round the bar ends.
				if (y == y0 || y == y0+h-1) && x != x0+1 {
					continue
				}
				img.Set(x, y, ink)
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func (a *App) createTray() {
	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(false), IconIsTemplate: runtime.GOOS == "darwin", ToolTip: a.ui.Copy().Text(copy.TrayToolTip)})
	if err != nil {
		return
	}
	a.tray = tray
	tray.OnClick(a.togglePanel)
	tray.OnRightClick(func() {
		a.closeTrayHint()
		if a.panel != nil {
			a.panel.Close()
		}
		label := a.ui.Copy().Text(copy.TrayStart)
		if a.state.Dictation.State == arion.StateCapturing {
			label = a.ui.Copy().Text(copy.TrayStop)
		}
		items := []*mygo.MenuItem{
			{Label: label, Click: func(*mygo.MenuItem, *mygo.Window) { a.core.Toggle() }},
			mygo.Separator(),
			{Label: a.ui.Copy().Text(copy.TraySettings), Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettings() }},
		}
		if notice, ok := update.Notice(a.ui); ok {
			items = append(items, &mygo.MenuItem{Label: notice, Click: func(*mygo.MenuItem, *mygo.Window) { actions{a}.OpenUpdate() }})
		}
		items = append(items, &mygo.MenuItem{Label: a.ui.Copy().Text(copy.TrayQuit), Accelerator: "CmdOrCtrl+Q", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }})
		menu := mygo.NewMenu(items)
		tray.SetMenu(menu)
		defer tray.SetMenu(nil)
		tray.PopUpMenu()
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { tray.Destroy() })
}

var trayLive bool

func (a *App) updateTray() {
	if a.tray == nil {
		return
	}
	live := a.state.Dictation.State == arion.StateCapturing
	if live != trayLive {
		trayLive = live
		_ = a.tray.SetIcon(trayIcon(live), runtime.GOOS == "darwin")
	}
}

func (a *App) togglePanel() {
	a.closeTrayHint()
	if a.panel != nil {
		a.panel.Close()
		return
	}
	a.peekScene()
	view := uiapp.Panel(a.ui)
	a.panel = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray: a.tray, Placement: mygo.PlacementBottom, Gap: 4, Width: uiapp.PanelWidth, Height: uiapp.PanelHeight,
		Focusable: true, Shadow: true, Popover: true, Content: view.Content,
	})
	a.track(a.panel, view, func() { a.panel = nil })
}

// showTrayHint points at the tray icon once onboarding finishes. It never
// takes the keyboard, and stays until dismissed or the icon is used.
func (a *App) showTrayHint() {
	if a.tray == nil || a.trayHint != nil {
		return
	}
	icon := a.tray.Bounds()
	area := mygo.Screen.DisplayMatching(icon).WorkArea
	a.state.TrayHint = arion.TrayHint{Below: icon.Y+icon.Height/2 > area.Y+area.Height/2}
	view := uiapp.TrayHint(a.ui)
	a.trayHint = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray: a.tray, Placement: mygo.PlacementBottom, Gap: 4, Width: uiapp.TrayHintWidth, Height: uiapp.TrayHintHeight,
		Shadow: true, Hidden: true, Content: view.Content,
	})
	a.track(a.trayHint, view, func() {
		a.trayHint = nil
		a.state.TrayHint = arion.TrayHint{}
	})
	a.trayHint.ShowInactive()
}

func (a *App) closeTrayHint() {
	if a.trayHint != nil {
		a.trayHint.Close()
	}
}

// track registers a window for invalidation and its close hook.
func (a *App) track(window *mygo.Window, view uiapp.View, closed func()) {
	a.views[window] = view
	window.OnClosed(func() {
		view.Close()
		delete(a.views, window)
		closed()
	})
}

// createOverlay makes the click-through window that hosts the pill. It never
// takes focus, so the user's editor keeps it while dictating.
func (a *App) createOverlay() {
	view := uiapp.Overlay(a.ui)
	a.overlay = mygo.NewWindow(mygo.WindowOptions{
		Title: a.ui.Copy().Text(copy.Title), Width: uiapp.OverlayWidth, Height: uiapp.OverlayHeight, Frameless: true, Transparent: true,
		AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true, DisableMinimize: true, DisableMaximize: true,
		DisableFullScreen: true, DisableShadow: true, Hidden: true, Content: view.Content,
	})
	a.overlay.SetIgnoreMouseEvents(true)
	a.overlay.SetVisibleOnAllWorkspaces(true)
	configureOverlay(a.overlay)
	a.track(a.overlay, view, func() { a.overlay = nil })
}

// positionOverlay places the overlay at the bottom of the display under the
// pointer, where the user is working, and shows it without activation.
func (a *App) positionOverlay() {
	if a.overlay == nil {
		return
	}
	display := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	area := display.WorkArea
	x := area.X + (area.Width-uiapp.OverlayWidth)/2
	y := area.Y + area.Height - uiapp.OverlayHeight
	a.overlay.SetBounds(mygo.Rectangle{X: x, Y: y, Width: uiapp.OverlayWidth, Height: uiapp.OverlayHeight})
	if !a.overlay.IsVisible() {
		a.overlay.ShowInactive()
	}
}

func (a *App) openSettings() {
	if a.panel != nil {
		a.panel.Close()
	}
	if a.settingsWin != nil {
		mygo.App.Focus()
		a.settingsWin.Focus()
		return
	}
	view := uiapp.Settings(a.ui)
	a.settingsWin = mygo.NewWindow(mygo.WindowOptions{
		Title: a.ui.Copy().Text(copy.SettingsTitle), Width: uiapp.SettingsWidth, Height: uiapp.SettingsHeight, MinWidth: 640, MinHeight: 440,
		TitleBarStyle: mygo.TitleBarHidden, Vibrancy: mygo.VibrancySidebar, StateKey: "settings", Content: view.Content,
	})
	a.track(a.settingsWin, view, func() { actions{a}.CancelInputMethodSetup(); a.settingsWin = nil })
	mygo.App.Focus()
	a.settingsWin.Focus()
}

func (a *App) openOnboarding() {
	if a.onboarding != nil {
		a.onboarding.Focus()
		return
	}
	view := uiapp.Onboarding(a.ui)
	a.onboarding = mygo.NewWindow(mygo.WindowOptions{
		Title: a.ui.Copy().Text(copy.Title), Width: uiapp.OnboardingWidth, Height: uiapp.OnboardingHeight, TitleBarStyle: mygo.TitleBarHidden,
		DisableResize: true, DisableMaximize: true, DisableFullScreen: true, Content: view.Content,
	})
	a.track(a.onboarding, view, func() { a.onboarding = nil })
	mygo.App.Focus()
	a.onboarding.Focus()
}

// updateLanguage refreshes native chrome without recreating windows or sessions.
func (a *App) updateLanguage() {
	messages := a.ui.Copy()
	for _, window := range []*mygo.Window{a.overlay, a.onboarding} {
		if window != nil {
			window.SetTitle(messages.Text(copy.Title))
		}
	}
	if a.settingsWin != nil {
		a.settingsWin.SetTitle(messages.Text(copy.SettingsTitle))
	}
	if a.tray != nil {
		a.tray.SetToolTip(messages.Text(copy.TrayToolTip))
	}
	a.invalidate()
}
