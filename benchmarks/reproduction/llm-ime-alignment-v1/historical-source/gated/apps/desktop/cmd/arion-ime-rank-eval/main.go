// Command arion-ime-rank-eval runs reference-free keyboard snapshots through
// the same completion adapter used by local workers or explicit API providers.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"arion.local/core/contracts"
	"arion.local/desktop/infrastructure/cloudpolish"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/ranking"
)

type capture struct {
	backend  ranking.Completer
	Request  prompt.ChatRequest `json:"chatRequest"`
	Response engine.Result      `json:"chatResponse"`
	Error    string             `json:"chatError,omitempty"`
}

func (c *capture) Complete(ctx context.Context, req prompt.ChatRequest) (engine.Result, error) {
	c.Request = req
	out, err := c.backend.Complete(ctx, req)
	c.Response = out
	if err != nil {
		c.Error = err.Error()
	}
	return out, err
}
func digest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	input := flag.String("requests", "", "reference-free JSONL requests")
	output := flag.String("out", "", "new observation directory")
	worker := flag.String("worker", "", "local arion-llm executable")
	runtime := flag.String("runtime", "", "local runtime directory")
	model := flag.String("model", "", "local model directory")
	modelHash := flag.String("model-sha256", "", "required local weight digest")
	provider := flag.String("provider", "", "explicit API provider; never stored credentials")
	modelID := flag.String("model-id", "", "API model ID")
	base := flag.String("base", "", "explicit compatible API base")
	budget := flag.Duration("timeout", 1500*time.Millisecond, "per-request wall deadline")
	ungated := flag.Bool("ungated", false, "explicit no-context inference ablation")
	flag.Parse()
	if *input == "" || *output == "" || *budget <= 0 {
		return errors.New("requests, new out directory and positive timeout required")
	}
	if err := os.Mkdir(*output, 0700); err != nil {
		return err
	}
	status := map[string]any{"status": "failed", "template": ranking.Version, "timeoutMs": float64(*budget / time.Millisecond)}
	status["allowWithoutContext"] = *ungated
	defer func() {
		b, _ := json.MarshalIndent(status, "", "  ")
		_ = os.WriteFile(filepath.Join(*output, "status.json"), append(b, '\n'), 0600)
	}()
	in, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer in.Close()
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	requests := []contracts.KeyboardRankRequest{}
	seen := map[string]bool{}
	for scanner.Scan() {
		var req contracts.KeyboardRankRequest
		d := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		d.DisallowUnknownFields()
		if err = d.Decode(&req); err != nil {
			return err
		}
		if _, err = ranking.Prepare(req); err != nil {
			return err
		}
		key := fmt.Sprintf("%s#%d#%d", req.SessionID, req.Generation, req.Revision)
		if seen[key] {
			return errors.New("duplicate request identity")
		}
		seen[key] = true
		requests = append(requests, req)
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if len(requests) == 0 {
		return errors.New("empty requests")
	}
	inputHash, err := digest(*input)
	if err != nil {
		return err
	}
	status["requestsSha256"] = inputHash
	var backend ranking.Completer
	if *provider != "" {
		if *worker != "" {
			return errors.New("choose local worker or explicit provider")
		}
		s, err := cloudpolish.NewAPICompletion(*provider, os.Getenv("ARION_RANK_API_KEY"), *modelID, *base)
		if err != nil {
			return err
		}
		defer s.CloseCompletion()
		backend = s
		status["provider"], status["modelID"] = *provider, *modelID
	} else {
		if *worker == "" || *runtime == "" || *model == "" || *modelHash == "" {
			return errors.New("local worker, runtime, model and model-sha256 required")
		}
		hash, err := digest(filepath.Join(*model, "Qwen3.5-2B-Q4_K_M.gguf"))
		if err != nil {
			return err
		}
		if hash != *modelHash {
			return errors.New("local weight digest mismatch")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		c, info, err := client.Start(ctx, client.Config{Worker: *worker, Runtime: *runtime, Model: *model, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
		cancel()
		if err != nil {
			return err
		}
		defer c.Close()
		backend = c
		status["load"], status["modelSha256"] = info, hash
		for _, item := range []struct{ name, path string }{{"workerSha256", *worker}, {"runtimeSha256", filepath.Join(*runtime, "llama-server")}} {
			h, err := digest(item.path)
			if err != nil {
				return err
			}
			status[item.name] = h
		}
	}
	f, err := os.OpenFile(filepath.Join(*output, "observations.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	errorsCount := 0
	for i, req := range requests {
		recorded := &capture{backend: backend}
		r := &ranking.Ranker{Chat: recorded, AllowWithoutContext: *ungated}
		started := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), *budget)
		result, callErr := r.RankKeyboard(ctx, req)
		elapsed := float64(time.Since(started).Microseconds()) / 1000
		cancel()
		failure := ""
		if callErr != nil {
			failure = callErr.Error()
			errorsCount++
		}
		if err = enc.Encode(map[string]any{"request": req, "result": result, "completion": recorded, "error": failure, "wallMs": elapsed}); err != nil {
			return err
		}
		if (i+1)%20 == 0 {
			fmt.Fprintf(os.Stderr, "Completed %d/%d\n", i+1, len(requests))
		}
	}
	status["status"], status["count"], status["errors"] = "complete", len(requests), errorsCount
	return nil
}
