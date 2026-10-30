package bootstrap

import (
	"context"
	"errors"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/diagnostics"
)

type observedRankFunc func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error)

func (f observedRankFunc) RankKeyboard(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	return f(ctx, req)
}

func TestKeyboardRankingObservationsContainOnlyOutcomes(t *testing.T) {
	for _, test := range []struct {
		outcome, first string
		err            error
	}{
		{"accepted", "second", nil}, {"unchanged", "first", nil},
		{"timeout", "", context.DeadlineExceeded}, {"cancelled", "", context.Canceled},
		{"unavailable", "", domain.Fail(domain.CodeModelMissing, "private error detail")},
		{"rejected", "", errors.New("private response")},
	} {
		t.Run(test.outcome, func(t *testing.T) {
			req := contracts.KeyboardRankRequest{Input: "private input", Before: "private context", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "first", Text: "private candidate"}}}}
			result := contracts.KeyboardRankResult{Candidates: []domain.RankedCandidate{{ID: test.first}}}
			var entries []diagnostics.Entry
			ranker := observedKeyboardRanker{ranker: observedRankFunc(func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
				return result, test.err
			}), record: func(e diagnostics.Entry) { entries = append(entries, e) }}
			got, err := ranker.RankKeyboard(context.Background(), req)
			if !errors.Is(err, test.err) || got.Candidates[0].ID != test.first {
				t.Fatal("observation changed result")
			}
			if len(entries) != 1 {
				t.Fatal(entries)
			}
			e := entries[0]
			if e.Event != "keyboard_ranking" || e.Outcome != test.outcome || e.Count != 1 || e.DurationMs < 0 {
				t.Fatal(e)
			}
			e.Event, e.Outcome, e.Count, e.DurationMs = "", "", 0, 0
			if e != (diagnostics.Entry{}) {
				t.Fatal("unexpected metadata retained", e)
			}
		})
	}
}
