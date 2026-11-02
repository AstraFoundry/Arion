package ranking

import (
	"context"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
)

type orderChat func(context.Context, prompt.ChatRequest) (engine.Result, error)

func (f orderChat) Complete(c context.Context, p prompt.ChatRequest) (engine.Result, error) {
	return f(c, p)
}
func TestOrderRejectsInvalidPermutations(t *testing.T) {
	req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "test", Generation: 1}, Revision: 1, Input: "shishi", Before: "请", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "c0", Text: "实施", Source: "phrase"}, End: 6},
		{Candidate: domain.Candidate{ID: "c1", Text: "事实", Source: "phrase"}, End: 6},
	}}
	for _, raw := range []string{`{"order":["c0","c0"]}`, `{"order":["c0"]}`, `{"order":["c0","fake"]}`, `{"order":["c1","c0"],"confidence":1}`, `{"order":["c1","c0"]} {}`} {
		_, err := Order(context.Background(), orderChat(func(context.Context, prompt.ChatRequest) (engine.Result, error) { return engine.Result{Text: raw}, nil }), req, "")
		if err == nil {
			t.Fatalf("admitted %s", raw)
		}
	}
	order, err := Order(context.Background(), orderChat(func(context.Context, prompt.ChatRequest) (engine.Result, error) {
		return engine.Result{Text: `{"order":["c1","c0"]}`}, nil
	}), req, "")
	if err != nil || order[0] != "c1" {
		t.Fatal(order, err)
	}
}
