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
	state := &runtime.State{Hold: runtime.Hold{Active: true}, Preferences: runtime.Preferences{HoldKey: runtime.HoldRightOption}}
	flow := Flow{Index: int(HoldKey)}
	for _, want := range []Step{Translate, TranslateKey, Try} {
		if !flow.Move(1, state) || flow.Step() != want {
			t.Fatalf("got step %d, want %d", flow.Step(), want)
		}
	}
	// A platform without a hold key has no translation key either.
	state.Preferences.HoldKey = ""
	if !flow.Move(-1, state) || flow.Step() != HoldKey {
		t.Fatalf("back entered translation without a hold key: %d", flow.Step())
	}
	if !flow.Move(1, state) || flow.Step() != Try {
		t.Fatalf("translation offered without a hold key: %d", flow.Step())
	}
}
