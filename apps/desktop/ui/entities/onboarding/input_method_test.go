package onboarding

import (
	"testing"

	"arion.local/desktop/ui/shared/runtime"
)

func TestInputMethodSetupIsOptionalButSelectionGatesContinue(t *testing.T) {
	for _, state := range []runtime.InputMethodState{runtime.InputMethodAbsent, runtime.InputMethodPending, runtime.InputMethodInstalled} {
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
