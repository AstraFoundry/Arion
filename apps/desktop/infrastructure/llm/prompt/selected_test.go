package prompt

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/spelling"
)

func TestSelectedSelectorParityAndEmptyReference(t *testing.T) {
	data, err := os.ReadFile("testdata/payload-parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Selectors []struct {
			Prepared domain.SpellingPreparation `json:"prepared"`
			Request  ChatRequest                `json:"request"`
		} `json:"selectors"`
		Formatters []struct {
			Source  string      `json:"source"`
			Scene   string      `json:"scene"`
			Request ChatRequest `json:"request"`
		} `json:"formatters"`
	}
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	l, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := Selected(l)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures.Selectors) != 101 || len(fixtures.Formatters) != 16 {
		t.Fatal("incomplete payload fixtures")
	}
	compare := func(got, want ChatRequest) {
		t.Helper()
		// Thinking stays off on the request. The local worker writes the template
		// argument when it sends the request.
		if got.AllowThinking || len(got.TemplateArgs) != 0 {
			t.Fatal("thinking enabled")
		}
		var a, b any
		json.Unmarshal([]byte(jsonText(got)), &a)
		json.Unmarshal([]byte(jsonText(want)), &b)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("payload changed\ngot %s\nwant %s", jsonText(got), jsonText(want))
		}
	}
	for _, c := range fixtures.Selectors {
		compare(recipe.Selection(c.Prepared), c.Request)
	}
	for _, c := range fixtures.Formatters {
		got, err := recipe.Format(c.Source, c.Scene, "", domain.RewriteContext{})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range got.Messages {
			if bytes.Contains([]byte(m.Content), []byte(`"context":`)) {
				t.Fatal("formatter received context")
			}
		}
	}
}

func TestSelectedTemplateOriginalBytes(t *testing.T) {
	for source, got := range map[string][]byte{
		"baseline-prompts.json": selectedSelection,
	} {
		want, err := os.ReadFile("testdata/" + source)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("template changed: %s", source)
		}
	}
}

// Payload tests check routing and isolation; actual model outputs are reviewed
// separately by TestNativeSelectedPipeline.
func TestSelectedNaturalLayoutRouting(t *testing.T) {
	l, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := Selected(l)
	if err != nil {
		t.Fatal(err)
	}
	for _, scene := range domain.Scenes {
		for _, source := range []string{"请确认是否准备好，暂时不要重启。", "Can you check readiness? Do not restart it.", "请检查 fetchState，不改配置。"} {
			request, err := recipe.Format(source, string(scene), "", domain.RewriteContext{})
			if err != nil {
				t.Fatalf("%s: %v", scene, err)
			}
			last := request.Messages[len(request.Messages)-1].Content
			if last != "<source>"+source+"</source>" || strings.Contains(last, "sourceScripts") {
				t.Fatalf("source wrapper changed: %q", last)
			}
			if request.MaxTokens != 256 || request.AllowThinking || len(request.TemplateArgs) != 0 {
				t.Fatal("formatter generation settings changed")
			}
			effective := scene
			if effective == domain.SceneSearch {
				effective = domain.SceneChat
			}
			expected := recipe.format.Scenes[string(effective)].Instruction
			if route(source) == "han" {
				expected = recipe.format.Scenes[string(effective)].InstructionHan
			}
			if !strings.Contains(request.Messages[0].Content, expected) {
				t.Fatalf("wrong recipe for %s/%s", scene, route(source))
			}
			policy := recipe.format.ProsePolicy
			if route(source) == "han" {
				policy = recipe.format.ProsePolicyHan
			}
			if policy == "" || strings.Contains(request.Messages[0].Content, policy) != (scene != domain.SceneList) {
				t.Fatalf("prose/list policy conflict for %s/%s", scene, route(source))
			}
		}
	}
}

