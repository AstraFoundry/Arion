package gguf_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/client"
)

// Authored text probes exercise the actual worker, not acoustic recognition.
func TestNativeNumericFillers(t *testing.T) {
	runtimeDir, modelDir, out := os.Getenv("ARION_GGUF_RUNTIME"), os.Getenv("ARION_GGUF_MODEL"), os.Getenv("ARION_GGUF_OBSERVATIONS")
	if runtimeDir == "" || modelDir == "" || out == "" {
		t.Skip("native GGUF artifacts and output path required")
	}
	data, err := os.ReadFile("../../../../../benchmarks/reproduction/rewrite-numeric-filler-v1/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Scene, Text    string
		Contains, Excludes []string
	}
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
	c, info, err := client.Start(context.Background(), client.Config{Worker: filepath.Join(runtimeDir, "arion-llm"), Runtime: runtimeDir, Model: modelDir, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, tc := range cases {
		for _, live := range []bool{true, false} {
			name := tc.ID + "/final"
			if live {
				name = tc.ID + "/live"
			}
			t.Run(name, func(t *testing.T) {
				id := contracts.Identity{SessionID: name, Generation: 1}
				req := ports.RewriteRequest{Identity: id, Text: tc.Text, Scene: tc.Scene, Live: live}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				start := time.Now()
				result, err := c.Rewrite(ctx, req)
				message := ""
				if err != nil {
					message = err.Error()
				}
				if err := encoder.Encode(map[string]any{"id": tc.ID, "request": req, "response": result, "error": message, "elapsedMs": float64(time.Since(start).Microseconds()) / 1000, "load": info}); err != nil {
					t.Fatal(err)
				}
				c.End(id)
				if err != nil {
					t.Fatal(err)
				}
				for _, text := range tc.Contains {
					if !strings.Contains(result.Text, text) {
						t.Errorf("missing %q in %q", text, result.Text)
					}
				}
				for _, text := range tc.Excludes {
					if strings.Contains(result.Text, text) {
						t.Errorf("unexpected %q in %q", text, result.Text)
					}
				}
			})
		}
	}
}
