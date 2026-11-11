// Package selection is the popover window over another app's selected text.
package selection

import (
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/widgets/selectionmenu"
	"github.com/egoist/mygo/ui"
)

const Width = selectionmenu.Width

// Height sizes the window for the current offer.
func Height(store runtime.Store) int { return selectionmenu.Height(store.State.Selection.Actions) }

// Page keeps the menu's highlight and question.
type Page struct{ menu selectionmenu.Menu }

// Render draws the menu and returns the height it needs.
func (p *Page) Render(c *ui.Context, store runtime.Store) int { return p.menu.Render(c, store) }
