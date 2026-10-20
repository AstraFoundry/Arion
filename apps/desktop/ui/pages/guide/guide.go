// Package guide is the borderless window that floats under System Settings
// while the user adds Arion to a privacy list.
package guide

import (
	"arion.local/desktop/ui/features/keyboard"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Height sizes the window; its width follows the System Settings pane.
// The instruction row holds two lines, so wrapped copy keeps the layout.
const Height = 112

// HeightFor keeps switch-only instructions compact.
func HeightFor(pane runtime.GuidePane) int {
	if pane == runtime.GuideMicrophone || pane == runtime.GuideScreenRecording {
		return 64
	}
	return Height
}

// Render draws the guide card over a transparent window.
func Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	if store.State.Guide.Pane == runtime.GuideKeyboard {
		keyboard.Guide(c, store, tokens)
	} else {
		permissions.Guide(c, store, tokens)
	}
}
