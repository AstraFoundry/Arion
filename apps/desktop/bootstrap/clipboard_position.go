package bootstrap

import (
	uiapp "arion.local/desktop/ui/app"
	"github.com/egoist/mygo"
)

// Prefer below the input, flip above at the bottom edge, then constrain the
// window to its monitor's usable area. All coordinates are screen DIPs.
func clipboardBounds(anchor, area mygo.Rectangle, width, height int) mygo.Rectangle {
	width = min(width, area.Width)
	height = min(height, area.Height)
	x := min(max(anchor.X, area.X), area.X+area.Width-width)
	y := anchor.Y + anchor.Height + 4
	if y+height > area.Y+area.Height {
		y = anchor.Y - height - 4
	}
	y = min(max(y, area.Y), area.Y+area.Height-height)
	return mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
}

func (a *App) positionClipboard() {
	if a.clipboardWin == nil || a.clipboard == nil {
		return
	}
	anchor := a.clipboard.anchor
	bounds := clipboardBounds(anchor, mygo.Screen.DisplayMatching(anchor).WorkArea, uiapp.ClipboardWidth, uiapp.ClipboardHeight(a.state.Clipboard.Items))
	a.clipboardWin.SetContentSize(bounds.Width, bounds.Height)
}
