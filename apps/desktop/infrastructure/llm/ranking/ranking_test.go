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
	return contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "test", Generation: 1}, Revision: 1, Input: "beijing", Before: "这幅图的", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "a", Text: "北京", Source: "rime"}, End: 7}, {Candidate: domain.Candidate{ID: "b", Text: "背景", Source: "rime"}, End: 7}, {Candidate: domain.Candidate{ID: "c", Text: "被", Source: "rime"}, End: 3}}}
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

func TestUnknownCoverageAbstainsWithoutCallingModel(t *testing.T) {
	req := snapshot()
	for i := range req.Candidates {
		req.Candidates[i].End = 0
	}
	r := Ranker{Chat: completionFunc(func(context.Context, prompt.ChatRequest) (engine.Result, error) {
		t.Fatal("model called without comparable producer ranges")
		return engine.Result{}, nil
	})}
	out, err := r.RankKeyboard(context.Background(), req)
	if err != nil || out.Decision != "abstained_no_comparable_candidates" || out.Candidates[0].ID != req.Candidates[0].ID {
		t.Fatal(out, err)
	}
	for _, span := range [][2]int{{-1, 3}, {3, 2}, {8, 9}, {2, 2}} {
		bad := snapshot()
		bad.Candidates[1].Start, bad.Candidates[1].End = span[0], span[1]
		if _, err := Prepare(bad); err == nil {
			t.Fatalf("accepted invalid span %v", span)
		}
	}
	bad := snapshot()
	bad.Input = "你hao"
	bad.Candidates[0].End = len(bad.Input)
	bad.Candidates[1].Start, bad.Candidates[1].End = 1, len(bad.Input)
	if _, err := Prepare(bad); err == nil {
		t.Fatal("accepted start inside UTF-8 code point")
	}
}

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

func TestOnlyComparableCandidatesReachCompletion(t *testing.T) {
	req := snapshot()
	req.Input = "nixian"
	req.Candidates[0].Start, req.Candidates[0].End = 2, 6
	req.Candidates[0].Text = "现"
	req.Candidates[1].Start, req.Candidates[1].End = 2, 6
	req.Candidates[1].Text = "西安"
	req.Candidates[2].Start, req.Candidates[2].End = 0, 6
	calls := 0
	r := Ranker{Chat: completionFunc(func(_ context.Context, in prompt.ChatRequest) (engine.Result, error) {
		calls++
		var data struct {
			Candidates []domain.Candidate `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(in.Messages[1].Content), &data); err != nil {
			t.Fatal(err)
		}
		if len(data.Candidates) != 2 || data.Candidates[0].ID != "a" || data.Candidates[1].ID != "b" {
			t.Fatalf("incomparable candidate sent: %+v", data)
		}
		return engine.Result{Text: `{"pick":"b"}`}, nil
	})}
	out, err := r.RankKeyboard(context.Background(), req)
	if err != nil || calls != 1 || len(out.Candidates) != 3 || out.Candidates[0].ID != "b" || out.Candidates[1].ID != "a" || out.Candidates[2].ID != "c" {
		t.Fatal(out, err, calls)
	}
}

func TestNoComparableCandidatesNeedsNoTransport(t *testing.T) {
	for _, test := range []struct {
		name   string
		modify func(*contracts.KeyboardRankRequest)
	}{
		{"single", func(r *contracts.KeyboardRankRequest) { r.Candidates = r.Candidates[:1] }},
		{"partial", func(r *contracts.KeyboardRankRequest) { r.Candidates[1].End = 4 }},
		{"different-start", func(r *contracts.KeyboardRankRequest) { r.Candidates[1].Start = 1 }},
		{"unknown-first", func(r *contracts.KeyboardRankRequest) { r.Candidates[0].End = 0 }},
		{"unknown-alternative", func(r *contracts.KeyboardRankRequest) { r.Candidates[1].End = 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := snapshot()
			test.modify(&req)
			// Even the no-context diagnostic override cannot waive range eligibility.
			req.Before = ""
			r := Ranker{AllowWithoutContext: true}
			out, err := r.RankKeyboard(context.Background(), req)
			if err != nil || out.Decision != "abstained_no_comparable_candidates" || len(out.Candidates) != len(req.Candidates) {
				t.Fatal(out, err)
			}
			for i, candidate := range req.Candidates {
				if out.Candidates[i].ID != candidate.ID {
					t.Fatal("abstention changed producer order")
				}
			}
		})
	}
}

func TestCandidateFieldsCannotShareUTF8Bytes(t *testing.T) {
	req := snapshot()
	req.Candidates[1].ID = string([]byte{0xc2})
	req.Candidates[1].Text = string([]byte{0xa2})
	if _, err := Prepare(req); err == nil {
		t.Fatal("invalid field bytes joined into valid UTF-8")
	}
}
