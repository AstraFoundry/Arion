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
		a.resizeSelection(height)
		if !a.selectionWin.IsVisible() {
			a.selectionWin.ShowInactive()
		}
		return
	}
	view := uiapp.Selection(a.ui, a.resizeSelection)
	a.selectionHeight = height
	a.selectionWin = mygo.NewFlyout(mygo.FlyoutOptions{
		ScreenAnchor: &a.selectionAnchor, Placement: mygo.PlacementBottomStart, Gap: 4,
		Width: uiapp.SelectionWidth, Height: height, Focusable: true, Shadow: true, Hidden: true, Content: view.Content,
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

func (a *App) resizeSelection(height int) {
	if a.selectionWin == nil || height <= 0 || height == a.selectionHeight {
		return
	}
	a.selectionHeight = height
	a.selectionWin.SetContentSize(uiapp.SelectionWidth, height)
}

func (a *App) dismissSelection() {
	if a.selectionWin != nil {
		a.selectionWin.Close()
	}
}
