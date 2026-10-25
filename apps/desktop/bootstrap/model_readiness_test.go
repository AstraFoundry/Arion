package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/infrastructure/modelruntime"
	arion "arion.local/desktop/ui/shared/runtime"
)

type readinessAudio struct{}

func (readinessAudio) Start(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error) {
	return nil, domain.Fail(domain.CodeDeviceUnavailable, "test capture must stay idle")
}

type readinessRecognizer struct{}

func (readinessRecognizer) NewRecognizer(ctx context.Context, _ contracts.Identity) (ports.Recognizer, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func readinessApp(t *testing.T) *App {
	t.Helper()
	core, err := application.NewCoordinator(application.Dependencies{Audio: readinessAudio{}, Recognizers: readinessRecognizer{}, NewID: func() string { return "readiness" }}, application.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(core.Cancel)
	return &App{core: core}
}

func TestHoldWaitsForModelWithoutArmingDelayedCapture(t *testing.T) {
	for _, state := range []arion.ModelState{arion.ModelUnselected, arion.ModelProvisioning, arion.ModelLoading, arion.ModelFailed} {
		t.Run(string(state), func(t *testing.T) {
			a := readinessApp(t)
			a.state.Model.State = state
			at := time.Now()
			a.onHoldKey(holdkey.Down, at)
			if snapshot := a.core.Snapshot(); snapshot.State != domain.Idle || a.state.Hold.Holding {
				t.Fatalf("model %s started an unavailable session: %+v", state, snapshot)
			}
			a.state.Model.State = arion.ModelReady
			a.onHoldKey(holdkey.Down, at.Add(time.Second)) // Held-key repeat must not start later.
			a.onHoldKey(holdkey.Up, at.Add(2*time.Second))
			if a.core.Snapshot().State != domain.Idle {
				t.Fatal("a rejected gesture armed a delayed capture")
			}
			a.onHoldKey(holdkey.Down, at.Add(3*time.Second))
			if a.core.Snapshot().State != domain.Preparing || !a.state.Hold.Holding {
				t.Fatal("a fresh gesture did not start after readiness")
			}
		})
	}
}

func TestToggleWaitsForModelButCanCancelActivePreparation(t *testing.T) {
	a := readinessApp(t)
	a.state.Model.State = arion.ModelLoading
	(actions{a}).ToggleDictation()
	if a.core.Snapshot().State != domain.Idle {
		t.Fatal("toggle started an unavailable session")
	}
	a.state.Model.State = arion.ModelReady
	(actions{a}).ToggleDictation()
	if a.core.Snapshot().State != domain.Preparing {
		t.Fatal("ready toggle did not start")
	}
	a.state.Model.State = arion.ModelLoading
	(actions{a}).ToggleDictation()
	if a.core.Snapshot().State != domain.Cancelled {
		t.Fatal("model transition blocked cancellation of an active session")
	}
}

func TestReadyClearsOnlyUnavailableModelFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		d      arion.Dictation
		active bool
		clear  bool
	}{
		{name: "unloaded", d: arion.Dictation{State: domain.Failed, Failure: domain.CodeModelMissing, FailureDetail: "no verified model loaded"}, clear: true},
		{name: "worker closed", d: arion.Dictation{State: domain.Failed, Failure: domain.CodeModelMissing, FailureDetail: "native ASR engine closed"}},
		{name: "checksum", d: arion.Dictation{State: domain.Failed, Failure: domain.CodeArtifactUnverified}},
		{name: "target", d: arion.Dictation{State: domain.Failed, Failure: domain.CodeTargetUnsupported}},
		{name: "retained draft", d: arion.Dictation{State: domain.Failed, Failure: domain.CodeModelMissing, FailureDetail: "no verified model loaded", Draft: "owned fixture"}},
		{name: "preparing", d: arion.Dictation{State: domain.Preparing, Failure: domain.CodeModelMissing, FailureDetail: "no verified model loaded"}},
		{name: "newer active", d: arion.Dictation{State: domain.Failed, Failure: domain.CodeModelMissing, FailureDetail: "no verified model loaded"}, active: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := readinessApp(t)
			if tc.active {
				a.core.Start()
			}
			a.state.Model.State = arion.ModelLoading
			a.state.Dictation = tc.d
			a.applyModel(modelruntime.Status{State: modelruntime.StateReady})
			if tc.clear {
				if a.state.Dictation.State != domain.Idle || a.state.Dictation.Failure != "" {
					t.Fatalf("ready retained the obsolete model error: %+v", a.state.Dictation)
				}
			} else if !reflect.DeepEqual(a.state.Dictation, tc.d) {
				t.Fatalf("ready changed an unrelated failure/session: %+v", a.state.Dictation)
			}
		})
	}
}

func TestLateUnavailableModelFailureDoesNotOverwriteReady(t *testing.T) {
	a := readinessApp(t)
	a.state.Model.State = arion.ModelReady
	a.state.Dictation.SessionID = "older"
	a.apply(contracts.Event{Envelope: contracts.Envelope{Kind: contracts.ErrorOccurred, Identity: contracts.Identity{SessionID: "older"}}, State: domain.Failed,
		Error: domain.CodeModelMissing, Detail: "no verified model loaded"})
	if a.state.Dictation.State != domain.Idle || a.state.Dictation.Failure != "" {
		t.Fatalf("late unavailable error contradicted ready: %+v", a.state.Dictation)
	}
}
