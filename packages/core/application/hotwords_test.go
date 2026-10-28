package application

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type hotwordFactory struct{ requests chan ports.RecognitionRequest }

func (f hotwordFactory) NewRecognizer(_ context.Context, r ports.RecognitionRequest) (ports.Recognizer, error) {
	f.requests <- r
	return &fakeRecognizer{words: []string{"hello"}}, nil
}

type hotwordFinalizer struct {
	mu       sync.Mutex
	requests []ports.FinalizeRequest
}

func (f *hotwordFinalizer) Finalize(_ context.Context, r ports.FinalizeRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r)
	return r.Draft, nil
}

func TestHotwordsFrozenAcrossAcousticPasses(t *testing.T) {
	for _, polish := range []bool{false, true} {
		t.Run(map[bool]string{false: "verbatim", true: "polish"}[polish], func(t *testing.T) {
			terms := []string{"Arion"}
			audio := &fakeAudio{}
			factory := hotwordFactory{requests: make(chan ports.RecognitionRequest, 2)}
			finalizer := &hotwordFinalizer{}
			rewriter := &scriptedRewriter{answer: func(r ports.RewriteRequest) (string, error) { return r.Text, nil }}
			c, err := NewCoordinator(Dependencies{Audio: audio, Recognizers: factory, Finalizer: finalizer, Rewriter: rewriter, NewID: newID}, Options{
				Hotwords: func() []string { return terms }, Rewrite: func() RewritePolicy { return RewritePolicy{Enabled: polish} },
			})
			if err != nil {
				t.Fatal(err)
			}
			events := &recorder{}
			c.Subscribe(events.listen)
			for _, want := range []string{"Arion", "Changed"} {
				c.Start()
				request := <-factory.requests
				events.wait(t, func(e contracts.Event) bool { return e.State == domain.Capturing && e.Generation == request.Generation })
				if !reflect.DeepEqual(request.Hotwords, []string{want}) {
					t.Fatalf("session terms %v", request.Hotwords)
				}
				terms[0] = "Changed"
				// A port cannot alter the coordinator's later acoustic request snapshot.
				request.Hotwords[0] = "Mutated by recognizer"
				audio.push(1, .2)
				c.Stop()
				events.wait(t, func(e contracts.Event) bool {
					return e.Kind == contracts.CorrectionReady && e.Generation == request.Generation
				})
				finalizer.mu.Lock()
				last := finalizer.requests[len(finalizer.requests)-1]
				finalizer.mu.Unlock()
				if !reflect.DeepEqual(last.Hotwords, []string{want}) {
					t.Fatalf("final terms %v", last.Hotwords)
				}
				c.Cancel()
			}
		})
	}
}

func TestSegmentFinalizerReceivesSessionHotwords(t *testing.T) {
	finalizer := &hotwordFinalizer{}
	c := &Coordinator{deps: Dependencies{Finalizer: finalizer}, opts: Options{SegmentFinalize: true}}
	r := &run{ctx: context.Background(), hotwords: []string{"Arion"}, sampleRate: 16000}
	if _, err := c.refine(r, unit{text: "hello", samples: []float32{.2}}); err != nil {
		t.Fatal(err)
	}
	if len(finalizer.requests) != 1 || !reflect.DeepEqual(finalizer.requests[0].Hotwords, r.hotwords) {
		t.Fatal(finalizer.requests)
	}
}
