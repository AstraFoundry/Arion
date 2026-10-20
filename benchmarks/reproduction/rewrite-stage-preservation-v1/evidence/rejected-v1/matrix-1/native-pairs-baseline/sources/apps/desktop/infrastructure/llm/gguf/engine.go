// Package gguf runs the selected GGUF pipeline through a private Unix socket.
// The child runtime is offline, owns one Metal slot, and never exposes TCP.
package gguf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/spelling"
)

// ModelFile is the catalog's single digest-verified weight file.
const ModelFile = "Qwen3.5-2B-Q4_K_M.gguf"

// Engine serializes requests from the pipe worker. Private context is erased
// between sessions; no prompt cache is saved to disk.
type Engine struct {
	config    engine.Config
	cmd       *exec.Cmd
	exited    chan struct{}
	socketDir string
	http      *http.Client
	recipe    *prompt.SelectedTemplates
	lexicon   *domain.SpellingLexicon
	session   string
}

// Load starts the pinned runtime with a private socket and fixed budgets.
func Load(config engine.Config, runtimeDir string) (*Engine, float64, error) {
	started := time.Now()
	lexicon, err := spelling.Load()
	if err != nil {
		return nil, 0, err
	}
	recipe, err := prompt.Selected(lexicon)
	if err != nil {
		return nil, 0, err
	}
	dir, err := os.MkdirTemp("", "arion-llm-")
	if err != nil {
		return nil, 0, err
	}
	socket := filepath.Join(dir, "worker.sock")
	if len(socket) > 100 {
		os.RemoveAll(dir)
		return nil, 0, errors.New("private runtime socket path exceeds platform limit")
	}
	e := &Engine{config: config, socketDir: dir, recipe: recipe, lexicon: lexicon, exited: make(chan struct{})}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	e.http = &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	args := []string{"--model", filepath.Join(config.ModelDir, ModelFile), "--host", socket, "--threads", fmt.Sprint(config.Threads), "--threads-batch", fmt.Sprint(config.Threads), "--n-gpu-layers", "99", "--ctx-size", fmt.Sprint(config.MaxLength), "--parallel", "1", "--batch-size", "4096", "--ubatch-size", "4096", "--reasoning", "off", "--jinja", "--offline", "--no-warmup", "--no-context-shift", "--cache-ram", "0", "--slots", "--slot-save-path", dir + string(os.PathSeparator), "--no-webui", "--log-disable"}
	e.cmd = exec.Command(filepath.Join(runtimeDir, "llama-server"), args...)
	// Model/runtime options come exclusively from the pinned product config.
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "LLAMA_ARG_") {
			e.cmd.Env = append(e.cmd.Env, v)
		}
	}
	e.cmd.Stdout = io.Discard
	e.cmd.Stderr = io.Discard
	if err = e.cmd.Start(); err != nil {
		e.Close()
		return nil, 0, errors.New("GGUF runtime start failed")
	}
	go func() { _ = e.cmd.Wait(); close(e.exited) }()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	for {
		if err = e.request(ctx, "GET", "/health", nil, &struct{}{}); err == nil {
			break
		}
		select {
		case <-e.exited:
			e.Close()
			return nil, 0, errors.New("GGUF runtime exited during load")
		case <-ctx.Done():
			e.Close()
			return nil, 0, errors.New("GGUF model load timed out")
		case <-time.After(50 * time.Millisecond):
		}
	}
	return e, float64(time.Since(started).Microseconds()) / 1000, nil
}

func (e *Engine) request(ctx context.Context, method, path string, body, output any) error {
	var reader io.Reader
	if body != nil {
		var data bytes.Buffer
		encoder := json.NewEncoder(&data)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(body); err != nil {
			return err
		}
		reader = &data
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://arion-local"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := e.http.Do(req)
	if err != nil {
		return errors.New("private GGUF request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("private GGUF request failed: HTTP %d", response.StatusCode)
	}
	if output == nil {
		_, err = io.Copy(io.Discard, response.Body)
		return err
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(output); err != nil {
		return errors.New("invalid GGUF response protocol")
	}
	return nil
}

// PrefixTokens is zero because the runtime owns template tokenization/cache.
func (e *Engine) PrefixTokens() int { return 0 }

// EndSession erases the sole native slot, including hybrid recurrent state.
func (e *Engine) EndSession() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := e.request(ctx, "POST", "/slots/0?action=erase", nil, &struct{}{}); err != nil {
		return err
	}
	e.session = ""
	return nil
}

// Close stops the native child before removing its private socket.
func (e *Engine) Close() {
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Signal(os.Interrupt)
		select {
		case <-e.exited:
		case <-time.After(time.Second):
			_ = e.cmd.Process.Kill()
			<-e.exited
		}
	}
	if e.http != nil {
		e.http.CloseIdleConnections()
	}
	if e.socketDir != "" {
		_ = os.RemoveAll(e.socketDir)
	}
}

type completion struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			Reasoning string `json:"reasoning_content"`
		} `json:"message"`
		Finish string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		Input  int `json:"prompt_tokens"`
		Output int `json:"completion_tokens"`
	} `json:"usage"`
	Timings struct {
		PromptMs float64 `json:"prompt_ms"`
		DecodeMs float64 `json:"predicted_ms"`
		Cached   int     `json:"cache_n"`
	} `json:"timings"`
}

