package bootstrap

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"arion.local/desktop/infrastructure/modelruntime"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/desktop/infrastructure/updatestate"
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

const updateQuiet = 2 * time.Second
const updateLater = 24 * time.Hour
const updateNotificationPrefix = "arion.update."

type updatePolicy struct {
	store             updatestate.Store
	saved             updatestate.State
	failed            bool
	lastBusy          time.Time
	reminder          *mygo.Window
	notification      string
	notifying         bool
	permissionPending bool
	nextCheck         time.Time
	applying          bool
	verifiedInstalled bool
	release           func()
	stop              chan struct{}
}

func (a *App) scheduleUpdates() {
	a.updates.record = a.record
	p := &updatePolicy{store: updatestate.Store{Path: filepath.Join(a.opts.DataDir, "updates.json")}, lastBusy: time.Now(), stop: make(chan struct{})}
	a.updatePolicy = p
	saved, err := p.store.Load()
	if err != nil {
		p.failed = true
		a.state.Update.StateError = true
		a.record("update_state", err)
		a.updates.recoveryFailure(err)
		if release, ok := a.core.TryMaintenance(); ok {
			p.release = release
		}
	} else {
		p.saved = saved
	}
	a.updates.persist = func(stage preparedRelease, automatic bool) error {
		next := p.saved
		next.Stage, next.Version, next.Notes = "", "", ""
		next.Automatic = false
		if stage != nil {
			next.Stage, next.Version, next.Notes, next.Beta, next.Automatic = stage.Path(), stage.Version(), a.state.Update.Notes, a.updates.includePrereleases, automatic
		}
		if err := p.store.Save(next); err != nil {
			p.failed = true
			a.state.Update.StateError = true
			return err
		}
		p.saved = next
		p.failed = false
		a.state.Update.StateError = false
		return nil
	}
	a.updates.preferences(a.state.Preferences.CheckUpdates, a.state.Preferences.AutoInstallUpdates)
	if saved.Stage != "" && a.updates.source != nil {
		a.updates.recovering = true
		restoreRelease, restoreLocked := a.core.TryMaintenance()
		if !restoreLocked {
			a.updates.recoveryFailure(errors.New("input started before update recovery"))
			a.updates.recovering = false
		} else {
			go func() {
				stage, recoverErr := mygo.Updater.RecoverPrepared(saved.Stage)
				a.post(func() {
					a.updates.recovering = false
					defer func() {
						if !a.updates.blocked {
							restoreRelease()
						}
					}()
					if a.updates.stopped {
						return
					}
					if recoverErr != nil {
						a.record("update_install", recoverErr)
						a.state.Update.Version = saved.Version
						a.state.Update.Notes = saved.Notes
						a.updates.recoveryFailure(recoverErr)
						p.failed = true
						a.state.Update.StateError = true
						return
					}
					if stage.Version() != saved.Version {
						a.updates.recoveryFailure(errors.New("staged version identity changed"))
						p.failed = true
						a.state.Update.StateError = true
						return
					}
					if stage.Applied() {
						if stage.Version() != mygo.App.Version() {
							a.updates.recoveryFailure(errors.New("completed update does not match the running host"))
							p.failed = true
							a.state.Update.StateError = true
							return
						}
						if err := a.updates.acknowledgeApplied(stage, saved.Automatic); err != nil {
							p.verifiedInstalled = true
							a.state.Update.Version, a.state.Update.Notes = saved.Version, saved.Notes
							a.updates.recoveryFailure(err)
							p.failed = true
							a.state.Update.StateError = true
							// Recovery already verified the installed release's signed bytes.
							// The current host can use its coherent resources even when storage
							// errors prevent receipt cleanup; further update transactions stay blocked.
							restoreRelease()
							return
						}
						a.state.Update = arion.Update{Phase: arion.UpdateIdle}
						return
					}
					if saved.Beta != a.updates.includePrereleases || (saved.Automatic && (!a.updates.autoCheck || !a.updates.autoInstall)) {
						if err := stage.Discard(); err != nil {
							a.record("update_install", err)
							p.failed = true
							a.state.Update.StateError = true
							return
						}
						a.record("update_state", a.updates.persist(nil, false))
						return
					}
					a.updates.prepared = stage
					a.updates.automatic = saved.Automatic
					a.state.Update = arion.Update{Phase: arion.UpdateStaged, Version: saved.Version, Notes: saved.Notes}
				})
			}()
		}
	}
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		if a.state.Update.Phase == arion.UpdateApplying || (p.applying && a.state.Update.RestartQueued) {
			e.PreventDefault()
		}
	})
	mygo.App.OnQuit(func() { close(p.stop); a.updates.close() })
	mygo.App.OnDidBecomeActive(func() { a.refreshUpdatePermission() })
	a.refreshUpdatePermission()
	a.updateTick(time.Now())
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case now := <-ticker.C:
				a.post(func() { a.updateTick(now) })
			}
		}
	}()
}

