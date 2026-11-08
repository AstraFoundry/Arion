package clipboard

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
)

// Access explains optional file history before the preference is enabled.
func Access(c *ui.Context, store runtime.Store, tokens theme.Tokens) int {
	if !store.State.Clipboard.FileConfirmation {
		return 0
	}
	if c.Shortcut(0, ui.KeyEscape) {
		store.Actions.DismissClipboardAccess()
		return 0
	}
	messages := store.Copy()
	card := ui.Column(c).FillWidth().Shrink(0).Padding(20).Gap(12).Radius(theme.RadiusCard).Background(tokens.Card)
	card.Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, messages.Text(copy.ClipboardFilesTitle)).FontSize(theme.Heading).FontWeight(600).Grow(1).Shrink(1)
			if kit.IconButton(c.Key("dismiss-access"), tokens, icons.X, messages.Text(copy.ClipboardCancel)).Clicked() {
				store.Actions.DismissClipboardAccess()
			}
		})
		ui.Text(c, messages.Text(copy.ClipboardFilesExplanation)).FontSize(theme.Body).TextColor(tokens.Muted)
		var help copy.Key
		switch {
		case strings.HasPrefix(store.State.Platform, "darwin-"):
			help = copy.ClipboardFilesPermissionsMac
		case strings.HasPrefix(store.State.Platform, "windows-"):
			help = copy.ClipboardFilesPermissionsWindows
		}
		if help != "" {
			ui.Text(c, messages.Text(help)).FontSize(theme.Body).TextColor(tokens.Muted)
		}
		if store.State.Clipboard.Error == copy.ClipboardPreferencesSaveFailed {
			kit.Notice(c, tokens, tokens.Danger, messages.Text(copy.ClipboardPreferencesSaveFailed), nil)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if kit.Button(c.Key("cancel-access"), tokens, kit.Ghost, messages.Text(copy.ClipboardCancel), true).Clicked() {
				store.Actions.DismissClipboardAccess()
			}
			if kit.Button(c.Key("confirm-access"), tokens, kit.Primary, messages.Text(copy.ClipboardFilesConfirm), true).Clicked() {
				store.Actions.ConfirmClipboardFiles()
			}
		})
	})
	return int(math.Ceil(float64(card.Bounds().H)))
}

func accessSettings(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	kit.Divider(c, tokens)
	kit.TipRow(c, tokens, messages.Text(copy.ClipboardFilesEnabled), messages.Text(copy.ClipboardFilesHint), func() {
		on := store.State.Preferences.Clipboard.FilesEnabled
		if ui.Switch(c.Key("files-enabled"), &on).Label(messages.Text(copy.ClipboardFilesEnabled)).Changed() {
			store.Actions.SetClipboardFilesEnabled(on)
		}
	})
}
