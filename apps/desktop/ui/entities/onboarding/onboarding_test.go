package onboarding

import (
	"arion.local/desktop/ui/shared/runtime"
	"testing"
)

func readyState() *runtime.State {
	return &runtime.State{Permissions: runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted}, Model: runtime.Model{State: runtime.ModelReady}, Hold: runtime.Hold{Active: true}}
}

func TestRequiredFlowSkipsGrantedPermissions(t *testing.T) {
	s := readyState()
	f := Flow{}
	for _, want := range []Step{Model, Try, Finished} {
		if !f.Move(1, s) || f.Step() != want {
			t.Fatalf("step = %v, want %v", f.Step(), want)
		}
	}
	if f.Move(1, s) {
		t.Fatal("advanced past finish")
	}
	f.Move(-1, s)
	if !f.Move(-1, s) || f.Step() != Model {
		t.Fatal("back entered an already granted permission")
	}
	s.Permissions.Microphone = runtime.ConsentDenied
	if !f.Move(1, s) || f.Step() != Microphone || f.Move(1, s) {
		t.Fatal("missing permission was bypassed")
	}
}

func TestPracticeRequiresReadiness(t *testing.T) {
	for _, step := range []Step{Try, Finished} {
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

func TestDownloadConfirmationAndStableProgress(t *testing.T) {
	s := readyState()
	s.Model.State = runtime.ModelAwaitingDownload
	f := Flow{Index: int(Model)}
	if f.Move(1, s) {
		t.Fatal("advanced without download confirmation")
	}
	s.Model.State = runtime.ModelProvisioning
	if !f.Move(1, s) || f.Step() != Try {
		t.Fatal("download did not continue in background")
	}
	current, total := Progress(f, s)
	s.Preferences.Polish = true
	s.Preferences.PolishEngine = runtime.PolishCloud
	c, n := Progress(f, s)
	if current != c || total != n {
		t.Fatal("optional choices changed progress")
	}
}
