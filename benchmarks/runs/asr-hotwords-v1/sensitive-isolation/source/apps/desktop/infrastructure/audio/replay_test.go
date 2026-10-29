package audio

import (
	"context"
	"testing"
	"time"

	"arion.local/core/contracts"
)

// A stopped replay still signals Done, so waiters never hang.
func TestReplayDoneOnStop(t *testing.T) {
	replay := &Replay{Samples: make([]float32, SampleRate*5), ChunkMs: 20}
	stream, err := replay.Start(context.Background(), contracts.Identity{SessionID: "s", Generation: 1}, func(contracts.Frame) {})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	_ = stream.Stop()
	select {
	case <-replay.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed after Stop")
	}
}

// A complete replay delivers every sample in order, in real time.
func TestReplayDeliversAllSamples(t *testing.T) {
	replay := &Replay{Samples: make([]float32, SampleRate/4), ChunkMs: 20}
	var got int
	var last uint64
	start := time.Now()
	stream, err := replay.Start(context.Background(), contracts.Identity{SessionID: "s", Generation: 1}, func(f contracts.Frame) {
		if f.Sequence != 0 && f.Sequence != last+1 {
			t.Error("sequence gap")
		}
		last = f.Sequence
		got += len(f.Samples)
	})
	if err != nil {
		t.Fatal(err)
	}
	<-replay.Done()
	_ = stream.Stop()
	if got != SampleRate/4 || time.Since(start) < 240*time.Millisecond {
		t.Fatalf("got %d samples in %v", got, time.Since(start))
	}
}
