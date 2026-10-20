// Package trayhint is the card that floats beside the tray icon once
// onboarding finishes, showing where Arion stays.
package trayhint

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Width and Height size the window around the single-line card.
const (
	Width  = 260
	Height = 64
)

// Render draws the hint over a transparent window, its arrow toward the icon.
func Render(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	arrow := icons.ArrowUp
	if store.State.TrayHint.Below {
		arrow = icons.ArrowDown
	}
	kit.Callout(c, tokens, 0, arrow, messages.Text(copy.TrayHintHere), messages.Text(copy.GuideClose), store.Actions.CloseTrayHint, nil)
}
