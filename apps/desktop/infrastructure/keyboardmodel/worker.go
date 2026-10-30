// Package keyboardmodel runs the bounded, local per-keystroke neural model.
package keyboardmodel

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/ranking"
)

const ModelID = "chinese-ime-lm-4m-int8"

type Worker struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	input  io.WriteCloser
	output *bufio.Scanner
	closed bool
}

// Open waits for model loading before any typing request is admitted.
func Open(ctx context.Context, executable, model string) (*Worker, error) {
	cmd := exec.Command(executable, model)
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		_ = input.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard
	w := &Worker{cmd: cmd, input: input, output: bufio.NewScanner(output)}
	w.output.Buffer(make([]byte, 1024), 4096)
	if err = cmd.Start(); err != nil {
		_ = input.Close()
		_ = output.Close()
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = cmd.Process.Kill() })
	defer stop()
	if !w.output.Scan() || w.output.Text() != `{"ready":true}` {
		w.Close()
		return nil, errors.New("keyboard model failed to initialize")
	}
	if err = ctx.Err(); err != nil {
		w.Close()
		return nil, err
	}
	return w, nil
}
func (w *Worker) Close() { w.mu.Lock(); defer w.mu.Unlock(); w.closeLocked() }
func (w *Worker) closeLocked() {
	if !w.closed {
		w.closed = true
		_ = w.input.Close()
		_ = w.cmd.Process.Kill()
		_ = w.cmd.Wait()
	}
}

// A cancelled request is still drained, so its reply cannot become the next
// request's answer. The model's input and candidate bounds cap this work.
func (w *Worker) RankKeyboard(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	if _, err := ranking.Prepare(req); err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	if w.closed {
		return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeModelMissing, "keyboard worker closed")
	}
	type candidate struct {
		ID     string `json:"id"`
		Text   string `json:"text"`
		Source string `json:"source"`
		Start  int    `json:"start"`
		End    int    `json:"end"`
	}
	candidates := make([]candidate, len(req.Candidates))
	for i, c := range req.Candidates {
		candidates[i] = candidate{c.ID, c.Text, c.Source, c.Start, c.End}
	}
	in := struct {
		Session    string      `json:"session"`
		Input      string      `json:"input"`
		Before     string      `json:"before"`
		Candidates []candidate `json:"candidates"`
	}{req.SessionID, req.Input, req.Before, candidates}
	watchdog := time.AfterFunc(time.Second, func() { _ = w.cmd.Process.Kill() })
	defer watchdog.Stop()
	if err := json.NewEncoder(w.input).Encode(in); err != nil {
		w.closeLocked()
		return contracts.KeyboardRankResult{}, errors.New("keyboard worker write failed")
	}
	if !w.output.Scan() {
		w.closeLocked()
		return contracts.KeyboardRankResult{}, errors.New("keyboard worker response failed")
	}
	if err := ctx.Err(); err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	out, err := ranking.Apply(req, w.output.Text())
	if err != nil {
		w.closeLocked()
	}
	out.Template = "chinese-ime-lm-0.2.0"
	return out, err
}

func (w *Worker) Alive() bool { w.mu.Lock(); defer w.mu.Unlock(); return !w.closed }
