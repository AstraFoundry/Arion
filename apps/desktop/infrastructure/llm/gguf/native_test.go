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
	"arion.local/desktop/infrastructure/llm/prompt"
)

// This opt-in test runs the actual packaged worker and pinned local weight.
// Its observations are integration evidence, not held-out quality estimates.
func TestNativeSelectedPipeline(t *testing.T) {
	runtimeDir, modelDir, out := os.Getenv("ARION_GGUF_RUNTIME"), os.Getenv("ARION_GGUF_MODEL"), os.Getenv("ARION_GGUF_OBSERVATIONS")
	if runtimeDir == "" || modelDir == "" || out == "" {
		t.Skip("native GGUF artifacts and output path required")
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
	if info.PromptID != prompt.SelectedVersion {
		t.Fatal(info.PromptID)
	}
	record := func(kind string, request ports.RewriteRequest, result ports.RewriteResult, err error, elapsed time.Duration) {
		t.Helper()
		message := ""
		if err != nil {
			message = err.Error()
		}
		if err := encoder.Encode(map[string]any{"kind": kind, "request": request, "response": result, "error": message, "elapsedMs": float64(elapsed.Microseconds()) / 1000, "load": info}); err != nil {
			t.Fatal(err)
		}
	}
	id := contracts.Identity{SessionID: "native-product-integration", Generation: 1}
	cases := []ports.RewriteRequest{
		{Identity: id, Scene: "work", Text: "小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。"},
		{Identity: id, Scene: "email", Text: "周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。"},
		{Identity: id, Scene: "prompt", Text: "给 fetchState 加校验，不改返回字段，也别动数据库。"},
	}
	var reference string
	for i, r := range cases {
		started := time.Now()
		result, err := c.Rewrite(context.Background(), r)
		record("rewrite", r, result, err, time.Since(started))
		if err != nil || result.Text == "" || result.Template != prompt.SelectedVersion {
			t.Fatalf("%+v %v", result, err)
		}
		if i == 0 {
			reference = result.Text
		}
	}
	c.End(id)
	r := cases[0]
	r.Identity.Generation++
	started := time.Now()
	result, err := c.Rewrite(context.Background(), r)
	record("new-session", r, result, err, time.Since(started))
	if err != nil || result.Text != reference {
		t.Fatalf("session isolation/parity changed: %q vs %q (%v)", result.Text, reference, err)
	}
	r.Prefill = true
	started = time.Now()
	result, err = c.Rewrite(context.Background(), r)
	record("preparation", r, result, err, time.Since(started))
	if err != nil || result.OutputTokens != 0 || result.Text != "" {
		t.Fatalf("preparation sampled tokens: %+v %v", result, err)
	}
	r.Prefill = false
	r.Text = "请整理工作内容，第一小陈负责设计，第二小刘负责测试，第三今天先核对，不要发出去。"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	started = time.Now()
	result, err = c.Rewrite(ctx, r)
	cancel()
	record("cancelled", r, result, err, time.Since(started))
	if err == nil || time.Since(started) > time.Second {
		t.Fatalf("cancellation failed: %v", err)
	}
	r = cases[1]
	started = time.Now()
	result, err = c.Rewrite(context.Background(), r)
	record("after-cancellation", r, result, err, time.Since(started))
	if err != nil || result.Text == "" {
		t.Fatalf("worker failed after cancellation: %+v %v", result, err)
	}
}
