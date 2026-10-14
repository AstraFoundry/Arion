package prompt

import (
	"strings"
	"testing"
)

func TestPromptExtendsSource(t *testing.T) {
	tpl := Live()
	if tpl.Version == "" || len(tpl.Examples) == 0 {
		t.Fatal("embedded template incomplete")
	}
	in := Input{Scene: "email", Terms: []string{"瑞迪斯 → Redis"}, Text: "你好"}
	source, prompt := tpl.Source(in), tpl.Prompt(in)
	if !strings.HasPrefix(source, tpl.Prefix()) || !strings.HasPrefix(prompt, source) {
		t.Fatal("prompt does not extend the shared prefix and source")
	}
	if !strings.HasSuffix(prompt, "<|im_start|>assistant\n<think>\n\n</think>\n\n") {
		t.Fatal("non-thinking answer turn missing")
	}
	// A longer transcript extends the shorter one's source text.
	longer := tpl.Source(Input{Scene: "email", Terms: in.Terms, Text: "你好，世界"})
	if !strings.HasPrefix(longer, source) {
		t.Fatal("growing transcript does not extend its source prompt")
	}
}

func TestUnknownSceneIsChat(t *testing.T) {
	if got := user(Input{Scene: "poem", Text: "x"}); !strings.HasPrefix(got, "[chat]") {
		t.Fatalf("unknown scene rendered as %q", got)
	}
}

// Demonstrations must not contain chat control markers or reasoning.
func TestExamplesAreClean(t *testing.T) {
	for _, e := range Live().Examples {
		for _, s := range []string{e.Input, e.Output} {
			if strings.Contains(s, "<|") || strings.Contains(s, "<think>") {
				t.Fatalf("control text in example %q", s)
			}
		}
	}
}
