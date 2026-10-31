package imebridge

import (
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type keyboardRankFunc func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error)

func (f keyboardRankFunc) RankKeyboard(c context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	return f(c, r)
}
func rankRequest(rev uint64) message {
	r := &contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "editor", Generation: 1}, Revision: rev, Input: "xian", Before: "这条", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "c0", Text: "西安", Source: "phrase"}, End: 4},
		{Candidate: domain.Candidate{ID: "c1", Text: "线", Source: "phrase"}, End: 4}}}
	return message{Op: "keyboard_rank", Activation: "editor", Revision: rev, RankRequest: r}
}
func rankAnswer(r contracts.KeyboardRankRequest) contracts.KeyboardRankResult {
	return contracts.KeyboardRankResult{Identity: r.Identity, Revision: r.Revision, Candidates: []domain.RankedCandidate{{ID: "c1"}, {ID: "c0"}}}
}
func rankingHost(t *testing.T, f keyboardRankFunc) (*IME, *host) {
	t.Helper()
	m := New("", nil, nil)
	t.Cleanup(m.Close)
	m.RequireClientVerification(func(PeerIdentity, ClientHello) error { return nil })
	if err := m.ConfigureKeyboardRanking(f); err != nil {
		t.Fatal(err)
	}
	alive := new(atomic.Bool)
	alive.Store(true)
	h, _ := verifiedHost(t, m, message{Op: "hello", Build: "ranking-test", Capabilities: []string{rankingCapability}}, alive)
	wait(t, func() bool { return m.ConnectedBuild() != "" })
	h.send(t, message{Op: "activate", Activation: "editor"})
	wait(t, m.ActiveClient)
	return m, h
}
func TestKeyboardRankingBoundToVerifiedSnapshot(t *testing.T) {
	_, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	h.send(t, rankRequest(1))
	r := h.read(t)
	if r.Op != "keyboard_ranked" || r.Activation != "editor" || r.Revision != 1 || r.Status != "ok" || r.RankResult == nil || r.RankResult.Candidates[0].ID != "c1" {
		t.Fatal(r)
	}
}
func TestKeyboardRankingInvalidationCancelsBackend(t *testing.T) {
	for _, mode := range []string{"freeze", "focus", "update", "disable", "close"} {
		t.Run(mode, func(t *testing.T) {
			started, cancelled := make(chan struct{}), make(chan struct{})
			m, h := rankingHost(t, func(ctx context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
				close(started)
				<-ctx.Done()
				close(cancelled)
				return rankAnswer(r), nil
			})
			h.send(t, rankRequest(1))
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("not started")
			}
			switch mode {
			case "freeze":
				h.send(t, message{Op: "keyboard_freeze", Activation: "editor", Revision: 2})
			case "focus":
				h.send(t, message{Op: "deactivate", Activation: "editor"})
			case "update":
				m.BeginUpdate()
			case "disable":
				if err := m.ConfigureKeyboardRanking(nil); err != nil {
					t.Fatal(err)
				}
			case "close":
				m.Close()
			}
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("inference not cancelled")
			}
		})
	}
}
func TestKeyboardRankingLatestRevisionWins(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	_, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		if r.Revision == 1 {
			close(started)
			<-release
		}
		return rankAnswer(r), nil
	})
	h.send(t, rankRequest(1))
	<-started
	h.send(t, rankRequest(3))
	h.send(t, rankRequest(2))
	close(release)
	if r := h.read(t); r.Revision != 3 || r.Status != "ok" {
		t.Fatal(r)
	}
}

func TestKeyboardRankingRejectsForeignActivationAndMalformedIdentity(t *testing.T) {
	var calls atomic.Int32
	_, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		calls.Add(1)
		return rankAnswer(r), nil
	})
	foreign := rankRequest(1)
	foreign.Activation = "foreign"
	h.send(t, foreign)
	malformed := rankRequest(2)
	malformed.RankRequest.Generation = 2
	h.send(t, malformed)
	h.send(t, rankRequest(3))
	if r := h.read(t); r.Revision != 3 || r.Status != "ok" || calls.Load() != 1 {
		t.Fatalf("unsafe admission: %+v, calls %d", r, calls.Load())
	}
}

func TestKeyboardRankingRejectsInventedCandidate(t *testing.T) {
	_, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		result := rankAnswer(r)
		result.Candidates[0].ID = "invented"
		return result, nil
	})
	h.send(t, rankRequest(1))
	if r := h.read(t); r.Status != "unavailable" || r.RankResult != nil {
		t.Fatal(r)
	}
}

func TestKeyboardRankingTwoLanesShareSnapshot(t *testing.T) {
	m, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	fast := keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	settled := keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		if r.Candidates[0].ID != "c0" {
			t.Error("settled saw reordered input")
		}
		out := rankAnswer(r)
		out.Candidates = []domain.RankedCandidate{{ID: "c0"}, {ID: "c1"}}
		return out, nil
	})
	if err := m.ConfigureKeyboardRankingStages(fast, settled); err != nil {
		t.Fatal(err)
	}
	h.send(t, rankRequest(1))
	if got := h.read(t); got.RankResult == nil || got.RankResult.Candidates[0].ID != "c1" {
		t.Fatal(got)
	}
	if got := h.read(t); got.RankResult == nil || got.RankResult.Candidates[0].ID != "c0" {
		t.Fatal(got)
	}
}

