package bootstrap

import (
	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardos"
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
	"runtime"
)

func (x actions) OpenClipboard() {
	a := x.a
	cc := a.clipboard
	if cc == nil || cc.delivering || a.updateApplying() {
		return
	}
	if a.clipboardWin != nil {
		a.clipboardWin.Close()
		return
	}
	if cc.target != nil {
		cc.target.Close()
	}
	cc.target = clipboardos.CaptureTarget()
	cursor := mygo.Screen.CursorScreenPoint()
	cc.anchor = mygo.Rectangle{X: cursor.X, Y: cursor.Y}
	a.state.Clipboard.CopyOnly = true
	if cc.target != nil {
		if ax, ay, aw, ah, ok := cc.target.Anchor(); ok {
			cc.anchor = mygo.Rectangle{X: ax, Y: ay, Width: aw, Height: ah}
			a.state.Clipboard.CopyOnly = false
		}
	}
	cc.windowSeq++
	if a.panel != nil {
		a.panel.Close()
	}
	if cc.service != nil && !a.state.Clipboard.Busy {
		limits := a.state.Preferences.Clipboard.Limits
		a.clipboardWork(func() ([]domain.Clip, error) { return refreshClipboard(cc.service, limits) }, copy.ClipboardLoadFailed)
	}
	view := uiapp.Clipboard(a.ui)
	bounds := clipboardBounds(cc.anchor, mygo.Screen.DisplayMatching(cc.anchor).WorkArea, uiapp.ClipboardWidth, uiapp.ClipboardHeight(a.state.Clipboard.Items))
	a.clipboardWin = mygo.NewFlyout(mygo.FlyoutOptions{
		ScreenAnchor: &cc.anchor, Placement: mygo.PlacementBottomStart, Gap: 4,
		Width: bounds.Width, Height: bounds.Height, Hidden: true, Focusable: true, Shadow: true, Content: view.Content,
	})
	configureClipboardWindow(a.clipboardWin)
	window := a.clipboardWin
	window.OnBlur(window.Close)
	a.track(a.clipboardWin, view, func() {
		cc.windowSeq++
		a.dismissClipboardPreview()
		a.clipboardWin = nil
		if !cc.delivering && cc.target != nil {
			cc.target.Close()
			cc.target = nil
		}
	})
	a.clipboardWin.Show()
	a.clipboardWin.Focus()
}
func (x actions) CloseClipboard() {
	if x.a.clipboardWin != nil {
		x.a.clipboardWin.Close()
	}
}

func (x actions) OpenClipboardSettings() {
	x.CloseClipboard()
	x.a.state.Focus = arion.SettingsFocus{Section: settings.Clipboard, Seq: x.a.state.Focus.Seq + 1}
	x.a.openSettings()
}
func (x actions) RetryClipboard() {
	if x.a.clipboard != nil {
		x.a.loadClipboard()
	}
}
func (x actions) SetClipboardShortcut(shortcut string) {
	a := x.a
	key, ok := application.ClipboardShortcutKey(runtime.GOOS, shortcut)
	p := a.state.Preferences
	if !ok || application.HoldKeysConflict(key, p.HoldKey) || application.HoldKeysConflict(key, p.TranslateKey) {
		a.state.Clipboard.Error = copy.ClipboardShortcutFailed
		return
	}
	x.UpdatePreferences(func(p *application.Preferences) { p.Clipboard.Shortcut = shortcut })
}
func (x actions) SetClipboardLimits(limits arion.ClipLimits) {
	a := x.a
	if !limits.Valid() {
		a.state.Clipboard.Error = copy.ClipboardInvalidLimits
		return
	}
	if a.clipboard == nil || a.state.Clipboard.Busy {
		return
	}
	if a.clipboard.service == nil {
		x.UpdatePreferences(func(p *application.Preferences) { p.Clipboard.Limits = limits })
		return
	}
	if _, err := domain.PruneClips(a.state.Clipboard.Items, limits); err != nil {
		a.state.Clipboard.Error = clipboardError(err, copy.ClipboardInvalidLimits)
		return
	}
	p := a.state.Preferences
	p.Clipboard.Limits = limits
	if !a.savePreferences(p) {
		a.state.Clipboard.Error = copy.ClipboardSaveFailed
		return
	}
	a.clipboardWork(func() ([]domain.Clip, error) { return refreshClipboard(a.clipboard.service, limits) }, copy.ClipboardSaveFailed)
}
func (x actions) PinClipboard(id string, pin bool) {
	a := x.a
	if a.clipboard == nil || a.clipboard.service == nil {
		return
	}
	limits := a.state.Preferences.Clipboard.Limits
	a.clipboardWork(func() ([]domain.Clip, error) { return a.clipboard.service.Pin(id, pin, limits) }, copy.ClipboardSaveFailed)
}