func (a *App) updateBusy() bool {
	if a.resourceJobs.Load() != 0 {
		return true
	}
	if a.core != nil && a.core.Snapshot().Active {
		return true
	}
	if a.state.Hold.Holding || a.state.InputMethodInstalling || a.state.InputMethodRemoving || a.state.InputMethodActivating {
		return true
	}
	if a.manager != nil {
		s := a.manager.Status().State
		if s == modelruntime.StateLoading || s == modelruntime.StateProvisioning {
			return true
		}
	}
	if a.rewriter != nil {
		s := a.rewriter.Status().State
		if s == rewriting.Loading || s == rewriting.Provisioning {
			return true
		}
	}
	// Legacy IMK hosts cannot report keyboard preedit. An active client is
	// conservatively busy; opening About deactivates it without killing the host.
	if m, ok := a.ime.(interface{ UpdateBusy() bool }); ok && m.UpdateBusy() {
		return true
	}
	return false
}

func (a *App) updateTick(now time.Time) {
	p := a.updatePolicy
	if p != nil {
		a.state.Update.StateError = p.failed
	}
	if p == nil || a.updates == nil || a.updates.stopped || p.applying {
		return
	}
	if a.state.Preferences.CheckUpdates && !now.Before(p.nextCheck) && a.updates.due(now) {
		p.nextCheck = now.Add(updatePoll)
		a.updates.check(now, false)
	}
	if a.updateBusy() {
		p.lastBusy = now
		a.closeUpdateReminder()
		return
	}
	if now.Sub(p.lastBusy) < updateQuiet {
		return
	}
	if a.state.Update.RestartQueued {
		a.restartUpdate()
		return
	}
	if p.failed || p.notifying || !a.state.Preferences.OnboardingCompleted || a.onboarding != nil {
		return
	}
	phase := a.state.Update.Phase
	if phase != arion.UpdateAvailable && phase != arion.UpdateStaged && phase != arion.UpdateReady {
		return
	}
	key := a.reminderKey()
	for _, r := range p.saved.Reminders {
		if r.Key == key && (r.Dismissed || now.Before(r.Later) || (!r.Submitted.IsZero() && r.Later.IsZero())) {
			return
		}
	}
	if err := a.saveReminder(key, func(r *updatestate.Reminder) { r.Submitted = now; r.Later = time.Time{} }); err != nil {
		return
	}
	if a.state.Preferences.SystemUpdateReminders && a.state.UpdateNotifications.Permission == arion.UpdateNotificationsAllowed {
		a.showUpdateNotification(key)
	} else {
		a.showUpdateReminder()
	}
}

func (a *App) reminderKey() string {
	channel := "stable"
	if a.state.Preferences.AllowPrereleaseUpdates {
		channel = "beta"
	}
	phase := "available"
	if a.state.Update.Phase == arion.UpdateStaged || a.state.Update.Phase == arion.UpdateReady {
		phase = "ready"
	}
	return updateNotificationPrefix + channel + "." + a.state.Update.Version + "." + phase
}

func (a *App) saveReminder(key string, change func(*updatestate.Reminder)) error {
	p := a.updatePolicy
	if p == nil {
		return errors.New("update policy not ready")
	}
	next := p.saved
	next.Reminders = append([]updatestate.Reminder(nil), p.saved.Reminders...)
	i := -1
	for j, r := range next.Reminders {
		if r.Key == key {
			i = j
			break
		}
	}
	if i < 0 {
		if len(next.Reminders) >= 32 {
			next.Reminders = next.Reminders[1:]
		}
		next.Reminders = append(next.Reminders, updatestate.Reminder{Key: key})
		i = len(next.Reminders) - 1
	}
	change(&next.Reminders[i])
	if err := p.store.Save(next); err != nil {
		p.failed = true
		a.state.Update.StateError = true
		a.record("update_state", err)
		return err
	}
	p.saved = next
	p.failed = false
	a.state.Update.StateError = false
	return nil
}

