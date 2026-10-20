package prompt

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/spelling"
)

func TestSelectedPayloadParity(t *testing.T) {
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
		compare(got, c.Request)
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
		"rewrite-work-scope-v4/prompts.json":                         selectedFormat,
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
