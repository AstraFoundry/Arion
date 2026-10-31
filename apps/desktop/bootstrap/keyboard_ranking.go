package bootstrap

import (
	"context"
	"errors"
	goruntime "runtime"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/keyboardmodel"
	"arion.local/desktop/infrastructure/llm/ranking"
	"arion.local/desktop/infrastructure/rewriting"
)

type keyboardRankingReceiver interface {
	ConfigureKeyboardRankingStages(ports.KeyboardRanker, ports.KeyboardRanker) error
}

func (a *App) recordKeyboardDiagnostic(entry diagnostics.Entry) {
	if a.preferences().DeveloperMode {
		a.recordEntry(entry)
	}
}

func (a *App) configureKeyboardRanking() {
	receiver, ok := a.ime.(keyboardRankingReceiver)
	if !ok {
		return
	}
	if observer, ok := a.ime.(interface{ OnKeyboardRankingDiagnostic(func(string, int)) }); ok {
		observer.OnKeyboardRankingDiagnostic(func(reason string, count int) {
			a.recordKeyboardDiagnostic(diagnostics.Entry{Event: "keyboard_ranking_trace", Reason: reason, Count: count})
		})
	}
	var fast, ranker ports.KeyboardRanker
	if a.state.Preferences.Keyboard.NeuralRanking {
		fast = observedKeyboardRanker{ranker: a.keyboardContextRanker(a.keyboardModel, false), modelID: keyboardmodel.ModelID, record: a.recordKeyboardDiagnostic}
		ranker = observedKeyboardRanker{ranker: a.keyboardContextRanker(&ranking.Ranker{Chat: a.rewriter}, true), modelID: rewriting.DefaultModel, record: a.recordKeyboardDiagnostic}
	}
	a.record("keyboard_ranking_configuration", receiver.ConfigureKeyboardRankingStages(fast, ranker))
}

// Observe proposals at the model boundary. Acceptance here is not confirmation
// that the native host applied the proposal before the user changed input.
type observedKeyboardRanker struct {
	ranker  ports.KeyboardRanker
	modelID string
	record  func(diagnostics.Entry)
}

func (r observedKeyboardRanker) RankKeyboard(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	started := time.Now()
	result, err := r.ranker.RankKeyboard(ctx, req)
	outcome := "unchanged"
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		outcome = "timeout"
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		outcome = "cancelled"
	case domain.CodeOf(err) == domain.CodeModelMissing:
		outcome = "unavailable"
	case err != nil:
		outcome = "rejected"
	case result.Decision == "abstained_no_context":
		outcome = "no_context"
	case result.Decision == "abstained_no_comparable_candidates":
		outcome = "no_comparable_candidates"
	case len(req.Candidates) > 0 && len(result.Candidates) > 0 && result.Candidates[0].ID != req.Candidates[0].ID:
		outcome = "accepted"
	}
	r.record(diagnostics.Entry{Event: "keyboard_ranking", ModelID: r.modelID, Outcome: outcome, DurationMs: float64(time.Since(started).Microseconds()) / 1000, Count: len(req.Candidates), Code: diagnostics.Failure(err)})
	return result, err
}

func (a *App) keyboardContextRanker(ranker ports.KeyboardRanker, screen bool) ports.KeyboardRanker {
	if goruntime.GOOS != "darwin" {
		return ranker
	}
	enabled := func() bool {
		p := a.preferences().Keyboard
		return p.NeuralRanking && (p.ContextCursor || screen && p.ContextScreen)
	}
	targets := a.platform.NewTargets(func() application.Delivery { return a.preferences().Delivery }, TargetGates{
		Cursor: func() bool { p := a.preferences().Keyboard; return p.NeuralRanking && p.ContextCursor },
		Screen: func() bool { p := a.preferences().Keyboard; return p.NeuralRanking && screen && p.ContextScreen },
	})
	return keyboardContext{trace: func(reason string, err error) {
		a.recordKeyboardDiagnostic(diagnostics.Entry{Event: "keyboard_ranking_trace", Reason: reason, Code: diagnostics.Failure(err)})
	}, targets: targets, ranker: ranker, enabled: enabled, cursor: func() bool { return a.preferences().Keyboard.ContextCursor }}
}
