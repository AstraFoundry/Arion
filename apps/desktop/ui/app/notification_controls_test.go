package app

import (
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestNotificationControlsUseSystemSettings(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(language), func(t *testing.T) {
			state, _, store := fixture()
			state.Preferences.UILanguage = language
			state.Account.Phase = runtime.AccountSignedIn
			state.Update = runtime.Update{Phase: runtime.UpdateStaged, Version: "0.2.0"}
			messages := store.Copy()
			for _, section := range []string{settings.General, settings.About} {
				page := settings.NewPage()
				page.Section = section
				tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
				for _, permission := range []runtime.UpdateNotificationPermission{
					runtime.UpdateNotificationsUnknown, runtime.UpdateNotificationsAllowed,
					runtime.UpdateNotificationsDenied, runtime.UpdateNotificationsUnsupported,
					runtime.UpdateNotificationsFailed,
				} {
					state.UpdateNotifications.Permission = permission
					settle(tt)
					for _, obsolete := range []string{"Product notifications", "System update reminders", "产品通知", "系统更新提醒"} {
						if hasText(tt, obsolete) {
							t.Fatalf("%s permission %s still exposes %q", section, permission, obsolete)
						}
					}
					if section == settings.General {
						for _, key := range []copy.Key{copy.LanguageLabel, copy.MicPermission, copy.AXPermission, copy.LaunchAtLogin, copy.Sounds} {
							if !hasText(tt, messages.Text(key)) {
								t.Fatalf("General lost %s with permission %s", key, permission)
							}
						}
					} else {
						for _, key := range []copy.Key{copy.UpdateAuto, copy.UpdateAutoInstall, copy.UpdatePrerelease, copy.UpdateRestart} {
							if !hasText(tt, messages.Text(key)) {
								t.Fatalf("About lost %s with permission %s", key, permission)
							}
						}
					}
				}
				if section == settings.About {
					reveal(t, tt, messages.Text(copy.UpdatePrerelease))
				}
				snapshot(t, tt, section+"-os-notifications-"+string(language))
			}
		})
	}
}
