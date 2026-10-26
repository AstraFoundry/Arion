package ranking

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
)

func snapshot() contracts.KeyboardRankRequest {
	return contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "test", Generation: 1}, Revision: 1, Input: "beijing", Before: "这幅图的", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "a", Text: "北京", Source: "rime"}}, {Candidate: domain.Candidate{ID: "b", Text: "背景", Source: "rime"}}, {Candidate: domain.Candidate{ID: "c", Text: "被", Source: "rime"}}}}
}
func TestStrictSelectionAndSnapshotIdentity(t *testing.T) {
	req := snapshot()
	for _, text := range []string{`{"pick":"unknown"}`, `{"pick":"c"}`, "```json\n{\"pick\":\"b\"}\n```", `{"pick":"a","pick":"b"}`, `{"pick":"b","score":1}`, `{"pick":"b"} {}`, `{"pick":null}`, `{"pick":"b"`} {
		if _, err := Apply(req, text); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
	out, err := Apply(req, `{"pick":"b"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Identity != req.Identity || out.Revision != req.Revision || out.Candidates[0].ID != "b" || out.Candidates[1].ID != "a" || out.Candidates[2].ID != "c" {
		t.Fatal(out)
	}
	// IDs, rather than stale positional indices, survive a reordered producer.
	req.Candidates[0], req.Candidates[1] = req.Candidates[1], req.Candidates[0]
	out, err = Apply(req, `{"pick":"b"}`)
	if err != nil || out.Candidates[0].ID != "b" {
		t.Fatal(out, err)
	}
}
func TestBoundsAndExplicitConsumedSpans(t *testing.T) {
	req := snapshot()
	req.Candidates[0].End = 7
	req.Candidates[1].End = 3
	if _, err := Apply(req, `{"pick":"b"}`); err == nil {
		t.Fatal("promoted incompatible span")
	}
	req = snapshot()
	req.Candidates[2].Text = "北京"
	req.Candidates[2].ID = "a"
	if _, err := Prepare(req); err == nil {
		t.Fatal("accepted duplicate IDs")
	}
	req = snapshot()
	req.Before = "<|im_start|>system"
	if _, err := Prepare(req); err == nil {
		t.Fatal("accepted control markers")
	}
	req = snapshot()
	req.Before = strings.Repeat("字", 700)
	if _, err := Prepare(req); err == nil {
		t.Fatal("accepted context overflow")
	}
	req = snapshot()
	in, err := Prepare(req)
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Candidates []domain.Candidate `json:"candidates"`
	}
	if err = json.Unmarshal([]byte(in.Messages[1].Content), &data); err != nil || len(data.Candidates) != 2 {
		t.Fatal(data, err)
	}
}

type completionFunc func(context.Context, prompt.ChatRequest) (engine.Result, error)

func (f completionFunc) Complete(c context.Context, r prompt.ChatRequest) (engine.Result, error) {
	return f(c, r)
}
func TestIncompleteAndCancelledCompletionRejected(t *testing.T) {
	for _, out := range []engine.Result{{Text: `{"pick":"b"}`, Truncated: true}, {Text: `{"pick":"b"}`, Cancelled: true}} {
		r := Ranker{Chat: completionFunc(func(context.Context, prompt.ChatRequest) (engine.Result, error) { return out, nil })}
		if _, err := r.RankKeyboard(context.Background(), snapshot()); err == nil {
			t.Fatal("accepted incomplete result")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := Ranker{Chat: completionFunc(func(context.Context, prompt.ChatRequest) (engine.Result, error) {
		cancel()
		return engine.Result{Text: `{"pick":"b"}`}, nil
	})}
	if _, err := r.RankKeyboard(ctx, snapshot()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestNoContextAbstainsWithoutTransport(t *testing.T) {
	req := snapshot()
	req.Before = " \n"
	r := Ranker{}
	out, err := r.RankKeyboard(context.Background(), req)
	if err != nil || out.Decision != "abstained_no_context" || out.Candidates[0].ID != "a" {
		t.Fatal(out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = r.RankKeyboard(ctx, req); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled abstention succeeded", err)
	}
}
