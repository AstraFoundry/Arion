package imebridge

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func acknowledgeVocabulary(t *testing.T, h *host, query message, revision uint64) message {
	t.Helper()
	if query.Op != "vocabulary_query" {
		t.Fatalf("expected state query, got %s", query.Op)
	}
	h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request, Status: "ok", VocabularyRevision: revision})
	publish := h.read(t)
	if publish.Op != "vocabulary_publish" || publish.VocabularyRevision != revision+1 {
		t.Fatalf("unexpected publication: %+v", publish)
	}
	h.send(t, message{Op: "vocabulary_published", Activation: "editor", Request: publish.Request, Status: "ok", VocabularyRevision: revision + 1})
	return publish
}

func TestVocabularyDeliveryResumesAfterVoiceAndUpdate(t *testing.T) {
	for _, reason := range []string{"voice", "update"} {
		t.Run(reason, func(t *testing.T) {
			m, h := vocabularyHost(t)
			id := contracts.Identity{SessionID: "voice"}
			var snapshot contracts.TargetSnapshot
			if reason == "voice" {
				captured := make(chan error, 1)
				go func() { var err error; snapshot, err = m.Capture(context.Background(), id); captured <- err }()
				if h.read(t).Op != "begin" {
					t.Fatal("voice capture did not begin")
				}
				if err := vocabularyResult(t, captured); err != nil {
					t.Fatal(err)
				}
			} else {
				m.BeginUpdate()
			}
			if err := m.SetKeyboardVocabulary(nil, nil); err != nil {
				t.Fatal(err)
			}
			if reason == "voice" {
				cancelled := make(chan error, 1)
				go func() { cancelled <- m.Cancel(context.Background(), id, snapshot.Target) }()
				if h.read(t).Op != "cancel" {
					t.Fatal("queued vocabulary displaced voice cancellation")
				}
				if err := vocabularyResult(t, cancelled); err != nil {
					t.Fatal(err)
				}
			} else {
				m.EndUpdate()
			}
			acknowledgeVocabulary(t, h, h.read(t), 0)
		})
	}
}

func TestVocabularyDeliveryCoalescesAndStops(t *testing.T) {
	m, h := vocabularyHost(t)
	outcomes := make(chan error, 4)
	report := func(err error) { outcomes <- err }
	terms := []domain.KeyboardTerm{{ID: "one", Text: "星槎", Reading: "xing cha"}}
	if err := m.setKeyboardVocabulary(terms, report, 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	old := h.read(t)
	// Replace the pending creation with deletion before the state query replies.
	if err := m.setKeyboardVocabulary(nil, report, 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: old.Request, Status: "ok", VocabularyRevision: 80})
	query := h.read(t)
	if query.Request == old.Request {
		t.Fatal("new desired snapshot reused cancelled request")
	}
	publish := acknowledgeVocabulary(t, h, query, 9)
	if publish.Terms == nil || len(*publish.Terms) != 0 {
		t.Fatal("superseded private term was published")
	}
	if err := vocabularyResult(t, outcomes); err != nil {
		t.Fatal(err)
	}
	if err := m.SetKeyboardVocabulary(nil, report); err != nil {
		t.Fatal(err)
	}
	h.conn.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	if _, err := h.input.Peek(1); err == nil {
		t.Fatal("identical snapshot caused another write")
	} else if e, ok := err.(net.Error); !ok || !e.Timeout() {
		t.Fatal(err)
	}
	h.conn.SetReadDeadline(time.Time{})
	m.mu.Lock()
	done := m.vocabularyDelivery.done
	m.mu.Unlock()
	m.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("delivery worker survived close")
	}
	if err := m.SetKeyboardVocabulary(terms, report); err == nil {
		t.Fatal("closed bridge accepted new desired state")
	}
}

func TestVocabularyDeliveryRetriesBusyAndReconnects(t *testing.T) {
	m, h := vocabularyHost(t)
	outcomes := make(chan error, 8)
	terms := []domain.KeyboardTerm{{ID: "one", Text: "星槎", Reading: "xing cha"}}
	if err := m.setKeyboardVocabulary(terms, func(err error) { outcomes <- err }, 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	query := h.read(t)
	h.send(t, message{Op: "vocabulary_state", Activation: "editor", Request: query.Request, Status: "busy"})
	if err := vocabularyResult(t, outcomes); !errors.Is(err, ErrVocabularyUnavailable) {
		t.Fatal(err)
	}
	publish := acknowledgeVocabulary(t, h, h.read(t), 0)
	if publish.Terms == nil || len(*publish.Terms) != 1 || (*publish.Terms)[0] != terms[0] {
		t.Fatal("busy retry lost desired term")
	}
	if err := vocabularyResult(t, outcomes); err != nil {
		t.Fatal(err)
	}
	alive := new(atomic.Bool)
	alive.Store(true)
	next, _ := verifiedHost(t, m, message{Op: "hello", Build: "replacement", Capabilities: []string{vocabularyCapability}}, alive)
	wait(t, func() bool { return m.ConnectedBuild() == "replacement" })
	next.send(t, message{Op: "activate", Activation: "editor"})
	publish = acknowledgeVocabulary(t, next, next.read(t), 1)
	if publish.Terms == nil || len(*publish.Terms) != 1 || (*publish.Terms)[0] != terms[0] {
		t.Fatal("reconnected host did not receive current snapshot")
	}
}

func TestVocabularyDeliveryReportsEachAcknowledgedSnapshot(t *testing.T) {
	m, h := vocabularyHost(t)
	outcomes := make(chan error, 4)
	for i, terms := range [][]domain.KeyboardTerm{
		{{ID: "one", Text: "星槎", Reading: "xing cha"}},
		nil,
	} {
		if err := m.SetKeyboardVocabulary(terms, func(err error) { outcomes <- err }); err != nil {
			t.Fatal(err)
		}
		acknowledgeVocabulary(t, h, h.read(t), uint64(i))
		if err := vocabularyResult(t, outcomes); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVocabularyDeliveryReplaysAcknowledgementAfterRejectedEdit(t *testing.T) {
	m, h := vocabularyHost(t)
	terms := []domain.KeyboardTerm{{ID: "one", Text: "星槎", Reading: "xing cha"}}
	first := make(chan error, 1)
	if err := m.SetKeyboardVocabulary(terms, func(err error) { first <- err }); err != nil {
		t.Fatal(err)
	}
	acknowledgeVocabulary(t, h, h.read(t), 0)
	if err := vocabularyResult(t, first); err != nil {
		t.Fatal(err)
	}
	if err := m.SetKeyboardVocabulary([]domain.KeyboardTerm{{ID: "invalid"}}, nil); err == nil {
		t.Fatal("invalid edit was accepted")
	}
	restored := make(chan error, 1)
	if err := m.SetKeyboardVocabulary(terms, func(err error) { restored <- err }); err != nil {
		t.Fatal(err)
	}
	if err := vocabularyResult(t, restored); err != nil {
		t.Fatal(err)
	}
	h.conn.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	if _, err := h.input.Peek(1); err == nil {
		t.Fatal("restoring an acknowledged snapshot caused another write")
	} else if e, ok := err.(net.Error); !ok || !e.Timeout() {
		t.Fatal(err)
	}
	h.conn.SetReadDeadline(time.Time{})
}
