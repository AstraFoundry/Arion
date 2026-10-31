package completion

import (
	"context"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
)

type chatFunc func(context.Context, prompt.ChatRequest) (engine.Result, error)

func (f chatFunc) Complete(c context.Context, r prompt.ChatRequest) (engine.Result, error) {
	return f(c, r)
}
func request() contracts.KeyboardRankRequest {
	return contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "test", Generation: 1}, Revision: 1, AllowCompletion: true, Input: "chonggyxia", Preedit: "chong g y xia", Before: "我想要把这个项目", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "崇高一下", Source: "sentence"}, End: 10}}}
}
func TestReadingConsumesAllInput(t *testing.T) {
	for _, c := range []struct {
		input, text string
		valid       bool
	}{
		{"chonggyxia", "重构一下", true}, {"chong'g'y'xia", "重构一下", true},
		{"moxbjjdyy", "模型被拒绝的原因", true}, {"niurouzizhufangti", "牛肉自助放题", true},
		{"niurouzizhufangti", "牛肉自助", false}, {"chonggyxia", "重构", false},
		{"chonggyxia", "重构一下。", false}, {"chonggyxia", "删除项目", false},
		{"chong'g'y'xia'", "重构一下", false}, {"chonggyxia", "重构一下全部", false},
	} {
		preedit := map[string]string{"chonggyxia": "chong g y xia", "chong'g'y'xia": "chong g y xia", "moxbjjdyy": "mo x b j j d y y", "niurouzizhufangti": "niu rou zi zhu fang ti"}[c.input]
		reading := SegmentedReading(c.input, preedit, c.text)
		if (reading != "") != c.valid {
			t.Errorf("%s/%s: %q", c.input, c.text, reading)
		}
		if reading != "" && !domain.ValidKeyboardCompletion(domain.KeyboardCompletion{Input: c.input, Text: c.text, Reading: reading}, c.input) {
			t.Fatal("validator disagreement")
		}
	}
}
func TestGeneratedCompletionIsSeparateFromPermutation(t *testing.T) {
	for _, c := range []struct {
		text             string
		valid, generated bool
	}{
		{"NONE", true, false}, {"重构一下", true, true}, {"崇高一下", true, false}, {"重构", false, false}, {"解释：重构一下", false, false},
	} {
		r := Ranker{Chat: chatFunc(func(context.Context, prompt.ChatRequest) (engine.Result, error) {
			return engine.Result{Text: c.text}, nil
		})}
		out, err := r.RankKeyboard(context.Background(), request())
		if (err == nil) != c.valid || (out.Completion != nil) != c.generated {
			t.Errorf("%q: %+v %v", c.text, out, err)
		}
	}
}
func TestNoContextAndTruncation(t *testing.T) {
	calls := 0
	r := Ranker{Chat: chatFunc(func(context.Context, prompt.ChatRequest) (engine.Result, error) {
		calls++
		return engine.Result{Text: "重构一下", Truncated: true}, nil
	})}
	req := request()
	req.Before = ""
	out, err := r.RankKeyboard(context.Background(), req)
	if err != nil || calls != 0 || out.Decision != "abstained_no_context" {
		t.Fatal(out, err, calls)
	}
	if _, err = r.RankKeyboard(context.Background(), request()); err == nil {
		t.Fatal("truncated result admitted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = r.RankKeyboard(ctx, request()); err == nil {
		t.Fatal("cancelled result admitted")
	}
}

func TestSegmentationCannotExpandFullSyllablesIntoInitials(t *testing.T) {
	if SegmentedReading("chonggyxia", "chong g y xia", "策划区内规划与细案") != "" {
		t.Fatal("full syllable expanded into initials")
	}
	if SegmentedReading("chonggyxia", "chong g y xia", "重构一下") == "" {
		t.Fatal("valid mixed input rejected")
	}
	for _, preedit := range []string{"chong g y xia", "chong'g'y'xia"} {
		grammar, err := Grammar("chonggyxia", preedit)
		if err != nil || len(grammar) == 0 || len(grammar) > 256*1024 {
			t.Fatal(err)
		}
	}
	for _, preedit := range []string{"chong g y", "chong g y xia extra", "重构 g y xia"} {
		if _, err := Grammar("chonggyxia", preedit); err == nil {
			t.Fatal("invalid engine segmentation admitted")
		}
	}
}

func TestSingleSyllableRetainsIDRanking(t *testing.T) {
	req := request()
	req.Input = "shi"
	req.Preedit = "shi"
	req.Candidates = []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "是", Source: "phrase"}, End: 3}, {Candidate: domain.Candidate{ID: "c1", Text: "时", Source: "phrase"}, End: 3}}
	r := Ranker{Chat: chatFunc(func(_ context.Context, p prompt.ChatRequest) (engine.Result, error) {
		if p.Grammar != "" {
			t.Fatal("single-syllable generation")
		}
		return engine.Result{Text: `{"pick":"c1"}`}, nil
	})}
	out, err := r.RankKeyboard(context.Background(), req)
	if err != nil || out.Completion != nil || len(out.Candidates) != 2 || out.Candidates[0].ID != "c1" {
		t.Fatal(out, err)
	}
}
