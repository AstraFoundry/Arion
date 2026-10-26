package asr

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type lifecycleEvent struct {
	Event string   `json:"event"`
	PID   int      `json:"pid"`
	ID    string   `json:"id,omitempty"`
	Args  []string `json:"args,omitempty"`
}

// lifecycleWorker is a real subprocess protocol fixture, not model evidence.
// Its first generation can withhold a cancellation acknowledgement so tests
// exercise the production watchdog without sleeping for the production budget.
func lifecycleWorker() {
	root := os.Getenv("ARION_ASR_LIFECYCLE_ROOT")
	modeBytes, err := os.ReadFile(filepath.Join(root, "mode"))
	if err != nil {
		os.Exit(10)
	}
	mode := string(modeBytes)
	events := readLifecycleEvents(filepath.Join(root, "trace.jsonl"))
	generation := 1
	for _, event := range events {
		if event.Event == "start" {
			generation++
		}
	}
	log, err := os.OpenFile(filepath.Join(root, "trace.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		os.Exit(11)
	}
	defer log.Close()
	trace := func(event, id string) {
		_ = json.NewEncoder(log).Encode(lifecycleEvent{Event: event, PID: os.Getpid(), ID: id, Args: os.Args[1:]})
	}
	trace("start", "")
	out := json.NewEncoder(os.Stdout)
	if generation > 1 && mode == "bad-ready" {
		_ = out.Encode(message{Type: "ready", Protocol: 99})
	} else if generation == 1 || (mode != "hold-ready" && !(mode == "hold-ready-once" && generation == 2)) {
		_ = out.Encode(message{Type: "ready", Protocol: 3})
	}
	counts := map[string]int{}
	held := ""
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 64<<10), 4<<20)
	for in.Scan() {
		var request struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			PCM  string `json:"pcm"`
		}
		if json.Unmarshal(in.Bytes(), &request) != nil {
			os.Exit(12)
		}
		switch request.Type {
		case "chunk":
			pcm, _ := base64.StdEncoding.DecodeString(request.PCM)
			counts[request.ID] += len(pcm) / 2
		case "transcribe":
			trace("transcribe", request.ID)
			if mode == "crash" {
				os.Exit(13)
			}
			if generation == 1 && held == "" {
				held = request.ID
				continue
			}
			_ = out.Encode(message{Type: "end", ID: request.ID, Stop: "complete", Text: strconv.Itoa(counts[request.ID])})
		case "cancel":
			trace("cancel", request.ID)
			if request.ID != held || mode == "ack" {
				_ = out.Encode(message{Type: "cancelled", ID: request.ID})
			}
		}
	}
}

func readLifecycleEvents(path string) []lifecycleEvent {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var events []lifecycleEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event lifecycleEvent
		if json.Unmarshal(scanner.Bytes(), &event) == nil {
			events = append(events, event)
		}
	}
	return events
}

func lifecycleSetup(t *testing.T, mode string) (Spec, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mode"), []byte(mode), 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARION_ASR_WORKER", executable)
	t.Setenv("ARION_ASR_LIFECYCLE_WORKER", "1")
	t.Setenv("ARION_ASR_LIFECYCLE_ROOT", root)
	return Spec{Kind: QwenGGUF, Directory: root, Files: map[string]string{"model": "pinned.gguf"}, SampleRate: 16000, Threads: 4, EndpointSilence: 0.8}, root
}

func waitLifecycle(t *testing.T, root, event string, count int) []lifecycleEvent {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events := readLifecycleEvents(filepath.Join(root, "trace.jsonl"))
		n := 0
		for _, item := range events {
			if item.Event == event {
				n++
			}
		}
		if n >= count {
			return events
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("worker never recorded %d %s events", count, event)
	return nil
}

func retireLifecycle(t *testing.T, e *engine, root string) {
	t.Helper()
	e.cancelGrace = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := e.decode(ctx, make([]float32, 16000)); done <- err }()
	waitLifecycle(t, root, "transcribe", 1)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled request: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancellation watchdog did not reap its process")
	}
}

