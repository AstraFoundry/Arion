// Package engine runs rewrite requests on one resident GenAI model inside
// the LLM worker process. A single KV state is reused by its longest common
// token prefix: the fixed instruction prefix is computed once at load, and
// a growing transcript is prefilled incrementally while the user speaks.
package engine

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/genai"
	"arion.local/desktop/infrastructure/llm/prompt"
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
	// Prefill computes the source tokens only and generates nothing.
	Prefill bool
	// Live marks a preview of still-open speech; its output budget is tighter.
	Live bool
}

// budget bounds new tokens by the source size: a faithful edit is about as
// long as its input, so a runaway answer or repetition loop stops early and
// surfaces as truncated. Scene formatting needs more room than a preview.
func (e *Engine) budget(inputTokens int, live bool) int {
	n := inputTokens*2 + 48
	if live {
		n = inputTokens*3/2 + 16
	}
	return min(n, e.config.MaxNewTokens)
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

// Engine is not safe for concurrent use; the worker serializes requests.
type Engine struct {
	config     Config
	template   prompt.Template
	contextual prompt.Template
	// active is the version of the template whose prefix is in KV.
	active string
	model  *genai.Model
	gen    *genai.Generator
	// tokens mirrors the generator's KV contents token-for-token.
	tokens  []int32
	prefix  int
	session string
}

// Load opens the model and computes the plain template's shared prefix.
// Requests with context use contextual, whose prefix replaces the plain one
// in the KV state while those requests run.
func Load(config Config, template, contextual prompt.Template) (*Engine, float64, error) {
	start := time.Now()
	model, err := genai.OpenModel(config.ModelDir, genai.Options{IntraOpThreads: config.Threads, InterOpThreads: 1})
	if err != nil {
		return nil, 0, err
	}
	gen, err := model.NewGenerator(config.MaxLength)
	if err != nil {
		model.Close()
		return nil, 0, err
	}
	e := &Engine{config: config, template: template, contextual: contextual, model: model, gen: gen}
	if err := e.use(template); err != nil {
		e.Close()
		return nil, 0, err
	}
	return e, ms(time.Since(start)), nil
}

// use makes t the template whose prefix fills the start of the KV state.
func (e *Engine) use(t prompt.Template) error {
	if e.active == t.Version && e.tokens != nil {
		return nil
	}
	prefix, err := e.model.Encode(t.Prefix())
	if err != nil {
		return err
	}
	if e.tokens != nil {
		if err := e.rewind(0); err != nil {
			return err
		}
	}
	if err := e.gen.Append(prefix); err != nil {
		e.tokens, e.active = nil, ""
		return err
	}
	e.tokens, e.prefix, e.active = prefix, len(prefix), t.Version
	return nil
}

// PrefixTokens is the size of the shared instruction prefix.
func (e *Engine) PrefixTokens() int { return e.prefix }

// Close releases the KV state and weights.
func (e *Engine) Close() {
	if e.gen != nil {
		e.gen.Close()
		e.gen = nil
	}
	if e.model != nil {
		e.model.Close()
		e.model = nil
	}
}

// EndSession rewinds to the shared prefix so no private KV state outlives
// its session.
func (e *Engine) EndSession() error {
	e.session = ""
	return e.rewind(e.prefix)
}

func (e *Engine) rewind(n int) error {
	if n == len(e.tokens) {
		return nil
	}
	if err := e.gen.RewindTo(n); err != nil {
		return err
	}
	e.tokens = e.tokens[:n]
	return nil
}

// Run serves one request. cancelled is polled between tokens.
func (e *Engine) Run(req Request, cancelled *atomic.Bool) (Result, error) {
	if strings.Contains(req.Text, "<|") || strings.Contains(req.Text, "|>") {
		return Result{}, ErrControlInput
	}
	if req.Session != e.session {
		if err := e.EndSession(); err != nil {
			return Result{}, err
		}
		e.session = req.Session
	}
	start := time.Now()
	tpl := e.template
	if !req.Context.Bounded().Empty() {
		tpl = e.contextual
	}
	if err := e.use(tpl); err != nil {
		_ = e.EndSession()
		return Result{}, err
	}
	cleanup := domain.CleanSpeech(req.Text)
	in := prompt.Input{Scene: req.Scene, Terms: req.Terms, Context: req.Context, Text: cleanup.Text}
	text := tpl.Prompt(in)
	if req.Prefill {
		text = tpl.Source(in)
	}
	ids, err := e.model.Encode(text)
	if err != nil {
		return Result{}, err
	}
	if len(ids)+e.config.MaxNewTokens > e.config.MaxLength {
		return Result{}, ErrInputBudget
	}
	// Keep the longest common prefix; the shared prefix is never recomputed.
	common := 0
	for common < len(ids) && common < len(e.tokens) && ids[common] == e.tokens[common] {
		common++
	}
	if common < e.prefix {
		return Result{}, errors.New("instruction prefix tokenization changed")
	}
	if err := e.rewind(common); err != nil {
		return Result{}, err
	}
	if err := e.gen.Append(ids[common:]); err != nil {
		_ = e.EndSession()
		return Result{}, err
	}
	e.tokens = append(e.tokens, ids[common:]...)
	result := Result{Cleanup: cleanup, Template: tpl.Version, InputTokens: len(ids) - e.prefix, ReusedTokens: common - e.prefix, PrefillMs: ms(time.Since(start))}
	if req.Prefill {
		return result, nil
	}
	limit := e.budget(result.InputTokens, req.Live)
	var out []int32
	decode := time.Now()
	for {
		if cancelled.Load() {
			result.Cancelled = true
			break
		}
		if len(out) >= limit || e.gen.Done() {
			result.Truncated = true
			break
		}
		token, err := e.gen.Next()
		if err != nil {
			_ = e.EndSession()
			return Result{}, err
		}
		if len(out) == 0 {
			result.FirstTokenMs = ms(time.Since(start))
		}
		e.tokens = append(e.tokens, token)
		if e.model.IsEOS(token) {
			break
		}
		out = append(out, token)
	}
	result.DecodeMs = ms(time.Since(decode))
	result.OutputTokens = len(out)
	answer, err := e.model.Decode(out)
	if err != nil {
		return Result{}, err
	}
	// The answer and its EOS stay in KV only until the next request rewinds
	// to the shared source prefix.
	if !utf8.ValidString(answer) {
		return Result{}, errors.New("decoded answer is not valid UTF-8")
	}
	result.Text = strings.TrimSpace(answer)
	return result, nil
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
