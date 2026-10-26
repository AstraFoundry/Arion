package bootstrap

import (
	"context"
	"errors"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/ui/shared/copy"
)

// ActivateInputMethod owns only this explicit attempt; background refresh never
// registers, enables or selects a source. Repeated clicks cannot start a rival.
func (x actions) ActivateInputMethod() error {
	a := x.a
	if a.ime == nil {
		return errInputMethodUnsupported
	}
	if a.updateApplying() || a.state.InputMethodInstalling || a.state.InputMethodActivating || a.state.InputMethodRemoving {
		return nil
	}
	if a.core != nil {
		a.core.CancelMode(domain.SystemIMEMode)
	}
	a.imeSetupGeneration++
	generation := a.imeSetupGeneration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.imeSetupCancel = cancel
	a.state.InputMethodActivating = true
	a.state.InputMethodError = ""
	a.record("ime_setup_started", nil)
	go func() {
		err := a.ime.Activate(ctx)
		cancel()
		state := a.ime.State()
		a.post(func() {
			// An older callback may not complete or clear a newer attempt.
			if generation != a.imeSetupGeneration {
				return
			}
			a.imeSetupCancel = nil
			a.state.InputMethodActivating = false
			a.setInputMethodState(state)
			switch {
			case errors.Is(err, context.Canceled):
				a.record("ime_setup_cancelled", nil)
			case err != nil:
				a.state.InputMethodError = inputMethodSetupMessage(err)
				phase := ""
				var staged interface{ InputMethodSetupStage() string }
				if errors.As(err, &staged) {
					switch value := staged.InputMethodSetupStage(); value {
					case "discovery", "enable", "selection", "connection":
						phase = value
					}
				}
				outcome, code := "failed", diagnostics.Failure(err)
				if errors.Is(err, context.DeadlineExceeded) {
					outcome, code = "timeout", domain.CodeTargetUnsupported
				}
				a.recordEntry(diagnostics.Entry{Event: "ime_setup_failed", Phase: phase, Outcome: outcome, Code: code})
			default:
				a.record("ime_setup_completed", nil)
			}
		})
	}()
	return nil
}

// CancelInputMethodSetup leaves installed files/authorization intact. The worker
// retains its busy guard until cancellation is acknowledged on the UI thread.
func (x actions) CancelInputMethodSetup() {
	if x.a.imeSetupCancel != nil {
		x.a.imeSetupCancel()
	} else {
		// Also revoke post-install continuation when its window closes during copy.
		x.a.imeSetupGeneration++
	}
}

func inputMethodSetupMessage(err error) copy.Key {
	if domain.CodeOf(err) == domain.CodeArtifactUnverified {
		return copy.IMEPackageUnverified
	}
	var staged interface{ InputMethodSetupStage() string }
	if errors.As(err, &staged) {
		switch staged.InputMethodSetupStage() {
		case "discovery":
			return copy.IMEDiscoveryFailed
		case "enable":
			return copy.IMEEnableFailed
		case "selection":
			return copy.IMESelectFailed
		case "connection":
			return copy.IMEConnectionFailed
		}
	}
	return copy.IMEFailed
}
