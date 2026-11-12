// Package selection is the bar window over another app's selected text.
package selection

import (
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/widgets/selectionmenu"
	"github.com/egoist/mygo/ui"
)

// Width sizes the window as an offer opens, until the bar has measured
// itself.
const Width = selectionmenu.Width

// Height sizes the window as an offer opens: the bar, with no menu open.
func Height(runtime.Store) int { return selectionmenu.BarHeight }

// Page keeps the bar's open menu, highlight and question.
type Page struct{ menu selectionmenu.Menu }

// Render draws the bar and returns the window size it needs.
func (p *Page) Render(c *ui.Context, store runtime.Store) (width, height int) {
	return p.menu.Render(c, store)
}
