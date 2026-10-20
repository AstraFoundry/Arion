package keyboard

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Guide explains the input-source controls; this pane does not accept an app
// bundle dragged into a privacy list.
func Guide(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	kit.SettingsGuide(c, tokens, store.State.Guide.Shown, messages.Text(copy.GuideKeyboardTitle), messages.Text(copy.GuideClose), store.Actions.CloseGuide, func() {
		ui.Text(c, messages.Text(copy.GuideKeyboardAdd)).FontSize(theme.Caption).FixedLineHeight(18).TextColor(tokens.Muted).Margin(0, 0, 0, 28).Shrink(1)
	})
}
