package application

import (
	"sync"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type diagnosticRecorder struct {
	mu      sync.Mutex
	entries []ports.Diagnostic
}

func (d *diagnosticRecorder) Observe(e ports.Diagnostic) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries = append(d.entries, e)
}

func TestDiagnosticsObservePipelineBoundaries(t *testing.T) {
	target := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "private-target"}}
	c, audio, events := setup(t, &fakeRecognizer{words: []string{"private-transcript"}}, target, Options{AutoCommit: true})
	observer := &diagnosticRecorder{}
	c.deps.Diagnostics = observer
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, .2)
	events.wait(t, kindIs(contracts.DraftReplaced))
	c.Stop()
	final := events.wait(t, kindIs(contracts.CommitResultReady))
	observer.mu.Lock()
	defer observer.mu.Unlock()
	want := []ports.DiagnosticStage{ports.DiagnosticTarget, ports.DiagnosticRecognizer, ports.DiagnosticCapture, ports.DiagnosticCommit}
	if len(observer.entries) != len(want) {
		t.Fatalf("diagnostics %+v", observer.entries)
	}
	for i, e := range observer.entries {
		if e.Stage != want[i] || e.Identity != final.Identity || e.Duration < 0 || e.Failure != "" {
			t.Errorf("stage %d: %+v", i, e)
		}
	}
	if target.commits.Load() != 1 {
		t.Fatal("diagnostics changed delivery")
	}
}

func TestDiagnosticsPreserveTargetFailure(t *testing.T) {
	target := &fakeTarget{err: domain.Fail(domain.CodeTargetChanged, "private detail")}
	c, _, events := setup(t, &fakeRecognizer{}, target, Options{AutoCommit: true})
	observer := &diagnosticRecorder{}
	c.deps.Diagnostics = observer
	c.Start()
	events.wait(t, kindIs(contracts.ErrorOccurred))
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if len(observer.entries) != 1 || observer.entries[0].Failure != domain.CodeTargetChanged {
		t.Fatalf("diagnostics %+v", observer.entries)
	}
	if target.commits.Load() != 0 {
		t.Fatal("failed target committed")
	}
}
