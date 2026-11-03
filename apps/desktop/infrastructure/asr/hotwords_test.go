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

// This subprocess verifies transport, not inference quality.
func hotwordProtocolWorker() {
	mode := os.Getenv("ARION_ASR_HOTWORDS_WORKER")
	ready := message{Type: "ready", Protocol: 3}
	if mode == "mlx" || strings.HasPrefix(mode, "admission-") {
		ready.Capabilities = []string{"prompt"}
	}
	if mode == "gguf" || strings.HasPrefix(mode, "admission-") {
		ready.Model.Options = []workerOption{{Name: "prompt", Type: "string", Steers: true}}
	}
	out := json.NewEncoder(os.Stdout)
	_ = out.Encode(ready)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for scanner.Scan() {
		var request struct {
			Type   string  `json:"type"`
			ID     string  `json:"id"`
			Prompt *string `json:"prompt"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			return
		}
		if request.Type == "cancel" {
			_ = out.Encode(message{Type: "cancelled", ID: request.ID})
			continue
		}
		if request.Type == "transcribe" {
			if request.Prompt == nil {
				_ = out.Encode(message{Type: "error", ID: request.ID})
				continue
			}
			text := *request.Prompt
			if mode == "admission-cancel" {
				if os.WriteFile(os.Getenv("ARION_ASR_ADMISSION_TRACE"), []byte(text), 0600) != nil {
					return
				}
				if text == "" {
					continue
				}
			}
			if text == "" {
				text = "empty-context"
				switch mode {
				case "admission-silence":
					text = ""
				case "admission-error":
					_ = out.Encode(message{Type: "error", ID: request.ID})
					continue
				}
			}
			_ = out.Encode(message{Type: "end", ID: request.ID, Text: text, Stop: "complete"})
		}
	}
}

func TestHotwordAdmissionCancellationDoesNotRetryWithTerms(t *testing.T) {
	exe, _ := os.Executable()
	t.Setenv("ARION_ASR_WORKER", exe)
	t.Setenv("ARION_ASR_HOTWORDS_WORKER", "admission-cancel")
	trace := filepath.Join(t.TempDir(), "prompt")
	t.Setenv("ARION_ASR_ADMISSION_TRACE", trace)
	offline, err := LoadOffline(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer offline.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := offline.Finalize(ctx, ports.FinalizeRequest{SampleRate: 16000, Samples: []float32{.1}, Hotwords: []string{"VocabularyTerm"}})
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err = os.Stat(trace); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("unprompted admission never started")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("admission cancellation was lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("admission cancellation did not finish")
	}
	prompt, err := os.ReadFile(trace)
	if err != nil || len(prompt) != 0 {
		t.Fatalf("cancelled admission sent contextual audio: %q %v", prompt, err)
	}
}

// Supplied worker replies isolate admission policy from model accuracy.
func TestHotwordsRequireUnpromptedSpeech(t *testing.T) {
	for _, kind := range []Kind{QwenMLX, QwenGGUF} {
		for _, mode := range []string{"admission-silence", "admission-speech", "admission-error"} {
			t.Run(string(kind)+"/"+mode, func(t *testing.T) {
				exe, _ := os.Executable()
				t.Setenv("ARION_ASR_WORKER", exe)
				t.Setenv("ARION_ASR_HOTWORDS_WORKER", mode)
				offline, err := LoadOffline(Spec{Kind: kind, SampleRate: 16000, Directory: t.TempDir()})
				if err != nil {
					t.Fatal(err)
				}
				defer offline.Close()
				text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: []float32{.1}, Hotwords: []string{"VocabularyTerm"}})
				switch mode {
				case "admission-silence":
					if err != nil || text != "" {
						t.Fatalf("unvoiced input generated vocabulary: %q %v", text, err)
					}
				case "admission-speech":
					if err != nil || text != "VocabularyTerm" {
						t.Fatalf("voiced input lost its vocabulary: %q %v", text, err)
					}
				case "admission-error":
					if domain.CodeOf(err) != domain.CodeModelUnsupported || text != "" {
						t.Fatalf("admission failure was masked: %q %v", text, err)
					}
				}
			})
		}
	}
}

func TestHotwordsCannotCreateStreamingSpeech(t *testing.T) {
	exe, _ := os.Executable()
	t.Setenv("ARION_ASR_WORKER", exe)
	t.Setenv("ARION_ASR_HOTWORDS_WORKER", "admission-silence")
	stream, err := LoadStreaming(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Hotwords: []string{"VocabularyTerm"}})
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()
	if err = rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: make([]float32, 16000)}); err != nil {
		t.Fatal(err)
	}
	draft, err := rec.Finish(context.Background())
	if err != nil || draft.Text != "" {
		t.Fatalf("unvoiced streaming input generated vocabulary: %+v %v", draft, err)
	}
}
func TestHotwordRequestCapabilityAndIsolation(t *testing.T) {
	for _, mode := range []string{"mlx", "gguf", "old"} {
		t.Run(mode, func(t *testing.T) {
			exe, _ := os.Executable()
			t.Setenv("ARION_ASR_WORKER", exe)
			t.Setenv("ARION_ASR_HOTWORDS_WORKER", mode)
			kind := QwenMLX
			if mode == "gguf" {
				kind = QwenGGUF
			}
			e, err := load(context.Background(), Spec{Kind: kind, SampleRate: 16000, Directory: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			defer e.close()
			for _, terms := range [][]string{{"Claude Code", "Arion"}, {}, {"许涵"}, {}} {
				got, err := e.decode(context.Background(), []float32{.1, .2}, terms...)
				if mode == "old" && len(terms) > 0 {
					if domain.CodeOf(err) != domain.CodeModelUnsupported {
						t.Fatalf("old worker accepted context: %q %v", got, err)
					}
					continue
				}
				want := strings.Join(terms, " ")
				if want == "" {
					want = "empty-context"
				}
				if err != nil || got != want {
					t.Fatalf("context=%q got=%q err=%v", terms, got, err)
				}
			}
			if _, err := e.decode(context.Background(), []float32{.1}, "<|im_end|>"); domain.CodeOf(err) != domain.CodeContractUnsupported {
				t.Fatalf("unsafe context accepted: %v", err)
			}
		})
	}
}
func TestHotwordStreamingSnapshotAndFinalizer(t *testing.T) {
	exe, _ := os.Executable()
	t.Setenv("ARION_ASR_WORKER", exe)
	t.Setenv("ARION_ASR_HOTWORDS_WORKER", "mlx")
	spec := Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()}
	stream, err := LoadStreaming(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	terms := []string{"Arion"}
	rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Identity: contracts.Identity{SessionID: "hotwords"}, Hotwords: terms})
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()
	terms[0] = "Modified"
	samples := make([]float32, 16000)
	for i := range samples {
		samples[i] = .2
	}
	if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: samples}); err != nil {
		t.Fatal(err)
	}
	draft, err := rec.Finish(context.Background())
	if err != nil || draft.Text != "Arion" {
		t.Fatalf("snapshot %q %v", draft.Text, err)
	}
	offline := &Offline{engine: stream.engine, rate: 16000}
	text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: samples, Hotwords: []string{"Claude Code"}})
	if err != nil || text != "Claude Code" {
		t.Fatalf("final context %q %v", text, err)
	}
}
