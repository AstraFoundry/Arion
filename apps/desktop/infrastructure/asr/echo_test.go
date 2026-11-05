package asr

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain/ports"
)

func TestHotwordEchoShapes(t *testing.T) {
	terms := []string{"Arion", "许涵", "Claude Code"}
	cities := []string{"北京", "上海", "广州", "深圳"}
	cases := []struct {
		name  string
		text  string
		terms []string
		echo  bool
	}{
		{name: "empty", text: "", terms: terms, echo: false},
		{name: "one-term-list", text: "Arion", terms: []string{"Arion"}, echo: false},
		{name: "single-mention", text: "Arion", terms: terms, echo: false},
		{name: "full-list", text: "Arion 许涵", terms: []string{"Arion", "许涵"}, echo: true},
		{name: "punctuation-and-case", text: "arion，许涵。", terms: []string{"Arion", "许涵"}, echo: true},
		{name: "spaced-phrase", text: "Claude  Code   Arion", terms: []string{"Claude Code", "Arion"}, echo: true},
		{name: "breath-label", text: "气声：Arion 许涵", terms: []string{"Arion", "许涵"}, echo: true},
		{name: "ascii-colon", text: "气声: Arion 许涵", terms: []string{"Arion", "许涵"}, echo: true},
		{name: "repeated-label", text: "气声：气声：北京 上海", terms: cities, echo: true},
		{name: "label-only", text: "气声", terms: terms, echo: false},
		{name: "colon-only", text: "气声：", terms: terms, echo: false},
		{name: "one-term-with-label", text: "气声：Arion", terms: []string{"Arion"}, echo: false},
		{name: "trailing-run", text: "我们开始。Arion 许涵", terms: []string{"Arion", "许涵"}, echo: true},
		{name: "leading-run", text: "Arion 许涵，我们开始", terms: []string{"Arion", "许涵"}, echo: true},
		{name: "one-trailing-term", text: "Hello Arion", terms: terms, echo: false},
		{name: "middle-run", text: "今天 Arion 许涵 开会", terms: []string{"Arion", "许涵"}, echo: false},
		{name: "reversed", text: "许涵 Arion", terms: []string{"Arion", "许涵"}, echo: false},
		{name: "non-contiguous", text: "Arion Claude Code", terms: terms, echo: false},
		{name: "concatenated-han", text: "北京上海广州", terms: cities, echo: true},
		{name: "glued-trailing", text: "我去北京上海", terms: []string{"北京", "上海"}, echo: true},
		{name: "ascii-glued", text: "GoTo", terms: []string{"Go", "To"}, echo: false},
		{name: "term-boundary", text: "ArionX 许涵", terms: []string{"Arion", "许涵"}, echo: false},
		{name: "repeated-list", text: "Arion 许涵 Arion 许涵", terms: []string{"Arion", "许涵"}, echo: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hotwordEcho(tc.text, tc.terms); got != tc.echo {
				t.Fatalf("hotwordEcho(%q) = %v, want %v", tc.text, got, tc.echo)
			}
		})
	}
}

