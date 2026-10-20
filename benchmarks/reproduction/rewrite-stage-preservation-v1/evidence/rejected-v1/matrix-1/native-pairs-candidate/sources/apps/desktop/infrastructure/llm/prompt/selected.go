package prompt

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"arion.local/core/domain"
)

// SelectedVersion identifies both independently evaluated stages.
const SelectedVersion = "selector-contextual-v2+natural-layout-v3"

//go:embed selection.json
var selectedSelection []byte

//go:embed format.json
var selectedFormat []byte

// Message is a chat turn. Source/context remain untrusted user data.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest uses the checkpoint's embedded Jinja template, with thinking off.
type ChatRequest struct {
	Messages       []Message       `json:"messages"`
	ResponseFormat any             `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
	Seed           int             `json:"seed"`
	MaxTokens      int             `json:"max_tokens"`
	CachePrompt    bool            `json:"cache_prompt"`
	Stream         bool            `json:"stream"`
	TemplateArgs   map[string]bool `json:"chat_template_kwargs"`
}

func jsonText(value any) string {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(value); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}
func chat(messages []Message, tokens int) ChatRequest {
	return ChatRequest{Messages: messages, Seed: 42, MaxTokens: tokens, CachePrompt: true, TemplateArgs: map[string]bool{"enable_thinking": false}}
}

type selectionExample struct {
	Scene   string            `json:"scene"`
	Source  string            `json:"source"`
	Context map[string]string `json:"context"`
	Choice  string            `json:"choice"`
	Expect  int               `json:"expectHypotheses"`
}
type selectionRecipe struct {
	System   string             `json:"selectionSystem"`
	Examples []selectionExample `json:"selectionExamples"`
}
type formatExample struct {
	Source string `json:"source"`
	Output string `json:"output"`
}
type sceneRecipe struct {
	Instruction    string          `json:"instruction"`
	InstructionHan string          `json:"instructionHan"`
	Examples       []formatExample `json:"examples"`
}
type formatRecipe struct {
	EditingExamples []formatExample        `json:"editingExamples"`
	System          string                 `json:"formatSystem"`
	SystemHan       string                 `json:"formatSystemHan"`
	ProsePolicy     string                 `json:"prosePolicy"`
	ProsePolicyHan  string                 `json:"prosePolicyHan"`
	Routes          map[string]string      `json:"routeInstructions"`
	Scenes          map[string]sceneRecipe `json:"format"`
}

// SelectedTemplates holds immutable parsed templates and their fixed selector turns.
type SelectedTemplates struct {
	selection []Message
	format    formatRecipe
}

// Selected parses the shipped templates and checks example producer coverage.
func Selected(lexicon *domain.SpellingLexicon) (*SelectedTemplates, error) {
	var s selectionRecipe
	var f formatRecipe
	if err := json.Unmarshal(selectedSelection, &s); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(selectedFormat, &f); err != nil {
		return nil, err
	}
	t := &SelectedTemplates{selection: []Message{{Role: "system", Content: s.System}}, format: f}
	for _, e := range s.Examples {
		p := lexicon.Prepare(e.Source, e.Context, e.Scene)
		if len(p.Hypotheses) != e.Expect {
			return nil, fmt.Errorf("selector example producer coverage changed")
		}
		choices := map[string]string{}
		for _, h := range p.Hypotheses {
			choice := "keep"
			if e.Choice == "replaceUnique" && h.Status == "unique" {
				choice = "replace"
			}
			choices[h.ID] = choice
		}
		t.selection = append(t.selection, Message{Role: "user", Content: selectionText(p)}, Message{Role: "assistant", Content: jsonText(choices)})
	}
	return t, nil
}

type selectionHypothesis struct {
	ID         string   `json:"id"`
	Source     string   `json:"sourceSpan"`
	Candidates []string `json:"candidateSpellings"`
	Status     string   `json:"status"`
	Kind       string   `json:"kind"`
}

func selectionText(p domain.SpellingPreparation) string {
	hypotheses := []selectionHypothesis{}
	for _, h := range p.Hypotheses {
		hypotheses = append(hypotheses, selectionHypothesis{h.ID, h.Source, h.Candidates, h.Status, h.Kind})
	}
	return jsonText(struct {
		Source     string                `json:"source"`
		Context    map[string]string     `json:"context"`
		Scene      string                `json:"scene"`
		Unknown    []string              `json:"unknownChineseCharacters"`
		Hypotheses []selectionHypothesis `json:"hypotheses"`
	}{p.Source, p.Context, p.Scene, p.Unknown, hypotheses})
}

// Selection builds the bounded keep/replace protocol. Ambiguous hypotheses
// can only keep; this is protocol coverage, not a semantic output validator.
func (t *SelectedTemplates) Selection(p domain.SpellingPreparation) ChatRequest {
	properties := map[string]any{}
	required := []string{}
	for _, h := range p.Hypotheses {
		choices := []string{"keep"}
		if h.Status == "unique" {
			choices = append(choices, "replace")
		}
		properties[h.ID] = map[string]any{"type": "string", "enum": choices}
		required = append(required, h.ID)
	}
	req := chat(append(append([]Message{}, t.selection...), Message{Role: "user", Content: selectionText(p)}), 128)
	req.ResponseFormat = map[string]any{"type": "json_object", "schema": map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}}
	return req
}

type sourceScripts struct {
	Han   bool `json:"hasHan"`
	Latin bool `json:"hasLatin"`
}

func profile(source string) sourceScripts {
	p := sourceScripts{}
	for _, r := range source {
		if unicode.In(r, unicode.Han) && (r >= 0x3400 || r >= 0xf900) {
			p.Han = true
		}
		if unicode.In(r, unicode.Latin) {
			p.Latin = true
		}
	}
	return p
}
func route(source string) string {
	p := profile(source)
	if p.Han && !p.Latin {
		return "han"
	}
	if p.Latin && !p.Han {
		return "latin"
	}
	return "mixed"
}
func formatText(source string) string {
	return jsonText(struct {
		Source  string        `json:"source"`
		Scripts sourceScripts `json:"sourceScripts"`
	}{source, profile(source)})
}

// Format receives only the explicitly edited source, never OCR or surrounding
// context. Scene hints preserve natural prose unless list layout is explicit.
func (t *SelectedTemplates) Format(source, scene string) (ChatRequest, error) {
	if scene == string(domain.SceneSearch) {
		scene = string(domain.SceneChat)
	}
	recipe, ok := t.format.Scenes[scene]
	if !ok {
		return ChatRequest{}, fmt.Errorf("unsupported formatting scene %q", scene)
	}
	kind := route(source)
	system := t.format.System + "\n" + recipe.Instruction
	if kind == "han" {
		system = t.format.SystemHan + "\n" + recipe.InstructionHan
	}
	if scene != string(domain.SceneList) {
		policy := t.format.ProsePolicy
		if kind == "han" {
			policy = t.format.ProsePolicyHan
		}
		system += "\n" + policy
	}
	system += "\n" + t.format.Routes[kind]
	messages := []Message{{Role: "system", Content: system}}
	groups := [][]formatExample{recipe.Examples, t.format.EditingExamples}
	// Keep explicit-list demonstrations last so prose editing examples do not
	// override the user-selected layout.
	if scene == string(domain.SceneList) {
		groups[0], groups[1] = groups[1], groups[0]
	}
	for group, examples := range groups {
		for _, e := range examples {
			if kind != "mixed" && route(e.Source) != kind {
				continue
			}
			output := e.Output
			if scene == string(domain.SceneList) && group == 0 {
				output = "1. " + output
			}
			messages = append(messages, Message{Role: "user", Content: formatText(e.Source)}, Message{Role: "assistant", Content: output})
		}
	}
	if len(messages) == 1 {
		return ChatRequest{}, fmt.Errorf("missing formatting examples for %s/%s", scene, kind)
	}
	return chat(append(messages, Message{Role: "user", Content: formatText(source)}), 256), nil
}
