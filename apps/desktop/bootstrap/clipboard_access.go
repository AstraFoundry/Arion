package bootstrap

import (
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// File history is enabled only after an explicit settings confirmation.
func (a *App) showClipboardAccess() {
	cc := a.clipboard
	if cc == nil || !a.state.Preferences.Clipboard.Enabled || a.updateApplying() {
		return
	}
	a.dismissClipboardAccess()
	a.state.Clipboard.FileConfirmation = true
	var window *mygo.Window
	layout := &clipboardAccessLayout{}
	view := uiapp.ClipboardAccess(a.ui, func(next int) {
		if window == nil || (layout.revealed && layout.height == next) {
			return
		}
		a.post(func() {
			if cc.accessWindow == window {
				layout.apply(window, next)
			}
		})
	})
	area := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint()).WorkArea
	window = mygo.NewWindow(mygo.WindowOptions{Title: a.ui.Copy().Text(copy.ClipboardFilesEnabled),
		Width: uiapp.ClipboardAccessWidth, Height: uiapp.ClipboardAccessHeight,
		Frameless: true, Transparent: true, AlwaysOnTop: true, SkipTaskbar: true,
		DisableResize: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true,
		Hidden: true, Content: view.Content})
	window.SetBounds(mygo.Rectangle{X: area.X + (area.Width-uiapp.ClipboardAccessWidth)/2,
		Y: area.Y + (area.Height-uiapp.ClipboardAccessHeight)/2, Width: uiapp.ClipboardAccessWidth, Height: uiapp.ClipboardAccessHeight})
	configureClipboardAccessWindow(window)
	cc.accessWindow = window
	a.track(window, view, func() {
		if cc.accessWindow == window {
			cc.accessWindow = nil
			a.state.Clipboard.FileConfirmation = false
		}
	})
	// Let native frames measure content without exposing provisional geometry.
	window.SetOpacity(0)
	window.ShowInactive()
}

func (a *App) dismissClipboardAccess() {
	cc := a.clipboard
	if cc == nil {
		return
	}
	window := cc.accessWindow
	cc.accessWindow = nil
	a.state.Clipboard.FileConfirmation = false
	if window != nil {
		window.Close()
	}
}

func (x actions) SetClipboardFilesEnabled(on bool) {
	a := x.a
	if !a.state.Preferences.Clipboard.Enabled || a.updateApplying() {
		return
	}
	if on {
		if !a.state.Preferences.Clipboard.FilesEnabled {
			a.showClipboardAccess()
		}
		return
	}
	a.setClipboardFiles(false)
}

func (a *App) setClipboardFiles(on bool) bool {
	previous := a.state.Preferences.Clipboard
	next := a.state.Preferences
	next.Clipboard.FilesEnabled = on
	if !a.savePreferences(next) {
		a.state.Clipboard.Error = copy.ClipboardPreferencesSaveFailed
		return false
	}
	a.dismissClipboardAccess()
	a.state.Clipboard.Error = ""
	a.configureClipboard(previous)
	return true
}

func (x actions) ConfirmClipboardFiles() {
	a := x.a
	if !a.state.Clipboard.FileConfirmation || !a.state.Preferences.Clipboard.Enabled || a.updateApplying() {
		return
	}
	a.setClipboardFiles(true)
}

func (x actions) DismissClipboardAccess() { x.a.dismissClipboardAccess() }

func (x actions) OpenClipboardFileSettings() {
	if x.a.platform != nil {
		x.a.platform.OpenGuidePane(arion.GuideFilesAndFolders)
	}
	x.a.state.Clipboard.Error = copy.ClipboardAccessReCopy
}
