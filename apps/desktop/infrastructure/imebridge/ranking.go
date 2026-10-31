package imebridge

import (
	"context"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain/ports"
)

// OnKeyboardRankingDiagnostic observes content-free native eligibility skips.
// The observer must return immediately and must not reenter the bridge.
func (m *IME) OnKeyboardRankingDiagnostic(observer func(string, int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRankingDiagnostic = observer
}

func (m *IME) traceRankingLocked(reason string, count int) {
	if m.onRankingDiagnostic != nil {
		m.onRankingDiagnostic(reason, count)
	}
}

const rankingCapability = "keyboard-ranking-v2"

// ConfigureKeyboardRanking replaces the local ranker and revokes every proposal
// made under the previous policy. A nil port disables inference immediately.
func (m *IME) ConfigureKeyboardRanking(ranker ports.KeyboardRanker) error {
	return m.ConfigureKeyboardRankingStages(nil, ranker)
}

// ConfigureKeyboardRankingStages gives each model its own latest-only lane.
// A settled result takes precedence over any late fast result for that revision.
func (m *IME) ConfigureKeyboardRankingStages(fast, settled ports.KeyboardRanker) error {
	var schedulers [2]*application.KeyboardRanking
	for i, ranker := range []ports.KeyboardRanker{fast, settled} {
		if ranker == nil {
			continue
		}
		debounce := time.Duration(0)
		budget := 100 * time.Millisecond
		if i == 1 {
			debounce = 150 * time.Millisecond
			budget = 3 * time.Second
		}
		scheduler, err := application.NewKeyboardRanking(ranker, budget, debounce)
		if err != nil {
			for _, s := range schedulers {
				if s != nil {
					s.Close()
				}
			}
			return err
		}
		schedulers[i] = scheduler
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		for _, s := range schedulers {
			if s != nil {
				s.Close()
			}
		}
		return changed()
	}
	if m.ranking != nil {
		m.ranking.Close()
	}
	if m.rankingFast != nil {
		m.rankingFast.Close()
	}
	m.rankingFast, m.ranking = schedulers[0], schedulers[1]
	m.rankingSettledRevision = 0
	return nil
}

func (m *IME) invalidateRankingLocked() {
	if m.ranking != nil {
		m.ranking.Freeze()
	}
	if m.rankingFast != nil {
		m.rankingFast.Freeze()
	}
}

// Caller holds the connection mutex. Keyboard identities never reuse voice
// session state. Activation changes reset their monotonically increasing epoch.
func (m *IME) receiveRankingLocked(msg message) {
	if msg.Activation == "" || msg.Activation != m.activation || !m.rankingCapable ||
		!m.clientVerified || !m.peer.Alive() || m.updating || m.token != "" || m.closed {
		return
	}
	if msg.Revision == 0 || msg.Revision <= m.rankingRevision {
		m.traceRankingLocked("stale_revision", 0)
		return
	}
	m.rankingRevision = msg.Revision
	if m.ranking == nil && m.rankingFast == nil {
		m.traceRankingLocked("ranking_disabled", 0)
		return
	}
	if msg.Op == "keyboard_rank_skipped" {
		m.invalidateRankingLocked()
		if msg.EligibleCount != nil && *msg.EligibleCount >= 0 && *msg.EligibleCount < 2 {
			switch msg.RankingReason {
			case "no_comparable_candidates", "navigation", "input_bounds", "caret_not_at_end", "non_chinese":
				m.traceRankingLocked(msg.RankingReason, *msg.EligibleCount)
			}
		}
		return
	}
	if msg.Op == "keyboard_freeze" {
		m.traceRankingLocked("freeze", 0)
		m.invalidateRankingLocked()
		return
	}
	req := msg.RankRequest
	if req == nil || req.SessionID != msg.Activation || req.Generation != 1 || req.Revision != msg.Revision {
		m.traceRankingLocked("invalid_request", 0)
		m.invalidateRankingLocked()
		return
	}
	req.Application = m.client
	req.AllowCompletion = req.AllowCompletion && m.completionCapable
	for tier, scheduler := range []*application.KeyboardRanking{m.rankingFast, m.ranking} {
		if scheduler == nil {
			continue
		}
		conn, activation := m.conn, m.activation
		laneRequest := *req
		if tier == 0 {
			laneRequest.AllowCompletion = false
		}
		outcome := scheduler.Submit(laneRequest)
		go func(tier int, scheduler *application.KeyboardRanking) {
			out := <-outcome
			m.mu.Lock()
			defer m.mu.Unlock()
			current := m.rankingFast
			if tier == 1 {
				current = m.ranking
			}
			if m.closed || current != scheduler || m.conn != conn || m.activation != activation ||
				m.rankingRevision != msg.Revision || m.updating || m.token != "" || !m.clientVerified || !m.peer.Alive() ||
				(tier == 0 && m.rankingSettledRevision == msg.Revision) {
				m.traceRankingLocked("superseded", len(req.Candidates))
				return
			}
			// An abstaining settled model has no new evidence; it must not undo a
			// valid fast promotion or suppress a still-running fast response.
			if tier == 1 && out.Err == nil && (out.Result.Decision == "abstained_uncertain" || out.Result.Decision == "abstained_no_context" || out.Result.Decision == "abstained_no_comparable_candidates") {
				return
			}
			reply := message{Op: "keyboard_ranked", Revision: msg.Revision, Status: "unavailable"}
			if out.Err == nil && scheduler.CanApply(out) {
				reply.Status = "ok"
				result := out.Result
				reply.RankResult = &result
				if tier == 1 {
					m.rankingSettledRevision = msg.Revision
				}
			} else if tier == 0 {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := m.sendLocked(ctx, reply); err != nil {
				m.traceRankingLocked("delivery_failed", len(req.Candidates))
			} else if reply.Status == "ok" {
				m.traceRankingLocked("proposal_sent", len(req.Candidates))
			}
		}(tier, scheduler)
	}
}