// Editing demonstrations remain in the system instruction, not chat history.
func TestSelectedEditingExamplesRouting(t *testing.T) {
	lexicon, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := Selected(lexicon)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"把阈值设成零点五。", "Set the threshold to zero point five.", "把 threshold 设成零点五。"} {
		for _, scene := range domain.Scenes {
			req, err := templates.Format(source, string(scene), "", domain.RewriteContext{})
			if err != nil {
				t.Fatal(err)
			}
			if len(req.Messages) != 2 || req.Messages[0].Role != "system" || req.Messages[1].Role != "user" {
				t.Fatalf("editing examples became conversation history: %+v", req.Messages)
			}
			for _, example := range scriptExamples(templates.format.EditingExamples, route(source)) {
				if !strings.Contains(req.Messages[0].Content, example.Source) || !strings.Contains(req.Messages[0].Content, example.Output) {
					t.Fatalf("missing compact example for %s/%s", scene, route(source))
				}
			}
		}
	}
}

func TestCleanupPromptWrapperAndPunctuationModes(t *testing.T) {
	lexicon, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := Selected(lexicon)
	if err != nil {
		t.Fatal(err)
	}
	for _, scene := range domain.Scenes {
		for _, source := range []string{"请确认是否准备好。", "Please check readiness.", "请检查 fetchState。"} {
			base, err := templates.Format(source, string(scene), domain.PunctuationSmart, domain.RewriteContext{})
			if err != nil {
				t.Fatal(err)
			}
			system := base.Messages[0].Content
			for _, heading := range []string{"# Role", "# Task", "# Input", "# Rules", "# Scene", "# Context", "# Examples", "# Output"} {
				if !strings.Contains(system, heading) {
					t.Fatalf("missing %s", heading)
				}
			}
			for _, line := range strings.Split(system, "\n") {
				if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "## 1.") {
					t.Fatalf("unordered rule or numbered heading: %q", line)
				}
			}
			punct := templates.format.Punctuation
			if route(source) == "han" {
				punct = templates.format.PunctuationHan
			}
			if strings.Contains(system, punct) {
				t.Fatal("smart appended complete-punctuation preference")
			}
			for _, mode := range []domain.Punctuation{domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces} {
				got, err := templates.Format(source, string(scene), mode, domain.RewriteContext{})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(got.Messages[0].Content, punct) ||
					strings.Replace(got.Messages[0].Content, "# Punctuation\n\n"+punct+"\n\n", "", 1) != system ||
					got.Messages[1] != base.Messages[1] {
					t.Fatalf("punctuation preference changed source or other rules: %s/%s", scene, mode)
				}
			}
		}
	}
}

func TestFormatDynamicTokenBudget(t *testing.T) {
	lexicon, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := Selected(lexicon)
	if err != nil {
		t.Fatal(err)
	}
	short, err := templates.Format("请确认是否准备好。", string(domain.SceneChat), domain.PunctuationSmart, domain.RewriteContext{})
	if err != nil || short.MaxTokens != 256 {
		t.Fatalf("short MaxTokens = %d, want 256 (%v)", short.MaxTokens, err)
	}
	mediumSource := strings.Repeat("今天先检查日志并确认参数状态。", 20) // 300 runes
	medium, err := templates.Format(mediumSource, string(domain.SceneChat), domain.PunctuationSmart, domain.RewriteContext{})
	if err != nil || medium.MaxTokens != 300+128 {
		t.Fatalf("medium MaxTokens = %d, want %d (%v)", medium.MaxTokens, 300+128, err)
	}
	longSource := strings.Repeat("Please check the service logs and keep the configuration unchanged. ", 40)
	long, err := templates.Format(longSource, string(domain.SceneChat), domain.PunctuationSmart, domain.RewriteContext{})
	if err != nil || long.MaxTokens != 2048 {
		t.Fatalf("long MaxTokens = %d, want 2048 (%v)", long.MaxTokens, err)
	}
}