func (e *Engine) generate(ctx context.Context, request prompt.ChatRequest, prefill bool) (completion, error) {
	if e.config.MaxNewTokens > 0 && request.MaxTokens > e.config.MaxNewTokens {
		request.MaxTokens = e.config.MaxNewTokens
	}

	var applied struct {
		Prompt string `json:"prompt"`
	}
	if err := e.request(ctx, "POST", "/apply-template", request, &applied); err != nil {
		return completion{}, err
	}
	var tokens struct {
		Tokens []int `json:"tokens"`
	}
	if err := e.request(ctx, "POST", "/tokenize", map[string]any{"content": applied.Prompt, "add_special": true}, &tokens); err != nil {
		return completion{}, err
	}
	if len(tokens.Tokens)+request.MaxTokens > e.config.MaxLength {
		return completion{}, engine.ErrInputBudget
	}
	if prefill {
		// Preparation checks the complete native template and token budget only.
		// A zero-token chat completion still samples one token in the pinned
		// runtime, so never use that endpoint for a generation-free preparation.
		result := completion{}
		result.Usage.Input = len(tokens.Tokens)
		return result, nil
	}
	var result completion
	if err := e.request(ctx, "POST", "/v1/chat/completions", request, &result); err != nil {
		return completion{}, err
	}
	if len(result.Choices) != 1 || result.Choices[0].Message.Reasoning != "" {
		return completion{}, errors.New("unexpected GGUF completion protocol")
	}
	if !prefill && result.Choices[0].Finish != "stop" {
		return result, errors.New("GGUF output budget exhausted")
	}
	if !utf8.ValidString(result.Choices[0].Message.Content) {
		return result, errors.New("GGUF completion is not UTF-8")
	}
	return result, nil
}

// Run applies local deletion, optional bounded spelling selection, then
// formatting. The formatter never receives OCR, cursor text, or history.
func (e *Engine) Run(req engine.Request, stop *atomic.Bool) (engine.Result, error) {
	if strings.Contains(req.Text, "<|") || strings.Contains(req.Text, "|>") {
		return engine.Result{}, engine.ErrControlInput
	}
	if stop.Load() {
		return engine.Result{Cancelled: true}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-finished:
				return
			case <-ticker.C:
				if stop.Load() {
					cancel()
					return
				}
			}
		}
	}()
	if req.Session != e.session {
		if err := e.EndSession(); err != nil {
			return engine.Result{}, err
		}
		e.session = req.Session
	}
	cleanup := domain.CleanSpeech(req.Text)
	contextData := map[string]string{}
	c := req.Context.Bounded()
	if c.Screen != "" {
		contextData["Screen"] = c.Screen
	}
	if c.Before != "" {
		contextData["Before"] = c.Before
	}
	if c.After != "" {
		contextData["After"] = c.After
	}
	if len(c.History) > 0 {
		contextData["History"] = strings.Join(c.History, "\n")
	}
	if len(req.Terms) > 0 {
		contextData["Terms"] = strings.Join(req.Terms, "\n")
	}
	if req.PreviousText != "" && req.PreviousText != req.Text {
		contextData["EarlierSource"] = (domain.RewriteContext{Screen: req.PreviousText}).Bounded().Screen
	}
	p := e.lexicon.PrepareContextual(cleanup.Text, contextData, req.Scene)
	result := engine.Result{Cleanup: cleanup, Template: prompt.SelectedVersion}
	edited := cleanup.Text
	add := func(out completion) {
		result.InputTokens += out.Usage.Input
		result.OutputTokens += out.Usage.Output
		result.ReusedTokens += out.Timings.Cached
		result.PrefillMs += out.Timings.PromptMs
		result.DecodeMs += out.Timings.DecodeMs
	}
	if len(p.Hypotheses) > 0 {
		out, err := e.generate(ctx, e.recipe.Selection(p), req.Prefill)
		if err != nil {
			if ctx.Err() != nil {
				result.Cancelled = true
				return result, nil
			}
			return result, err
		}
		add(out)
		if !req.Prefill {
			edited, err = domain.ApplySpellingChoices(out.Choices[0].Message.Content, p)
			if err != nil {
				return result, err
			}
		}
	}
	request, err := e.recipe.Format(edited, req.Scene)
	if err != nil {
		return result, err
	}
	out, err := e.generate(ctx, request, req.Prefill)
	if err != nil {
		if ctx.Err() != nil {
			result.Cancelled = true
			return result, nil
		}
		return result, err
	}
	add(out)
	if !req.Prefill {
		result.Text = strings.TrimSpace(out.Choices[0].Message.Content)
	}
	result.Cancelled = ctx.Err() != nil || stop.Load()
	return result, nil
}
