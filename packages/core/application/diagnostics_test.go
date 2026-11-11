package application

import (
	"context"
	"sync"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type finishFailureRecognizer struct {
	*fakeRecognizer
}

func (r finishFailureRecognizer) Finish(context.Context) (contracts.Draft, error) {
	return contracts.Draft{}, domain.Fail(domain.CodeModelUnsupported, "private native failure")
}

type diagnosticFactory struct{ recognizer ports.Recognizer }

func (f diagnosticFactory) NewRecognizer(context.Context, ports.RecognitionRequest) (ports.Recognizer, error) {
	return f.recognizer, nil
}

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
	want := []ports.DiagnosticStage{ports.DiagnosticRecognizer, ports.DiagnosticCapture, ports.DiagnosticTarget, ports.DiagnosticFinish, ports.DiagnosticCommit}
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
	if len(observer.entries) != 3 || observer.entries[0].Stage != ports.DiagnosticRecognizer || observer.entries[1].Stage != ports.DiagnosticCapture || observer.entries[2].Stage != ports.DiagnosticTarget || observer.entries[2].Failure != domain.CodeTargetChanged {
		t.Fatalf("diagnostics %+v", observer.entries)
	}
	if target.commits.Load() != 0 {
		t.Fatal("failed target committed")
	}
}

// A recognizer can fail before the separate acoustic finalizer runs. Both
// dictation paths must retain that stage's identity and failure without a commit.
func TestDiagnosticsPreserveRecognizerFinishFailure(t *testing.T) {
	for _, polishing := range []bool{false, true} {
		name := "plain"
		if polishing {
			name = "polishing"
		}
		t.Run(name, func(t *testing.T) {
			target := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "private-target"}}
			opts := Options{AutoCommit: true, Rewrite: func() RewritePolicy { return RewritePolicy{Enabled: polishing} }}
			c, _, events := setup(t, &fakeRecognizer{}, target, opts)
			c.deps.Recognizers = diagnosticFactory{finishFailureRecognizer{&fakeRecognizer{}}}
			observer := &diagnosticRecorder{}
			c.deps.Diagnostics = observer
			c.Start()
			events.wait(t, stateIs(domain.Capturing))
			c.Stop()
			failure := events.wait(t, kindIs(contracts.ErrorOccurred))
			observer.mu.Lock()
			defer observer.mu.Unlock()
			var finishes int
			for _, entry := range observer.entries {
				switch entry.Stage {
				case ports.DiagnosticFinish:
					finishes++
					if entry.Identity != failure.Identity || entry.Duration < 0 || entry.Failure != domain.CodeModelUnsupported {
						t.Fatalf("finish diagnostic lost its failure: %+v", entry)
					}
				case ports.DiagnosticFinalize, ports.DiagnosticRewrite, ports.DiagnosticCommit:
					t.Fatalf("failed recognition continued: %+v", entry)
				}
			}
			if finishes != 1 || target.commits.Load() != 0 {
				t.Fatalf("finish count %d; commit count %d", finishes, target.commits.Load())
			}
		})
	}
}
