package bootstrap

import (
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/runtime"
)

type routeIME struct {
	InputMethod
	state runtime.InputMethodState
}

func (m routeIME) State() runtime.InputMethodState { return m.state }

func TestDictationRouteUsesActualSelection(t *testing.T) {
	if (&App{}).dictationMode() != domain.ShortcutMode {
		t.Fatal("absent adapter must use external delivery")
	}
	for _, state := range []runtime.InputMethodState{runtime.InputMethodAbsent, runtime.InputMethodPending, runtime.InputMethodInstalled, runtime.InputMethodSelected, runtime.InputMethodUnsupported} {
		a := &App{ime: routeIME{state: state}}
		want := domain.ShortcutMode
		if state == runtime.InputMethodSelected {
			want = domain.SystemIMEMode
		}
		if got := a.dictationMode(); got != want {
			t.Fatalf("%s: got %s want %s", state, got, want)
		}
	}
}
