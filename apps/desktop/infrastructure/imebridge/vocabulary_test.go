package imebridge

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func vocabularyHost(t *testing.T) (*IME, *host) {
	t.Helper()
	m := New("", nil, nil)
	t.Cleanup(m.Close)
	m.RequireClientVerification(func(PeerIdentity, ClientHello) error { return nil })
	alive := new(atomic.Bool)
	alive.Store(true)
	h, _ := verifiedHost(t, m, message{Op: "hello", Build: "vocabulary-test", Capabilities: []string{vocabularyCapability}}, alive)
	wait(t, func() bool { return m.ConnectedBuild() != "" })
	h.send(t, message{Op: "activate", Activation: "editor"})
	wait(t, m.ActiveClient)
	return m, h
}

func syncVocabulary(m *IME, ctx context.Context, terms []domain.KeyboardTerm) <-chan error {
	done := make(chan error, 1)
	go func() { done <- m.SyncKeyboardVocabulary(ctx, terms) }()
	return done
}

func vocabularyResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("vocabulary exchange did not finish")
		return nil
	}
}

func TestVocabularyCompleteSnapshotAndDeletion(t *testing.T) {
	m, h := vocabularyHost(t)
	for _, terms := range [][]domain.KeyboardTerm{{{ID: "term", Text: "星槎", Reading: "xing cha"}}, {}} {
		done := syncVocabulary(m, context.Background(), terms)
		query := h.read(t)
		if query.Op != "vocabulary_query" || query.Terms != nil {
			t.Fatal("private snapshot preceded revision query")
		}
		// Wrong request and activation cannot advance the exchange.
		h.send(t, message{Op: "vocabulary_state", Activation: "old-editor", Request: query.Request, Status: "ok", VocabularyRevision: 99})
		h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request + 1, Status: "ok", VocabularyRevision: 99})
		h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request, Status: "ok", VocabularyRevision: 7})
		publish := h.read(t)
		if publish.Op != "vocabulary_publish" || publish.VocabularyRevision != 8 || publish.Terms == nil || len(*publish.Terms) != len(terms) {
			t.Fatalf("bad complete publication: %+v", publish)
		}
		if len(terms) > 0 && (*publish.Terms)[0] != terms[0] {
			t.Fatal("term changed in transport")
		}
		h.send(t, message{Op: "vocabulary_published", Activation: "editor", Request: publish.Request, Status: "ok", VocabularyRevision: 8})
		if err := vocabularyResult(t, done); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVocabularyRejectsUnsafeAdmission(t *testing.T) {
	for _, mode := range []string{"unverified", "unsupported", "voice", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			m, _ := vocabularyHost(t)
			terms := []domain.KeyboardTerm{{ID: "term", Text: "星槎", Reading: "xing cha"}}
			m.mu.Lock()
			switch mode {
			case "unverified":
				m.clientVerified = false
			case "unsupported":
				m.vocabularyCapable = false
			case "voice":
				m.token = "owned"
			case "invalid":
				terms[0].Reading = ""
			}
			m.mu.Unlock()
			if err := m.SyncKeyboardVocabulary(context.Background(), terms); err == nil {
				t.Fatal("unsafe publication accepted")
			}
		})
	}
}

func TestVocabularyInvalidationAndRetry(t *testing.T) {
	for _, mode := range []string{"focus", "disconnect", "update", "cancel", "busy", "bad-ack"} {
		t.Run(mode, func(t *testing.T) {
			m, h := vocabularyHost(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := syncVocabulary(m, ctx, nil)
			query := h.read(t)
			if err := m.SyncKeyboardVocabulary(ctx, nil); !errors.Is(err, ErrVocabularyUnavailable) {
				t.Fatal("overlapping publication allowed")
			}
			switch mode {
			case "focus":
				h.send(t, message{Op: "deactivate", Activation: "editor"})
			case "disconnect":
				h.conn.Close()
			case "update":
				m.BeginUpdate()
			case "cancel":
				cancel()
			case "busy":
				h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request, Status: "busy"})
			case "bad-ack":
				h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request, Status: "ok", VocabularyRevision: 4})
				publish := h.read(t)
				h.send(t, message{Op: "vocabulary_published", Activation: "editor", Request: publish.Request, Status: "ok", VocabularyRevision: 4})
			}
			if err := vocabularyResult(t, done); err == nil {
				t.Fatal("invalidated publication succeeded")
			}
			m.mu.Lock()
			pending := m.vocabulary != nil
			m.mu.Unlock()
			if pending {
				t.Fatal("failed exchange retained transaction")
			}
			if mode != "busy" {
				return
			}
			// A retry queries actual persisted state rather than replaying a guessed revision.
			done = syncVocabulary(m, ctx, nil)
			retry := h.read(t)
			if retry.Request == query.Request || retry.Op != "vocabulary_query" {
				t.Fatal("retry reused old transaction")
			}
			cancel()
			if err := vocabularyResult(t, done); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestVocabularyLostAcknowledgementQueriesPersistedRevision(t *testing.T) {
	m, h := vocabularyHost(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := syncVocabulary(m, ctx, nil)
	query := h.read(t)
	h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request, Status: "ok", VocabularyRevision: 10})
	publish := h.read(t)
	if publish.VocabularyRevision != 11 {
		t.Fatal("incorrect publication revision")
	}
	// The host persists 11 but the caller cancels before its reply arrives.
	cancel()
	if err := vocabularyResult(t, done); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	done = syncVocabulary(m, context.Background(), nil)
	retry := h.read(t)
	h.send(t, message{Op: "vocabulary_published", Activation: "editor", Request: publish.Request, Status: "ok", VocabularyRevision: 11})
	h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: retry.Request, Status: "ok", VocabularyRevision: 11})
	next := h.read(t)
	if next.VocabularyRevision != 12 {
		t.Fatal("retry guessed an obsolete revision")
	}
	h.send(t, message{Op: "vocabulary_published", Activation: "editor", Request: next.Request, Status: "ok", VocabularyRevision: 12})
	if err := vocabularyResult(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestVocabularyConnectionReplacementRevokesSameActivation(t *testing.T) {
	m, h := vocabularyHost(t)
	done := syncVocabulary(m, context.Background(), nil)
	h.read(t)
	alive := new(atomic.Bool)
	alive.Store(true)
	next, _ := verifiedHost(t, m, message{Op: "hello", Build: "replacement", Capabilities: []string{vocabularyCapability}}, alive)
	wait(t, func() bool { return m.ConnectedBuild() == "replacement" })
	next.send(t, message{Op: "activate", Activation: "editor"})
	if err := vocabularyResult(t, done); err == nil {
		t.Fatal("old transaction survived connection replacement")
	}
}

func TestVoiceCaptureInvalidatesVocabularyExchange(t *testing.T) {
	m, h := vocabularyHost(t)
	done := syncVocabulary(m, context.Background(), nil)
	h.read(t)
	captured := make(chan error, 1)
	go func() {
		_, err := m.Capture(context.Background(), contracts.Identity{SessionID: "dictation"})
		captured <- err
	}()
	if begin := h.read(t); begin.Op != "begin" || begin.Session != "dictation" {
		t.Fatal("voice capture was blocked by vocabulary")
	}
	if err := vocabularyResult(t, captured); err != nil {
		t.Fatal(err)
	}
	if err := vocabularyResult(t, done); err == nil {
		t.Fatal("vocabulary survived voice ownership")
	}
}
