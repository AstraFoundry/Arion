package bootstrap

import (
	"context"
	"strings"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/ranking"
)

// keyboardContext reads only after consent and validates the native client's
// application against the captured target before asking for any text. Each
// request captures afresh; no field text survives a focus or revision change.
type keyboardContext struct {
	targets Targets
	ranker  ports.KeyboardRanker
	enabled func() bool
	cursor  func() bool
}

func (r keyboardContext) RankKeyboard(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	if err := ctx.Err(); err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	if _, err := ranking.Prepare(req); err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	if !r.enabled() {
		return r.ranker.RankKeyboard(ctx, req)
	}
	if req.Application == "" || domain.ContextDenied(req.Application) {
		return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeTargetUnsupported, "keyboard context target unavailable")
	}
	target, err := r.targets.Capture(ctx, req.Identity)
	if err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	defer r.targets.Discard(target.Target)
	if target.Sensitive || target.Application != req.Application {
		return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeTargetChanged, "keyboard context target changed")
	}
	snapshot, err := r.targets.ReadContext(ctx, req.Identity, target.Target)
	if err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	if !snapshot.SafeToRead || snapshot.Target != target.Target || snapshot.Identity != req.Identity {
		return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeTargetUnsupported, "keyboard context unreadable")
	}
	if err = ctx.Err(); err != nil {
		return contracts.KeyboardRankResult{}, err
	}
	if r.cursor() {
		// The native host supplies its exact owned marked text. Refuse a changed
		// caret instead of treating that uncommitted spelling as preceding context.
		if req.Preedit == "" || !strings.HasSuffix(snapshot.Before, req.Preedit) {
			return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeTargetChanged, "keyboard marked text changed")
		}
		snapshot.Before = strings.TrimSuffix(snapshot.Before, req.Preedit)
	}
	bounded := (domain.RewriteContext{Before: snapshot.Before, After: snapshot.After}).Bounded()
	req.Before, req.After = bounded.Before, bounded.After
	// Screen context is bounded separately; do not silently truncate evidence.
	if len(snapshot.Screen) > 8192 {
		return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeContractUnsupported, "keyboard screen context exceeds bounds")
	}
	req.Screen = strings.TrimSpace(snapshot.Screen)
	return r.ranker.RankKeyboard(ctx, req)
}
