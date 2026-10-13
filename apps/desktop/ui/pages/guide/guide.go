// Package guide is the borderless window that floats under System Settings
// while the user adds Arion to a privacy list.
package guide

import (
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Height sizes the window; its width follows the System Settings pane.
// The instruction row holds two lines, so wrapped copy keeps the layout.
const Height = 112

// Render draws the guide card over a transparent window.
func Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	permissions.Guide(c, store, tokens)
}
