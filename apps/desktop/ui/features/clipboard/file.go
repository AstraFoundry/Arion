package clipboard

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// File references keep their full accessible label on the containing button.
func fileContent(c *ui.Context, item runtime.Clip, tokens theme.Tokens, messages copy.Catalog) {
	name, tag := clipTitle(item, messages), ""
	if len(item.Payload.Files) == 1 {
		ext := filepath.Ext(name)
		if stem := strings.TrimSuffix(name, ext); stem != "" && len(ext) > 1 && utf8.RuneCountInString(ext) <= 9 {
			name, tag = stem, strings.ToUpper(strings.TrimPrefix(ext, "."))
		}
	} else if len(item.Payload.Files) > 1 {
		tag = messages.Count(len(item.Payload.Files))
	}
	ui.Row(c).Grow(1).Shrink(1).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.File).Size(16, 16).TextColor(tokens.Muted)
		ui.Text(c, name).SingleLine().FontSize(theme.Body).TextColor(tokens.Text).Grow(1).Shrink(1)
		if tag != "" {
			ui.Row(c).Padding(2, 4).Radius(theme.RadiusSM).Background(tokens.Subtle).Children(func() {
				ui.Text(c, tag).SingleLine().FontSize(theme.Caption).TextColor(tokens.Muted)
			})
		}
	})
}
