package app

import (
	"arion.local/desktop/ui/features/productnotifications"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestProductNotificationPreferenceIsExplicit(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = language
		state.ProductNotifications.Supported = true
		state.Account.Phase = runtime.AccountSignedIn
		messages := store.Copy()
		tt := ui.NewTester(func(c *ui.Context) { productnotifications.Card(c, store, theme.Apply(c)) }, 490, 200)
		settle(tt)
		if !hasText(tt, messages.Text(copy.ProductNotificationsLabel)) || !hasText(tt, messages.Text(copy.ProductNotificationsDetail)) {
			t.Fatal(tt.Texts())
		}
		toggle := func(detail copy.Key) {
			t.Helper()
			label, ok := tt.Find(messages.Text(copy.ProductNotificationsLabel))
			if !ok {
				t.Fatal("missing preference")
			}
			text, ok := tt.Find(messages.Text(detail))
			if !ok {
				t.Fatal("missing detail")
			}
			tt.ClickAt(465, (label.Y+text.Y+text.H)/2)
		}
		toggle(copy.ProductNotificationsDetail)
		settle(tt)
		if !actions.called("enable-product-notifications") || state.Preferences.ProductNotifications {
			t.Fatal("switch bypassed explicit authorization")
		}
		state.Preferences.ProductNotifications = true
		settle(tt)
		toggle(copy.ProductNotificationsDetail)
		settle(tt)
		if state.Preferences.ProductNotifications {
			t.Fatal("switch did not disable")
		}
		for _, step := range []struct {
			name              string
			permission        runtime.UpdateNotificationPermission
			failed, uncertain bool
			copy              copy.Key
		}{{"denied", runtime.UpdateNotificationsDenied, false, false, copy.ProductNotificationsDenied}, {"failed", runtime.UpdateNotificationsAllowed, true, false, copy.ProductNotificationsFailed}, {"uncertain", runtime.UpdateNotificationsAllowed, false, true, copy.ProductNotificationsUncertain}} {
			state.UpdateNotifications.Permission = step.permission
			state.ProductNotifications.Failed = step.failed
			state.ProductNotifications.Uncertain = step.uncertain
			settle(tt)
			if !hasText(tt, messages.Text(step.copy)) {
				t.Fatalf("%s: %v", step.name, tt.Texts())
			}
			snapshot(t, tt, "product-notifications-"+string(language)+"-"+step.name)
		}
		state.ProductNotifications.Failed = false
		state.ProductNotifications.Uncertain = false
		state.Account.Phase = runtime.AccountSignedOut
		settle(tt)
		if !hasText(tt, messages.Text(copy.ProductNotificationsSignIn)) {
			t.Fatal("missing account requirement")
		}
		actions.calls = nil
		toggle(copy.ProductNotificationsSignIn)
		settle(tt)
		if actions.called("enable-product-notifications") {
			t.Fatal("signed-out account could enable")
		}
	}
}