func (a *App) showUpdateReminder() {
	p := a.updatePolicy
	if p == nil || p.reminder != nil || a.tray == nil {
		return
	}
	a.state.Update.ReminderVisible = true
	view := uiapp.UpdateReminder(a.ui)
	p.reminder = mygo.NewFlyout(mygo.FlyoutOptions{Tray: a.tray, Placement: mygo.PlacementBottom, Gap: 4, Width: uiapp.UpdateReminderWidth, Height: uiapp.UpdateReminderHeight, Shadow: true, Hidden: true, Content: view.Content})
	a.track(p.reminder, view, func() { p.reminder = nil; a.state.Update.ReminderVisible = false })
	p.reminder.ShowInactive()
}
func (a *App) closeUpdateReminder() {
	if a.updatePolicy != nil && a.updatePolicy.reminder != nil {
		a.updatePolicy.reminder.Close()
	}
	a.state.Update.ReminderVisible = false
}
func (a *App) clearUpdateNotification() {
	p := a.updatePolicy
	if p == nil {
		return
	}
	if p.notification != "" {
		mygo.RemoveNotification(p.notification)
		p.notification = ""
	}
	// History survives app exit; remove only IDs previously owned by this policy.
	for _, r := range p.saved.Reminders {
		mygo.RemoveNotification(r.Key)
	}
}
func (a *App) showUpdateNotification(key string) {
	p := a.updatePolicy
	messages := a.ui.Copy()
	body := messages.UpdateAvailable(a.state.Update.Version)
	if a.state.Update.Phase == arion.UpdateStaged {
		body = messages.UpdateDownloaded(a.state.Update.Version)
	}
	if a.state.Update.Phase == arion.UpdateReady {
		body = messages.UpdateReadyVersion(a.state.Update.Version)
	}
	if p.notification != "" && p.notification != key {
		mygo.RemoveNotification(p.notification)
	}
	generation := a.updates.generation
	p.notification = key
	p.notifying = true
	notification := mygo.NewNotification(mygo.NotificationOptions{ID: key, Title: messages.Text(copy.UpdateNotificationTitle), Body: body, Silent: true, Group: "updates", Actions: []mygo.NotificationAction{{ID: "view", Title: messages.Text(copy.UpdateView), Foreground: true}, {ID: "later", Title: messages.Text(copy.UpdateLater)}}})
	go func() {
		err := notification.Show()
		a.post(func() {
			p.notifying = false
			if generation != a.updates.generation || a.reminderKey() != key || !a.state.Preferences.SystemUpdateReminders || a.updateBusy() || a.updateApplying() {
				mygo.RemoveNotification(key)
				return
			}
			if err != nil {
				a.record("update_notification", err)
				a.refreshUpdatePermission()
			}
		})
	}()
}

func (a *App) notificationAction(id, action string) {
	if strings.HasPrefix(id, "arion.product.") {
		a.productNotificationAction(id, action)
		return
	}
	// Only previously persisted notification IDs are accepted, including cold starts.
	if a.updatePolicy == nil {
		a.pendingUpdateAction = [2]string{id, action}
		return
	}
	known := false
	for _, r := range a.updatePolicy.saved.Reminders {
		if r.Key == id {
			known = true
			break
		}
	}
	if !known {
		return
	}
	switch action {
	case "", "view":
		(actions{a}).OpenUpdate()
	case "later":
		_ = a.saveReminder(id, func(r *updatestate.Reminder) { r.Later = time.Now().Add(updateLater) })
		mygo.RemoveNotification(id)
		a.closeUpdateReminder()
	}
}

func (a *App) refreshUpdatePermission() {
	p := a.updatePolicy
	if p == nil || p.permissionPending {
		return
	}
	p.permissionPending = true
	go func() {
		permission, err := mygo.NotificationPermission()
		a.post(func() { p.permissionPending = false; a.applyUpdatePermission(permission, err) })
	}()
}
func (a *App) applyUpdatePermission(permission mygo.NotificationAuthorization, err error) {
	state := arion.UpdateNotificationsUnknown
	switch permission {
	case mygo.NotificationAuthorizationAllowed:
		state = arion.UpdateNotificationsAllowed
	case mygo.NotificationAuthorizationDenied:
		state = arion.UpdateNotificationsDenied
	case mygo.NotificationAuthorizationUnsupported:
		state = arion.UpdateNotificationsUnsupported
	}
	if err != nil {
		state = arion.UpdateNotificationsFailed
		a.record("update_notification", err)
	}
	a.state.UpdateNotifications.Permission = state
	a.configureProductNotifications()
}

