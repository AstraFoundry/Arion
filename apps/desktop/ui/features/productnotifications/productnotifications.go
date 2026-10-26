// Package productnotifications owns the user's account-notice preference.
package productnotifications

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	status := store.State.ProductNotifications
	permission := store.State.UpdateNotifications.Permission
	enabled := store.State.Preferences.ProductNotifications
	detail := copy.ProductNotificationsDetail
	switch {
	case !status.Supported:
		detail = copy.ProductNotificationsUnsupported
	case store.State.Account.Phase != runtime.AccountSignedIn:
		detail = copy.ProductNotificationsSignIn
	case permission == runtime.UpdateNotificationsDenied:
		detail = copy.ProductNotificationsDenied
	case permission == runtime.UpdateNotificationsUnsupported:
		detail = copy.ProductNotificationsUnavailable
	case permission == runtime.UpdateNotificationsFailed || status.Failed:
		detail = copy.ProductNotificationsFailed
	case status.Uncertain:
		detail = copy.ProductNotificationsUncertain
	}
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.ProductNotificationsLabel), messages.Text(detail), func() {
			disabled := !enabled && (!status.Supported || store.State.Account.Phase != runtime.AccountSignedIn || permission == runtime.UpdateNotificationsUnsupported)
			if ui.Switch(c.Key("product-notifications"), &enabled).Label(messages.Text(copy.ProductNotificationsLabel)).Disabled(disabled).Changed() {
				if enabled {
					store.Actions.EnableProductNotifications()
				} else {
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.ProductNotifications = false })
				}
			}
		})
		if status.Supported && store.State.Account.Phase == runtime.AccountSignedIn && (permission == runtime.UpdateNotificationsDenied || permission == runtime.UpdateNotificationsFailed || status.Failed) {
			ui.Row(c).Padding(0, 16, 12, 16).Children(func() {
				if kit.Button(c.Key("retry-product-notifications"), tokens, kit.Outline, messages.Text(copy.Retry), true).Clicked() {
					store.Actions.EnableProductNotifications()
				}
			})
		}
	})
}