func TestHotwordEchoRedecode(t *testing.T) {
	t.Run("breath-label", func(t *testing.T) {
		assertRedecode(t, "气声：Arion 许涵", "轻声那句", []string{"Arion", "许涵"}, "轻声那句", 2)
	})
	t.Run("trailing-run", func(t *testing.T) {
		assertRedecode(t, "我们开始。 Arion 许涵", "我们开始。", []string{"Arion", "许涵"}, "我们开始。", 2)
	})
	t.Run("empty-second-pass", func(t *testing.T) {
		assertRedecode(t, "北京 上海 广州 深圳", "", []string{"北京", "上海", "广州", "深圳"}, "", 2)
	})
	t.Run("sentence-kept", func(t *testing.T) {
		assertRedecode(t, "今天开会", "", []string{"Arion", "许涵"}, "今天开会", 1)
	})
	t.Run("reversed-kept", func(t *testing.T) {
		assertRedecode(t, "许涵 Arion", "", []string{"Arion", "许涵"}, "许涵 Arion", 1)
	})
	t.Run("single-term-kept", func(t *testing.T) {
		assertRedecode(t, "气声：Arion", "", []string{"Arion"}, "气声：Arion", 1)
	})
	t.Run("preview-and-finish", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "prompts.log")
		if err := os.WriteFile(logPath, nil, 0600); err != nil {
			t.Fatal(err)
		}
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("ARION_ASR_WORKER", exe)
		t.Setenv("ARION_ASR_HOTWORDS_WORKER", "mlx")
		t.Setenv("ARION_ASR_PROMPT_LOG", logPath)
		t.Setenv("ARION_ASR_BIAS_TEXT", "气声：Arion 许涵")
		t.Setenv("ARION_ASR_PLAIN_TEXT", "轻声那句")
		stream, err := LoadStreaming(Spec{Kind: QwenMLX, SampleRate: 16000, Directory: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(stream.Close)
		rec, err := stream.NewRecognizer(context.Background(), ports.RecognitionRequest{Identity: contracts.Identity{SessionID: "echo"}, Hotwords: []string{"Arion", "许涵"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = rec.Close() })
		if err := rec.Accept(context.Background(), contracts.Frame{SampleRate: 16000, Channels: 1, Samples: audible(16000)}); err != nil {
			t.Fatal(err)
		}
		draft := waitRevision(t, rec, 1)
		calls := readPromptLog(t, logPath)
		if draft.Text != "轻声那句" || len(calls) != 2 || calls[0].prompt != "Arion 许涵" || calls[1].prompt != "" {
			t.Fatalf("preview published %q calls=%+v", draft.Text, calls)
		}
		final, err := rec.Finish(context.Background())
		if err != nil || final.Text != "轻声那句" || len(readPromptLog(t, logPath)) != 2 {
			t.Fatalf("finish %q %v calls=%+v", final.Text, err, readPromptLog(t, logPath))
		}
	})
	t.Run("offline", func(t *testing.T) {
		t.Setenv("ARION_ASR_BIAS_TEXT", "气声：北京 上海")
		t.Setenv("ARION_ASR_PLAIN_TEXT", "轻声那句")
		e, logPath := hotwordEngine(t, "mlx")
		offline := &Offline{engine: e, rate: 16000}
		text, err := offline.Finalize(context.Background(), ports.FinalizeRequest{SampleRate: 16000, Samples: audible(16000), Hotwords: []string{"北京", "上海"}})
		calls := readPromptLog(t, logPath)
		if err != nil || text != "轻声那句" || len(calls) != 2 || calls[0].prompt != "北京 上海" || calls[1].prompt != "" || calls[1].samples != 16000 {
			t.Fatalf("offline %q %v calls=%+v", text, err, calls)
		}
	})
}

func assertRedecode(t *testing.T, bias, plain string, terms []string, want string, passes int) {
	t.Helper()
	t.Setenv("ARION_ASR_BIAS_TEXT", bias)
	t.Setenv("ARION_ASR_PLAIN_TEXT", plain)
	e, logPath := hotwordEngine(t, "mlx")
	got, err := e.decode(context.Background(), audible(320), terms...)
	calls := readPromptLog(t, logPath)
	if err != nil || got != want || len(calls) != passes {
		t.Fatalf("got %q %v calls=%+v, want %q in %d passes", got, err, calls, want, passes)
	}
	if calls[0].prompt != joinTerms(terms) {
		t.Fatalf("first prompt %q", calls[0].prompt)
	}
	if passes == 2 && (calls[1].prompt != "" || calls[1].samples != calls[0].samples) {
		t.Fatalf("second pass %+v", calls[1])
	}
}

func joinTerms(terms []string) string {
	out := ""
	for i, term := range terms {
		if i > 0 {
			out += " "
		}
		out += term
	}
	return out
}
