//go:build arion_e2e && (darwin || windows)

package e2e

import (
	"path/filepath"
	"testing"
	"time"

	"arion.local/desktop/infrastructure/updatestate"
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/pages/settings"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// The native reminder uses the production host and view. Its synthetic release
// state never reaches a download, updater swap, relaunch or permission request.
func TestUpdateReminderNativeFocusAndSuppression(t *testing.T) {
	app.OpenSettings()
	var owned *mygo.Window
	eventually(t, "owned focus window opened", func() bool {
		owned = app.Windows()["settings"]
		return owned != nil && owned.IsVisible()
	})
	var before arion.Update
	var language arion.UILanguage
	var beta bool
	app.Update(func(s *arion.State) {
		before, language, beta = s.Update, s.Preferences.UILanguage, s.Preferences.AllowPrereleaseUpdates
	})
	defer func() {
		app.Update(func(s *arion.State) {
			app.Store().Actions.DismissUpdateReminder()
			s.Update, s.Preferences.AllowPrereleaseUpdates = before, beta
			app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = language })
		})
		owned.Close()
	}()
	for _, locale := range []arion.UILanguage{arion.LanguageEnglish, arion.LanguageChinese} {
		for _, phase := range []arion.UpdatePhase{arion.UpdateAvailable, arion.UpdateStaged, arion.UpdateReady} {
			app.Update(func(s *arion.State) {
				app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = locale })
				s.Preferences.AllowPrereleaseUpdates = false
				s.Update = arion.Update{Phase: phase, Version: "8.7.1"}
			})
			owned.Focus()
			eventually(t, "owned window focused before reminder", owned.IsFocused)
			app.ShowUpdateReminderForTest()
			var reminder *mygo.Window
			eventually(t, "native update reminder shown", func() bool {
				reminder = app.Windows()["update-reminder"]
				return reminder != nil && reminder.IsVisible()
			})
			if reminder.IsFocused() || !owned.IsFocused() {
				t.Fatal("passive update reminder stole owned window focus")
			}
			eventually(t, "native reminder content drawn", func() bool {
				return nonblank(capture(t, reminder, "update-reminder-"+string(locale)+"-"+string(phase), uiapp.UpdateReminderWidth, uiapp.UpdateReminderHeight))
			})
			app.Update(func(s *arion.State) { app.Store().Actions.LaterUpdate() })
			eventually(t, "Later closed reminder", reminder.IsDestroyed)
			if !owned.IsFocused() {
				t.Fatal("Later changed owned window focus")
			}
		}
	}
	saved, err := (updatestate.Store{Path: filepath.Join(profileDir, "updates.json")}).Load()
	if err != nil {
		t.Fatal(err)
	}
	later := false
	for _, r := range saved.Reminders {
		if r.Key == "arion.update.stable.8.7.1.ready" {
			later = r.Later.After(time.Now().Add(23*time.Hour)) && r.Later.Before(time.Now().Add(25*time.Hour))
		}
	}
	if !later {
		t.Fatalf("Later decision did not survive disk reload: %+v", saved.Reminders)
	}
	app.Update(func(s *arion.State) { s.Update = arion.Update{Phase: arion.UpdateAvailable, Version: "8.7.2"} })
	app.ShowUpdateReminderForTest()
	var reminder *mygo.Window
	eventually(t, "dismiss reminder shown", func() bool {
		reminder = app.Windows()["update-reminder"]
		return reminder != nil && reminder.IsVisible()
	})
	app.Update(func(s *arion.State) { app.Store().Actions.DismissUpdateReminder() })
	eventually(t, "Dismiss closed reminder", reminder.IsDestroyed)
	saved, err = (updatestate.Store{Path: filepath.Join(profileDir, "updates.json")}).Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range saved.Reminders {
		if r.Key == "arion.update.stable.8.7.2.available" && r.Dismissed {
			return
		}
	}
	t.Fatalf("Dismiss decision did not survive disk reload: %+v", saved.Reminders)
}

func TestUpdateSettingsNativeStates(t *testing.T) {
	app.OpenSettings()
	var window *mygo.Window
	eventually(t, "update settings opened", func() bool { window = app.Windows()["settings"]; return window != nil && window.IsVisible() })
	var before arion.Update
	var language arion.UILanguage
	app.Update(func(s *arion.State) { before, language = s.Update, s.Preferences.UILanguage })
	defer func() {
		app.Update(func(s *arion.State) {
			s.Update = before
			app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = language })
		})
		window.Close()
	}()
	// The taller owned window puts the full update card in each native capture.
	window.SetSize(820, 900)
	eventually(t, "update settings resized", func() bool { w, h := window.Size(); return w == 820 && h == 900 })
	for _, locale := range []arion.UILanguage{arion.LanguageEnglish, arion.LanguageChinese} {
		for _, step := range []struct {
			name  string
			state arion.Update
		}{
			{"staged", arion.Update{Phase: arion.UpdateStaged, Version: "8.7.3"}},
			{"applying", arion.Update{Phase: arion.UpdateApplying, Version: "8.7.3"}},
			{"queued", arion.Update{Phase: arion.UpdateStaged, Version: "8.7.3", RestartQueued: true}},
		} {
			app.Update(func(s *arion.State) {
				app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) { p.UILanguage = locale })
				s.Update = step.state
				s.Focus = arion.SettingsFocus{Section: settings.About, Seq: s.Focus.Seq + 1}
			})
			time.Sleep(350 * time.Millisecond)
			if !nonblank(capture(t, window, "settings-update-"+string(locale)+"-"+step.name, 760, 800)) {
				t.Fatalf("%s %s: update settings blank", locale, step.name)
			}
			if step.state.RestartQueued {
				app.Update(func(s *arion.State) { app.Store().Actions.CancelUpdateRestart() })
				var queued bool
				app.Update(func(s *arion.State) { queued = s.Update.RestartQueued })
				if queued {
					t.Fatal("queued restart could not be canceled")
				}
			}
		}
	}
}
