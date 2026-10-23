package onboarding

import (
	"testing"

	"arion.local/desktop/ui/shared/runtime"
)

func TestLocalPolishWaitsForDownloadConfirmation(t *testing.T) {
	s := &runtime.State{Preferences: runtime.Preferences{Polish: true, PolishEngine: runtime.PolishLocal}}
	for _, tc := range []struct {
		state runtime.PolishState
		ready bool
	}{
		{runtime.PolishOff, false},
		{runtime.PolishFailed, false},
		{runtime.PolishProvisioning, true},
		{runtime.PolishLoading, true},
		{runtime.PolishReady, true},
	} {
		s.Polish.State = tc.state
		if got := Ready(Polish, s); got != tc.ready {
			t.Errorf("%s: ready = %v, want %v", tc.state, got, tc.ready)
		}
	}
	s.Polish.State = runtime.PolishOff
	s.Preferences.PolishEngine = runtime.PolishCloud
	if !Ready(Polish, s) {
		t.Fatal("cloud selection waited for a local download")
	}
	s.Preferences.PolishEngine, s.Preferences.Polish = runtime.PolishLocal, false
	if !Ready(Polish, s) {
		t.Fatal("disabled polishing waited for a download")
	}
}
