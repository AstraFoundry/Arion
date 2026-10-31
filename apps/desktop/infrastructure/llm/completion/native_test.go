package completion

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
)

// This small authored smoke checks the local worker, not general accuracy.
func TestLocalCompletion(t *testing.T) {
	runtime, model := os.Getenv("ARION_KEYBOARD_TEST_RUNTIME"), os.Getenv("ARION_KEYBOARD_TEST_MODEL")
	if runtime == "" || model == "" || os.Getenv("ARION_KEYBOARD_TEST_WORKER") == "" {
		t.Skip("explicit local runtime and model required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, _, err := client.Start(ctx, client.Config{Worker: os.Getenv("ARION_KEYBOARD_TEST_WORKER"), Runtime: runtime, Model: model, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	r := Ranker{Chat: chatFunc(func(ctx context.Context, req prompt.ChatRequest) (engine.Result, error) {
		out, err := c.Complete(ctx, req)
		t.Logf("synthetic model output=%q error=%v", out.Text, err)
		return out, err
	})}
	for _, test := range []struct{ input, preedit, before, screen, want string }{
		{"chonggyxia", "chong g y xia", "我想要把这个项目", "", "重构一下"},
		{"moxbjjdyy", "mo x b j j d y y", "请检查", "日志中记录了模型被拒绝的原因。", "模型被拒绝的原因"},
		{"niurouzizhufangti", "niu rou zi zhu fang ti", "想要下楼吃一些", "餐厅菜单：牛肉自助放题。", "牛肉自助放题"},
	} {
		req := request()
		req.Input = test.input
		req.Preedit = test.preedit
		req.Before = test.before
		req.Screen = test.screen
		req.Candidates[0].End = len(req.Input)
		started := time.Now()
		out, err := r.RankKeyboard(ctx, req)
		t.Logf("input=%s elapsed=%s result=%+v error=%v", test.input, time.Since(started), out.Completion, err)
		if err != nil || out.Completion == nil || out.Completion.Text != test.want {
			t.Errorf("expected contextual suggestion %s", test.want)
		}
	}
}
