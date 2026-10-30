package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
			if e.Code != diagnostics.Failure(test.err) {
				t.Fatal(e.Code)
			}
			e.Code = ""
			e.Event, e.Outcome, e.Count, e.DurationMs = "", "", 0, 0
			if e != (diagnostics.Entry{}) {
				t.Fatal("unexpected metadata retained", e)
			}
		})
	}
}

func TestDeveloperModeGatesKeyboardDiagnostics(t *testing.T) {
	dir := t.TempDir()
	recorder, err := diagnostics.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := &App{diagnostics: recorder}
	a.recordKeyboardDiagnostic(diagnostics.Entry{Event: "keyboard_ranking_trace", Reason: "caret_not_at_end"})
	a.prefs.DeveloperMode = true
	a.recordKeyboardDiagnostic(diagnostics.Entry{Event: "keyboard_ranking_trace", Reason: "no_comparable_candidates", Count: 1})
	a.prefs.DeveloperMode = false
	a.recordKeyboardDiagnostic(diagnostics.Entry{Event: "keyboard_ranking_trace", Reason: "non_chinese"})
	if err := recorder.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "events.0.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "no_comparable_candidates") || strings.Contains(string(data), "caret_not_at_end") || strings.Contains(string(data), "non_chinese") {
		t.Fatal(string(data))
	}
}

func TestKeyboardAbstentionDiagnostics(t *testing.T) {
	for _, reason := range []string{"no_context", "no_comparable_candidates"} {
		var entry diagnostics.Entry
		ranker := observedKeyboardRanker{ranker: observedRankFunc(func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
			return contracts.KeyboardRankResult{Decision: "abstained_" + reason}, nil
		}), record: func(e diagnostics.Entry) { entry = e }}
		_, err := ranker.RankKeyboard(context.Background(), contracts.KeyboardRankRequest{})
		if err != nil || entry.Outcome != reason {
			t.Fatal(entry, err)
		}
	}
}
