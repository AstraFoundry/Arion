package bootstrap

import (
	uiapp "arion.local/desktop/ui/app"
	"github.com/egoist/mygo"
)

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
func (a *App) presentSelection(anchor mygo.Rectangle, hasAnchor bool) {
	if a.views == nil {
		return
	}
	cursor := mygo.Point{}
	if !hasAnchor {
		cursor = mygo.Screen.CursorScreenPoint()
	}
	anchor = selectionPlacement(anchor, hasAnchor, cursor)
	height := uiapp.SelectionHeightFor(a.ui)
	a.selectionAnchor = anchor
	if a.selectionWin != nil {
		a.selectionWin.SetAnchor(a.selectionAnchor)
		a.resizeSelection(a.selectionWidth, height)
		if !a.selectionWin.IsVisible() {
			a.selectionWin.ShowInactive()
		}
		return
	}
	view := uiapp.Selection(a.ui, a.resizeSelection)
	if a.selectionWidth <= 0 {
		a.selectionWidth = uiapp.SelectionWidth
	}
	a.selectionHeight = height
	a.selectionWin = mygo.NewFlyout(mygo.FlyoutOptions{
		ScreenAnchor: &a.selectionAnchor, Placement: mygo.PlacementBottomStart, Gap: 4,
		Width: a.selectionWidth, Height: height, Focusable: true, Shadow: true, Hidden: true, Content: view.Content,
	})
	configureOverlay(a.selectionWin)
	window := a.selectionWin
	a.track(window, view, func() {
		if a.selectionWin == window {
			a.selectionWin = nil
			a.selectionHeight = 0
		}
	})
	window.ShowInactive()
}

// resizeSelection fits the popover to the size its view reports: the bar's
// own width, so the window has no empty space beside it. A zero width, from
// a bar not laid out yet, keeps the width the last bar measured.
func (a *App) resizeSelection(width, height int) {
	if width <= 0 {
		width = a.selectionWidth
	}
	if a.selectionWin == nil || width <= 0 || height <= 0 || width == a.selectionWidth && height == a.selectionHeight {
		return
	}
	a.selectionWidth, a.selectionHeight = width, height
	a.selectionWin.SetContentSize(width, height)
}

func (a *App) dismissSelection() {
	if a.selectionWin != nil {
		a.selectionWin.Close()
	}
}
