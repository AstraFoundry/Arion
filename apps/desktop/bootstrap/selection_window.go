package bootstrap

import (
	"math"

	uiapp "arion.local/desktop/ui/app"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// selectionPopup is a window the bar opened beside it for a menu or a
// failure line. leaving is set once the bar wants another popup: the
// window then plays its exit and closes.
type selectionPopup struct {
	key     string
	popup   uiapp.SelectionPopup
	window  *mygo.Window
	above   bool
	leaving bool
}

// selectionPlacement is the popover anchor in screen DIPs. The selection
// rectangle wins. Without one, the anchor is the pointer.
func selectionPlacement(bounds mygo.Rectangle, hasBounds bool, cursor mygo.Point) mygo.Rectangle {
	if hasBounds {
		return bounds
	}
	return mygo.Rectangle{X: cursor.X, Y: cursor.Y}
}

// presentSelection hosts uiapp.Selection beside the selection. It does not
// activate Arion, so the editor keeps the focused field the action rechecks.
// The bar's window keeps its height: its menus open in windows of their own.
func (a *App) presentSelection(anchor mygo.Rectangle, hasAnchor bool) {
	if a.views == nil {
		return
	}
	cursor := mygo.Point{}
	if !hasAnchor {
		cursor = mygo.Screen.CursorScreenPoint()
	}
	a.selectionAnchor = selectionPlacement(anchor, hasAnchor, cursor)
	if a.selectionWin != nil {
		a.selectionWin.SetAnchor(a.selectionAnchor)
		if !a.selectionWin.IsVisible() {
			a.selectionWin.ShowInactive()
		}
		return
	}
	views := uiapp.Selection(a.ui, a.syncSelection)
	if a.selectionWidth <= 0 {
		a.selectionWidth = uiapp.SelectionWidth
	}
	a.selectionViews = views
	a.selectionWin = mygo.NewFlyout(mygo.FlyoutOptions{
		ScreenAnchor: &a.selectionAnchor, Placement: mygo.PlacementBottomStart, Gap: uiapp.SelectionGap,
		Width: a.selectionWidth, Height: uiapp.SelectionHeight, Focusable: true, Shadow: true, Hidden: true, Content: views.Bar.Content,
	})
	configureOverlay(a.selectionWin)
	window := a.selectionWin
	a.track(window, views.Bar, func() {
		if a.selectionWin == window {
			a.selectionWin = nil
			a.selectionPopups = nil
		}
	})
	window.ShowInactive()
}

// syncSelection follows the bar after each frame of it or a popup: it fits
// the bar window to the bar's width, redraws the other windows when the
// menus changed, and opens or retires the popup the bar wants. A zero
// width, from a bar not laid out yet, keeps the width the last bar
// measured.
func (a *App) syncSelection(width int, popup uiapp.SelectionPopup, revision int) {
	if a.selectionWin == nil {
		return
	}
	if width > 0 && width != a.selectionWidth {
		a.selectionWidth = width
		a.selectionWin.SetContentSize(width, uiapp.SelectionHeight)
	}
	if revision != a.selectionRevision {
		a.selectionRevision = revision
		a.selectionWin.Invalidate()
		for _, p := range a.selectionPopups {
			p.window.Invalidate()
		}
	}
	a.placeSelectionPopup(popup)
}

// placeSelectionPopup opens the popup the bar wants in a window of its
// own, a flyout of the bar's window. MyGo places it under the bar, or
// above it where the screen has no room below, lined up with the control
// that opened it. The window is created at the popup's size and keeps it,
// so nothing resizes as a menu comes and goes. The popup it replaces
// leaves first.
func (a *App) placeSelectionPopup(want uiapp.SelectionPopup) {
	var current *selectionPopup
	for _, p := range a.selectionPopups {
		if !p.leaving {
			current = p
		}
	}
	if current != nil && current.key == want.Key && current.popup == want {
		return
	}
	if current != nil {
		current.leaving = true
		current.window.Invalidate()
	}
	if want.Key == "" || want.Width <= 0 || want.Height <= 0 {
		return
	}
	placement := mygo.PlacementBottomStart
	if want.End {
		placement = mygo.PlacementBottomEnd
	}
	p := &selectionPopup{key: want.Key, popup: want}
	view := a.selectionViews.Popup(want.Key, func() bool { return p.above }, func() {
		a.post(func() {
			p.leaving = true
			p.window.Close()
		})
	})
	p.window = mygo.NewFlyout(mygo.FlyoutOptions{
		Parent: a.selectionWin, Anchor: popupAnchor(want.Anchor), Placement: placement, Gap: uiapp.SelectionGap,
		Width: want.Width, Height: want.Height, Focusable: want.Focusable, Shadow: true, Hidden: true, Content: view.Content,
	})
	configureOverlay(p.window)
	p.above = p.window.Bounds().Y < a.selectionWin.Bounds().Y
	// The system closes a focusable popup on a press on its anchor or on
	// Escape: the bar closes the menu instead, and the popup leaves.
	p.window.OnClose(func(e *mygo.CloseEvent) {
		if !p.leaving {
			e.PreventDefault()
			p.leaving = true
			a.selectionViews.Dismiss(p.key)
			a.invalidate()
		}
	})
	window := p.window
	a.track(window, view, func() {
		for i, q := range a.selectionPopups {
			if q == p {
				a.selectionPopups = append(a.selectionPopups[:i], a.selectionPopups[i+1:]...)
				break
			}
		}
	})
	a.selectionPopups = append(a.selectionPopups, p)
	window.ShowInactive()
}

// popupAnchor is a popup's anchor in the bar window's whole DIPs.
func popupAnchor(r ui.Rect) mygo.Rectangle {
	x, y := int(math.Floor(float64(r.X))), int(math.Floor(float64(r.Y)))
	return mygo.Rectangle{X: x, Y: y, Width: int(math.Ceil(float64(r.X+r.W))) - x, Height: int(math.Ceil(float64(r.Y+r.H))) - y}
}

// selectionWindows are the bar's window and its popups, where a press
// chooses in the bar rather than ends a selection.
func (a *App) selectionWindows() []*mygo.Window {
	if a.selectionWin == nil {
		return nil
	}
	windows := []*mygo.Window{a.selectionWin}
	for _, p := range a.selectionPopups {
		windows = append(windows, p.window)
	}
	return windows
}

func (a *App) dismissSelection() {
	if a.selectionWin != nil {
		a.selectionWin.Close()
	}
}
