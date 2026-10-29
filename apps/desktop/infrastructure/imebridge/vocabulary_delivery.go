package imebridge

import (
	"context"
	"slices"
	"time"

	"arion.local/core/domain"
)

type vocabularySnapshot struct{ terms []domain.KeyboardTerm }

// All delivery state is guarded by the bridge mutex. There is one worker and
// one replaceable desired snapshot, not one goroutine per vocabulary edit.
type vocabularyDelivery struct {
	desired       *vocabularySnapshot
	stop          context.CancelFunc
	cancelAttempt context.CancelFunc
	done          chan struct{}
	report        func(error)
	observer      uint64
}

// SetKeyboardVocabulary queues the latest validated full snapshot without
// blocking the UI on native composition or connection readiness. Report receives
// changes in delivery outcome off the caller's thread and must not block.
func (m *IME) SetKeyboardVocabulary(terms []domain.KeyboardTerm, report func(error)) error {
	return m.setKeyboardVocabulary(terms, report, time.Second)
}

func (m *IME) setKeyboardVocabulary(terms []domain.KeyboardTerm, report func(error), retry time.Duration) error {
	canonical, err := canonicalKeyboardTerms(terms)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrVocabularyUnavailable
	}
	if m.vocabularyDelivery == nil {
		ctx, stop := context.WithCancel(context.Background())
		m.vocabularyDelivery = &vocabularyDelivery{stop: stop, done: make(chan struct{})}
		go m.deliverKeyboardVocabulary(ctx, m.vocabularyDelivery, retry)
	}
	d := m.vocabularyDelivery
	d.report = report
	d.observer++
	m.notifyLocked()
	if d.desired != nil && slices.Equal(d.desired.terms, canonical) {
		return nil
	}
	d.desired = &vocabularySnapshot{terms: canonical}
	if d.cancelAttempt != nil {
		d.cancelAttempt()
	}
	m.notifyLocked()
	return nil
}

func (m *IME) deliverKeyboardVocabulary(ctx context.Context, d *vocabularyDelivery, retry time.Duration) {
	defer close(d.done)
	var delivered *vocabularySnapshot
	var admission uint64
	var activation string
	var previousError string
	var observedSnapshot *vocabularySnapshot
	var observedObserver uint64
	for {
		m.mu.Lock()
		if m.closed || ctx.Err() != nil {
			m.mu.Unlock()
			return
		}
		snapshot, changed := d.desired, m.changed
		currentAdmission, currentActivation := m.publishedAdmission, m.activation
		ready := snapshot != nil && m.conn != nil && m.clientVerified && m.vocabularyCapable && m.activation != "" && !m.updating && m.token == ""
		needed := snapshot != delivered || admission != currentAdmission || activation != currentActivation
		if ready && !needed && d.observer != observedObserver {
			report := d.report
			observedObserver = d.observer
			m.mu.Unlock()
			if report != nil {
				report(nil)
			}
			continue
		}
		if !ready || !needed {
			m.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-changed:
			}
			continue
		}
		attempt, cancel := context.WithCancel(ctx)
		d.cancelAttempt = cancel
		m.mu.Unlock()
		err := m.SyncKeyboardVocabulary(attempt, snapshot.terms)
		cancel()
		m.mu.Lock()
		d.cancelAttempt = nil
		if m.closed || ctx.Err() != nil {
			m.mu.Unlock()
			return
		}
		if d.desired != snapshot {
			m.mu.Unlock()
			continue
		}
		if err == nil {
			delivered = snapshot
			admission = currentAdmission
			activation = currentActivation
		}
		report := d.report
		observer := d.observer
		changed = m.changed
		m.mu.Unlock()
		message := ""
		if err != nil {
			message = err.Error()
		}
		if snapshot != observedSnapshot || message != previousError || observer != observedObserver {
			observedSnapshot = snapshot
			previousError = message
			observedObserver = observer
			if report != nil {
				report(err)
			}
		}
		if err == nil {
			continue
		}
		timer := time.NewTimer(retry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-changed:
			timer.Stop()
		case <-timer.C:
		}
	}
}
