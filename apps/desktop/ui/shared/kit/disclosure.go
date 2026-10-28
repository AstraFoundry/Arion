package kit

import (
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Disclosure reveals measured content below a separate control. Closing keeps
// the content mounted but disables input; reversing retargets the current height.
func Disclosure(c *ui.Context, open bool, body func()) {
	panel := ui.Column(c).Shrink(0).ClipY().Disabled(!open)
	target := float32(0)
	if open {
		target = 1
	}
	progress := panel.AnimateTo("reveal", target, theme.Disclosure)
	if !open && progress == 0 {
		panel.Height(0)
		return
	}
	var content ui.Element
	panel.Opacity(progress).Children(func() {
		content = ui.Column(c).Shrink(0).Children(body)
	})
	if progress < 1 {
		panel.Height(progress * content.Bounds().H)
	}
}
