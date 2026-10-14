// Package prompt builds the versioned rewrite prompt in the Qwen chat
// format. Trusted instructions and demonstrations form a fixed prefix whose
// KV state is computed once; dictated text only appears in the last user
// turn, as data.
package prompt

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed live.json
var live []byte

// Example is one demonstration turn.
type Example struct {
	Scene  string   `json:"scene"`
	Terms  []string `json:"terms,omitempty"`
	Input  string   `json:"input"`
	Output string   `json:"output"`
}

// Template is an immutable prompt version.
type Template struct {
	Version  string    `json:"version"`
	System   string    `json:"system"`
	Examples []Example `json:"examples"`
}

// Scenes the template formats; anything else is rendered as chat.
var scenes = map[string]bool{"chat": true, "email": true, "list": true}

// Live is the shipped template; its version is recorded with every run.
func Live() Template {
	var t Template
	if err := json.Unmarshal(live, &t); err != nil {
		panic("embedded prompt invalid: " + err.Error())
	}
	return t
}

// Input is one rewrite request's untrusted data.
type Input struct {
	Scene string
	Terms []string
	Text  string
}

// Prefix is the fixed system and demonstration part shared by all requests.
func (t Template) Prefix() string {
	var b strings.Builder
	turn(&b, "system", t.System)
	for _, e := range t.Examples {
		turn(&b, "user", user(Input{Scene: e.Scene, Terms: e.Terms, Text: e.Input}))
		turn(&b, "assistant", e.Output)
	}
	return b.String()
}

// Source is the prefix plus the user turn up to and including its text, so
// a growing transcript extends the previous prompt token-for-token as long
// as the tokenizer keeps earlier boundaries.
func (t Template) Source(in Input) string {
	return t.Prefix() + "<|im_start|>user\n" + user(in)
}

// Prompt closes the user turn and opens a non-thinking assistant answer.
func (t Template) Prompt(in Input) string {
	return t.Source(in) + "<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n"
}

func user(in Input) string {
	scene := in.Scene
	if !scenes[scene] {
		scene = "chat"
	}
	var b strings.Builder
	b.WriteString("[" + scene + "]")
	if len(in.Terms) > 0 {
		b.WriteString(" [terms: " + strings.Join(in.Terms, "; ") + "]")
	}
	b.WriteString("\n")
	b.WriteString(in.Text)
	return b.String()
}

func turn(b *strings.Builder, role, content string) {
	b.WriteString("<|im_start|>" + role + "\n" + content + "<|im_end|>\n")
}
