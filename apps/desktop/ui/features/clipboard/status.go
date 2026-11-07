package clipboard

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
	"strings"
)

func Status(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	s := store.State.Clipboard
	if s.Error == "" {
		return
	}
	var retry func()
	if s.Error == copy.ClipboardLoadFailed {
		retry = func() {
			if kit.Button(c, tokens, kit.Ghost, store.Copy().Text(copy.Retry), !s.Busy).Clicked() {
				store.Actions.RetryClipboard()
			}
		}
	}
	if store.State.Preferences.Clipboard.FilesEnabled && strings.HasPrefix(store.State.Platform, "darwin-") && (s.Error == copy.ClipboardAccessRequired || s.Error == copy.ClipboardAccessReCopy) {
		retry = func() {
			if kit.Button(c, tokens, kit.Ghost, store.Copy().Text(copy.ClipboardAccessSettings), true).Clicked() {
				store.Actions.OpenClipboardFileSettings()
			}
		}
	}
	kit.Notice(c, tokens, tokens.Danger, store.Copy().Text(s.Error), retry)
}
