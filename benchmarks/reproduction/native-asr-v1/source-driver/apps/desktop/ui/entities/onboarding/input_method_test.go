package onboarding

import (
	"arion.local/desktop/ui/shared/copy"
	"testing"

	"arion.local/desktop/ui/shared/runtime"
)

func TestInputMethodSetupIsOptionalButSelectionGatesContinue(t *testing.T) {
	for _, state := range []runtime.InputMethodState{runtime.InputMethodAbsent, runtime.InputMethodDiscovering, runtime.InputMethodPending, runtime.InputMethodInstalled} {
		s := &runtime.State{InputMethod: state}
		flow := Flow{Index: int(InputMethod)}
		if flow.Next(s) {
			t.Fatalf("continued before selecting source: %s", state)
		}
		if !flow.SkipInputMethod(s) || flow.Step() != Model {
			t.Fatalf("could not defer setup: %s", state)
		}
	}
	s := &runtime.State{InputMethod: runtime.InputMethodSelected, InputMethodInstalling: true}
	flow := Flow{Index: int(InputMethod)}
	if flow.SkipInputMethod(s) || flow.Next(s) || flow.Move(-1, s) {
		t.Fatal("left setup while installation was replacing the input source")
	}
	s.InputMethodInstalling = false
	if !flow.Next(s) || flow.Step() != Model {
		t.Fatal("selected source could not continue")
	}
}

func TestUnsupportedInputMethodSkippedInBothDirections(t *testing.T) {
	s := &runtime.State{InputMethod: runtime.InputMethodUnsupported, Permissions: runtime.Permissions{Accessibility: runtime.ConsentGranted}}
	flow := Flow{Index: int(Accessibility)}
	if !flow.Move(1, s) || flow.Step() != Model {
		t.Fatal("unsupported source exposed during forward navigation")
	}
	if !flow.Move(-1, s) || flow.Step() != Accessibility {
		t.Fatal("unsupported source exposed during backward navigation")
	}
}

func TestInputMethodConnectionFailureGatesContinueAfterSelection(t *testing.T) {
	s := &runtime.State{InputMethod: runtime.InputMethodSelected, InputMethodError: copy.IMEConnectionFailed}
	if Ready(InputMethod, s) {
		t.Fatal("selected but unconnected host completed setup")
	}
	s.InputMethodError = ""
	s.InputMethodActivating = true
	if Ready(InputMethod, s) {
		t.Fatal("continued before activation worker completed")
	}
	s.InputMethodActivating = false
	if !Ready(InputMethod, s) {
		t.Fatal("verified selected host could not continue")
	}
}
