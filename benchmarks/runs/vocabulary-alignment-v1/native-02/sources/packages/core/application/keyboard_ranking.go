package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// KeyboardRankOutcome is a proposal. The UI must check CanApply on its own
// event loop immediately before updating a still-owned candidate menu.
type KeyboardRankOutcome struct {
	Ticket uint64
	Result contracts.KeyboardRankResult
	Err    error
}

type keyboardRankWork struct {
	ticket  uint64
	request contracts.KeyboardRankRequest
	answer  chan KeyboardRankOutcome
}

// KeyboardRanking owns one running request and one replaceable pending slot.
// Debounce and deadline are explicit configuration, not measured defaults.
type KeyboardRanking struct {
	mu               sync.Mutex
	ranker           ports.KeyboardRanker
	budget, debounce time.Duration
	wake             chan struct{}
	done             chan struct{}
	pending          *keyboardRankWork
	cancel           context.CancelFunc
	current          contracts.KeyboardRankRequest
	ticket           uint64
	closed, frozen   bool
}

func NewKeyboardRanking(ranker ports.KeyboardRanker, budget, debounce time.Duration) (*KeyboardRanking, error) {
	if ranker == nil || budget <= 0 || debounce < 0 {
		return nil, errors.New("invalid keyboard ranking configuration")
	}
	s := &KeyboardRanking{ranker: ranker, budget: budget, debounce: debounce, wake: make(chan struct{}, 1), done: make(chan struct{})}
	go s.run()
	return s, nil
}

func (s *KeyboardRanking) invalidateLocked() {
	s.ticket++
	if s.cancel != nil {
		s.cancel()
	}
	if s.pending != nil {
		s.pending.answer <- KeyboardRankOutcome{Ticket: s.pending.ticket, Err: context.Canceled}
		close(s.pending.answer)
		s.pending = nil
	}
}

// Submit copies the exact snapshot. A new composition update revokes old work.
func (s *KeyboardRanking) Submit(req contracts.KeyboardRankRequest) <-chan KeyboardRankOutcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	answer := make(chan KeyboardRankOutcome, 1)
	if s.closed {
		answer <- KeyboardRankOutcome{Err: errors.New("keyboard ranker closed")}
		close(answer)
		return answer
	}
	s.invalidateLocked()
	req.Candidates = append([]domain.KeyboardCandidate(nil), req.Candidates...)
	s.current = req
	s.frozen = false
	s.pending = &keyboardRankWork{ticket: s.ticket, request: req, answer: answer}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return answer
}

// Freeze is used on navigation, commit, cancel, focus/mode change and voice
// takeover. Only a subsequent explicit composition update may submit again.
func (s *KeyboardRanking) Freeze() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidateLocked()
	s.frozen = true
}

func (s *KeyboardRanking) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.invalidateLocked()
	s.closed = true
	close(s.done)
}

func (s *KeyboardRanking) CanApply(out KeyboardRankOutcome) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.canApplyLocked(out)
}

func (s *KeyboardRanking) canApplyLocked(out KeyboardRankOutcome) bool {
	if s.closed || s.frozen || out.Err != nil || out.Ticket != s.ticket || out.Result.Identity != s.current.Identity || out.Result.Revision != s.current.Revision {
		return false
	}
	plain := make([]domain.Candidate, len(s.current.Candidates))
	for i, c := range s.current.Candidates {
		plain[i] = c.Candidate
	}
	if domain.ValidateRanking(plain, out.Result.Candidates) != nil {
		return false
	}
	for _, c := range s.current.Candidates {
		if c.ID == out.Result.Candidates[0].ID {
			return domain.KeyboardRankEligible(s.current.Candidates[0], c)
		}
	}
	return false
}

func (s *KeyboardRanking) run() {
	for {
		select {
		case <-s.done:
			return
		case <-s.wake:
		}
		for {
			s.mu.Lock()
			work := s.pending
			s.pending = nil
			if work == nil || s.closed {
				s.mu.Unlock()
				break
			}
			ctx, cancel := context.WithTimeout(context.Background(), s.budget)
			s.cancel = cancel
			s.mu.Unlock()
			out := KeyboardRankOutcome{Ticket: work.ticket}
			timer := time.NewTimer(s.debounce)
			select {
			case <-ctx.Done():
				out.Err = ctx.Err()
			case <-timer.C:
				out.Result, out.Err = s.ranker.RankKeyboard(ctx, work.request)
			}
			timer.Stop()
			if ctx.Err() != nil {
				out.Err = ctx.Err()
			}
			cancel()
			s.mu.Lock()
			if out.Err == nil && !s.canApplyLocked(out) {
				out.Err = errors.New("stale or invalid keyboard ranking")
			}
			work.answer <- out
			close(work.answer)
			s.cancel = nil
			s.mu.Unlock()
		}
	}
}
