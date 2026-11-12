// Package selection is the bar window over another app's selected text,
// and the popup windows its menus open beside it.
package selection

import (
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/widgets/selectionmenu"
	"github.com/egoist/mygo/ui"
)

// Width sizes the bar window as an offer opens, until the bar has measured
// itself. Height is the bar window's height, with or without a menu open.
const (
	Width  = selectionmenu.Width
	Height = selectionmenu.BarHeight
	Gap    = selectionmenu.Gap
)

// Popup is the menu or failure line the bar shows in a window beside it.
type Popup = selectionmenu.Popup

// Surface is one popup window's view of the bar's menus.
type Surface = selectionmenu.Surface

// Page keeps the bar's open menu, highlight and question for the bar and
// its popups.
type Page struct{ menu selectionmenu.Menu }

// Render draws the bar and returns its width.
func (p *Page) Render(c *ui.Context, store runtime.Store) int { return p.menu.Render(c, store) }

// RenderPopup draws a popup window and reports when it has left.
func (p *Page) RenderPopup(c *ui.Context, store runtime.Store, s *Surface) bool {
	return p.menu.RenderPopup(c, store, s)
}

// Width is the bar's width as last laid out, or 0 before its first frame.
func (p *Page) Width() int { return p.menu.Width() }

// Want is the popup the bar shows now.
func (p *Page) Want() Popup { return p.menu.Want() }

// Revision changes whenever what the bar or a popup shows changes.
func (p *Page) Revision() int { return p.menu.Revision() }

// Dismiss closes the popup key that the system closed.
func (p *Page) Dismiss(key string) { p.menu.Dismiss(key) }
