package bootstrap

import (
	"arion.local/desktop/infrastructure/clipboardos"
	uiapp "arion.local/desktop/ui/app"
	"github.com/egoist/mygo"
)

// Geometry affects placement only. The captured target independently decides
// whether activation may attempt a paste, and is checked again before writing.
func clipboardTargetPresentation(target clipboardos.Target, pointer mygo.Rectangle) (anchor mygo.Rectangle, copyOnly bool) {
	if target == nil {
		return pointer, true
	}
	copyOnly = target.Check() != nil
	anchor = pointer
	if x, y, width, height, ok := target.Anchor(); ok {
		anchor = mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
	}
	return anchor, copyOnly
}

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
