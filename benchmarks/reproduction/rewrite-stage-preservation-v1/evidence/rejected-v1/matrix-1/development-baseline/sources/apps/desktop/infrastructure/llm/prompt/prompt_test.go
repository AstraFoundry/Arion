package prompt

import (
	"strings"
	"testing"

	"arion.local/core/domain"
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

// Every layout the coordinator can ask for is demonstrated, and
// demonstrations use no other layout.
func TestLegacyModelSceneIsDemonstrated(t *testing.T) {
	shown := map[domain.Scene]bool{}
	for _, e := range Live().Examples {
		shown[domain.Scene(e.Scene)] = true
	}
	for _, scene := range []domain.Scene{domain.SceneChat, domain.SceneEmail, domain.SceneList} {
		if !shown[scene] {
			t.Errorf("no demonstration for %s", scene)
		}
		delete(shown, scene)
	}
	for scene := range shown {
		t.Errorf("demonstration uses %s, which the coordinator never asks for", scene)
	}
}

func TestContextScenesAreNotModelTags(t *testing.T) {
	for _, scene := range []domain.Scene{domain.SceneWork, domain.ScenePrompt, domain.SceneSearch} {
		if got := user(Input{Scene: string(scene), Text: "x"}); !strings.HasPrefix(got, "[chat]") {
			t.Errorf("%s rendered as %q", scene, got)
		}
	}
}

func TestFullOCRIsSeparateFromDictation(t *testing.T) {
	screen := strings.Repeat("会议内容。", 300) + "订阅费用可按报销流程申请。"
	got := user(Input{Scene: "chat", Context: domain.RewriteContext{Screen: screen, Before: "问题：", After: "谢谢"}, Text: "codex能播吗？"})
	for _, part := range []string{"<screen>\n" + screen + "\n</screen>", "<before_cursor>\n问题：\n</before_cursor>", "<after_cursor>\n谢谢\n</after_cursor>", "<dictation>\ncodex能播吗？"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing section %q", part[len(part)-min(40, len(part)):])
		}
	}
	if strings.Contains(got, "[terms:") {
		t.Fatal("OCR must not be reduced to terms")
	}
}
