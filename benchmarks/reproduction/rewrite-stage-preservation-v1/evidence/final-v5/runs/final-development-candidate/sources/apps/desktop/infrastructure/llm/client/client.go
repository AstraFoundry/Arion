// Package client runs the arion-llm worker process and adapts it to the
// core Rewriter port. The worker owns the LLM runtime; this process only
// exchanges protocol lines over pipes and never loads it.
package client

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/protocol"
)

// Config locates the worker, runtime and model of one rewriter.
type Config struct {
	Worker       string
	Runtime      string
	Model        string
	Threads      int
	MaxLength    int
	MaxNewTokens int
}

// LoadInfo is the worker's report after loading.
type LoadInfo struct {
	LoadMs       float64
	PromptID     string
	PrefixTokens int
	PeakRSS      int64
}

// Client is one running worker. A dead worker fails every request with
// MODEL_UNSUPPORTED; it is never silently replaced.
type Client struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	next  atomic.Uint64

	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[uint64]chan protocol.Response
	dead    error
	done    chan struct{}
}

// Start launches the worker and loads the model. stderr is discarded by the
// caller's choice of diag writer; it never contains content.
func Start(ctx context.Context, config Config, diag io.Writer) (*Client, LoadInfo, error) {
	cmd := exec.Command(config.Worker)
	prepareProcess(cmd)
	cmd.Stderr = diag
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, LoadInfo{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, LoadInfo{}, err
	}
	if err := cmd.Start(); err != nil {
		return nil, LoadInfo{}, domain.Fail(domain.CodeModelUnsupported, "llm worker start")
	}
	c := &Client{cmd: cmd, stdin: stdin, pending: map[uint64]chan protocol.Response{}, done: make(chan struct{})}
	go c.read(stdout)
	response, err := c.call(ctx, protocol.Request{Op: protocol.OpLoad, Runtime: config.Runtime, Model: config.Model,
		Threads: config.Threads, MaxLength: config.MaxLength, MaxNewTokens: config.MaxNewTokens})
	if err != nil {
		c.Close()
		return nil, LoadInfo{}, err
	}
	return c, LoadInfo{LoadMs: response.LoadMs, PromptID: response.PromptID, PrefixTokens: response.PrefixTokens, PeakRSS: response.PeakRSS}, nil
}

func (c *Client) read(stdout io.Reader) {
	in := bufio.NewScanner(stdout)
	in.Buffer(make([]byte, 1<<20), 8<<20)
	for in.Scan() {
		var response protocol.Response
		if json.Unmarshal(in.Bytes(), &response) != nil {
			break
		}
		c.mu.Lock()
		ch := c.pending[response.ID]
		delete(c.pending, response.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- response
		}
	}
	_ = c.cmd.Wait()
	stopProcessTree(c.cmd)
	c.mu.Lock()
	c.dead = domain.Fail(domain.CodeModelUnsupported, "llm worker exited")
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()
	close(c.done)
}

func (c *Client) send(req protocol.Request) error {
	req.Version = protocol.Version
	line, err := json.Marshal(req)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.stdin.Write(append(line, '\n'))
	return err
}

// call sends a request and waits. A cancelled context asks the worker to
// stop at its next token and abandons the response.
func (c *Client) call(ctx context.Context, req protocol.Request) (protocol.Response, error) {
	req.ID = c.next.Add(1)
	ch := make(chan protocol.Response, 1)
	c.mu.Lock()
	if c.dead != nil {
		c.mu.Unlock()
		return protocol.Response{}, c.dead
	}
	c.pending[req.ID] = ch
	c.mu.Unlock()
	if err := c.send(req); err != nil {
		return protocol.Response{}, domain.Fail(domain.CodeModelUnsupported, "llm worker write")
	}
	select {
	case response, ok := <-ch:
		if !ok {
			return protocol.Response{}, domain.Fail(domain.CodeModelUnsupported, "llm worker exited")
		}
		if !response.OK {
			return response, domain.Fail(domain.CodeModelUnsupported, response.Error)
		}
		return response, nil
	case <-ctx.Done():
		_ = c.send(protocol.Request{Op: protocol.OpCancel, Target: req.ID})
		return protocol.Response{}, domain.Fail(domain.CodeCancelled, "rewrite cancelled")
	}
}

// Rewrite satisfies ports.Rewriter.
func (c *Client) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
	op := protocol.OpRewrite
	if req.Prefill {
		op = protocol.OpPrefill
	}
	request := protocol.Request{Op: op, Session: session(req.Identity), Scene: req.Scene, Terms: req.Terms, Text: req.Text, PreviousText: req.PreviousText, Live: req.Live}
	if c := req.Context.Bounded(); !c.Empty() {
		request.Context = &c
	}
	response, err := c.call(ctx, request)
	if err != nil {
		return ports.RewriteResult{}, err
	}
	if response.Cancelled {
		return ports.RewriteResult{}, domain.Fail(domain.CodeCancelled, "rewrite cancelled")
	}
	var cleanup domain.SpeechCleanup
	if response.Cleanup != nil {
		cleanup = *response.Cleanup
	}
	return ports.RewriteResult{Cleanup: cleanup, Text: response.Text, Truncated: response.Truncated, Template: response.Template, PrefillMs: response.PrefillMs, DecodeMs: response.DecodeMs,
		InputTokens: response.InputTokens, ReusedTokens: response.ReusedTokens, OutputTokens: response.OutputTokens}, nil
}

// End releases the session's KV state in the worker.
func (c *Client) End(id contracts.Identity) {
	_, _ = c.call(context.Background(), protocol.Request{Op: protocol.OpEnd, Session: session(id)})
}

func session(id contracts.Identity) string {
	return id.SessionID + "#" + strconv.FormatUint(id.Generation, 10)
}

// Close unloads the model and stops the worker, killing it if it does not
// exit promptly.
func (c *Client) Close() {
	_ = c.send(protocol.Request{ID: c.next.Add(1), Op: protocol.OpClose})
	_ = c.stdin.Close()
	select {
	case <-c.done:
	case <-time.After(5 * time.Second):
		stopProcessTree(c.cmd)
		<-c.done
	}
}

// Alive reports whether the worker process is still serving.
func (c *Client) Alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dead == nil
}

var _ ports.Rewriter = (*Client)(nil)
