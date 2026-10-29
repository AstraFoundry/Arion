package imebridge

import (
	"context"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain/ports"
)

const rankingCapability = "keyboard-ranking-v1"

// ConfigureKeyboardRanking replaces the local ranker and revokes every proposal
// made under the previous policy. A nil port disables inference immediately.
func (m *IME) ConfigureKeyboardRanking(ranker ports.KeyboardRanker) error {
	var scheduler *application.KeyboardRanking
	if ranker != nil {
		var err error
		scheduler, err = application.NewKeyboardRanking(ranker, 750*time.Millisecond, 75*time.Millisecond)
		if err != nil {
			return err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		if scheduler != nil {
			scheduler.Close()
		}
		return changed()
	}
	if m.ranking != nil {
		m.ranking.Close()
	}
	m.ranking = scheduler
	return nil
}

func (m *IME) invalidateRankingLocked() {
	if m.ranking != nil {
		m.ranking.Freeze()
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
		return
	}
	m.rankingRevision = msg.Revision
	if m.ranking == nil {
		return
	}
	if msg.Op == "keyboard_freeze" {
		m.ranking.Freeze()
		return
	}
	req := msg.RankRequest
	if req == nil || req.SessionID != msg.Activation || req.Generation != 1 || req.Revision != msg.Revision {
		m.ranking.Freeze()
		return
	}
	scheduler, conn, activation := m.ranking, m.conn, m.activation
	outcome := scheduler.Submit(*req)
	go func() {
		out := <-outcome
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.closed || m.ranking != scheduler || m.conn != conn || m.activation != activation ||
			m.rankingRevision != msg.Revision || m.updating || m.token != "" || !m.clientVerified || !m.peer.Alive() {
			return
		}
		reply := message{Op: "keyboard_ranked", Revision: msg.Revision, Status: "unavailable"}
		if out.Err == nil && scheduler.CanApply(out) {
			reply.Status = "ok"
			result := out.Result
			reply.RankResult = &result
		}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_ = m.sendLocked(ctx, reply)
	}()
}