func TestCancellationRetiresAndReloadsSameWorker(t *testing.T) {
	spec, root := lifecycleSetup(t, "retire")
	offline, err := LoadOffline(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer offline.Close()
	retireLifecycle(t, offline.engine, root)
	// Neither a caller mutating its map nor changing the override may change
	// the runtime/model selected for this engine's next process generation.
	spec.Files["model"] = "different.gguf"
	t.Setenv("ARION_ASR_WORKER", filepath.Join(root, "must-not-run.exe"))
	text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: make([]float32, 321)})
	if err != nil || text != "321" {
		t.Fatalf("next request after retirement: %q %v", text, err)
	}
	events := readLifecycleEvents(filepath.Join(root, "trace.jsonl"))
	var starts, requests []lifecycleEvent
	for _, event := range events {
		if event.Event == "start" {
			starts = append(starts, event)
		}
		if event.Event == "transcribe" {
			requests = append(requests, event)
		}
	}
	if len(starts) != 2 || starts[0].PID == starts[1].PID || len(requests) != 2 || requests[0].PID == requests[1].PID {
		t.Fatalf("cancelled request retried or replacement not isolated: %+v", events)
	}
	if fmt.Sprint(starts[0].Args) != fmt.Sprint(starts[1].Args) {
		t.Fatalf("replacement changed its immutable load specification: %+v", starts)
	}
}

