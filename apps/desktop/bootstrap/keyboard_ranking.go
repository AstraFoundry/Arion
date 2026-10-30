package bootstrap

import (
	"context"
	"errors"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/llm/ranking"
)

type keyboardRankingReceiver interface {
	ConfigureKeyboardRanking(ports.KeyboardRanker) error
}

func (a *App) configureKeyboardRanking() {
	receiver, ok := a.ime.(keyboardRankingReceiver)
	if !ok {
		return
	}
	var ranker ports.KeyboardRanker
	if a.state.Preferences.Keyboard.NeuralRanking {
		ranker = observedKeyboardRanker{ranker: &ranking.Ranker{Chat: a.rewriter}, record: a.recordEntry}
	}
	a.record("keyboard_ranking_configuration", receiver.ConfigureKeyboardRanking(ranker))
}

// Observe proposals at the model boundary. Acceptance here is not confirmation
// that the native host applied the proposal before the user changed input.
type observedKeyboardRanker struct {
	ranker ports.KeyboardRanker
	record func(diagnostics.Entry)
}

func (r observedKeyboardRanker) RankKeyboard(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	started := time.Now()
	result, err := r.ranker.RankKeyboard(ctx, req)
	outcome := "unchanged"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		outcome = "timeout"
	case errors.Is(err, context.Canceled):
		outcome = "cancelled"
	case domain.CodeOf(err) == domain.CodeModelMissing:
		outcome = "unavailable"
	case err != nil:
		outcome = "rejected"
	case len(req.Candidates) > 0 && len(result.Candidates) > 0 && result.Candidates[0].ID != req.Candidates[0].ID:
		outcome = "accepted"
	}
	r.record(diagnostics.Entry{Event: "keyboard_ranking", Outcome: outcome, DurationMs: float64(time.Since(started).Microseconds()) / 1000, Count: len(req.Candidates)})
	return result, err
}
