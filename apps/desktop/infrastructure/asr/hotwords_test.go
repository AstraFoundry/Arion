package asr

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// This subprocess verifies transport, not inference quality.
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
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	counts := map[string]int{}
	for scanner.Scan() {
		var request struct {
			Type   string  `json:"type"`
			ID     string  `json:"id"`
			PCM    string  `json:"pcm"`
			Prompt *string `json:"prompt"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			return
		}
		switch request.Type {
		case "chunk":
			if pcm, err := base64.StdEncoding.DecodeString(request.PCM); err == nil {
				counts[request.ID] += len(pcm) / 2
			}
		case "transcribe":
			if request.Prompt == nil {
				_ = out.Encode(message{Type: "error", ID: request.ID})
				continue
			}
			if path := os.Getenv("ARION_ASR_PROMPT_LOG"); path != "" {
				file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					return
				}
				_, _ = fmt.Fprintf(file, "%q\t%d\n", *request.Prompt, counts[request.ID])
				_ = file.Close()
			}
			text := *request.Prompt
			if text == "" {
				if plain, ok := os.LookupEnv("ARION_ASR_PLAIN_TEXT"); ok {
					text = plain
				} else {
					text = "empty-context"
				}
			} else if bias, ok := os.LookupEnv("ARION_ASR_BIAS_TEXT"); ok {
				text = bias
			}
			_ = out.Encode(message{Type: "end", ID: request.ID, Text: text, Stop: "complete"})
		}
	}
}
func TestHotwordRequestCapabilityAndIsolation(t *testing.T) {
	for _, mode := range []string{"mlx", "gguf", "old"} {
		t.Run(mode, func(t *testing.T) {
			exe, _ := os.Executable()
			logPath := filepath.Join(t.TempDir(), "prompts.log")
			if err := os.WriteFile(logPath, nil, 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ARION_ASR_WORKER", exe)
			t.Setenv("ARION_ASR_HOTWORDS_WORKER", mode)
			t.Setenv("ARION_ASR_PROMPT_LOG", logPath)
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
				if err := os.WriteFile(logPath, nil, 0600); err != nil {
					t.Fatal(err)
				}
				got, err := e.decode(context.Background(), []float32{.1, .2}, terms...)
				calls := readPromptLog(t, logPath)
				joined := strings.Join(terms, " ")
				if mode == "old" && len(terms) > 0 {
					if domain.CodeOf(err) != domain.CodeModelUnsupported || len(calls) != 0 {
						t.Fatalf("old worker accepted context: %q %v calls=%+v", got, err, calls)
					}
					continue
				}
				if len(terms) >= 2 {
					if err != nil || got != "empty-context" || len(calls) != 2 || calls[0].prompt != joined || calls[1].prompt != "" {
						t.Fatalf("list echo context=%q got=%q err=%v calls=%+v", terms, got, err, calls)
					}
					continue
				}
				want := joined
				if want == "" {
					want = "empty-context"
				}
				if err != nil || got != want || len(calls) != 1 || calls[0].prompt != joined {
					t.Fatalf("context=%q got=%q err=%v calls=%+v", terms, got, err, calls)
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
