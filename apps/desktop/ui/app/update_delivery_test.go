package app

import (
	"testing"

	"arion.local/desktop/ui/features/update"
	"arion.local/desktop/ui/pages/updatereminder"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func TestUpdateDeliveryStates(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = language
		messages := store.Copy()
		tt := ui.NewTester(func(c *ui.Context) { update.Card(c, store, theme.Apply(c)) }, 490, 480)
		for _, step := range []struct {
			update runtime.Update
			text   copy.Key
			button copy.Key
			call   string
		}{
			{runtime.Update{Phase: runtime.UpdateStaged, Version: "0.2.0"}, copy.UpdateStaged, copy.UpdateRestart, "restart-update"},
			{runtime.Update{Phase: runtime.UpdateStaged, Version: "0.2.0", RestartQueued: true}, copy.UpdateRestartQueued, copy.UpdateCancelRestart, "cancel-update-restart"},
			{runtime.Update{Phase: runtime.UpdateApplying, Version: "0.2.0"}, copy.UpdateApplying, "", ""},
			{runtime.Update{Phase: runtime.UpdateReady, Version: "0.2.0"}, copy.UpdateReady, copy.UpdateRestart, "restart-update"},
			{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepPrepare, Version: "0.2.0"}, copy.UpdatePrepareFailed, copy.Retry, "install-update"},
			{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepApply, Version: "0.2.0"}, copy.UpdateApplyFailed, copy.Retry, "restart-update"},
			{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepRestart, Version: "0.2.0"}, copy.UpdateRestartFailed, copy.Retry, "restart-update"},
			{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepRecovery, Version: "0.2.0"}, copy.UpdateRecoveryRequired, copy.Quit, "quit"},
		} {
			state.Update = step.update
			actions.calls = nil
			settle(tt)
			if !hasText(tt, messages.Text(step.text)) {
				t.Fatalf("%s %s: missing %q in %v", language, step.update.Phase, messages.Text(step.text), tt.Texts())
			}
			if step.button != "" {
				if err := tt.Click(messages.Text(step.button)); err != nil || !actions.called(step.call) {
					t.Fatalf("%s %s: %v calls=%v", language, step.update.Phase, err, actions.calls)
				}
			}
			if step.update.Phase == runtime.UpdateApplying && hasText(tt, messages.Text(copy.UpdateCancelRestart)) {
				t.Fatal("atomic install offered cancellation")
			}
			if step.update.Failed == runtime.UpdateStepRecovery {
				if _, retry := tt.Find(messages.Text(copy.Retry)); retry {
					t.Fatal("unsafe recovery offered another swap")
				}
				if !hasText(tt, messages.Text(copy.UpdateRecoveryDetail)) {
					t.Fatal("recovery omitted repair instructions")
				}
			}
			name := "update-delivery-" + string(language) + "-" + string(step.update.Phase) + "-" + string(step.update.Failed)
			if step.update.RestartQueued {
				name += "-queued"
			}
			snapshot(t, tt, name)
		}
	}
}

