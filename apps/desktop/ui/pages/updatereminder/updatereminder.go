// Package updatereminder composes the nonactivating post-input update surface.
package updatereminder

import (
	"arion.local/desktop/ui/features/update"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

const (
	Width  = 400
	Height = 132
)

func Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	update.Reminder(c, store, tokens)
}
