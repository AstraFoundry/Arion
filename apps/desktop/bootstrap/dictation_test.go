package bootstrap

import (
	"errors"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

type routeIME struct {
	InputMethod
	state runtime.InputMethodState
}

type observedRouteIME struct {
	routeIME
	selected bool
	err      error
}

func (m observedRouteIME) SelectedForSession() bool { return m.selected }
func (m observedRouteIME) ObservationError() error  { return m.err }

func TestDictationRoutePrefersSessionObservation(t *testing.T) {
	for _, selected := range []bool{false, true} {
		state := runtime.InputMethodSelected
		want := domain.ShortcutMode
		if selected {
			state, want = runtime.InputMethodAbsent, domain.SystemIMEMode
		}
		a := &App{ime: observedRouteIME{routeIME: routeIME{state: state}, selected: selected}}
		if got := a.dictationMode(); got != want {
			t.Fatalf("session selection %v with stale UI %s: got %s, want %s", selected, state, got, want)
		}
	}
}

func TestInputMethodObservationErrorRecovers(t *testing.T) {
	a := &App{ime: observedRouteIME{err: errors.New("observation failed")}}
	a.refreshInputMethod()
	if a.state.InputMethodError != copy.IMEObservationFailed {
		t.Fatal("native observation failure is not visible")
	}
	a.ime = observedRouteIME{}
	a.refreshInputMethod()
	if a.state.InputMethodError != "" {
		t.Fatal("recovered observation retained an error")
	}
	a.state.InputMethodError = copy.IMERemoveFailed
	a.refreshInputMethod()
	if a.state.InputMethodError != copy.IMERemoveFailed {
		t.Fatal("successful observation erased an unrelated setup error")
	}
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
