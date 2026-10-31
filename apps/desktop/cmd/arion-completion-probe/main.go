// Command arion-completion-probe exposes local completion for private studies.
// It receives reference-free prompts, never corpus labels or scoring rules.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/llm/prompt"
)

type request struct {
	Protocol  int                `json:"protocol"`
	RequestID string             `json:"requestId"`
	Op        string             `json:"op"`
	Config    client.Config      `json:"config"`
	Chat      prompt.ChatRequest `json:"chat"`
	TimeoutMs int                `json:"timeoutMs"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var worker *client.Client
	defer func() {
		if worker != nil {
			worker.Close()
		}
	}()
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	out := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			return err
		}
		if req.Protocol != 1 || req.RequestID == "" {
			return errors.New("invalid protocol identity")
		}
		response := map[string]any{"protocol": 1, "requestId": req.RequestID}
		var err error
		started := time.Now()
		if req.TimeoutMs <= 0 || req.TimeoutMs > 120000 {
			return errors.New("invalid timeout")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutMs)*time.Millisecond)
		switch req.Op {
		case "load":
			if worker != nil {
				err = errors.New("worker already loaded")
			} else {
				var info client.LoadInfo
				worker, info, err = client.Start(ctx, req.Config, io.Discard)
				response["load"] = info
			}
		case "complete":
			if worker == nil {
				err = errors.New("worker not loaded")
			} else {
				result, failure := worker.Complete(ctx, req.Chat)
				response["result"], err = result, failure
			}
		default:
			err = errors.New("unsupported operation")
		}
		cancel()
		response["wallMs"] = float64(time.Since(started).Microseconds()) / 1000
		response["ok"] = err == nil
		if err != nil {
			response["error"] = err.Error()
		}
		if err := out.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}
