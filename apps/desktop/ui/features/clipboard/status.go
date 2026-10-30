package clipboard

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func Status(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	s := store.State.Clipboard
	if s.Error != "" {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, store.Copy().Text(s.Error)).FontSize(theme.Caption).TextColor(tokens.Danger).Grow(1).Shrink(1)
			if s.Error == copy.ClipboardLoadFailed && kit.Button(c, tokens, kit.Ghost, store.Copy().Text(copy.Retry), !s.Busy).Clicked() {
				store.Actions.RetryClipboard()
			}
		})
	}
}
