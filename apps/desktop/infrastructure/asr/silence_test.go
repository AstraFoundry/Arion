package asr

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

func audible(n int) []float32 {
	samples := make([]float32, n)
	for i := range samples {
		samples[i] = 0.1
	}
	return samples
}

func fill(n int, value float32) []float32 {
	samples := make([]float32, n)
	for i := range samples {
		samples[i] = value
	}
	return samples
}

type promptCall struct {
	prompt  string
	samples int
}

func readPromptLog(t *testing.T, path string) []promptCall {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil
	}
	var calls []promptCall
	for _, line := range strings.Split(text, "\n") {
		var call promptCall
		if _, err := fmt.Sscanf(line, "%q\t%d", &call.prompt, &call.samples); err != nil {
			t.Fatalf("prompt log %q: %v", line, err)
		}
		calls = append(calls, call)
	}
	return calls
}

func hotwordEngine(t *testing.T, mode string) (*engine, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "prompts.log")
	if err := os.WriteFile(logPath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
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
	t.Cleanup(e.close)
	return e, logPath
}

func waitRevision(t *testing.T, rec ports.Recognizer, revision uint64) contracts.Draft {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last contracts.Draft
	for time.Now().Before(deadline) {
		draft, err := rec.Partial(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		last = draft
		if draft.Revision >= revision {
			return draft
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("revision %d not reached, last %+v", revision, last)
	return last
}

func TestSpeechFloorWindows(t *testing.T) {
	if speech, invalid := classifyPCM(nil); speech || invalid {
		t.Fatalf("empty audio: speech=%v invalid=%v", speech, invalid)
	}
	if speech, invalid := classifyPCM([]float32{float32(math.NaN())}); !invalid || speech {
		t.Fatalf("NaN: speech=%v invalid=%v", speech, invalid)
	}
	if speech, invalid := classifyPCM([]float32{float32(math.Inf(1))}); !invalid || speech {
		t.Fatalf("+Inf: speech=%v invalid=%v", speech, invalid)
	}
	if speech, invalid := classifyPCM([]float32{float32(math.Inf(-1))}); !invalid || speech {
		t.Fatalf("-Inf: speech=%v invalid=%v", speech, invalid)
	}
	if speech, _ := classifyPCM(make([]float32, 16000)); speech {
		t.Fatal("digital silence classified as speech")
	}
	if speech, _ := classifyPCM(fill(16000, 0.007)); speech {
		t.Fatal("noise below -42 dBFS classified as speech")
	}
	if speech, _ := classifyPCM(fill(320, 0.008)); !speech {
		t.Fatal("energy at the speech floor was ignored")
	}
	if speech, _ := classifyPCM([]float32{-0.2, -0.2}); !speech {
		t.Fatal("negative polarity was ignored")
	}
	click := make([]float32, 320)
	click[10] = 0.1
	if speech, _ := classifyPCM(click); speech {
		t.Fatal("one sub-floor sample in a 20 ms window counted as speech")
	}
	// Twenty loud samples sit inside one 20 ms span. Whole-buffer RMS is below the floor.
	spike := make([]float32, 16000)
	for i := 400; i < 420; i++ {
		spike[i] = 0.2
	}
	if speech, _ := classifyPCM(spike); !speech {
		t.Fatal("a short speech span was diluted by the surrounding quiet")
	}
}

func TestQuietAudioDoesNotEmitHotwords(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		e, logPath := hotwordEngine(t, "mlx")
		cases := []struct {
			name    string
			samples []float32
			terms   []string
			want    string
			sent    bool
			count   int
			passes  int
		}{
			{name: "zeros", samples: make([]float32, 16000), terms: []string{"Arion", "许涵"}, want: ""},
			{name: "below-floor", samples: fill(16000, 0.007), terms: []string{"Arion"}, want: ""},
			{name: "empty", samples: nil, terms: []string{"Arion"}, want: ""},
			{name: "single-click", samples: func() []float32 {
				s := make([]float32, 320)
				s[10] = 0.1
				return s
			}(), terms: []string{"Arion"}, want: ""},
			{name: "at-floor", samples: fill(320, 0.008), terms: []string{"Arion"}, want: "Arion", sent: true, count: 320},
			{name: "negative", samples: fill(320, -0.2), terms: []string{"Arion"}, want: "Arion", sent: true, count: 320},
			{name: "spike-keeps-context", samples: func() []float32 {
				s := make([]float32, 16000)
				for i := 400; i < 420; i++ {
					s[i] = 0.2
				}
				return s
			}(), terms: []string{"Claude Code", "Arion"}, want: "empty-context", sent: true, count: 16000, passes: 2},
			{name: "leading-silence-kept", samples: func() []float32 {
				s := make([]float32, 32000)
				for i := 16000; i < len(s); i++ {
					s[i] = 0.2
				}
				return s
			}(), terms: []string{"Arion", "许涵"}, want: "empty-context", sent: true, count: 32000, passes: 2},
			{name: "empty-context-speech", samples: audible(320), terms: nil, want: "empty-context", sent: true, count: 320},
			{name: "empty-context-silence", samples: make([]float32, 320), terms: nil, want: ""},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := os.WriteFile(logPath, nil, 0600); err != nil {
					t.Fatal(err)
				}
				got, err := e.decode(context.Background(), tc.samples, tc.terms...)
				if err != nil || got != tc.want {
					t.Fatalf("got %q %v, want %q", got, err, tc.want)
				}
				calls := readPromptLog(t, logPath)
				if !tc.sent {
					if len(calls) != 0 {
						t.Fatalf("quiet audio submitted context: %+v", calls)
					}
					return
				}
				passes := tc.passes
				if passes == 0 {
					passes = 1
				}
				joined := strings.Join(tc.terms, " ")
				if len(calls) != passes || calls[0].prompt != joined || calls[0].samples != tc.count {
					t.Fatalf("submission %+v, want %d prompt %q samples %d", calls, passes, joined, tc.count)
				}
				if passes == 2 && (calls[1].prompt != "" || calls[1].samples != tc.count) {
					t.Fatalf("echo was not redecoded without context: %+v", calls)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		e, logPath := hotwordEngine(t, "mlx")
		if _, err := e.decode(context.Background(), make([]float32, 16000), "<|im_end|>"); domain.CodeOf(err) != domain.CodeContractUnsupported {
			t.Fatalf("quiet invalid hotword: %v", err)
		}
		if _, err := e.decode(context.Background(), []float32{float32(math.NaN())}, "Arion"); domain.CodeOf(err) != domain.CodeContractUnsupported {
			t.Fatalf("NaN: %v", err)
		}
		quiet := make([]float32, 16000)
		quiet[10] = float32(math.Inf(1))
		if _, err := e.decode(context.Background(), quiet, "Arion"); domain.CodeOf(err) != domain.CodeContractUnsupported {
			t.Fatalf("Inf among silence: %v", err)
		}
		if calls := readPromptLog(t, logPath); len(calls) != 0 {
			t.Fatalf("invalid audio or terms reached the worker: %+v", calls)
		}
	})

	t.Run("old-worker", func(t *testing.T) {
		e, logPath := hotwordEngine(t, "old")
		got, err := e.decode(context.Background(), make([]float32, 16000), "Arion")
		if err != nil || got != "" {
			t.Fatalf("quiet unsupported worker: %q %v", got, err)
		}
		if _, err := e.decode(context.Background(), audible(320), "Arion"); domain.CodeOf(err) != domain.CodeModelUnsupported {
			t.Fatalf("speech on an unsupported worker: %v", err)
		}
		if calls := readPromptLog(t, logPath); len(calls) != 0 {
			t.Fatalf("unsupported worker received a prompt: %+v", calls)
		}
	})

	t.Run("preview-before-speech", func(t *testing.T) {
		_, logPath := hotwordEngine(t, "mlx")
		stream, err := LoadStreaming(Spec{Kind: QwenMLX, SampleRate: 16000, EndpointSilence: 0.8, Directory: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(stream.Close)
		rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Identity: contracts.Identity{SessionID: "quiet-preview"}, Hotwords: []string{"Arion"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = rec.Close() })
		if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: make([]float32, 16000)}); err != nil {
			t.Fatal(err)
		}
		draft := waitRevision(t, rec, 1)
		if draft.Text != "" || len(readPromptLog(t, logPath)) != 0 {
			t.Fatalf("preview before speech: %q log=%v", draft.Text, readPromptLog(t, logPath))
		}
		spoken := audible(16000)
		for i := range spoken {
			spoken[i] = 0.2
		}
		if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: spoken}); err != nil {
			t.Fatal(err)
		}
		draft = waitRevision(t, rec, 2)
		calls := readPromptLog(t, logPath)
		if draft.Text != "Arion" || len(calls) != 1 || calls[0].prompt != "Arion" || calls[0].samples != 32000 {
			t.Fatalf("speech after silence: text=%q calls=%+v", draft.Text, calls)
		}
		final, err := rec.Finish(context.Background())
		if err != nil || final.Text != "Arion" {
			t.Fatalf("finish: %q %v", final.Text, err)
		}
	})

	t.Run("finish-before-preview", func(t *testing.T) {
		_, logPath := hotwordEngine(t, "mlx")
		stream, err := LoadStreaming(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(stream.Close)
		rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Hotwords: []string{"Arion", "许涵"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = rec.Close() })
		if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: make([]float32, 8000)}); err != nil {
			t.Fatal(err)
		}
		draft, err := rec.Finish(context.Background())
		if err != nil || draft.Text != "" || !draft.Final || len(readPromptLog(t, logPath)) != 0 {
			t.Fatalf("short silence finish: %+v %v log=%v", draft, err, readPromptLog(t, logPath))
		}
	})

	t.Run("quiet-tail", func(t *testing.T) {
		_, logPath := hotwordEngine(t, "mlx")
		stream, err := LoadStreaming(Spec{Kind: QwenMLX, SampleRate: 16000, EndpointSilence: 0.8, Directory: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(stream.Close)
		rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Hotwords: []string{"Arion"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = rec.Close() })
		accept := func(samples []float32) {
			t.Helper()
			if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: samples}); err != nil {
				t.Fatal(err)
			}
		}
		accept(fill(16000, 0.2))
		accept(make([]float32, 12800))
		var committed contracts.Draft
		deadline := time.Now().Add(5 * time.Second)
		for {
			committed, err = rec.Partial(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(committed.Segments) > 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("speech never closed: %+v", committed)
			}
			time.Sleep(time.Millisecond)
		}
		if committed.Text != "Arion" {
			t.Fatalf("closed speech: %+v", committed)
		}
		logged := len(readPromptLog(t, logPath))
		if logged == 0 {
			t.Fatal("speech was not submitted")
		}
		seen := committed.Revision
		accept(make([]float32, 16000))
		later := waitRevision(t, rec, seen+1)
		if later.Text != "Arion" || later.Tail != "" || len(readPromptLog(t, logPath)) != logged {
			t.Fatalf("quiet tail changed the draft: %+v log=%v", later, readPromptLog(t, logPath))
		}
	})

	t.Run("long-silence", func(t *testing.T) {
		_, logPath := hotwordEngine(t, "mlx")
		stream, err := LoadStreaming(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(stream.Close)
		rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Hotwords: []string{"Arion", "许涵"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = rec.Close() })
		frame := make([]float32, 320)
		for range 20*50 + 2 {
			if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: frame}); err != nil {
				t.Fatal(err)
			}
		}
		draft, err := rec.Finish(context.Background())
		if err != nil || draft.Text != "" || len(draft.Segments) != 0 || len(readPromptLog(t, logPath)) != 0 {
			t.Fatalf("20s silence: %+v %v log=%v", draft, err, readPromptLog(t, logPath))
		}
	})

	t.Run("offline", func(t *testing.T) {
		e, logPath := hotwordEngine(t, "mlx")
		offline := &Offline{engine: e, rate: 16000}
		text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: make([]float32, 16000), Hotwords: []string{"Arion"}})
		if err != nil || text != "" || len(readPromptLog(t, logPath)) != 0 {
			t.Fatalf("offline silence: %q %v log=%v", text, err, readPromptLog(t, logPath))
		}
		spoken := make([]float32, 32000)
		for i := 16000; i < len(spoken); i++ {
			spoken[i] = 0.2
		}
		text, err = offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: spoken, Hotwords: []string{"Arion", "许涵"}})
		calls := readPromptLog(t, logPath)
		if err != nil || text != "empty-context" || len(calls) != 2 || calls[0].samples != 32000 || calls[0].prompt != "Arion 许涵" || calls[1].prompt != "" || calls[1].samples != 32000 {
			t.Fatalf("offline speech after silence: %q %v calls=%+v", text, err, calls)
		}
	})
}
