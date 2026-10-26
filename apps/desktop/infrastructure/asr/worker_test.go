package asr

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
)

func TestMain(m *testing.M) {
	if os.Getenv("ARION_ASR_LIFECYCLE_WORKER") == "1" {
		lifecycleWorker()
		os.Exit(0)
	}
	if os.Getenv("ARION_ASR_TEST_WORKER") == "1" {
		out := json.NewEncoder(os.Stdout)
		_ = out.Encode(message{Type: "ready", Protocol: 3})
		counts := map[string]int{}
		in := bufio.NewScanner(os.Stdin)
		in.Buffer(make([]byte, 64<<10), 4<<20)
		for in.Scan() {
			var req map[string]any
			_ = json.Unmarshal(in.Bytes(), &req)
			id, _ := req["id"].(string)
			switch req["type"] {
			case "chunk":
				data, _ := base64.StdEncoding.DecodeString(req["pcm"].(string))
				counts[id] += len(data) / 2
			case "transcribe":
				_ = out.Encode(message{Type: "end", ID: id, Text: strconv.Itoa(counts[id]), Stop: "complete"})
			case "cancel":
				_ = out.Encode(message{Type: "cancelled", ID: id})
			}
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// A real subprocess exercises transport and asynchronous capture. Finalization
// includes every uncommitted sample when the last preview was stale.
func TestNativeChunkedTailAndIdentity(t *testing.T) {
	exe, _ := os.Executable()
	t.Setenv("ARION_ASR_WORKER", exe)
	t.Setenv("ARION_ASR_TEST_WORKER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := LoadStreamingContext(ctx, Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	id := contracts.Identity{SessionID: "test", Generation: 2}
	r, err := stream.NewRecognizer(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for i := 0; i < 75; i++ {
		if err = r.Accept(ctx, contracts.Frame{Identity: id, SampleRate: 16000, Channels: 1, Samples: make([]float32, 320)}); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := r.Finish(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Identity != id || !draft.Final || strings.TrimSpace(draft.Text) != "24000" {
		t.Fatalf("lost tail or identity: %+v", draft)
	}
}

func TestRejectsWrongFormatBeforeAccept(t *testing.T) {
	s := &session{rate: 16000}
	if err := s.Accept(context.Background(), contracts.Frame{SampleRate: 48000, Channels: 1}); err == nil {
		t.Fatal("accepted wrong sample rate")
	}
}

// Long audio and pause boundaries must partition all original samples exactly
// once, independent of how many replaceable previews complete along the way.
func TestNativeStableChunksPreserveAllAudio(t *testing.T) {
	exe, _ := os.Executable()
	t.Setenv("ARION_ASR_WORKER", exe)
	t.Setenv("ARION_ASR_TEST_WORKER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := LoadStreamingContext(ctx, Spec{Kind: QwenMLX, SampleRate: 16000, EndpointSilence: 0.8, Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	r, err := stream.NewRecognizer(ctx, contracts.Identity{SessionID: "chunks"})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	total := 0
	for i := 0; i < 1600; i++ {
		samples := make([]float32, 320)
		if i%150 < 100 {
			for j := range samples {
				samples[j] = 0.1
			}
		}
		total += len(samples)
		if err = r.Accept(ctx, contracts.Frame{SampleRate: 16000, Channels: 1, Samples: samples}); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := r.Finish(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for _, count := range strings.Fields(draft.Text) {
		n, err := strconv.Atoi(count)
		if err != nil {
			t.Fatal(err)
		}
		got += n
	}
	if got != total {
		t.Fatalf("audio partition lost or duplicated samples: got %d want %d (%q)", got, total, draft.Text)
	}
}