// scripts are the writing systems each target's examples must use; Han
// appears only where the target is written with it.
var scripts = map[domain.TranslationLanguage][]*unicode.RangeTable{
	domain.TargetEnglish: {unicode.Latin}, domain.TargetChinese: {unicode.Han}, domain.TargetChineseTraditional: {unicode.Han},
	domain.TargetJapanese: {unicode.Hiragana, unicode.Katakana}, domain.TargetKorean: {unicode.Hangul},
	domain.TargetSpanish: {unicode.Latin}, domain.TargetFrench: {unicode.Latin}, domain.TargetGerman: {unicode.Latin},
	domain.TargetPortuguese: {unicode.Latin}, domain.TargetItalian: {unicode.Latin},
	domain.TargetRussian: {unicode.Cyrillic}, domain.TargetArabic: {unicode.Arabic},
}

func inScript(target domain.TranslationLanguage, out string) bool {
	tables, hanAllowed := scripts[target], target == domain.TargetChinese || target == domain.TargetChineseTraditional || target == domain.TargetJapanese
	found := false
	for _, r := range out {
		if unicode.Is(unicode.Han, r) && !hanAllowed {
			return false
		}
		if unicode.In(r, tables...) {
			found = true
		}
	}
	return found
}

func TestFormatTranslationPromptsAndTokenBudget(t *testing.T) {
	lexicon, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := Selected(lexicon)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range domain.TranslationLanguages {
		for _, scene := range domain.Scenes {
			for _, punct := range []domain.Punctuation{domain.PunctuationSmart, domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces} {
				source := "请检查 fetchState 的返回结果，暂时不要重启。"
				req, err := templates.FormatTranslation(source, string(scene), punct, target, domain.RewriteContext{})
				if err != nil {
					t.Fatalf("%s/%s/%s: %v", target, scene, punct, err)
				}
				if len(req.Messages) < 5 || req.Messages[len(req.Messages)-1].Content != "<source>"+source+"</source>" {
					t.Fatalf("unexpected messages for %s/%s", target, scene)
				}
				for i := 2; i < len(req.Messages)-1; i += 2 {
					if out := req.Messages[i].Content; !inScript(target, out) {
						t.Fatalf("%s translation example %d is not in the target's script: %q", target, i, out)
					}
				}
			}
		}
	}
	mediumSource := strings.Repeat("今天先检查日志并确认参数状态。", 20) // 300 runes
	trReq, err := templates.FormatTranslation(mediumSource, string(domain.SceneChat), domain.PunctuationSmart, domain.TargetEnglish, domain.RewriteContext{})
	if err != nil || trReq.MaxTokens != 300*2+128 {
		t.Fatalf("translation MaxTokens = %d, want %d (%v)", trReq.MaxTokens, 300*2+128, err)
	}
	if len(scripts) != len(domain.TranslationLanguages) {
		t.Fatalf("scripts cover %d of %d targets", len(scripts), len(domain.TranslationLanguages))
	}
	if _, err := templates.FormatTranslation("你好", "chat", "", "fr-CA", domain.RewriteContext{}); err == nil {
		t.Fatal("unsupported target language accepted")
	}
}

func TestPolishExamplesPreserveStructuredDetails(t *testing.T) {
	lexicon, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	templates, err := Selected(lexicon)
	if err != nil {
		t.Fatal(err)
	}
	for _, example := range templates.format.EditingExamples {
		switch {
		case strings.Contains(example.Source, "登录页"):
			for _, detail := range []string{"手机", "太挤", "间距", "错误提示", "更清楚", "提交后", "正在处理", "接口先别动", "周五应该", "测试通过后"} {
				if !strings.Contains(example.Output, detail) {
					t.Errorf("structured example dropped %q", detail)
				}
			}
		case strings.Contains(example.Source, "login"):
			for _, detail := range []string{"mobile", "crowded", "spacing", "error message", "processing", "API unchanged", "should", "Friday", "only after testing passes"} {
				if !strings.Contains(example.Output, detail) {
					t.Errorf("structured example dropped %q", detail)
				}
			}
		}
	}
}
