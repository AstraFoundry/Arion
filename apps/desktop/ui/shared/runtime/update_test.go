package runtime

import "testing"

func TestUpdatePendingIncludesRecoverableInstallation(t *testing.T) {
	for _, phase := range []UpdatePhase{UpdateAvailable, UpdateDownloading, UpdateStaged, UpdateApplying, UpdateReady} {
		if !(Update{Phase: phase, Version: "0.2.0"}).Pending() {
			t.Errorf("%s lost persistent update access", phase)
		}
	}
	for _, step := range []UpdateStep{UpdateStepInstall, UpdateStepPrepare, UpdateStepApply, UpdateStepRestart} {
		if !(Update{Phase: UpdateFailed, Failed: step, Version: "0.2.0"}).Pending() {
			t.Errorf("%s failure lost retry access", step)
		}
	}
	for _, u := range []Update{
		{Phase: UpdateUnsupported}, {Phase: UpdateIdle}, {Phase: UpdateChecking}, {Phase: UpdateCurrent}, {Phase: UpdateNoRelease},
		{Phase: UpdateFailed, Failed: UpdateStepCheck}, {Phase: UpdateFailed, Failed: UpdateStepPrepare},
	} {
		if u.Pending() {
			t.Errorf("nonpending update: %+v", u)
		}
	}
}
