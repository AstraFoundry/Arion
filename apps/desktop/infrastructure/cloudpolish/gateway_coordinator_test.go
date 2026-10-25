package cloudpolish

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// Only OS capture, recognition and editor insertion are synthetic. Scheduling,
// source filtering, rewrite, HTTP transport and delivery validation are real.
type gatewayAudio struct {
	mu   sync.Mutex
	sink ports.FrameSink
}
type gatewayStream struct{}

func (gatewayStream) Stop() error { return nil }
func (a *gatewayAudio) Start(_ context.Context, _ contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	a.mu.Lock()
	a.sink = sink
	a.mu.Unlock()
	return gatewayStream{}, nil
}
func (a *gatewayAudio) push() {
	a.mu.Lock()
	sink := a.sink
	a.mu.Unlock()
	samples := make([]float32, 320)
	for i := range samples {
		samples[i] = 0.2
	}
	sink(contracts.Frame{SampleRate: 16000, Channels: 1, Samples: samples})
}

type gatewayRecognition struct{}

func (gatewayRecognition) NewRecognizer(context.Context, contracts.Identity) (ports.Recognizer, error) {
	return gatewayRecognition{}, nil
}
func (gatewayRecognition) Accept(context.Context, contracts.Frame) error { return nil }
func (gatewayRecognition) Partial(context.Context) (contracts.Draft, error) {
	return contracts.Draft{Text: "检查 write value。"}, nil
}
func (gatewayRecognition) Finish(context.Context) (contracts.Draft, error) {
	return contracts.Draft{Text: "检查 write value。", Final: true}, nil
}
func (gatewayRecognition) Close() error { return nil }

type gatewayEditor struct {
	reads atomic.Int32
	mu    sync.Mutex
	texts []string
}

func (*gatewayEditor) Capture(context.Context, contracts.Identity) (contracts.TargetSnapshot, error) {
	return contracts.TargetSnapshot{Target: "synthetic-editor", Application: "com.apple.mail"}, nil
}
func (*gatewayEditor) Capabilities(context.Context, contracts.TargetToken) (contracts.Capabilities, error) {
	return contracts.Capabilities{SingleInsertion: true}, nil
}
func (e *gatewayEditor) ReadContext(context.Context, contracts.Identity, contracts.TargetToken) (contracts.ContextSnapshot, error) {
	e.reads.Add(1)
	return contracts.ContextSnapshot{SafeToRead: true, Before: "BEFORE_CANARY", After: "AFTER_CANARY", Screen: "SCREEN_CANARY"}, nil
}
func (e *gatewayEditor) Commit(_ context.Context, r contracts.CommitRequest) (contracts.CommitResult, error) {
	e.mu.Lock()
	e.texts = append(e.texts, r.Text)
	e.mu.Unlock()
	return contracts.CommitResult{Identity: r.Identity, Outcome: contracts.Applied}, nil
}

func gatewaySession(t *testing.T, s *Service, policy application.RewritePolicy) (*application.Coordinator, *gatewayAudio, *gatewayEditor, <-chan contracts.Event) {
	t.Helper()
	audio, editor := &gatewayAudio{}, &gatewayEditor{}
	history := application.NewHistory(time.Hour)
	history.Add("com.apple.mail", "HISTORY_CANARY")
	c, err := application.NewCoordinator(application.Dependencies{NewID: func() string { return "synthetic-session" }, Audio: audio, Recognizers: gatewayRecognition{}, Targets: editor, Input: editor, Rewriter: s, History: history}, application.Options{AutoCommit: true, Rewrite: func() application.RewritePolicy { return policy }})
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan contracts.Event, 128)
	unsubscribe := c.Subscribe(func(e contracts.Event) { events <- e })
	t.Cleanup(func() { c.Cancel(); unsubscribe() })
	c.Start()
	gatewayWait(t, events, func(e contracts.Event) bool { return e.State == domain.Capturing })
	audio.push()
	gatewayWait(t, events, func(e contracts.Event) bool { return e.Kind == contracts.DraftReplaced })
	return c, audio, editor, events
}
func gatewayWait(t *testing.T, events <-chan contracts.Event, match func(contracts.Event) bool) contracts.Event {
	t.Helper()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case e := <-events:
			if match(e) {
				return e
			}
		case <-timer.C:
			t.Fatal("coordinator event timed out")
			return contracts.Event{}
		}
	}
}

