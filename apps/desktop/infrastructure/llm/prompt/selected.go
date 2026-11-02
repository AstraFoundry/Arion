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

// SelectedVersion identifies the selector and contextual editing recipe.
const SelectedVersion = "selector-contextual-v5+contextual-editing-v1"

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
	Grammar        string          `json:"grammar,omitempty"`
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
	Instruction    string `json:"instruction"`
	InstructionHan string `json:"instructionHan"`
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
	ContextRules    string                           `json:"contextRules"`
	ContextRulesHan string                           `json:"contextRulesHan"`
	Output          string                           `json:"outputRule"`
	OutputHan       string                           `json:"outputRuleHan"`
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

// formatInput keeps reference data in the user turn, separate from the source.
// JSON escaping prevents reference text from closing either data wrapper.
func formatInput(source string, reference domain.RewriteContext) string {
	reference = reference.Bounded()
	if reference.Empty() {
		return formatText(source)
	}
	data := struct {
		Screen  string   `json:"screen,omitempty"`
		Before  string   `json:"before_cursor,omitempty"`
		After   string   `json:"after_cursor,omitempty"`
		History []string `json:"history,omitempty"`
	}{reference.Screen, reference.Before, reference.After, reference.History}
	encoded, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return "<context>\n" + string(encoded) + "\n</context>\n\n" + formatText(source)
}

// formatBudget scales the completion ceiling with the source length.
// perRune is 1 for a polish and 2 for a translation, which can expand.
// Short utterances stay on a 256-token floor. The ceiling is 2048.
func formatBudget(source string, perRune int) int {
	return min(maxFormatTokens, max(minFormatTokens, utf8.RuneCountInString(source)*perRune+128))
}

// rulesBlock is a preamble plus one rules list. The scene line follows the
// recipe rules, layout is omitted for list, and the route line is last.
// Full, omit, and spaces append one punctuation bullet; smart does not.
func rulesBlock(preamble, rules, sceneRule, layout, routeLine, scene string, punctuation domain.Punctuation, punct string) string {
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

// exampleGroups puts editing examples first for an explicit list, so prose
// demonstrations do not override the selected layout.
func exampleGroups(scene string, primary, editing []formatExample) [][]formatExample {
	groups := [][]formatExample{primary, editing}
	if scene == string(domain.SceneList) {
		groups[0], groups[1] = groups[1], groups[0]
	}
	return groups
}

func scriptExamples(examples []formatExample, kind string) []formatExample {
	if kind == "mixed" {
		return examples
	}
	matched := make([]formatExample, 0, len(examples))
	for _, e := range examples {
		if route(e.Source) == kind {
			matched = append(matched, e)
		}
	}
	return matched
}

func appendExampleTurns(messages []Message, groups [][]formatExample, scene string) []Message {
	for group, examples := range groups {
		for _, e := range examples {
			output := e.Output
			if scene == string(domain.SceneList) && group == 0 {
				output = "1. " + output
			}
			messages = append(messages, Message{Role: "user", Content: formatText(e.Source)}, Message{Role: "assistant", Content: output})
		}
	}
	return messages
}

// systemPrompt puts role, task, scene and reference policy in one instruction
// message. Compact editing examples are embedded rather than chat history.
func (t *SelectedTemplates) systemPrompt(kind, scene string, punctuation domain.Punctuation) string {
	preamble, rules, layout, routeLine, punct := t.format.System, t.format.Rules, t.format.ProsePolicy, t.format.Routes[kind], t.format.Punctuation
	contextRules, output := t.format.ContextRules, t.format.Output
	recipe := t.format.Scenes[scene]
	sceneRule := recipe.Instruction
	if kind == "han" {
		preamble, rules, sceneRule, layout, punct = t.format.SystemHan, t.format.RulesHan, recipe.InstructionHan, t.format.ProsePolicyHan, t.format.PunctuationHan
		contextRules, output = t.format.ContextRulesHan, t.format.OutputHan
	}
	sections := []string{preamble, rules, "# Scene\n\n" + sceneRule}
	if scene != string(domain.SceneList) {
		sections = append(sections, layout)
	}
	sections = append(sections, routeLine, "# Context\n\n"+contextRules)
	switch punctuation {
	case domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces:
		sections = append(sections, "# Punctuation\n\n"+punct)
	}
	examples := []string{}
	for _, e := range scriptExamples(t.format.EditingExamples, kind) {
		if kind == "han" {
			examples = append(examples, "原文："+e.Source+"\n输出："+e.Output)
		} else {
			examples = append(examples, "Input: "+e.Source+"\nOutput: "+e.Output)
		}
	}
	sections = append(sections, "# Examples\n\n"+strings.Join(examples, "\n\n"), "# Output\n\n"+output)
	return strings.Join(sections, "\n\n")
}

// Format edits the source with a scene policy and optional bounded reference
// data. Reference text never enters the system message or editing examples.
func (t *SelectedTemplates) Format(source, scene string, punctuation domain.Punctuation, reference domain.RewriteContext) (ChatRequest, error) {
	if scene == string(domain.SceneSearch) {
		scene = string(domain.SceneChat)
	}
	_, ok := t.format.Scenes[scene]
	if !ok {
		return ChatRequest{}, fmt.Errorf("unsupported formatting scene %q", scene)
	}
	kind := route(source)
	messages := []Message{{Role: "system", Content: t.systemPrompt(kind, scene, punctuation)}}
	return chat(append(messages, Message{Role: "user", Content: formatInput(source, reference)}), formatBudget(source, 1)), nil
}

func (t *SelectedTemplates) translationSystemPrompt(scene string, punctuation domain.Punctuation, target domain.TranslationLanguage) string {
	tr := t.format.Translate[string(target)]
	recipe := t.format.Scenes[scene]
	sceneRule, layout, punct := recipe.Instruction, t.format.ProsePolicy, t.format.Punctuation
	if target == domain.TargetChinese {
		sceneRule, layout, punct = recipe.InstructionHan, t.format.ProsePolicyHan, t.format.PunctuationHan
	}
	contextRules := t.format.ContextRules
	if target == domain.TargetChinese {
		contextRules = t.format.ContextRulesHan
	}
	return rulesBlock(tr.System, tr.Rules, sceneRule, layout, tr.Route, scene, punctuation, punct) + "\n\n# Context\n\n" + contextRules
}

// FormatTranslation builds the Stage 2 request that polishes and translates
// the edited source into target while preserving scene layout, punctuation
// rules, and code/identifier fidelity.
func (t *SelectedTemplates) FormatTranslation(source, scene string, punctuation domain.Punctuation, target domain.TranslationLanguage, reference domain.RewriteContext) (ChatRequest, error) {
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
	messages = appendExampleTurns(messages, exampleGroups(scene, sceneExamples, tr.EditingExamples), scene)
	return chat(append(messages, Message{Role: "user", Content: formatInput(source, reference)}), formatBudget(source, 2)), nil
}
