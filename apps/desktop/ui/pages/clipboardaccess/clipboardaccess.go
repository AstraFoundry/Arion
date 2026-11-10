// Package clipboardaccess composes the file-history enable confirmation surface.
package clipboardaccess

import (
	"arion.local/desktop/ui/features/clipboard"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

const Width = 360
const Height = 200

func Render(c *ui.Context, store runtime.Store) int {
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	return clipboard.Access(c, store, tokens)
}
