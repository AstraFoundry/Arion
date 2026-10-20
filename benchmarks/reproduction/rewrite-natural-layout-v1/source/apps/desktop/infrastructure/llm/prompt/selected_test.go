package prompt

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/spelling"
)

func TestSelectedSelectorParityAndFormatterIsolation(t *testing.T) {
	data, err := os.ReadFile("../../../../../benchmarks/reproduction/rewrite-product-integration-v1/payload-parity.json")
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
		// The old server had reasoning disabled globally. The product also pins
		// the equivalent per-request template argument explicitly.
		if !reflect.DeepEqual(got.TemplateArgs, map[string]bool{"enable_thinking": false}) {
			t.Fatal("thinking enabled")
		}
		got.TemplateArgs = nil
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
		got, err := recipe.Format(c.Source, c.Scene)
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
		"rewrite-selector-screen-v1/templates/baseline/prompts.json": selectedSelection,
	} {
		want, err := os.ReadFile("../../../../../benchmarks/reproduction/" + source)
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
			request, err := recipe.Format(source, string(scene))
			if err != nil {
				t.Fatalf("%s: %v", scene, err)
			}
			var last struct {
				Source string `json:"source"`
			}
			if err := json.Unmarshal([]byte(request.Messages[len(request.Messages)-1].Content), &last); err != nil || last.Source != source {
				t.Fatalf("source changed: %q (%v)", last.Source, err)
			}
			if request.MaxTokens != 256 || !reflect.DeepEqual(request.TemplateArgs, map[string]bool{"enable_thinking": false}) {
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
