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
