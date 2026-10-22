// Package overlay is the transparent, click-through window holding the
// dictation pill above every app.
package overlay

import (
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/capsule"
	"github.com/egoist/mygo/ui"
)

// Page keeps the overlay's state across frames.
type Page struct {
	pill capsule.State
}

func NewPage() *Page { return &Page{} }

// Render draws the overlay view.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	capsule.Render(c, store, tokens, &p.pill)
}
