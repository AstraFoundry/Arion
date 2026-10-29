package gguf_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/rewriting"
)

// Opt-in real Windows runtime observations with the product's bounded CPU
// rewrite deadline. Fixtures are authored text, never acoustic-quality evidence.
func TestNativeWindowsLifecycle(t *testing.T) {
	runtimeDir, modelDir, out := os.Getenv("ARION_GGUF_RUNTIME"), os.Getenv("ARION_GGUF_MODEL"), os.Getenv("ARION_GGUF_OBSERVATIONS")
	if runtimeDir == "" || modelDir == "" || out == "" {
		t.Skip("native GGUF artifacts and output path required")
	}
	data, err := os.ReadFile("testdata/windows-local-rewrite-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ ID, Scene, Text, Screen string }
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	deadline := rewriting.Timeout("windows-amd64")
	ctx, cancel := context.WithTimeout(context.Background(), 95*time.Second)
	c, info, err := client.Start(ctx, client.Config{Worker: filepath.Join(runtimeDir, "arion-llm.exe"), Runtime: runtimeDir, Model: modelDir, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := encoder.Encode(map[string]any{"kind": "load", "load": info}); err != nil {
		t.Fatal(err)
	}
	run := func(kind string, req ports.RewriteRequest, deadline time.Duration) (ports.RewriteResult, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), deadline)
		defer cancel()
		started := time.Now()
		result, err := c.Rewrite(ctx, req)
		message := ""
		if err != nil {
			message = err.Error()
		}
		if writeErr := encoder.Encode(map[string]any{"kind": kind, "request": req, "response": result, "error": message, "elapsedMs": float64(time.Since(started).Microseconds()) / 1000, "deadlineMs": deadline.Milliseconds()}); writeErr != nil {
			t.Fatal(writeErr)
		}
		return result, err
	}
	for i, item := range cases {
		req := ports.RewriteRequest{Identity: contracts.Identity{SessionID: item.ID, Generation: 1}, Scene: item.Scene, Text: item.Text}
		req.Context.Screen = item.Screen
		result, err := run(item.ID, req, deadline)
		if err != nil || result.Text == "" {
			t.Errorf("%s: %+v %v", item.ID, result, err)
		}
		if i == 0 {
			hot, err := run("hot-repeat", req, deadline)
			if err != nil || hot.Text == "" {
				t.Errorf("hot repeat: %+v %v", hot, err)
			}
		}
		c.End(req.Identity)
	}
	req := ports.RewriteRequest{Identity: contracts.Identity{SessionID: "cancel-retry", Generation: 1}, Scene: "chat", Text: cases[0].Text}
	if _, err := run("cancelled", req, 20*time.Millisecond); err == nil {
		t.Error("cancelled generation succeeded")
	}
	req.Identity.Generation++
	if result, err := run("after-cancellation", req, deadline); err != nil || result.Text == "" || result.ReusedTokens != 0 {
		t.Errorf("retry: %+v %v", result, err)
	}
	c.End(req.Identity)
	c.Close()
	if c.Alive() {
		t.Error("closed worker remains ready")
	}
}
