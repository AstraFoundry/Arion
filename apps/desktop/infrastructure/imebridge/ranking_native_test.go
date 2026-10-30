package imebridge

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/desktop/infrastructure/keyboardmodel"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/llm/ranking"
)

// Uses only an owned connection and synthetic text. No installed editor access.
func TestRealRankingAfterNativeFreeze(t *testing.T) {
	fastBinary, fastModel := os.Getenv("ARION_TEST_KEYBOARD_BINARY"), os.Getenv("ARION_TEST_KEYBOARD_MODEL")
	bundled, model := os.Getenv("ARION_KEYBOARD_TEST_RUNTIME"), os.Getenv("ARION_KEYBOARD_TEST_MODEL")
	if fastBinary == "" || fastModel == "" || bundled == "" || model == "" {
		t.Skip("explicit local model paths required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	fast, err := keyboardmodel.Open(ctx, fastBinary, fastModel)
	if err != nil {
		t.Fatal(err)
	}
	defer fast.Close()
	name := "arion-llm"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	slow, _, err := client.Start(ctx, client.Config{Worker: filepath.Join(bundled, name), Runtime: bundled, Model: model, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer slow.Close()
	m, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	if err = m.ConfigureKeyboardRankingStages(fast, &ranking.Ranker{Chat: slow}); err != nil {
		t.Fatal(err)
	}
	h.send(t, message{Op: "keyboard_freeze", Activation: "editor", Revision: 1})
	request := rankRequest(2)
	request.RankRequest.Input = "shijie"
	request.RankRequest.Before = "欢迎来到这个"
	for i, text := range []string{"时节", "世界"} {
		request.RankRequest.Candidates[i].Text = text
		request.RankRequest.Candidates[i].Source = "sentence"
		request.RankRequest.Candidates[i].End = 6
	}
	start := time.Now()
	h.send(t, request)
	for _, template := range []string{"chinese-ime-lm-0.2.0", ranking.Version} {
		reply := h.read(t)
		if reply.Status != "ok" || reply.RankResult == nil || reply.RankResult.Template != template || reply.RankResult.Candidates[0].ID != "c1" {
			t.Fatalf("real proposal: %+v", reply)
		}
		t.Logf("%s proposal through bridge after freeze: %s", template, time.Since(start))
	}
}