func TestKeyboardRankingSettledSuppressesLateFast(t *testing.T) {
	m, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	release := make(chan struct{})
	fast := keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		<-release
		return rankAnswer(r), nil
	})
	settled := keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	if err := m.ConfigureKeyboardRankingStages(fast, settled); err != nil {
		t.Fatal(err)
	}
	h.send(t, rankRequest(1))
	if got := h.read(t); got.Status != "ok" {
		t.Fatal(got)
	}
	close(release)
	h.send(t, rankRequest(2))
	if got := h.read(t); got.Revision != 2 {
		t.Fatal("late fast overwrote settled", got)
	}
}

func TestSettledAbstentionPreservesFast(t *testing.T) {
	m, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	settledDone := make(chan struct{}, 1)
	fast := keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	settled := keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		out := rankAnswer(r)
		out.Decision = "abstained_no_context"
		settledDone <- struct{}{}
		return out, nil
	})
	if err := m.ConfigureKeyboardRankingStages(fast, settled); err != nil {
		t.Fatal(err)
	}
	h.send(t, rankRequest(1))
	if got := h.read(t); got.Status != "ok" {
		t.Fatal(got)
	}
	<-settledDone
	h.send(t, rankRequest(2))
	if got := h.read(t); got.Revision != 2 {
		t.Fatal("abstention undid fast result", got)
	}
}

func TestKeyboardRankingRequestAfterNativeFreeze(t *testing.T) {
	_, h := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		return rankAnswer(r), nil
	})
	h.send(t, message{Op: "keyboard_freeze", Activation: "editor", Revision: 1})
	h.send(t, rankRequest(2))
	if got := h.read(t); got.Revision != 2 || got.Status != "ok" {
		t.Fatal("new native snapshot was discarded", got)
	}
}

func TestNativeRankingSkipDiagnostics(t *testing.T) {
	m, _ := rankingHost(t, func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		t.Error("skip must not invoke model")
		return rankAnswer(r), nil
	})
	var counts []int
	m.OnKeyboardRankingDiagnostic(func(reason string, count int) {
		if reason == "no_comparable_candidates" {
			counts = append(counts, count)
		}
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	one, invalid := 1, 2
	m.receiveRankingLocked(message{Op: "keyboard_rank_skipped", RankingReason: "no_comparable_candidates", Activation: "other", Revision: 1, EligibleCount: &one})
	m.receiveRankingLocked(message{Op: "keyboard_rank_skipped", RankingReason: "no_comparable_candidates", Activation: "editor", Revision: 1, EligibleCount: &one})
	m.receiveRankingLocked(message{Op: "keyboard_rank_skipped", RankingReason: "no_comparable_candidates", Activation: "editor", Revision: 1, EligibleCount: &one})
	m.receiveRankingLocked(message{Op: "keyboard_rank_skipped", RankingReason: "no_comparable_candidates", Activation: "editor", Revision: 2, EligibleCount: &invalid})
	m.receiveRankingLocked(message{Op: "keyboard_rank_skipped", RankingReason: "no_comparable_candidates", Activation: "editor", Revision: 3})
	m.receiveRankingLocked(message{Op: "keyboard_rank_skipped", RankingReason: "private input canary", Activation: "editor", Revision: 4, EligibleCount: &one})
	if len(counts) != 1 || counts[0] != 1 {
		t.Fatal(counts)
	}
}

func TestCompletionCapabilityIsNegotiated(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		m := New("", nil, nil)
		t.Cleanup(m.Close)
		m.RequireClientVerification(func(PeerIdentity, ClientHello) error { return nil })
		err := m.ConfigureKeyboardRanking(keyboardRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
			if r.AllowCompletion != enabled {
				t.Errorf("capability = %v", r.AllowCompletion)
			}
			if !enabled {
				return rankAnswer(r), nil
			}
			return contracts.KeyboardRankResult{Identity: r.Identity, Revision: r.Revision, Completion: &domain.KeyboardCompletion{Input: r.Input, Text: "重构一下", Reading: "chong gou yi xia"}}, nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		caps := []string{rankingCapability}
		if enabled {
			caps = append(caps, "keyboard-completion-v1")
		}
		alive := new(atomic.Bool)
		alive.Store(true)
		h, _ := verifiedHost(t, m, message{Op: "hello", Build: "completion-test", Capabilities: caps}, alive)
		wait(t, func() bool { return m.ConnectedBuild() != "" })
		h.send(t, message{Op: "activate", Activation: "editor"})
		wait(t, m.ActiveClient)
		req := rankRequest(1)
		req.RankRequest.AllowCompletion = true
		if enabled {
			req.RankRequest.Input = "chonggyxia"
			req.RankRequest.Candidates = req.RankRequest.Candidates[:1]
			req.RankRequest.Candidates[0].End = 10
		}
		h.send(t, req)
		reply := h.read(t)
		if reply.Status != "ok" || reply.RankResult == nil || (reply.RankResult.Completion != nil) != enabled {
			t.Fatal(reply)
		}
	}
}
