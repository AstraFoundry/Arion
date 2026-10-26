package prompt

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"arion.local/core/domain"
)

const (
	minFormatTokens = 256
	maxFormatTokens = 2048
)

// SelectedVersion identifies both independently evaluated stages.
const SelectedVersion = "selector-contextual-v5+cleanup-rules-v2"

//go:embed selection.json
var selectedSelection []byte

//go:embed format.json
var selectedFormat []byte

// Message is a chat turn. Source/context remain untrusted user data.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is one polishing stage. AllowThinking permits reasoning tokens.
// The zero value keeps thinking off. Adapters translate that policy; this
// struct does not carry a provider field.
type ChatRequest struct {
	Messages       []Message       `json:"messages"`
	ResponseFormat any             `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
	Seed           int             `json:"seed"`
	MaxTokens      int             `json:"max_tokens"`
	CachePrompt    bool            `json:"cache_prompt"`
	Stream         bool            `json:"stream"`
	TemplateArgs   map[string]bool `json:"chat_template_kwargs,omitempty"`
	AllowThinking  bool            `json:"-"`
}

// LocalWire copies the thinking policy into the llama.cpp template argument.
func (r ChatRequest) LocalWire() ChatRequest {
	args := make(map[string]bool, len(r.TemplateArgs)+1)
	for key, value := range r.TemplateArgs {
		args[key] = value
	}
	args["enable_thinking"] = r.AllowThinking
	r.TemplateArgs = args
	return r
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
	return ChatRequest{Messages: messages, Seed: 42, MaxTokens: tokens, CachePrompt: true}
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
type translateTargetRecipe struct {
	System          string                     `json:"system"`
	Rules           string                     `json:"rules"`
	Route           string                     `json:"route"`
	EditingExamples []formatExample            `json:"editingExamples"`
	Scenes          map[string][]formatExample `json:"scenes"`
}
type formatRecipe struct {
	EditingExamples []formatExample                  `json:"editingExamples"`
	System          string                           `json:"formatSystem"`
	SystemHan       string                           `json:"formatSystemHan"`
	Rules           string                           `json:"formatRules"`
	RulesHan        string                           `json:"formatRulesHan"`
	ProsePolicy     string                           `json:"prosePolicy"`
	ProsePolicyHan  string                           `json:"prosePolicyHan"`
	Routes          map[string]string                `json:"routeInstructions"`
	Punctuation     string                           `json:"punctuationRule"`
	PunctuationHan  string                           `json:"punctuationRuleHan"`
	Scenes          map[string]sceneRecipe           `json:"format"`
	Translate       map[string]translateTargetRecipe `json:"translate"`
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

// Selection builds the bounded keep/replace protocol. The wire token
// "replace" is an ASR correction of a nominated mishearing, not a synonym
// swap. Ambiguous hypotheses can only keep; this is protocol coverage, not
// a semantic output validator.
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

func profile(source string) (han, latin bool) {
	for _, r := range source {
		if unicode.In(r, unicode.Han) && (r >= 0x3400 || r >= 0xf900) {
			han = true
		}
		if unicode.In(r, unicode.Latin) {
			latin = true
		}
	}
	return han, latin
}
func route(source string) string {
	han, latin := profile(source)
	if han && !latin {
		return "han"
	}
	if latin && !han {
		return "latin"
	}
	return "mixed"
}
func formatText(source string) string {
	return "<source>" + source + "</source>"
}

// formatTokens scales the formatter output budget with the source length,
// keeping a 256-token floor for short utterances and a 2048-token ceiling.
func formatTokens(source string) int {
	return min(maxFormatTokens, max(minFormatTokens, utf8.RuneCountInString(source)+128))
}

// formatTranslationTokens allows 2x expansion per source rune so Chinese-to-English
// translations do not exhaust the completion budget on longer utterances.
func formatTranslationTokens(source string) int {
	return min(maxFormatTokens, max(minFormatTokens, utf8.RuneCountInString(source)*2+128))
}

// systemPrompt is a preamble plus one rules list. Scene text is rule 8,
// layout is rule 9 (omitted for list), and the script route is rule 10.
// Rule 2 is the wording constraint: no synonym paraphrase; ASR fixes are corrections.
// Full, omit, and spaces append one punctuation bullet; smart does not.
func (t *SelectedTemplates) systemPrompt(kind, scene string, punctuation domain.Punctuation) string {
	preamble, rules, sceneRule, layout, routeLine, punct := t.format.System, t.format.Rules, "", t.format.ProsePolicy, t.format.Routes[kind], t.format.Punctuation
	recipe := t.format.Scenes[scene]
	sceneRule = recipe.Instruction
	if kind == "han" {
		preamble, rules, sceneRule, layout, punct = t.format.SystemHan, t.format.RulesHan, recipe.InstructionHan, t.format.ProsePolicyHan, t.format.PunctuationHan
	}
	lines := strings.Split(rules, "\n")
	lines = append(lines, sceneRule)
	if scene != string(domain.SceneList) {
		lines = append(lines, layout)
	}
	lines = append(lines, routeLine)
	switch punctuation {
	case domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces:
		lines = append(lines, punct)
	}
	return preamble + "\n\n<rules>\n" + strings.Join(lines, "\n") + "\n</rules>"
}

// Format receives only the explicitly edited source, never OCR or surrounding
// context. Scene hints preserve natural prose unless list layout is explicit.
// punctuation appends the complete-punctuation bullet for full, omit, and
// spaces; smart and empty leave the approved prompt unchanged.
func (t *SelectedTemplates) Format(source, scene string, punctuation domain.Punctuation) (ChatRequest, error) {
	if scene == string(domain.SceneSearch) {
		scene = string(domain.SceneChat)
	}
	recipe, ok := t.format.Scenes[scene]
	if !ok {
		return ChatRequest{}, fmt.Errorf("unsupported formatting scene %q", scene)
	}
	kind := route(source)
	messages := []Message{{Role: "system", Content: t.systemPrompt(kind, scene, punctuation)}}
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
	return chat(append(messages, Message{Role: "user", Content: formatText(source)}), formatTokens(source)), nil
}

func (t *SelectedTemplates) translationSystemPrompt(scene string, punctuation domain.Punctuation, target domain.TranslationLanguage) string {
	tr := t.format.Translate[string(target)]
	recipe := t.format.Scenes[scene]
	sceneRule, layout, punct := recipe.Instruction, t.format.ProsePolicy, t.format.Punctuation
	if target == domain.TargetChinese {
		sceneRule, layout, punct = recipe.InstructionHan, t.format.ProsePolicyHan, t.format.PunctuationHan
	}
	lines := strings.Split(tr.Rules, "\n")
	lines = append(lines, sceneRule)
	if scene != string(domain.SceneList) {
		lines = append(lines, layout)
	}
	lines = append(lines, tr.Route)
	switch punctuation {
	case domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces:
		lines = append(lines, punct)
	}
	return tr.System + "\n\n<rules>\n" + strings.Join(lines, "\n") + "\n</rules>"
}

// FormatTranslation builds the Stage 2 request that polishes and translates
// the edited source into target while preserving scene layout, punctuation
// rules, and code/identifier fidelity.
func (t *SelectedTemplates) FormatTranslation(source, scene string, punctuation domain.Punctuation, target domain.TranslationLanguage) (ChatRequest, error) {
	if !target.Known() {
		return ChatRequest{}, fmt.Errorf("unsupported translation target %q", target)
	}
	if scene == string(domain.SceneSearch) {
		scene = string(domain.SceneChat)
	}
	if _, ok := t.format.Scenes[scene]; !ok {
		return ChatRequest{}, fmt.Errorf("unsupported formatting scene %q", scene)
	}
	tr, ok := t.format.Translate[string(target)]
	if !ok {
		return ChatRequest{}, fmt.Errorf("missing translation recipe for %q", target)
	}
	sceneExamples, ok := tr.Scenes[scene]
	if !ok || len(sceneExamples) == 0 {
		return ChatRequest{}, fmt.Errorf("missing translation examples for %s/%s", scene, target)
	}
	messages := []Message{{Role: "system", Content: t.translationSystemPrompt(scene, punctuation, target)}}
	groups := [][]formatExample{sceneExamples, tr.EditingExamples}
	if scene == string(domain.SceneList) {
		groups[0], groups[1] = groups[1], groups[0]
	}
	for group, examples := range groups {
		for _, e := range examples {
			output := e.Output
			if scene == string(domain.SceneList) && group == 0 {
				output = "1. " + output
			}
			messages = append(messages, Message{Role: "user", Content: formatText(e.Source)}, Message{Role: "assistant", Content: output})
		}
	}
	return chat(append(messages, Message{Role: "user", Content: formatText(source)}), formatTranslationTokens(source)), nil
}
