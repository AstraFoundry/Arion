package onboarding

import (
	"arion.local/desktop/ui/shared/runtime"
	"testing"
)

func TestDeviceCenterSkipsAccountStepBothDirections(t *testing.T) {
	state := &runtime.State{Account: runtime.Account{Disabled: true}}
	flow := Flow{Index: int(Try)}
	if !flow.Move(1, state) || flow.Step() != Finished {
		t.Fatal("device center offered sign-in")
	}
	if !flow.Move(-1, state) || flow.Step() != Try {
		t.Fatal("back entered unavailable sign-in")
	}
	state.Account.Disabled = false
	if !flow.Move(1, state) || flow.Step() != Sync {
		t.Fatal("OIDC account step missing")
	}
}