func TestOpenAIGatewayCoordinatorContextSwitches(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		for mask := 0; mask < 8; mask++ {
			t.Run(fmt.Sprintf("%s/mask%d", format, mask), func(t *testing.T) {
				g, s := newGateway(t, format, "", 0)
				// This matrix verifies context routing, not latency. Cold CI hosts
				// may spend over a second scheduling the two local HTTP stages.
				policy := application.RewritePolicy{Enabled: true, FinalOnly: true, Scene: "prompt", Terms: []string{"writeValue"}, Timeout: 10 * time.Second, Context: application.ContextPolicy{Cursor: mask&1 != 0, Screen: mask&2 != 0, History: mask&4 != 0}}
				c, _, editor, events := gatewaySession(t, s, policy)
				bodies, _ := g.snapshot()
				if len(bodies) != 0 {
					t.Fatal("cloud request during capture")
				}
				c.Stop()
				correction := gatewayWait(t, events, func(e contracts.Event) bool { return e.Kind == contracts.CorrectionReady })
				commit := gatewayWait(t, events, func(e contracts.Event) bool { return e.Kind == contracts.CommitResultReady })
				if correction.Text != "检查 writeValue。" || correction.Rewrite != string(domain.RewriteAccepted) || commit.Outcome != contracts.Applied {
					t.Fatalf("correction=%+v commit=%+v", correction, commit)
				}
				bodies, payloads := g.snapshot()
				if len(bodies) != 2 {
					t.Fatalf("HTTP calls=%d", len(bodies))
				}
				ctx, ok := payloads[0]["context"].(map[string]any)
				if !ok {
					t.Fatal("missing selector context")
				}
				for key, enabled := range map[string]bool{"Before": mask&1 != 0, "After": mask&1 != 0, "Screen": mask&2 != 0, "History": mask&4 != 0} {
					_, present := ctx[key]
					if present != enabled {
						t.Errorf("mask=%d %s present=%v", mask, key, present)
					}
				}
				if mask&3 == 0 && editor.reads.Load() != 0 {
					t.Fatal("read editor with cursor and screen disabled")
				}
				editor.mu.Lock()
				defer editor.mu.Unlock()
				if len(editor.texts) != 1 || editor.texts[0] != correction.Text {
					t.Fatalf("commits=%v", editor.texts)
				}
				t.Logf("mask=%d; no preview HTTP; two final HTTP stages; accepted; exactly one commit", mask)
			})
		}
	}
}

func TestOpenAIGatewayCoordinatorFailureDelivery(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		for _, mode := range []string{"400", "truncated", "block"} {
			t.Run(format+"/"+mode, func(t *testing.T) {
				_, s := newGateway(t, format, mode, 2)
				policy := application.RewritePolicy{Enabled: true, FinalOnly: true, Scene: "prompt", Terms: []string{"writeValue"}, Timeout: 200 * time.Millisecond}
				c, _, editor, events := gatewaySession(t, s, policy)
				c.Stop()
				correction := gatewayWait(t, events, func(e contracts.Event) bool { return e.Kind == contracts.CorrectionReady })
				gatewayWait(t, events, func(e contracts.Event) bool { return e.Kind == contracts.CommitResultReady })
				want := application.RewriteUnavailable
				if mode == "block" {
					want = application.RewriteTimeout
				}
				if correction.Text != "检查 write value。" || correction.Rewrite != want {
					t.Fatalf("failure delivery=%+v", correction)
				}
				editor.mu.Lock()
				defer editor.mu.Unlock()
				if len(editor.texts) != 1 || editor.texts[0] != correction.Text {
					t.Fatalf("commits=%v", editor.texts)
				}
			})
		}
	}
}
