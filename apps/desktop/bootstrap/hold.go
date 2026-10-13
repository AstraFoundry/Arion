package bootstrap

import (
	"log"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
	arion "arion.local/desktop/ui/shared/runtime"
)

// startHoldKey (re)starts the listener for the preferred hold key. A key
// that cannot listen stays chosen and reports why, so granting consent later
// starts it without the user choosing again.
func (a *App) startHoldKey() {
	a.platform.StopHoldKey()
	a.hold.Reset()
	a.state.Shortcut.HoldActive, a.state.Shortcut.HoldFailure, a.state.Shortcut.Holding = false, "", false
	key := a.state.Preferences.HoldKey
	if key == application.HoldOff {
		return
	}
	err := a.platform.StartHoldKey(key, func(e holdkey.Event) {
		now := time.Now()
		a.post(func() { a.onHoldKey(e, now) })
	})
	if err != nil {
		a.state.Shortcut.HoldFailure = domain.CodeOf(err)
		log.Printf("hold key %s unavailable: %v", key, err)
		return
	}
	a.state.Shortcut.HoldActive = true
}

// holdSession projects the coordinator state onto the gesture's view.
func (a *App) holdSession() application.HoldSession {
	snapshot := a.core.Snapshot()
	switch {
	case !snapshot.Active:
		return application.HoldIdle
	case snapshot.State == domain.Preparing || snapshot.State == domain.Capturing:
		return application.HoldCapturing
	}
	return application.HoldBusy
}

func (a *App) onHoldKey(e holdkey.Event, at time.Time) {
	if a.testingShortcut {
		if e == holdkey.Down {
			a.state.Shortcut.Pressed = at
			a.invalidate()
		}
		return
	}
	var action application.HoldAction
	switch e {
	case holdkey.Down:
		action = a.hold.Press(at, a.holdSession())
	case holdkey.Up:
		action = a.hold.Release(at)
	case holdkey.Other:
		action = a.hold.Other()
	}
	switch action {
	case application.HoldStart:
		a.state.Shortcut.Holding = true
		a.core.Start()
	case application.HoldStop:
		a.state.Shortcut.Holding = false
		a.finishDictation()
	case application.HoldCancel:
		a.state.Shortcut.Holding = false
		a.core.Cancel()
	}
	if e == holdkey.Up && action == application.HoldNone {
		// A tap: the session keeps dictating until the next press.
		a.state.Shortcut.Holding = false
	}
	a.invalidate()
}

// finishDictation ends capture, or cancels a session still preparing, which
// has captured nothing to finish.
func (a *App) finishDictation() {
	switch a.core.Snapshot().State {
	case domain.Capturing:
		a.core.Stop()
	case domain.Preparing:
		a.core.Cancel()
	}
}

// refreshHoldConsent restarts a waiting hold key once consent arrives.
func (a *App) refreshHoldConsent(previous arion.Consent) {
	if previous != arion.ConsentGranted && a.state.Permissions.InputMonitoring == arion.ConsentGranted &&
		a.state.Preferences.HoldKey != application.HoldOff && !a.state.Shortcut.HoldActive {
		a.startHoldKey()
	}
}
