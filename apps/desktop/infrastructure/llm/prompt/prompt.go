// Package prompt builds the versioned rewrite prompt in the Qwen chat
// format. Trusted instructions and demonstrations form a fixed prefix whose
// KV state is computed once; dictated text only appears in the last user
// turn, as data.
package prompt

import (
	_ "embed"
	"encoding/json"
	"strings"

	"arion.local/core/domain"
)

//go:embed live.json
var live []byte

//go:embed context.json
var contextual []byte

// Example is one demonstration turn.
type Example struct {
	Scene   string                 `json:"scene"`
	Terms   []string               `json:"terms,omitempty"`
	Context *domain.RewriteContext `json:"context,omitempty"`
	Input   string                 `json:"input"`
	Output  string                 `json:"output"`
}

// Template is an immutable prompt version.
type Template struct {
	Version  string    `json:"version"`
	System   string    `json:"system"`
	Examples []Example `json:"examples"`
}

// Live is the shipped template; its version is recorded with every run.
func Live() Template {
	var t Template
	if err := json.Unmarshal(live, &t); err != nil {
		panic("embedded prompt invalid: " + err.Error())
	}
	return t
}

// Context extends Live for requests that carry context sections: its system
// prompt adds the context rules and its demonstrations follow Live's. The
// product uses this template for both previews and final rewrites whenever
// session context is available (ADR 013).
func Context() Template {
	var extra struct {
		Version  string    `json:"version"`
		System   string    `json:"system"`
		Examples []Example `json:"examples"`
	}
	if err := json.Unmarshal(contextual, &extra); err != nil {
		panic("embedded context prompt invalid: " + err.Error())
	}
	base := Live()
	return Template{Version: extra.Version + "+" + base.Version, System: base.System + " " + extra.System,
		Examples: append(base.Examples, extra.Examples...)}
}

// Input is one rewrite request's untrusted data.
type Input struct {
	Scene   string
	Terms   []string
	Context domain.RewriteContext
	Text    string
}

// Prefix is the fixed system and demonstration part shared by all requests.
func (t Template) Prefix() string {
	var b strings.Builder
	turn(&b, "system", t.System)
	for _, e := range t.Examples {
		in := Input{Scene: e.Scene, Terms: e.Terms, Text: e.Input}
		if e.Context != nil {
			in.Context = *e.Context
		}
		turn(&b, "user", user(in))
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
	// The template formats the model scenes; anything else is chat.
	scene := string(domain.SceneChat)
	for _, s := range []domain.Scene{domain.SceneChat, domain.SceneEmail, domain.SceneList} {
		if string(s) == in.Scene {
			scene = in.Scene
		}
	}
	var b strings.Builder
	b.WriteString("[" + scene + "]")
	if len(in.Terms) > 0 {
		b.WriteString(" [terms: " + strings.Join(in.Terms, "; ") + "]")
	}
	b.WriteString("\n")
	if c := in.Context.Bounded(); !c.Empty() {
		// Context precedes the dictation so a growing dictation still
		// extends the previous prompt.
		section(&b, "screen", c.Screen)
		section(&b, "before_cursor", c.Before)
		section(&b, "after_cursor", c.After)
		section(&b, "history", strings.Join(c.History, "\n"))
		b.WriteString("<dictation>\n")
	}
	b.WriteString(in.Text)
	return b.String()
}

func turn(b *strings.Builder, role, content string) {
	b.WriteString("<|im_start|>" + role + "\n" + content + "<|im_end|>\n")
}

func section(b *strings.Builder, name, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	b.WriteString("<" + name + ">\n" + text + "\n</" + name + ">\n")
}
