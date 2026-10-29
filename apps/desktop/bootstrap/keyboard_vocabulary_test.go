package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
	"arion.local/desktop/infrastructure/vocabulary"
	"arion.local/desktop/ui/shared/copy"
)

type vocabularyReceiver struct {
	InputMethod
	snapshots [][]domain.KeyboardTerm
	reports   []func(error)
}

func (r *vocabularyReceiver) SetKeyboardVocabulary(terms []domain.KeyboardTerm, report func(error)) error {
	r.snapshots = append(r.snapshots, append([]domain.KeyboardTerm(nil), terms...))
	r.reports = append(r.reports, report)
	return nil
}

func TestKeyboardVocabularyNoticeTracksAcknowledgementAndRejectsStaleResults(t *testing.T) {
	receiver := &vocabularyReceiver{}
	a := &App{ime: receiver}
	flush := func() {
		a.queueMu.Lock()
		batch := a.queue
		a.queue = nil
		a.queueMu.Unlock()
		for _, fn := range batch {
			fn()
		}
	}
	entries := []vocabulary.Entry{{ID: "one", To: "星槎", Approved: true, Origin: string(domain.OriginUser), KeyboardPinyin: "xing cha"}}
	a.queueKeyboardVocabulary(entries)
	if a.state.VocabularyKeyboardNotice != copy.VocabKeyboardPending {
		t.Fatal("local save claimed native delivery")
	}
	receiver.reports[0](imebridge.ErrVocabularyRejected)
	flush()
	if a.state.VocabularyKeyboardNotice != copy.VocabKeyboardFailed {
		t.Fatal("native rejection was hidden")
	}
	a.queueKeyboardVocabulary(entries)
	if len(receiver.snapshots) != 1 || a.state.VocabularyKeyboardNotice != copy.VocabKeyboardFailed {
		t.Fatal("unchanged reload cleared the failure")
	}
	// A queued old success must not clear a subsequent deletion's pending state.
	receiver.reports[0](nil)
	a.queueKeyboardVocabulary(nil)
	flush()
	if a.state.VocabularyKeyboardNotice != copy.VocabKeyboardPending {
		t.Fatal("stale acknowledgement hid pending deletion")
	}
	receiver.reports[1](imebridge.ErrVocabularyUnavailable)
	flush()
	if a.state.VocabularyKeyboardNotice != copy.VocabKeyboardPending {
		t.Fatal("busy host lost pending state")
	}
	receiver.reports[1](nil)
	flush()
	if a.state.VocabularyKeyboardNotice != "" {
		t.Fatal("current native acknowledgement did not clear notice")
	}
}

func TestEmptyInitialKeyboardVocabularyHasNoNotice(t *testing.T) {
	receiver := &vocabularyReceiver{}
	a := &App{ime: receiver}
	a.queueKeyboardVocabulary(nil)
	if a.state.VocabularyKeyboardNotice != "" {
		t.Fatal("empty startup displayed an unrelated keyboard notice")
	}
}

func TestVocabularyLoadsQueueApprovedReadingsAndDeletion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vocabulary.bin")
	store, err := vocabulary.Open(path, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	term, err := store.AddHotword("星槎工坊")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.SetKeyboardPinyin(term.ID, term.To, "xing cha gong fang"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.AddHotword("语音专用"); err != nil {
		t.Fatal(err)
	}
	receiver := &vocabularyReceiver{}
	a := &App{ime: receiver, vocab: store, vocabPath: path}
	a.loadVocabulary(false)
	if len(receiver.snapshots) != 1 || len(receiver.snapshots[0]) != 1 || receiver.snapshots[0][0].Text != term.To {
		t.Fatalf("incorrect keyboard projection: %v", receiver.snapshots)
	}
	if err := store.Remove(term.ID); err != nil {
		t.Fatal(err)
	}
	a.loadVocabulary(false)
	if len(receiver.snapshots) != 2 || len(receiver.snapshots[1]) != 0 {
		t.Fatal("term removal did not queue complete deletion")
	}
	// A storage failure must never masquerade as an empty approved vocabulary.
	if err := os.WriteFile(path, []byte("corrupt authenticated storage"), 0600); err != nil {
		t.Fatal(err)
	}
	a.loadVocabulary(false)
	if len(receiver.snapshots) != 2 {
		t.Fatal("failed load replaced desired keyboard state")
	}
}

func TestAbsentVocabularyDoesNotEraseOfflineCache(t *testing.T) {
	receiver := &vocabularyReceiver{}
	a := &App{ime: receiver, vocabPath: filepath.Join(t.TempDir(), "absent.bin")}
	a.loadVocabulary(false)
	if len(receiver.snapshots) != 0 {
		t.Fatal("unknown absent app state erased native vocabulary")
	}
}
