package imebridge

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"arion.local/core/domain"
)

const vocabularyCapability = "keyboard-vocabulary-v1"

var ErrVocabularyUnavailable = errors.New("keyboard vocabulary host unavailable or busy")
var ErrVocabularyRejected = errors.New("keyboard vocabulary snapshot rejected")

type vocabularyExchange struct {
	request uint64
	op      string
	reply   chan message
}

func (m *IME) invalidateVocabularyLocked() {
	if m.vocabulary != nil {
		select {
		case m.vocabulary.reply <- message{}:
		default:
		}
		m.vocabulary = nil
	}
}

func (m *IME) receiveVocabularyLocked(msg message) {
	x := m.vocabulary
	if x == nil || msg.Request != x.request || msg.Op != x.op || msg.Activation != m.activation {
		return
	}
	select {
	case x.reply <- msg:
	default:
	}
	// A duplicate response cannot become the next phase's acknowledgement.
	x.op = ""
}

// SyncKeyboardVocabulary publishes a complete approved snapshot, including an
// empty deletion snapshot. The caller retains desired state on any error and
// retries from a fresh persisted-revision query; no uncertain write is assumed
// lost. Native handlers must persist before acknowledging publication.
func (m *IME) SyncKeyboardVocabulary(ctx context.Context, terms []domain.KeyboardTerm) error {
	canonical, err := canonicalKeyboardTerms(terms)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := m.waitForClientVerification(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	if m.closed || m.updating || m.conn == nil || m.activation == "" || m.token != "" || m.vocabulary != nil || !m.vocabularyCapable {
		m.mu.Unlock()
		return ErrVocabularyUnavailable
	}
	if !m.clientVerified || !m.peer.Alive() {
		m.mu.Unlock()
		return domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
	}
	if m.vocabularyRequest == math.MaxUint64 {
		m.mu.Unlock()
		return ErrVocabularyUnavailable
	}
	m.vocabularyRequest++
	x := &vocabularyExchange{request: m.vocabularyRequest, reply: make(chan message, 1)}
	m.vocabulary = x
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		if m.vocabulary == x {
			m.vocabulary = nil
		}
		m.mu.Unlock()
	}()

	exchange := func(out message, expected string) (message, error) {
		m.mu.Lock()
		if m.vocabulary != x || !m.peer.Alive() || m.updating || m.token != "" {
			m.mu.Unlock()
			return message{}, changed()
		}
		x.op = expected
		out.Request = x.request
		err := m.sendLocked(ctx, out)
		m.mu.Unlock()
		if err != nil {
			return message{}, err
		}
		select {
		case <-ctx.Done():
			return message{}, ctx.Err()
		case reply := <-x.reply:
			if err := ctx.Err(); err != nil {
				return message{}, err
			}
			m.mu.Lock()
			valid := m.vocabulary == x && m.peer.Alive() && !m.updating && m.token == ""
			m.mu.Unlock()
			if !valid || reply.Op != expected {
				return message{}, changed()
			}
			if reply.Status == "busy" {
				return message{}, ErrVocabularyUnavailable
			}
			if reply.Status != "ok" {
				return message{}, ErrVocabularyRejected
			}
			return reply, nil
		}
	}
	state, err := exchange(message{Op: "vocabulary_query"}, "vocabulary_state")
	if err != nil {
		return err
	}
	if state.VocabularyRevision == math.MaxUint64 {
		return ErrVocabularyRejected
	}
	revision := state.VocabularyRevision + 1
	ack, err := exchange(message{Op: "vocabulary_publish", VocabularyRevision: revision, Terms: &canonical}, "vocabulary_published")
	if err != nil {
		return err
	}
	if ack.VocabularyRevision != revision {
		return ErrVocabularyRejected
	}
	return nil
}

func canonicalKeyboardTerms(terms []domain.KeyboardTerm) ([]domain.KeyboardTerm, error) {
	// Reuse the domain projection's validation and deterministic ordering.
	if len(terms) > domain.MaxMemoryEntries {
		return nil, domain.ErrInvalidKeyboardVocabulary
	}
	entries := make([]domain.MemoryEntry, len(terms))
	for i, term := range terms {
		if term.Reading == "" {
			return nil, domain.ErrInvalidKeyboardVocabulary
		}
		entries[i] = domain.MemoryEntry{ID: term.ID, To: term.Text, KeyboardPinyin: term.Reading, Origin: domain.OriginUser}
	}
	canonical, err := domain.KeyboardTerms(entries)
	if err != nil {
		return nil, err
	}
	// Bound the complete JSON frame before any private content is sent.
	frame, err := json.Marshal(message{Op: "vocabulary_publish", Terms: &canonical})
	if err != nil || len(frame) > (1<<20)-4096 {
		return nil, domain.ErrInvalidKeyboardVocabulary
	}
	return canonical, nil
}
