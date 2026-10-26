package bootstrap

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/desktop/infrastructure/updatestate"
	"arion.local/desktop/internal/center"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

func permissionApp(t *testing.T) *App {
	t.Helper()
	profile, err := center.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := &App{center: profile, queueReady: make(chan struct{}, 1), updatePolicy: &updatePolicy{permissionChecked: true, lastBusy: time.Now().Add(-2 * updateQuiet)}}
	a.state.Preferences.OnboardingCompleted = true
	a.state.UpdateNotifications.Permission = arion.UpdateNotificationsUnknown
	return a
}

func finishPermission(t *testing.T, a *App) {
	t.Helper()
	select {
	case <-a.queueReady:
	case <-time.After(time.Second):
		t.Fatal("permission callback did not finish")
	}
	a.queueMu.Lock()
	queued := a.queue
	a.queue = nil
	a.queueMu.Unlock()
	for _, callback := range queued {
		callback()
	}
}

func TestSystemNotificationPermissionRequiresQueriedIdleStandardProfile(t *testing.T) {
	cases := map[string]func(*App){
		"unqueried":         func(a *App) { a.updatePolicy.permissionChecked = false },
		"inflight":          func(a *App) { a.updatePolicy.permissionPending = true },
		"onboarding":        func(a *App) { a.state.Preferences.OnboardingCompleted = false },
		"onboarding-window": func(a *App) { a.onboarding = &mygo.Window{} },
		"holding":           func(a *App) { a.state.Hold.Holding = true },
		"resources":         func(a *App) { a.resourceJobs.Store(1) },
		"quiet-interval":    func(a *App) { a.updatePolicy.lastBusy = time.Now() },
		"maintenance":       func(a *App) { a.updatePolicy.release = func() {} },
		"applying":          func(a *App) { a.updatePolicy.applying = true },
		"offline":           func(a *App) { a.opts.Offline = true },
		"custom-center":     func(a *App) { a.center = center.Profile{} },
		"harness":           func(a *App) { a.harness = &harness{} },
		"denied":            func(a *App) { a.state.UpdateNotifications.Permission = arion.UpdateNotificationsDenied },
		"failed":            func(a *App) { a.state.UpdateNotifications.Permission = arion.UpdateNotificationsFailed },
		"unsupported":       func(a *App) { a.state.UpdateNotifications.Permission = arion.UpdateNotificationsUnsupported },
		"allowed":           func(a *App) { a.state.UpdateNotifications.Permission = arion.UpdateNotificationsAllowed },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			a := permissionApp(t)
			a.updatePolicy.requestPermission = func() (mygo.NotificationAuthorization, error) { return mygo.NotificationAuthorizationAllowed, nil }
			change(a)
			a.requestSystemNotificationPermission()
			if a.notificationPermissionAttempted {
				t.Fatal("permission requested outside an eligible idle session")
			}
		})
	}
}

func TestSystemNotificationPermissionIsSingleInflightAndOncePerProcess(t *testing.T) {
	for _, tc := range []struct {
		name       string
		permission mygo.NotificationAuthorization
		err        error
		want       arion.UpdateNotificationPermission
	}{
		{"allowed", mygo.NotificationAuthorizationAllowed, nil, arion.UpdateNotificationsAllowed},
		{"denied", mygo.NotificationAuthorizationDenied, nil, arion.UpdateNotificationsDenied},
		{"unknown", mygo.NotificationAuthorizationUnknown, nil, arion.UpdateNotificationsUnknown},
		{"error", mygo.NotificationAuthorizationUnknown, errors.New("authorization query failed"), arion.UpdateNotificationsFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := permissionApp(t)
			var calls atomic.Int32
			finish := make(chan struct{})
			a.updatePolicy.requestPermission = func() (mygo.NotificationAuthorization, error) { calls.Add(1); <-finish; return tc.permission, tc.err }
			a.requestSystemNotificationPermission()
			a.requestSystemNotificationPermission()
			if !a.notificationPermissionAttempted || !a.updatePolicy.permissionPending {
				t.Fatal("request not protected before asynchronous dispatch")
			}
			close(finish)
			finishPermission(t, a)
			if calls.Load() != 1 || a.updatePolicy.permissionPending || a.state.UpdateNotifications.Permission != tc.want {
				t.Fatalf("request completion: calls=%d pending=%v permission=%s", calls.Load(), a.updatePolicy.permissionPending, a.state.UpdateNotifications.Permission)
			}
			a.state.UpdateNotifications.Permission = arion.UpdateNotificationsUnknown
			a.requestSystemNotificationPermission()
			if a.updatePolicy.permissionPending || calls.Load() != 1 {
				t.Fatal("authorization automatically retried in the same process")
			}
		})
	}
}

func TestUnknownUpdatePermissionDefersReminderPersistence(t *testing.T) {
	a := permissionApp(t)
	a.state.Update = arion.Update{Phase: arion.UpdateAvailable, Version: "1.2.3"}
	a.updates = &updates{state: &a.state.Update}
	a.updatePolicy.store = updatestate.Store{Path: filepath.Join(t.TempDir(), "updates.json")}
	finish := make(chan struct{})
	a.updatePolicy.requestPermission = func() (mygo.NotificationAuthorization, error) {
		<-finish
		return mygo.NotificationAuthorizationDenied, nil
	}
	a.updateTick(time.Now())
	if len(a.updatePolicy.saved.Reminders) != 0 || !a.updatePolicy.permissionPending {
		t.Fatal("permission request consumed reminder before its result")
	}
	close(finish)
	finishPermission(t, a)
	a.updateTick(time.Now())
	if len(a.updatePolicy.saved.Reminders) != 1 || a.updatePolicy.saved.Reminders[0].Submitted.IsZero() {
		t.Fatal("denied authorization lost the ordinary update reminder")
	}
}
