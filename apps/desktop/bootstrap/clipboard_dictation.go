package bootstrap

import (
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
)

// The adapter's positive no-input observation authorizes this one internal
// result. Clipboard polling continues to exclude every internal revision.
func (a *App) recordClipboardDictation(e contracts.Event) {
	cc := a.clipboard
	if cc == nil || !a.state.Preferences.Clipboard.Enabled || e.Kind != contracts.CommitResultReady ||
		e.Outcome != contracts.Copied || !e.NoInput || e.Error != "" || e.Text == "" {
		return
	}
	pending, err := domain.CaptureClip(cc.pendingDictations, domain.ClipTextPayload(e.Text), a.state.Preferences.Clipboard.Limits, time.Now())
	if err != nil {
		a.state.Clipboard.Error = clipboardError(err, copy.ClipboardSaveFailed)
		return
	}
	cc.pendingDictations = pending
	a.drainClipboardDictations()
}

// A bounded pending history uses the same count/age/byte rules as persisted
// history. Drain after the current resource mutation, preserving copy time.
func (a *App) drainClipboardDictations() {
	cc := a.clipboard
	if cc == nil || cc.service == nil || a.state.Clipboard.Busy || !a.state.Preferences.Clipboard.Enabled || len(cc.pendingDictations) == 0 || a.updateApplying() {
		return
	}
	item := cc.pendingDictations[len(cc.pendingDictations)-1]
	cc.pendingDictations = cc.pendingDictations[:len(cc.pendingDictations)-1]
	limits := a.state.Preferences.Clipboard.Limits
	a.clipboardWork(func() ([]domain.Clip, error) { return cc.service.Capture(item.Payload, limits, item.Copied) }, copy.ClipboardSaveFailed)
}
