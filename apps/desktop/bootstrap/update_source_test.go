package bootstrap

import (
	"testing"
	"time"

	arion "arion.local/desktop/ui/shared/runtime"
)

func TestUpdateChecksRequireConfiguredSource(t *testing.T) {
	state := arion.Update{Phase: arion.UpdateAvailable, Version: "1.2.3"}
	u := &updates{state: &state}
	if u.due(time.Now()) {
		t.Fatal("unconfigured updater scheduled a check")
	}
	u.check(time.Now(), true)
	if state.Phase != arion.UpdateAvailable {
		t.Fatal("unconfigured check changed state")
	}
}
