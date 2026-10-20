package gguf_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
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
		{Identity: id, Scene: "prompt", Text: "请确认 syncState 是不是还没准备好，暂时不要重启。"},
		{Identity: id, Scene: "prompt", Text: "Can you check why fetchState is slow? Keep the response shape and do not restart it."},
		{Identity: id, Scene: "work", Text: "Zion prepares receipts and Remy checks figures, receipts are due Monday and figures are due Thursday."},
		{Identity: id, Scene: "prompt", Text: "把间隔设成六十秒，不对，是六秒，保留重试次数，只改 Router.go。"},
		{Identity: id, Scene: "prompt", Text: "第一，给 fetchState 加日志。第二，保留返回字段。第三，不改配置。"},
		{Identity: id, Scene: "work", Text: "First, Jules checks inventory on Monday. Second, Morgan packs the orders on Thursday. Ship only after inspection clears."},
		{Identity: id, Scene: "list", Text: "核对账目，整理凭证，双方确认后再提交。"},
		{Identity: id, Scene: "chat", Text: "我可能晚点到，如果下雨就线上参加，别替我确认。"},
		{Identity: id, Scene: "search", Text: "如何检查 syncState 的状态？"},
		{Identity: id, Scene: "prompt", Text: "请把以下事项列成清单：给 fetchState 加日志，保留返回字段，不改配置。"},
		{Identity: id, Scene: "list", Text: "阈值设成零点八，超时五秒，不要重启。"},
	}
	listLine := regexp.MustCompile(`(?m)^\s*(?:[-*•]|[0-9]+[.)、])\s+`)

	var reference string
	for i, r := range cases {
		started := time.Now()
		result, err := c.Rewrite(context.Background(), r)
		record("rewrite", r, result, err, time.Since(started))
		if err != nil || result.Text == "" || result.Template != prompt.SelectedVersion {
			t.Fatalf("%+v %v", result, err)
		}

		wantsList := r.Scene == "list" || i == 12
		enumerated := domain.Enumerated(r.Text)
		hasList := listLine.MatchString(result.Text)
		if enumerated && !hasList && !domain.Enumerated(result.Text) {
			t.Errorf("lost explicit enumeration for case %d: %q", i, result.Text)
		}
		if !enumerated && hasList != wantsList {
			t.Errorf("unexpected list layout for case %d: %q", i, result.Text)
		}
		if !wantsList && r.Scene != "email" && regexp.MustCompile(`(?m)^#{1,6} |(?:要求|限制|Requirements|Constraints)[:：]`).MatchString(result.Text) {
			t.Errorf("invented heading for case %d: %q", i, result.Text)
		}
		if i == 13 && (!strings.Contains(result.Text, "0.8") || !strings.Contains(result.Text, "5") || !strings.Contains(result.Text, "不要重启")) {
			t.Errorf("lost list numeric formatting or constraint: %q", result.Text)
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
