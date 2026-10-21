// Package engine defines the native rewrite worker contract.
package engine

import (
	"arion.local/core/domain"
	"errors"
)

// Config bounds one loaded model.
type Config struct {
	ModelDir     string
	Threads      int
	MaxLength    int
	MaxNewTokens int
}

// Request is one rewrite or prefill of a session's current source text.
type Request struct {
	Session string
	Scene   string
	Terms   []string
	// Context, when present, selects the context template.
	Context domain.RewriteContext
	Text    string
	// PreviousText is earlier recognized source for spelling evidence only.
	PreviousText string
	// Prefill computes the source tokens only and generates nothing.
	Prefill bool
	// Live marks a preview of still-open speech; its output budget is tighter.
	Live bool
}

// Result reports the candidate and where its time went.
type Result struct {
	Text    string               `json:"text"`
	Cleanup domain.SpeechCleanup `json:"cleanup"`
	// Template is the version of the prompt that served the request.
	Template     string  `json:"template"`
	InputTokens  int     `json:"inputTokens"`
	ReusedTokens int     `json:"reusedTokens"`
	OutputTokens int     `json:"outputTokens"`
	PrefillMs    float64 `json:"prefillMs"`
	DecodeMs     float64 `json:"decodeMs"`
	FirstTokenMs float64 `json:"firstTokenMs"`
	Truncated    bool    `json:"truncated"`
	Cancelled    bool    `json:"cancelled"`
}

// Errors are explicit outcomes, never a silent original-text substitute.
var (
	ErrInputBudget  = errors.New("input exceeds token budget")
	ErrControlInput = errors.New("input contains chat control markers")
)
