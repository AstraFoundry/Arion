package onboarding

import (
	"testing"

	"arion.local/desktop/ui/shared/runtime"
)

func readyState() *runtime.State {
	return &runtime.State{
		Permissions: runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted},
		Model:       runtime.Model{State: runtime.ModelReady},
		Hold:        runtime.Hold{Active: true},
		Preferences: runtime.Preferences{HoldKey: "right-option"},
	}
}

func TestRequiredFlowSkipsGrantedPermissions(t *testing.T) {
	s := readyState()
	f := Flow{}
	for _, want := range []Step{Model, Polish, Permissions, Dictate, Translate} {
		if !f.Move(1, s) || f.Step() != want {
			t.Fatalf("step = %v, want %v", f.Step(), want)
		}
	}
	if f.Move(1, s) {
		t.Fatal("advanced past practice")
	}
	f = Flow{Index: int(Model)}
	if !f.Move(-1, s) || f.Step() != Welcome {
		t.Fatal("back left the welcome page")
	}
	s.Permissions.Microphone = runtime.ConsentDenied
	s.Permissions.Accessibility = runtime.ConsentDenied
	f = Flow{Index: int(Polish)}
	if !f.Move(1, s) || f.Step() != Permissions || f.Move(1, s) {
		t.Fatal("missing permission was bypassed")
	}
	s.Permissions.Microphone = runtime.ConsentGranted
	if f.Move(1, s) {
		t.Fatal("one grant was enough")
	}
	s.Permissions.Accessibility = runtime.ConsentGranted
	if !f.Move(1, s) || f.Step() != Dictate {
		t.Fatal("both grants did not reach practice")
	}
}

func TestPracticeRequiresReadiness(t *testing.T) {
	for _, step := range []Step{Dictate, Translate} {
		s := readyState()
		if !Ready(step, s) {
			t.Fatal("ready setup blocked")
		}
		for _, state := range []runtime.ModelState{runtime.ModelAwaitingDownload, runtime.ModelProvisioning, runtime.ModelLoading, runtime.ModelFailed} {
			s.Model.State = state
			if Ready(step, s) {
				t.Fatalf("completed with model %s", state)
			}
		}
		s = readyState()
		s.Hold.Active = false
		if Ready(step, s) {
			t.Fatal("completed without listener")
		}
		s = readyState()
		s.Permissions.Accessibility = runtime.ConsentDenied
		if Ready(step, s) {
			t.Fatal("completed after permission revocation")
		}
		s = readyState()
		s.Dictation.State = runtime.StateFinalizing
		if Ready(step, s) {
			t.Fatal("completed while dictation was pending")
		}
	}
}

func TestDownloadConfirmationAndPolishGate(t *testing.T) {
	s := readyState()
	s.Model.State = runtime.ModelAwaitingDownload
	f := Flow{Index: int(Model)}
	if f.Move(1, s) {
		t.Fatal("advanced without download confirmation")
	}
	s.Model.State = runtime.ModelProvisioning
	if !f.Move(1, s) || f.Step() != Polish {
		t.Fatal("confirmed download did not reach polish")
	}
	s.Preferences.Polish = true
	s.Preferences.PolishEngine = runtime.PolishLocal
	s.Polish.State = runtime.PolishOff
	f.Index = int(Polish)
	if f.Move(1, s) {
		t.Fatal("local polish continued before download confirmation")
	}
	s.Polish.State = runtime.PolishProvisioning
	if !f.Move(1, s) || f.Step() != Permissions {
		t.Fatalf("confirmed polish download advanced to %v", f.Step())
	}
}

func TestWelcomeBlocksSignInAndCloudStaysInsidePolish(t *testing.T) {
	s := readyState()
	s.Account.Disabled = true
	f := Flow{}
	if !f.Move(1, s) || f.Step() != Model {
		t.Fatal("disabled account added a page")
	}
	s = readyState()
	s.Account.Phase = runtime.AccountSigningIn
	if f.Index != int(Welcome) && !f.Move(-1, s) {
		t.Fatal("could not return to welcome")
	}
	f.Index = int(Welcome)
	if f.Move(1, s) {
		t.Fatal("moved during sign-in")
	}
	s = readyState()
	s.Preferences.Polish = true
	s.Preferences.PolishEngine = runtime.PolishCloud
	current, total := Progress(Flow{Index: int(Polish)}, readyState())
	c, n := Progress(Flow{Index: int(Polish)}, s)
	if c != current || n != total {
		t.Fatal("cloud setup changed the step count")
	}
	f = Flow{Index: int(Polish)}
	if !f.Move(1, s) || f.Step() != Permissions {
		t.Fatal("cloud setup became its own step")
	}
}

func TestSelectionStepCompletesSetup(t *testing.T) {
	s := readyState()
	s.SelectionReader = true
	f := Flow{Index: int(Dictate)}
	if Last(f, s) || !f.Move(1, s) || f.Step() != Translate || Last(f, s) {
		t.Fatal("translate no longer leads to the selection step")
	}
	if !f.Move(1, s) || f.Step() != Selection || !Last(f, s) {
		t.Fatalf("step = %v, want the last step Selection", f.Step())
	}
	s.Model.State = runtime.ModelLoading
	s.Hold.Active = false
	if !Ready(Selection, s) || s.Preferences.Selection.Enabled {
		t.Fatal("finishing with selection off was blocked")
	}
	if f.Move(1, s) {
		t.Fatal("advanced past the selection step")
	}
	if current, total := Progress(f, s); current != total {
		t.Fatalf("progress %d of %d on the last step", current, total)
	}
}

func TestSelectionStepSkips(t *testing.T) {
	s := readyState()
	f := Flow{Index: int(Dictate)}
	if !f.Move(1, s) || f.Step() != Translate || !Last(f, s) {
		t.Fatal("without a selection reader translate must finish")
	}
	s.SelectionReader = true
	s.Preferences.HoldKey = ""
	f = Flow{Index: int(Dictate)}
	if Last(f, s) || !f.Move(1, s) || f.Step() != Selection || !Last(f, s) {
		t.Fatalf("skipped translate did not reach selection: %v", f.Step())
	}
	if !f.Move(-1, s) || f.Step() != Dictate {
		t.Fatal("back from selection did not skip translate")
	}
	s.SelectionReader = false
	if !Last(f, s) {
		t.Fatal("dictate must finish when both later steps are skipped")
	}
}
