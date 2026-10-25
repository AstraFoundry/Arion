// Package gguf runs the selected GGUF pipeline through a private Unix socket.
// The child runtime is offline, owns one slot, and never exposes TCP.
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
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/internal/subprocess"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/rewrite"
)

// ModelFile is the catalog's single digest-verified weight file.
const ModelFile = "Qwen3.5-2B-Q4_K_M.gguf"

// ErrRuntimeUnavailable requires the pipe worker to exit after reporting the
// failed reload, so a dead native capability cannot remain apparently ready.
var ErrRuntimeUnavailable = errors.New("GGUF runtime reload failed")

// Engine serializes requests from the pipe worker. Private context is erased
// between sessions; no prompt cache is saved to disk.
type Engine struct {
	config     engine.Config
	runtimeDir string
	reload     bool
	cmd        *exec.Cmd
	process    *subprocess.Process
	exited     chan struct{}
	socketDir  string
	http       *http.Client
	pipeline   *rewrite.Pipeline
	session    string
}

// Load starts the pinned runtime with a private socket and fixed budgets.
func Load(config engine.Config, runtimeDir string) (*Engine, float64, error) {
	return load(config, runtimeDir, nil)
}

func load(config engine.Config, runtimeDir string, stop *atomic.Bool) (*Engine, float64, error) {
	started := time.Now()
	if stop != nil && stop.Load() {
		return nil, 0, context.Canceled
	}
	var err error
	runtimeDir, err = filepath.Abs(runtimeDir)
	if err != nil {
		return nil, 0, err
	}
	config.ModelDir, err = filepath.Abs(config.ModelDir)
	if err != nil {
		return nil, 0, err
	}
	pipeline, err := rewrite.Load()
	if err != nil {
		return nil, 0, err
	}
	dir, err := privateDirectory()
	if err != nil {
		return nil, 0, err
	}
	socket := filepath.Join(dir, "worker.sock")
	if len(socket) > 100 {
		os.RemoveAll(dir)
		return nil, 0, errors.New("private runtime socket path exceeds platform limit")
	}
	e := &Engine{config: config, runtimeDir: runtimeDir, socketDir: dir, pipeline: pipeline, exited: make(chan struct{})}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	e.http = &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	args := []string{"--model", filepath.Join(config.ModelDir, ModelFile), "--host", socket, "--threads", fmt.Sprint(config.Threads), "--threads-batch", fmt.Sprint(config.Threads), "--n-gpu-layers", "99", "--ctx-size", fmt.Sprint(config.MaxLength), "--parallel", "1", "--batch-size", "4096", "--ubatch-size", "4096", "--reasoning", "off", "--jinja", "--offline", "--no-warmup", "--no-context-shift", "--cache-ram", "0", "--slots", "--slot-save-path", dir + string(os.PathSeparator), "--no-webui", "--log-disable"}
	if runtime.GOOS == "windows" {
		// The pinned Windows baseline is CPU-only, independent of GPU drivers.
		// Bound CPU prefill batches and use physical-core-scale parallelism.
		for i := range args {
			if args[i] == "--n-gpu-layers" {
				args[i+1] = "0"
			}
			if args[i] == "--batch-size" {
				args[i+1] = "512"
			}
			if args[i] == "--ubatch-size" {
				args[i+1] = "512"
			}
			if args[i] == "--threads-batch" {
				args[i+1] = fmt.Sprint(max(config.Threads, min(8, runtime.NumCPU()/2)))
			}
		}
	}
	e.cmd = exec.Command(filepath.Join(runtimeDir, runtimeExecutable()), args...)
	// Keep the runtime's working directory private instead of inheriting an
	// editor or repository directory.
	e.cmd.Dir = dir
	// Model/runtime options come exclusively from the pinned product config.
	for _, v := range os.Environ() {
		upper := strings.ToUpper(v)
		if !strings.HasPrefix(upper, "LLAMA_ARG_") && !strings.HasPrefix(upper, "GGML_") {
			e.cmd.Env = append(e.cmd.Env, v)
		}
	}
	e.cmd.Stdout = io.Discard
	e.cmd.Stderr = io.Discard
	if e.process, err = subprocess.StartChild(e.cmd); err != nil {
		e.Close()
		return nil, 0, errors.New("GGUF runtime start failed")
	}
	go func() { _ = e.cmd.Wait(); close(e.exited) }()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	stopObserving := observeCancellation(stop, func() { e.process.Stop(); cancel() })
	defer stopObserving()
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
	if e.reload {
		return nil
	} // cancellation already destroyed the native process and its KV state
	deadline := 3 * time.Second
	if runtime.GOOS == "windows" {
		// The server observes a disconnected request on its next one-second
		// poll, then processes cancellation after the active CPU batch ends.
		deadline = 6 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	if err := e.request(ctx, "POST", "/slots/0?action=erase", nil, &struct{}{}); err != nil {
		return err
	}
	e.session = ""
	return nil
}

// Close stops the native child before removing its private socket.
func (e *Engine) Close() {
	if e.process != nil {
		_ = e.cmd.Process.Signal(os.Interrupt)
		select {
		case <-e.exited:
		case <-time.After(time.Second):
			e.process.Stop()
			<-e.exited
		}
		e.process.Stop()
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
	request = request.LocalWire()
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

// Run preserves the native session while the shared pipeline prepares stages.
func (e *Engine) Run(req engine.Request, stop *atomic.Bool) (engine.Result, error) {
	if stop.Load() {
		return engine.Result{Cancelled: true}, nil
	}
	if e.reload {
		fresh, _, err := load(e.config, e.runtimeDir, stop)
		if err != nil {
			if stop.Load() {
				return engine.Result{Cancelled: true}, nil
			}
			return engine.Result{}, ErrRuntimeUnavailable
		}
		*e = *fresh
		if stop.Load() {
			e.Close()
			e.reload = true
			return engine.Result{Cancelled: true}, nil
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopObserving := observeCancellation(stop, func() {
		if runtime.GOOS == "windows" {
			e.process.Stop()
		}
		cancel()
	})
	defer func() {
		stopObserving()
		if runtime.GOOS == "windows" && stop.Load() {
			// Erasure can race an HTTP handler that has not yet enqueued its
			// native task. Destroying the child guarantees that no cancelled
			// request can populate a later session's slot. Reload only when
			// another explicit rewrite arrives; never retry cancelled content.
			e.Close()
			e.reload = true
			e.session = ""
		}
	}()
	if req.Session != e.session {
		if err := e.EndSession(); err != nil {
			return engine.Result{}, err
		}
		e.session = req.Session
	}
	result, err := e.pipeline.Run(ctx, req, e.generateStage)
	result.Cancelled = result.Cancelled || stop.Load()
	return result, err
}

func observeCancellation(stop *atomic.Bool, onStop func()) func() {
	if stop == nil {
		return func() {}
	}
	finished, observed := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(observed)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-finished:
				return
			case <-ticker.C:
				if stop.Load() {
					onStop()
					return
				}
			}
		}
	}()
	return func() { close(finished); <-observed }
}

func (e *Engine) generateStage(ctx context.Context, req prompt.ChatRequest, prefill bool) (engine.Result, error) {
	out, err := e.generate(ctx, req, prefill)
	result := engine.Result{InputTokens: out.Usage.Input, OutputTokens: out.Usage.Output,
		ReusedTokens: out.Timings.Cached, PrefillMs: out.Timings.PromptMs, DecodeMs: out.Timings.DecodeMs}
	if len(out.Choices) == 1 {
		result.Text = out.Choices[0].Message.Content
	}
	return result, err
}
