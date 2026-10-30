// Package clipboard composes the clipboard resource picker.
package clipboard

import (
	clipboardfeature "arion.local/desktop/ui/features/clipboard"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

type Page struct{ panel clipboardfeature.Panel }

func (p *Page) Render(c *ui.Context, store runtime.Store) { p.panel.Render(c, store) }

const Width = clipboardfeature.Width

func Height(items []runtime.Clip) int { return clipboardfeature.Height(items) }

const PreviewWidth = clipboardfeature.PreviewWidth
const PreviewHeight = clipboardfeature.PreviewHeight

func Preview(c *ui.Context, item runtime.Clip, bitmap *ui.Bitmap) {
	clipboardfeature.Preview(c, item, bitmap)
}