func (a *App) restartUpdate() {
	p := a.updatePolicy
	if p == nil || p.applying || a.updates == nil {
		return
	}
	if p.release != nil && a.state.Update.Failed == arion.UpdateStepRestart {
		a.relaunchUpdate()
		return
	}
	if a.updates.prepared == nil && a.state.Update.Phase != arion.UpdateReady {
		return
	}
	a.state.Update.RestartQueued = true
	if a.updateBusy() {
		return
	}
	release, ok := a.core.TryMaintenance()
	if !ok {
		return
	}
	if m, has := a.ime.(interface{ TryMaintenance() (func(), bool) }); has {
		unlock, got := m.TryMaintenance()
		if !got {
			release()
			return
		}
		coreRelease := release
		release = func() { unlock(); coreRelease() }
	}
	p.verifiedInstalled = false
	p.applying = true
	p.release = release
	a.closeUpdateReminder()
	a.clearUpdateNotification()
	a.stopMonitor()
	finish := func(err error) {
		a.state.Update.RestartQueued = false
		if err != nil {
			if errors.Is(err, mygo.ErrUpdateRecoveryRequired) {
				return
			}
			p.release()
			p.release = nil
			p.applying = false
			return
		}
		// Keep the input lease until exit: old code must not load new resources.
		a.relaunchUpdate()
	}
	if a.state.Update.Phase == arion.UpdateReady {
		finish(nil)
		return
	}
	if a.edits != nil {
		a.edits.Settle()
	}
	go func() {
		if a.edits != nil {
			a.edits.Wait()
		}
		deadline := time.Now().Add(30 * time.Second)
		for a.resourceJobs.Load() != 0 {
			if time.Now().After(deadline) {
				a.post(func() {
					err := errors.New("resource shutdown did not finish")
					a.record("update_install", err)
					finish(err)
				})
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		a.post(func() { a.updates.apply(finish) })
	}()
}

func (x actions) LaterUpdate() {
	if x.a.updatePolicy == nil {
		return
	}
	_ = x.a.saveReminder(x.a.reminderKey(), func(r *updatestate.Reminder) { r.Later = time.Now().Add(updateLater) })
	x.a.closeUpdateReminder()
	x.a.clearUpdateNotification()
}
func (x actions) DismissUpdateReminder() {
	if x.a.updatePolicy == nil {
		return
	}
	_ = x.a.saveReminder(x.a.reminderKey(), func(r *updatestate.Reminder) { r.Dismissed = true })
	x.a.closeUpdateReminder()
	x.a.clearUpdateNotification()
}
func (x actions) CancelUpdateRestart() {
	if x.a.updatePolicy == nil || !x.a.updatePolicy.applying {
		x.a.state.Update.RestartQueued = false
	}
}
func (x actions) EnableUpdateReminders() {
	a := x.a
	p := a.updatePolicy
	if p == nil || p.permissionPending {
		return
	}
	p.permissionPending = true
	go func() {
		permission, err := mygo.RequestNotificationPermission()
		a.post(func() {
			p.permissionPending = false
			a.applyUpdatePermission(permission, err)
			if err == nil && permission == mygo.NotificationAuthorizationAllowed {
				(actions{a}).UpdatePreferences(func(p *arion.Preferences) { p.SystemUpdateReminders = true })
			}
		})
	}()
}

func (a *App) updateApplying() bool {
	if a.updatePolicy != nil && (a.updatePolicy.applying || a.updatePolicy.release != nil) {
		return true
	}
	return a.updates != nil && a.updates.blocked && (a.updatePolicy == nil || !a.updatePolicy.verifiedInstalled)
}

// Resource jobs are counted before dispatch, closing the gap before their
// asynchronous status callback reaches the main thread.
func (a *App) runResource(fn func()) {
	if a.updateApplying() {
		return
	}
	a.resourceJobs.Add(1)
	go func() { defer a.resourceJobs.Add(-1); fn() }()
}

func (a *App) relaunchUpdate() {
	p := a.updatePolicy
	p.applying = true
	a.state.Update.Phase = arion.UpdateReady
	a.state.Update.Failed = ""
	mygo.App.RelaunchWithResult(func(accepted bool) {
		if !accepted {
			p.applying = false
			a.state.Update.Phase = arion.UpdateFailed
			a.state.Update.Failed = arion.UpdateStepRestart
			a.invalidate()
		}
	})
}
