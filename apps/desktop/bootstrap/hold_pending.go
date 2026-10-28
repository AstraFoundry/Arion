package bootstrap

import (
	"time"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/holdkey"
)

type deferredHold struct {
	key   application.HoldKey
	at    time.Time
	timer *time.Timer
}

func (a *App) deferHold(key application.HoldKey, at time.Time) {
	a.clearDeferredHold()
	pending := &deferredHold{key: key, at: at}
	a.holdPending = pending
	generation := a.holdGeneration
	pending.timer = time.AfterFunc(time.Until(at.Add(application.DefaultHoldThreshold)), func() {
		a.post(func() { a.activateDeferredHold(pending, generation) })
	})
}

func (a *App) activateDeferredHold(pending *deferredHold, generation uint64) {
	if a.holdPending != pending || a.holdGeneration != generation {
		return
	}
	a.clearDeferredHold()
	a.onHoldKeyEvent(pending.key, holdkey.Down, pending.at)
}

func (a *App) clearDeferredHold() {
	if pending := a.holdPending; pending != nil {
		pending.timer.Stop()
		a.holdPending = nil
	}
}

func (a *App) resetHoldGesture() {
	a.clearDeferredHold()
	a.hold.Reset()
	a.holdKey = ""
	a.state.Hold.Holding = false
}
