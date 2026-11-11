package runtime

import "testing"

func TestUpdatePendingIncludesRecoverableInstallation(t *testing.T) {
	for _, phase := range []UpdatePhase{UpdateAvailable, UpdateDownloading, UpdateStaged, UpdateApplying, UpdateReady, UpdatePrepareFailed, UpdateRestartFailed, UpdateRecoveryFailed} {
		if !(Update{Phase: phase, Version: "0.2.0"}).Pending() {
			t.Errorf("%s lost persistent update access", phase)
		}
	}
	for _, u := range []Update{
		{Phase: UpdateUnsupported}, {Phase: UpdateIdle}, {Phase: UpdateChecking}, {Phase: UpdateCurrent}, {Phase: UpdateNoRelease},
		{Phase: UpdateCheckFailed}, {Phase: UpdateCheckFailed, Version: "0.2.0"}, {Phase: UpdatePrepareFailed},
		{Phase: UpdateRestartFailed}, {Phase: UpdateRecoveryFailed},
	} {
		if u.Pending() {
			t.Errorf("nonpending update: %+v", u)
		}
	}
}
