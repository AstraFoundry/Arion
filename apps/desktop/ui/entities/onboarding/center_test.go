package onboarding

import (
	"arion.local/desktop/ui/shared/runtime"
	"testing"
)

func TestDeviceCenterSkipsAccountStepBothDirections(t *testing.T) {
	state := &runtime.State{Account: runtime.Account{Disabled: true}}
	flow := Flow{Index: int(Welcome)}
	if !flow.Move(1, state) || flow.Step() != Microphone {
		t.Fatal("device center offered sign-in")
	}
	if !flow.Move(-1, state) || flow.Step() != Welcome {
		t.Fatal("back entered unavailable sign-in")
	}
	state.Account.Disabled = false
	if !flow.Move(1, state) || flow.Step() != Account {
		t.Fatal("OIDC account step missing")
	}
}

func TestTranslationStepsFollowHoldKey(t *testing.T) {
	state := &runtime.State{Hold: runtime.Hold{Active: true}, Polish: runtime.Polish{State: runtime.PolishReady},
		Preferences: runtime.Preferences{HoldKey: runtime.HoldRightOption, Polish: true, PolishEngine: runtime.PolishLocal}}
	flow := Flow{Index: int(HoldKey)}
	for _, want := range []Step{Translate, TranslateKey, Try} {
		if !flow.Move(1, state) || flow.Step() != want {
			t.Fatalf("got step %d, want %d", flow.Step(), want)
		}
	}
	// Without polishing there is no translation model.
	state.Preferences.Polish = false
	if !flow.Move(-1, state) || flow.Step() != HoldKey {
		t.Fatalf("back entered translation without polishing: %d", flow.Step())
	}
	if !flow.Move(1, state) || flow.Step() != Try {
		t.Fatalf("translation offered without polishing: %d", flow.Step())
	}
	// A platform without a hold key has no translation key either.
	state.Preferences.Polish = true
	state.Preferences.HoldKey = ""
	if !flow.Move(-1, state) || flow.Step() != HoldKey {
		t.Fatalf("back entered translation without a hold key: %d", flow.Step())
	}
	if !flow.Move(1, state) || flow.Step() != Try {
		t.Fatalf("translation offered without a hold key: %d", flow.Step())
	}
}