func TestUpdatePreferencesAreIndependentAndPermissionIsExplicit(t *testing.T) {
	state, actions, store := fixture()
	state.Update.Phase = runtime.UpdateCurrent
	state.Update.StateError = true
	messages := store.Copy()
	tt := ui.NewTester(func(c *ui.Context) { update.Card(c, store, theme.Apply(c)) }, 490, 480)
	settle(tt)
	if !hasText(tt, messages.Text(copy.UpdateStateFailed)) {
		t.Fatal("persistence failure hidden")
	}
	toggle := func(key copy.Key) {
		t.Helper()
		rect, ok := tt.Find(messages.Text(key))
		if !ok {
			t.Fatalf("missing %s", key)
		}
		tt.ClickAt(465, rect.Y+rect.H/2)
		settle(tt)
	}
	toggle(copy.UpdateAutoInstall)
	if !state.Preferences.AutoInstallUpdates || !state.Preferences.CheckUpdates {
		t.Fatal("automatic download did not persist independently")
	}
	toggle(copy.UpdateAuto)
	if state.Preferences.CheckUpdates || !state.Preferences.AutoInstallUpdates {
		t.Fatal("turning checks off lost automatic download choice")
	}
	actions.calls = nil
	toggle(copy.UpdateSystemReminders)
	if !actions.called("enable-update-reminders") || state.Preferences.SystemUpdateReminders {
		t.Fatalf("permission interaction bypassed controller: %v", actions.calls)
	}
	state.Preferences.SystemUpdateReminders = true
	settle(tt)
	toggle(copy.UpdateSystemReminders)
	if state.Preferences.SystemUpdateReminders {
		t.Fatal("reminders did not turn off")
	}
	for _, step := range []struct {
		permission runtime.UpdateNotificationPermission
		text       copy.Key
	}{
		{runtime.UpdateNotificationsDenied, copy.UpdatePermissionDenied},
		{runtime.UpdateNotificationsFailed, copy.UpdatePermissionFailed},
	} {
		state.UpdateNotifications.Permission = step.permission
		settle(tt)
		if !hasText(tt, messages.Text(step.text)) {
			t.Fatalf("missing notification permission status %s", step.permission)
		}
	}
	state.UpdateNotifications.Permission = runtime.UpdateNotificationsUnsupported
	actions.calls = nil
	settle(tt)
	toggle(copy.UpdateSystemReminders)
	if actions.called("enable-update-reminders") {
		t.Fatal("unsupported notification control enabled")
	}
	state.Update.Phase = runtime.UpdateApplying
	state.UpdateNotifications.Permission = runtime.UpdateNotificationsAllowed
	actions.calls = nil
	settle(tt)
	for _, key := range []copy.Key{copy.UpdateAuto, copy.UpdateAutoInstall, copy.UpdateSystemReminders, copy.UpdatePrerelease} {
		toggle(key)
	}
	if len(actions.calls) != 0 {
		t.Fatalf("protected apply offered changing preferences: %v", actions.calls)
	}

}

func TestPostInputUpdateReminder(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, dark := range []bool{false, true} {
			state, actions, store := fixture()
			state.Preferences.UILanguage = language
			messages := store.Copy()
			tt := ui.NewTester(func(c *ui.Context) { updatereminder.Render(c, store) }, UpdateReminderWidth, UpdateReminderHeight)
			tt.SetDark(dark)
			state.Update = runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}
			settle(tt)
			if hasText(tt, messages.Text(copy.UpdateView)) {
				t.Fatal("reminder visible without policy consent")
			}
			state.Update.ReminderVisible = true
			settle(tt)
			for _, step := range []struct {
				label copy.Key
				call  string
			}{
				{copy.UpdateView, "open-update"}, {copy.UpdateLater, "later-update"}, {copy.UpdateDismiss, "dismiss-update-reminder"},
			} {
				if err := tt.Click(messages.Text(step.label)); err != nil || !actions.called(step.call) {
					t.Fatalf("%s: %v %v", step.label, err, actions.calls)
				}
			}
			for _, phase := range []runtime.UpdatePhase{runtime.UpdateAvailable, runtime.UpdateStaged, runtime.UpdateReady} {
				state.Update.Phase = phase
				settle(tt)
				_, restart := tt.Find(messages.Text(copy.UpdateRestart))
				_, install := tt.Find(messages.Text(copy.UpdateInstall))
				if restart || install {
					t.Fatal("passive reminder can directly install or restart")
				}
				name := "update-reminder-" + string(language) + "-" + string(phase)
				if dark {
					name += "-dark"
				}
				snapshot(t, tt, name)
			}
			for _, phase := range []runtime.UpdatePhase{runtime.UpdateCurrent, runtime.UpdateDownloading, runtime.UpdateApplying, runtime.UpdateFailed} {
				state.Update = runtime.Update{Phase: phase, Version: "0.2.0", ReminderVisible: true}
				settle(tt)
				if hasText(tt, messages.Text(copy.UpdateView)) {
					t.Fatalf("stale reminder shown in %s", phase)
				}
			}
		}
	}
}
