package update

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Reminder is hosted in its own nonactivating window after input is complete.
// The controller decides when it is eligible; this surface never installs.
func Reminder(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	u := store.State.Update
	if !u.ReminderVisible || (u.Phase != runtime.UpdateAvailable && u.Phase != runtime.UpdateStaged && u.Phase != runtime.UpdateReady) {
		return
	}
	messages := store.Copy()
	title := messages.UpdateAvailable(u.Version)
	if u.Phase == runtime.UpdateStaged {
		title = messages.UpdateDownloaded(u.Version)
	} else if u.Phase == runtime.UpdateReady {
		title = messages.UpdateReadyVersion(u.Version)
	}
	ui.Column(c).Fill().Padding(16).Gap(12).Radius(theme.RadiusCard).Background(tokens.Card).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			kit.Mark(c, 28)
			ui.Text(c, title).FontSize(theme.Body).Grow(1).Shrink(1)
			if kit.IconButton(c.Key("dismiss-update"), tokens, icons.X, messages.Text(copy.UpdateDismiss)).Clicked() {
				store.Actions.DismissUpdateReminder()
			}
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			later := kit.Button(c.Key("later-update"), tokens, kit.Ghost, messages.Text(copy.UpdateLater), true)
			kit.Tooltip(c, tokens, later, messages.Text(copy.UpdateLaterDetail))
			if later.Clicked() {
				store.Actions.LaterUpdate()
			}
			if kit.Button(c.Key("view-update"), tokens, kit.Outline, messages.Text(copy.UpdateView), true).Clicked() {
				store.Actions.OpenUpdate()
			}
		})
	})
}
