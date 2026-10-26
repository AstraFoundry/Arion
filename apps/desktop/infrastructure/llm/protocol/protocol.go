// Package protocol defines the JSON-lines messages between the desktop
// process and its LLM worker. Requests are served in order; cancel is
// handled out of band and stops a running or queued request.
package protocol

import (
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/prompt"
)

// Version is bumped on any incompatible message change.
const Version = 2

// Ops.
const (
	OpLoad     = "load"
	OpRewrite  = "rewrite"
	OpComplete = "complete"
	OpPrefill  = "prefill"
	OpCancel   = "cancel"
	OpEnd      = "end"
	OpClose    = "close"
)

// Request is one line on the worker's stdin.
type Request struct {
	Version int    `json:"version"`
	ID      uint64 `json:"id"`
	Op      string `json:"op"`
	// Load.
	Runtime      string `json:"runtime,omitempty"`
	Model        string `json:"model,omitempty"`
	Threads      int    `json:"threads,omitempty"`
	MaxLength    int    `json:"maxLength,omitempty"`
	MaxNewTokens int    `json:"maxNewTokens,omitempty"`
	// Rewrite, prefill and end.
	Session string   `json:"session,omitempty"`
	Scene   string   `json:"scene,omitempty"`
	Terms   []string `json:"terms,omitempty"`
	// Context is the request's untrusted context, when it has any.
	Context      *domain.RewriteContext `json:"context,omitempty"`
	Text         string                 `json:"text,omitempty"`
	PreviousText string                 `json:"previousText,omitempty"`
	Live         bool                   `json:"live,omitempty"`
	Chat         *prompt.ChatRequest    `json:"chat,omitempty"`
	// Punctuation is the cleanup mode. Empty is smart.
	Punctuation string `json:"punctuation,omitempty"`
	// Cancel.
	Target uint64 `json:"target,omitempty"`
}

// Response is one line on the worker's stdout, answering Request.ID.
type Response struct {
	ID    uint64 `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// Load.
	LoadMs       float64 `json:"loadMs,omitempty"`
	PromptID     string  `json:"promptId,omitempty"`
	PrefixTokens int     `json:"prefixTokens,omitempty"`
	// Rewrite and prefill.
	Template     string                `json:"template,omitempty"`
	Text         string                `json:"text,omitempty"`
	Cleanup      *domain.SpeechCleanup `json:"cleanup,omitempty"`
	InputTokens  int                   `json:"inputTokens,omitempty"`
	ReusedTokens int                   `json:"reusedTokens,omitempty"`
	OutputTokens int                   `json:"outputTokens,omitempty"`
	PrefillMs    float64               `json:"prefillMs,omitempty"`
	DecodeMs     float64               `json:"decodeMs,omitempty"`
	FirstTokenMs float64               `json:"firstTokenMs,omitempty"`
	Truncated    bool                  `json:"truncated,omitempty"`
	Cancelled    bool                  `json:"cancelled,omitempty"`
	PeakRSS      int64                 `json:"peakRssBytes,omitempty"`
}
