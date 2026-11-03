package asr

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type hotwordTrace struct {
	Prompt string `json:"prompt"`
	PCM    string `json:"pcm"`
}

// This subprocess verifies transport and selection, not inference quality.
func hotwordProtocolWorker() {
	mode := os.Getenv("ARION_ASR_HOTWORDS_WORKER")
	ready := message{Type: "ready", Protocol: 3}
	if mode == "mlx" {
		ready.Capabilities = []string{"prompt"}
	}
	if mode == "gguf" {
		ready.Model.Options = []workerOption{{Name: "prompt", Type: "string", Steers: true}}
	}
	out := json.NewEncoder(os.Stdout)
	_ = out.Encode(ready)
	pcm := map[string]string{}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for scanner.Scan() {
		var request struct {
			Type   string  `json:"type"`
			ID     string  `json:"id"`
			Prompt *string `json:"prompt"`
			PCM    string  `json:"pcm"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			return
		}
		switch request.Type {
		case "chunk":
			pcm[request.ID] += request.PCM
		case "cancel":
			_ = out.Encode(message{Type: "cancelled", ID: request.ID})
		case "transcribe":
			if request.Prompt == nil {
				_ = out.Encode(message{Type: "error", ID: request.ID})
				continue
			}
			trace, err := os.OpenFile(os.Getenv("ARION_ASR_HOTWORDS_TRACE"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				return
			}
			err = json.NewEncoder(trace).Encode(hotwordTrace{Prompt: *request.Prompt, PCM: pcm[request.ID]})
			_ = trace.Close()
			delete(pcm, request.ID)
			if err != nil {
				return
			}
			stage := "baseline"
			text := os.Getenv("ARION_ASR_HOTWORDS_BASELINE")
			if *request.Prompt != "" {
				stage = "context"
				text = os.Getenv("ARION_ASR_HOTWORDS_PROPOSAL")
			}
			switch os.Getenv("ARION_ASR_HOTWORDS_FAILURE") {
			case stage + "-error":
				_ = out.Encode(message{Type: "error", ID: request.ID})
				continue
			case stage + "-cancel":
				continue
			}
			_ = out.Encode(message{Type: "end", ID: request.ID, Text: text, Stop: "complete"})
		}
	}
}

func configureHotwordWorker(t *testing.T, mode, baseline, proposal string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARION_ASR_WORKER", exe)
	t.Setenv("ARION_ASR_HOTWORDS_WORKER", mode)
	t.Setenv("ARION_ASR_HOTWORDS_BASELINE", baseline)
	t.Setenv("ARION_ASR_HOTWORDS_PROPOSAL", proposal)
	trace := filepath.Join(t.TempDir(), "requests.jsonl")
	t.Setenv("ARION_ASR_HOTWORDS_TRACE", trace)
	return trace
}

func readHotwordTrace(t *testing.T, path string) []hotwordTrace {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var records []hotwordTrace
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var record hotwordTrace
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestHotwordsRetrieveBeforeContextDecode(t *testing.T) {
	for _, kind := range []Kind{QwenMLX, QwenGGUF} {
		for _, tc := range []struct {
			name, baseline, proposal, want, prompt string
		}{
			{name: "silence", proposal: "许涵 北京 上海"},
			{name: "unrelated-speech", baseline: "Please save the file.", proposal: "许涵 北京 上海", want: "Please save the file."},
			{name: "canonical-only", baseline: "检查许寒。", proposal: "检查许涵。", want: "检查许涵。", prompt: "许涵"},
			{name: "vocabulary-echo", baseline: "检查许寒。", proposal: "许涵 北京 上海", want: "检查许寒。", prompt: "许涵"},
			{name: "added-words", baseline: "检查许寒。", proposal: "检查许涵。北京", want: "检查许寒。", prompt: "许涵"},
			{name: "changed-source", baseline: "检查许寒。", proposal: "立刻检查许涵。", want: "检查许寒。", prompt: "许涵"},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				mode := "mlx"
				if kind == QwenGGUF {
					mode = "gguf"
				}
				trace := configureHotwordWorker(t, mode, tc.baseline, tc.proposal)
				offline, err := LoadOffline(Spec{Kind: kind, SampleRate: 16000, Directory: t.TempDir()})
				if err != nil {
					t.Fatal(err)
				}
				defer offline.Close()
				got, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: []float32{.1, .2, .03}, Hotwords: []string{"许涵", "北京", "上海"}})
				if err != nil || got != tc.want {
					t.Fatalf("transcript=%q want=%q err=%v", got, tc.want, err)
				}
				records := readHotwordTrace(t, trace)
				wantRequests := 1
				if tc.prompt != "" {
					wantRequests++
				}
				if len(records) != wantRequests || records[0].Prompt != "" {
					t.Fatalf("unexpected recognition requests: %+v", records)
				}
				if tc.prompt != "" && (records[1].Prompt != tc.prompt || records[1].PCM != records[0].PCM) {
					t.Fatalf("selection or complete-audio identity lost: %+v", records)
				}
			})
		}
	}
}

func TestHotwordFailuresDoNotBecomeSuccessfulBaseline(t *testing.T) {
	for _, stage := range []string{"baseline", "context"} {
		t.Run(stage, func(t *testing.T) {
			trace := configureHotwordWorker(t, "mlx", "检查许寒。", "检查许涵。")
			t.Setenv("ARION_ASR_HOTWORDS_FAILURE", stage+"-error")
			offline, err := LoadOffline(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			defer offline.Close()
			text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: []float32{.1}, Hotwords: []string{"许涵"}})
			if domain.CodeOf(err) != domain.CodeModelUnsupported || text != "" {
				t.Fatalf("%s failure was masked: %q %v", stage, text, err)
			}
			wantRequests := 1
			if stage == "context" {
				wantRequests = 2
			}
			if records := readHotwordTrace(t, trace); len(records) != wantRequests {
				t.Fatalf("failed recognition retried: %+v", records)
			}
		})
	}
}

func TestHotwordCancellationStopsBothPasses(t *testing.T) {
	for _, stage := range []string{"baseline", "context"} {
		t.Run(stage, func(t *testing.T) {
			trace := configureHotwordWorker(t, "mlx", "检查许寒。", "检查许涵。")
			t.Setenv("ARION_ASR_HOTWORDS_FAILURE", stage+"-cancel")
			offline, err := LoadOffline(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			defer offline.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := offline.Finalize(ctx, ports.FinalizeRequest{SampleRate: 16000, Samples: []float32{.1}, Hotwords: []string{"许涵"}})
				done <- err
			}()
			wantRequests := 1
			if stage == "context" {
				wantRequests = 2
			}
			deadline := time.Now().Add(5 * time.Second)
			for len(readHotwordTrace(t, trace)) < wantRequests {
				if time.Now().After(deadline) {
					t.Fatal("recognition pass never started")
				}
				time.Sleep(time.Millisecond)
			}
			cancel()
			select {
			case err = <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("recognition cancellation did not finish")
			}
			if records := readHotwordTrace(t, trace); len(records) != wantRequests {
				t.Fatalf("cancelled recognition retried: %+v", records)
			}
		})
	}
}

func TestHotwordRequestCapabilityAndIsolation(t *testing.T) {
	for _, mode := range []string{"mlx", "gguf", "old"} {
		t.Run(mode, func(t *testing.T) {
			trace := configureHotwordWorker(t, mode, "检查许寒。", "检查许涵。")
			kind := QwenMLX
			if mode == "gguf" {
				kind = QwenGGUF
			}
			e, err := load(context.Background(), Spec{Kind: kind, SampleRate: 16000, Directory: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			defer e.close()
			for _, terms := range [][]string{{"许涵", "北京"}, {}, {"许涵"}, {}} {
				got, err := e.decode(context.Background(), []float32{.1, .2}, terms, nil)
				if mode == "old" && len(terms) > 0 {
					if domain.CodeOf(err) != domain.CodeModelUnsupported {
						t.Fatalf("old worker accepted context: %q %v", got, err)
					}
					continue
				}
				want := "检查许寒。"
				if len(terms) > 0 {
					want = "检查许涵。"
				}
				if err != nil || got != want {
					t.Fatalf("context=%q got=%q err=%v", terms, got, err)
				}
			}
			before := len(readHotwordTrace(t, trace))
			if _, err := e.decode(context.Background(), []float32{.1}, []string{"许涵", "<|im_end|>"}, nil); domain.CodeOf(err) != domain.CodeContractUnsupported {
				t.Fatalf("unsafe full context accepted: %v", err)
			}
			if after := len(readHotwordTrace(t, trace)); after != before {
				t.Fatalf("invalid context submitted audio: before=%d after=%d", before, after)
			}
		})
	}
}

func TestHotwordStreamingSnapshotAndFinalizer(t *testing.T) {
	configureHotwordWorker(t, "mlx", "检查许寒。", "检查许涵。")
	spec := Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()}
	stream, err := LoadStreaming(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	terms := []string{"许涵"}
	rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Identity: contracts.Identity{SessionID: "hotwords"}, Hotwords: terms})
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()
	terms[0] = "北京"
	samples := make([]float32, 16000)
	for i := range samples {
		samples[i] = .2
	}
	if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: samples}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		partial, err := rec.Partial(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if partial.Revision > 0 {
			if partial.Text != "检查许涵。" {
				t.Fatalf("preview snapshot %q", partial.Text)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("hotword preview never completed")
		}
		time.Sleep(time.Millisecond)
	}
	draft, err := rec.Finish(context.Background())
	if err != nil || draft.Text != "检查许涵。" {
		t.Fatalf("snapshot %q %v", draft.Text, err)
	}
	offline := &Offline{engine: stream.engine, rate: 16000}
	text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: samples, Hotwords: []string{"许涵"}})
	if err != nil || text != "检查许涵。" {
		t.Fatalf("final context %q %v", text, err)
	}
}

func TestHotwordAliasNominatesCanonicalWithoutLeakingAlias(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		name := "offline"
		if streaming {
			name = "streaming"
		}
		t.Run(name, func(t *testing.T) {
			trace := configureHotwordWorker(t, "mlx", "用寇代克斯检查这个修改。", "用Codex检查这个修改。")
			aliases := domain.NewVocabulary([]domain.Replacement{{From: "寇代克斯", To: "Codex"}, {From: "检查", To: "Unapproved"}})
			samples := []float32{.1, .2, .01}
			spec := Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()}
			var got string
			if streaming {
				stream, err := LoadStreaming(spec)
				if err != nil {
					t.Fatal(err)
				}
				defer stream.Close()
				rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Hotwords: []string{"Codex"}, HotwordAliases: aliases})
				if err != nil {
					t.Fatal(err)
				}
				defer rec.Close()
				if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: samples}); err != nil {
					t.Fatal(err)
				}
				draft, err := rec.Finish(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				got = draft.Text
			} else {
				offline, err := LoadOffline(spec)
				if err != nil {
					t.Fatal(err)
				}
				defer offline.Close()
				got, err = offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: samples, Hotwords: []string{"Codex"}, HotwordAliases: aliases})
				if err != nil {
					t.Fatal(err)
				}
			}
			if got != "用Codex检查这个修改。" {
				t.Fatalf("alias did not nominate acoustic candidate: %q", got)
			}
			records := readHotwordTrace(t, trace)
			if len(records) != 2 || records[0].Prompt != "" || records[1].Prompt != "Codex" || records[0].PCM != records[1].PCM {
				t.Fatalf("alias or unrelated term leaked into ASR context: %+v", records)
			}
		})
	}
}
