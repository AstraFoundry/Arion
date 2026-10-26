package bootstrap

import (
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
	arion "arion.local/desktop/ui/shared/runtime"
)

// startHoldKey (re)starts the listener for the preferred hold key and optional
// translation hold key. A key that cannot listen stays chosen and reports why,
// so granting consent later starts it without the user choosing again.
func (a *App) startHoldKey() {
	a.holdGeneration++
	generation := a.holdGeneration
	a.platform.StopHoldKey()
	a.hold.Reset()
	a.holdKey = ""
	a.state.Hold.Active, a.state.Hold.Failure, a.state.Hold.Holding = false, "", false
	key := a.state.Preferences.HoldKey
	if key == "" {
		// This platform has no hold key adapter.
		return
	}
	keys := []application.HoldKey{key}
	if t := a.state.Preferences.TranslateKey; t != "" && t != application.HoldOff && t != key {
		keys = append(keys, t)
	}
	err := a.platform.StartHoldKeys(keys, func(k application.HoldKey, e holdkey.Event) {
		now := time.Now()
		a.post(func() {
			if generation == a.holdGeneration {
				a.onHoldKeyEvent(k, e, now)
			}
		})
	})
	a.record("hold_register", err)
	if err != nil {
		a.state.Hold.Failure = domain.CodeOf(err)
		return
	}
	a.state.Hold.Active = true
}

// holdSession projects the coordinator state onto the gesture's view.
func (a *App) holdSession() application.HoldSession {
	snapshot := a.core.Snapshot()
	switch {
	case !snapshot.Active:
		if a.state.Model.State != arion.ModelReady {
			// Consume the gesture while loading without starting a failed
			// session or arming capture when the held key is later released.
			return application.HoldBusy
		}
		return application.HoldIdle
	case snapshot.State == domain.Preparing || snapshot.State == domain.Capturing:
		return application.HoldCapturing
	}
	return application.HoldBusy
}

func (a *App) onHoldKey(e holdkey.Event, at time.Time) {
	a.onHoldKeyEvent(a.state.Preferences.HoldKey, e, at)
}

func (a *App) onHoldKeyEvent(key application.HoldKey, e holdkey.Event, at time.Time) {
	if a.testingHold {
		if e == holdkey.Down {
			if key == "" {
				key = a.state.Preferences.HoldKey
			}
			a.state.Hold.Pressed, a.state.Hold.PressedKey = at, key
			a.invalidate()
		}
		return
	}
	if e == holdkey.Up && a.holdKey != "" && key != "" && key != a.holdKey {
		return
	}
	var action application.HoldAction
	switch e {
	case holdkey.Down:
		action = a.hold.Press(at, a.holdSession())
		if action != application.HoldNone {
			a.holdKey = key
		}
	case holdkey.Up:
		action = a.hold.Release(at)
	case holdkey.Other:
		action = a.hold.Other()
	}
	translate := key != "" && key != application.HoldOff && key == a.state.Preferences.TranslateKey && key != a.state.Preferences.HoldKey
	switch action {
	case application.HoldStart:
		a.record("hold_start", nil)
		a.state.Hold.Holding = true
		a.setTranslating(translate)
		if translate {
			a.core.StartTranslation(a.preferences().TranslateLanguage)
		} else {
			a.core.Start()
		}
	case application.HoldStop:
		a.record("hold_stop", nil)
		a.state.Hold.Holding = false
		a.finishDictation()
	case application.HoldCancel:
		a.record("hold_cancel", nil)
		a.state.Hold.Holding = false
		a.core.Cancel()
	}
	if e == holdkey.Up && action == application.HoldNone {
		// A tap: the session keeps dictating until the next press.
		a.state.Hold.Holding = false
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
	if a.state.Permissions.InputMonitoring != arion.ConsentGranted && a.state.Hold.Active {
		a.holdGeneration++
		a.platform.StopHoldKey()
		a.hold.Reset()
		a.state.Hold.Active, a.state.Hold.Holding = false, false
		a.state.Hold.Failure = domain.CodePermissionDenied
		a.core.Cancel()
	}
	if previous != arion.ConsentGranted && a.state.Permissions.InputMonitoring == arion.ConsentGranted &&
		a.state.Preferences.HoldKey != "" && !a.state.Hold.Active {
		a.startHoldKey()
	}
}
