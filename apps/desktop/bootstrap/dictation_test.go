package bootstrap

import (
	"errors"
	goruntime "runtime"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

func TestVoiceShortcutsUseCurrentCaretWithArionInputMethodSelected(t *testing.T) {
	for _, translation := range []bool{false, true} {
		t.Run(map[bool]string{false: "polishing", true: "translation"}[translation], func(t *testing.T) {
			core, err := application.NewCoordinator(application.Dependencies{
				Audio: readinessAudio{}, Recognizers: readinessRecognizer{},
				NewID:      func() string { return "shortcut-route" },
				SelectMode: func() domain.InputMode { return domain.SystemIMEMode },
			}, application.Options{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(core.Cancel)
			prefs := application.DefaultPreferences(goruntime.GOOS)
			a := &App{core: core, prefs: prefs}
			a.state.Preferences = prefs
			a.state.Model.State = runtime.ModelReady
			key := prefs.HoldKey
			if translation {
				key = prefs.TranslateKey
			}
			a.onHoldKeyEvent(key, holdkey.Down, time.Now())
			if snapshot := core.Snapshot(); snapshot.Mode != domain.ShortcutMode || snapshot.State != domain.Preparing {
				t.Fatalf("voice shortcut selected owned IME delivery: %+v", snapshot)
			}
		})
	}
}

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
