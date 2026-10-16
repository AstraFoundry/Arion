// Command arion-llm is the local rewrite worker. It loads a provisioned
// ONNX Runtime GenAI runtime and model in its own process, so the speech
// runtime's ONNX Runtime and the LLM's never share an address space, and
// serves protocol requests on stdin/stdout. It never opens a network
// connection; stderr carries content-free diagnostics only.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/genai"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/protocol"
)

type worker struct {
	engine *engine.Engine
	out    *json.Encoder

	mu        sync.Mutex
	cancelled map[uint64]bool
	running   uint64
	stop      atomic.Bool
}

func main() {
	w := &worker{out: json.NewEncoder(os.Stdout), cancelled: map[uint64]bool{}}
	queue := make(chan protocol.Request, 64)
	go w.read(queue)
	for req := range queue {
		response := w.serve(req)
		response.ID = req.ID
		if err := w.out.Encode(response); err != nil {
			os.Exit(1)
		}
		if req.Op == protocol.OpClose {
			return
		}
	}
	// stdin closed: the owning process is gone.
	w.close()
}

// read handles cancel immediately and queues everything else in order.
func (w *worker) read(queue chan<- protocol.Request) {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 8<<20)
	for in.Scan() {
		var req protocol.Request
		if err := json.Unmarshal(in.Bytes(), &req); err != nil || req.Version != protocol.Version {
			fmt.Fprintln(os.Stderr, "invalid request line")
			os.Exit(2)
		}
		if req.Op == protocol.OpCancel {
			w.mu.Lock()
			w.cancelled[req.Target] = true
			if w.running == req.Target {
				w.stop.Store(true)
			}
			w.mu.Unlock()
			continue
		}
		queue <- req
	}
	close(queue)
}

func (w *worker) serve(req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpLoad:
		return w.load(req)
	case protocol.OpRewrite, protocol.OpPrefill:
		return w.rewrite(req)
	case protocol.OpEnd:
		if w.engine == nil {
			return fail(errors.New("model not loaded"))
		}
		if err := w.engine.EndSession(); err != nil {
			return fail(err)
		}
		return protocol.Response{OK: true}
	case protocol.OpClose:
		w.close()
		return protocol.Response{OK: true}
	}
	return fail(fmt.Errorf("unsupported op %q", req.Op))
}

func (w *worker) load(req protocol.Request) protocol.Response {
	if w.engine != nil {
		return fail(errors.New("model already loaded"))
	}
	if req.Threads <= 0 || req.MaxLength <= 0 || req.MaxNewTokens <= 0 {
		return fail(errors.New("threads and token budgets are required"))
	}
	if err := genai.Load(req.Runtime); err != nil {
		return fail(err)
	}
	template := prompt.Live()
	e, loadMs, err := engine.Load(engine.Config{ModelDir: req.Model, Threads: req.Threads, MaxLength: req.MaxLength, MaxNewTokens: req.MaxNewTokens}, template, prompt.Context())
	if err != nil {
		return fail(err)
	}
	w.engine = e
	return protocol.Response{OK: true, LoadMs: loadMs, PromptID: template.Version, PrefixTokens: e.PrefixTokens(), PeakRSS: peakRSS()}
}

func (w *worker) rewrite(req protocol.Request) protocol.Response {
	if w.engine == nil {
		return fail(errors.New("model not loaded"))
	}
	w.mu.Lock()
	if w.cancelled[req.ID] {
		delete(w.cancelled, req.ID)
		w.mu.Unlock()
		return protocol.Response{OK: true, Cancelled: true}
	}
	w.running = req.ID
	w.stop.Store(false)
	w.mu.Unlock()
	in := engine.Request{Session: req.Session, Scene: req.Scene, Terms: req.Terms, Text: req.Text, Prefill: req.Op == protocol.OpPrefill, Live: req.Live}
	if req.Context != nil {
		in.Context = *req.Context
	}
	result, err := w.engine.Run(in, &w.stop)
	w.mu.Lock()
	w.running = 0
	delete(w.cancelled, req.ID)
	w.mu.Unlock()
	if err != nil {
		return fail(err)
	}
	return protocol.Response{OK: true, Text: result.Text, Template: result.Template, InputTokens: result.InputTokens, ReusedTokens: result.ReusedTokens,
		OutputTokens: result.OutputTokens, PrefillMs: result.PrefillMs, DecodeMs: result.DecodeMs, FirstTokenMs: result.FirstTokenMs,
		Truncated: result.Truncated, Cancelled: result.Cancelled, PeakRSS: peakRSS()}
}

func (w *worker) close() {
	if w.engine != nil {
		w.engine.Close()
		w.engine = nil
		genai.Shutdown()
	}
}

func fail(err error) protocol.Response { return protocol.Response{OK: false, Error: err.Error()} }
