// Package foundation presents the shell's actual implementation status.
package foundation

import (
	"arion.local/desktop/ui/shared/copy"
	"github.com/egoist/mygo/ui"
)

func Render(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(16).Padding(24).Children(func() {
		ui.Text(c, copy.Title).FontSize(26).Bold()
		ui.Text(c, copy.FoundationStatus).FontSize(16)
		ui.Text(c, copy.FoundationDetail).FontSize(14)
	})
}