func TestStreamingFinishReloadsRetiredPreview(t *testing.T) {
	spec, root := lifecycleSetup(t, "retire")
	stream, err := LoadStreaming(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	stream.engine.cancelGrace = 20 * time.Millisecond
	id := contracts.Identity{SessionID: "retired-preview", Generation: 7}
	recognizer, err := stream.NewRecognizer(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	defer recognizer.Close()
	for i := range 60 {
		if err = recognizer.Accept(context.Background(), contracts.Frame{Identity: id, Sequence: uint64(i), SampleOffset: uint64(i * 320), SampleRate: 16000, Channels: 1, Samples: make([]float32, 320)}); err != nil {
			t.Fatal(err)
		}
	}
	waitLifecycle(t, root, "transcribe", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	draft, err := recognizer.Finish(ctx)
	if err != nil || draft.Text != "19200" || draft.Identity != id || !draft.Final {
		t.Fatalf("tail flush after retiring obsolete preview: %+v %v", draft, err)
	}
	waitLifecycle(t, root, "start", 2)
}

func TestUnexpectedWorkerExitDoesNotReload(t *testing.T) {
	spec, root := lifecycleSetup(t, "crash")
	e, err := load(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer e.close()
	for range 2 {
		if _, err = e.decode(context.Background(), make([]float32, 320)); domain.CodeOf(err) != domain.CodeModelUnsupported {
			t.Fatalf("unexpected worker exit: %v", err)
		}
	}
	starts := 0
	for _, event := range readLifecycleEvents(filepath.Join(root, "trace.jsonl")) {
		if event.Event == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("unexpected failure restarted %d workers", starts)
	}
}

func TestReloadFailureIsReturnedWithoutRepeatedStarts(t *testing.T) {
	spec, root := lifecycleSetup(t, "bad-ready")
	e, err := load(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer e.close()
	retireLifecycle(t, e, root)
	for range 2 {
		if _, err = e.decode(context.Background(), make([]float32, 320)); domain.CodeOf(err) != domain.CodeModelUnsupported {
			t.Fatalf("reload protocol failure was hidden: %v", err)
		}
	}
	events := waitLifecycle(t, root, "start", 2)
	starts := 0
	for _, event := range events {
		if event.Event == "start" {
			starts++
		}
	}
	if starts != 2 {
		t.Fatalf("reload failure looped: %+v", events)
	}
}

func TestClosePreventsReloadAndInterruptsLoading(t *testing.T) {
	for _, duringLoad := range []bool{false, true} {
		t.Run(strconv.FormatBool(duringLoad), func(t *testing.T) {
			spec, root := lifecycleSetup(t, "hold-ready")
			e, err := load(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			defer e.close()
			retireLifecycle(t, e, root)
			var decoding chan error
			if duringLoad {
				decoding = make(chan error, 1)
				go func() { _, err := e.decode(context.Background(), make([]float32, 320)); decoding <- err }()
				waitLifecycle(t, root, "start", 2)
			}
			e.close()
			if duringLoad {
				select {
				case err := <-decoding:
					if err == nil {
						t.Fatal("Close allowed model loading to finish")
					}
				case <-time.After(5 * time.Second):
					t.Fatal("Close deadlocked with a loading request")
				}
			}
			if _, err = e.decode(context.Background(), make([]float32, 320)); domain.CodeOf(err) != domain.CodeModelMissing {
				t.Fatalf("closed engine accepted a request: %v", err)
			}
			starts := 0
			for _, event := range readLifecycleEvents(filepath.Join(root, "trace.jsonl")) {
				if event.Event == "start" {
					starts++
				}
			}
			want := 1
			if duringLoad {
				want = 2
			}
			if starts != want {
				t.Fatalf("Close resurrected a worker: got %d starts, want %d", starts, want)
			}
		})
	}
}

func TestConcurrentRequestsShareOneReplacement(t *testing.T) {
	spec, root := lifecycleSetup(t, "retire")
	e, err := load(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer e.close()
	retireLifecycle(t, e, root)
	var workers sync.WaitGroup
	results := make(chan error, 8)
	for i := range 8 {
		workers.Add(1)
		go func(samples int) {
			defer workers.Done()
			text, err := e.decode(context.Background(), make([]float32, samples))
			if err == nil && text != strconv.Itoa(samples) {
				err = fmt.Errorf("crossed request output: %s for %d samples", text, samples)
			}
			results <- err
		}(320 + i)
	}
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	starts := 0
	for _, event := range readLifecycleEvents(filepath.Join(root, "trace.jsonl")) {
		if event.Event == "start" {
			starts++
		}
	}
	if starts != 2 {
		t.Fatalf("concurrent requests loaded %d workers, want 2", starts)
	}
}

func TestCancelledReplacementLoadAllowsLaterRequest(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(strconv.FormatBool(deadline), func(t *testing.T) {
			spec, root := lifecycleSetup(t, "hold-ready-once")
			e, err := load(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			defer e.close()
			retireLifecycle(t, e, root)
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
			}
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := e.decode(ctx, make([]float32, 321)); done <- err }()
			waitLifecycle(t, root, "start", 2)
			if !deadline {
				cancel()
			}
			select {
			case err := <-done:
				if !errors.Is(err, ctx.Err()) || ctx.Err() == nil {
					t.Fatalf("replacement load cancellation: %v, caller %v", err, ctx.Err())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("cancelled replacement load did not reap its process")
			}
			text, err := e.decode(context.Background(), make([]float32, 322))
			if err != nil || text != "322" {
				t.Fatalf("later request after cancelled replacement: %q %v", text, err)
			}
			events := readLifecycleEvents(filepath.Join(root, "trace.jsonl"))
			starts, requests := 0, 0
			for _, event := range events {
				if event.Event == "start" {
					starts++
				}
				if event.Event == "transcribe" {
					requests++
				}
			}
			if starts != 3 || requests != 2 {
				t.Fatalf("incomplete load reused or cancelled request replayed: %+v", events)
			}
		})
	}
}

func TestAcknowledgedCancellationReusesSameProcess(t *testing.T) {
	spec, root := lifecycleSetup(t, "ack")
	e, err := load(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer e.close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := e.decode(ctx, make([]float32, 320)); done <- err }()
	waitLifecycle(t, root, "transcribe", 1)
	cancel()
	if err = <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	text, err := e.decode(context.Background(), make([]float32, 321))
	if err != nil || text != "321" {
		t.Fatalf("acknowledged cancellation poisoned worker: %q %v", text, err)
	}
	starts := 0
	for _, event := range readLifecycleEvents(filepath.Join(root, "trace.jsonl")) {
		if event.Event == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("acknowledged cancellation unnecessarily restarted %d workers", starts)
	}
}

func TestCloseStopsActiveAndQueuedRequests(t *testing.T) {
	spec, root := lifecycleSetup(t, "retire")
	e, err := load(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	defer e.close()
	results := make(chan error, 3)
	go func() { _, err := e.decode(context.Background(), make([]float32, 320)); results <- err }()
	waitLifecycle(t, root, "transcribe", 1)
	for range 2 {
		go func() { _, err := e.decode(context.Background(), make([]float32, 321)); results <- err }()
	}
	closed := make(chan struct{})
	go func() { e.close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close deadlocked behind an active decode")
	}
	for range 3 {
		select {
		case err := <-results:
			if err == nil {
				t.Fatal("Close allowed an active or queued request to complete")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Close left an active or queued decode blocked")
		}
	}
	for _, event := range readLifecycleEvents(filepath.Join(root, "trace.jsonl")) {
		if event.Event == "transcribe" && event.ID != "1" {
			t.Fatalf("queued request reached a permanently closed process: %+v", event)
		}
	}
}
