package application

import (
	"context"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type keyboardRankFunc func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error)

func (f keyboardRankFunc) RankKeyboard(c context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	return f(c, r)
}
func keyRequest(rev uint64) contracts.KeyboardRankRequest {
	return contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "keyboard", Generation: 1}, Revision: rev, Input: "shi", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "是", Source: "rime"}}, {Candidate: domain.Candidate{ID: "c1", Text: "时", Source: "rime"}}}}
}
func keyResult(req contracts.KeyboardRankRequest) contracts.KeyboardRankResult {
	return contracts.KeyboardRankResult{Identity: req.Identity, Revision: req.Revision, Candidates: []domain.RankedCandidate{{ID: "c1"}, {ID: "c0"}}}
}
func receiveRank(t *testing.T, ch <-chan KeyboardRankOutcome) KeyboardRankOutcome {
	t.Helper()
	select {
	case out := <-ch:
		return out
	case <-time.After(time.Second):
		t.Fatal("rank outcome timeout")
		return KeyboardRankOutcome{}
	}
}
func TestKeyboardRankingLatestPendingAndFreeze(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	calls := make(chan uint64, 4)
	f := keyboardRankFunc(func(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		calls <- req.Revision
		if req.Revision == 1 {
			close(started)
			<-release
		} // Simulate a backend ignoring cancellation.
		return keyResult(req), nil
	})
	s, err := NewKeyboardRanking(f, time.Second, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := s.Submit(keyRequest(1))
	<-started
	b := s.Submit(keyRequest(2))
	c := s.Submit(keyRequest(3))
	close(release)
	if receiveRank(t, b).Err == nil {
		t.Fatal("superseded pending work succeeded")
	}
	if receiveRank(t, a).Err == nil {
		t.Fatal("uncancellable stale backend applied")
	}
	out := receiveRank(t, c)
	if out.Err != nil || !s.CanApply(out) {
		t.Fatal(out)
	}
	if <-calls != 1 || <-calls != 3 {
		t.Fatal("pending queue did not coalesce")
	}
	s.Freeze()
	if s.CanApply(out) {
		t.Fatal("navigation did not freeze ranking")
	}
	req := keyRequest(3)
	req.Generation++
	next := receiveRank(t, s.Submit(req))
	if next.Err != nil || !s.CanApply(next) || s.CanApply(out) {
		t.Fatal("generation identity failed")
	}
}
func TestKeyboardRankingDeadlineAndIdentityValidation(t *testing.T) {
	f := keyboardRankFunc(func(ctx context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		<-ctx.Done()
		return keyResult(r), nil
	})
	s, err := NewKeyboardRanking(f, 15*time.Millisecond, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := receiveRank(t, s.Submit(keyRequest(1)))
	if out.Err == nil || s.CanApply(out) {
		t.Fatal("deadline result applied")
	}
	s.Close()
	if receiveRank(t, s.Submit(keyRequest(2))).Err == nil {
		t.Fatal("closed scheduler accepted work")
	}
	f = func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
		out := keyResult(r)
		out.Revision++
		return out, nil
	}
	s, err = NewKeyboardRanking(f, time.Second, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if receiveRank(t, s.Submit(keyRequest(1))).Err == nil {
		t.Fatal("stale result identity accepted")
	}
}
