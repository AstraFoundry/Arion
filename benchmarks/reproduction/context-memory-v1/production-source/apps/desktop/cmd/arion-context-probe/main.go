// Command arion-context-probe renders actual product prompts and validation
// for frozen, authored research. It performs no model inference itself.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/vocabulary"
)

type request struct {
	Operation string                `json:"operation"`
	Mode      string                `json:"mode"`
	Text      string                `json:"text"`
	Candidate string                `json:"candidate"`
	Truncated bool                  `json:"truncated"`
	Context   domain.RewriteContext `json:"context"`
	Entries   []vocabulary.Entry    `json:"entries"`
	Terms     []string              `json:"terms"`
	Template  *prompt.Template      `json:"template"`
}

func main() {
	var r request
	if err := json.NewDecoder(os.Stdin).Decode(&r); err != nil {
		panic(err)
	}
	v := domain.NewVocabulary(domain.Replacements(vocabulary.Memory(r.Entries)))
	source := r.Text
	terms := r.Terms
	var template prompt.Template
	var context domain.RewriteContext
	if r.Mode == "product" {
		source, _ = v.Apply(source)
		terms = append(v.Terms(source, domain.MaxRecallTerms), domain.ContextTerms(r.Context, source)...)
		template = prompt.Live()
	} else {
		context = r.Context.Bounded()
		template = prompt.Context()
		if r.Template != nil {
			template = *r.Template
		}
	}
	result := map[string]any{"source": source, "terms": terms, "modelContext": context, "template": template.Version}
	switch r.Operation {
	case "render":
		result["prompt"] = template.Prompt(prompt.Input{Scene: "chat", Terms: terms, Context: context, Text: source})
	case "validate":
		result["verdict"] = domain.ValidateRewrite(domain.RewriteCheck{Source: source, Candidate: r.Candidate, Truncated: r.Truncated, Terms: v.Terms(source, domain.MaxRecallTerms), Context: r.Context})
	default:
		fmt.Fprintln(os.Stderr, "render or validate required")
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
