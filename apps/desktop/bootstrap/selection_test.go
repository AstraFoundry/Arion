package bootstrap

import (
	"bytes"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/vocabulary"
)

type selectionPlatform struct {
	Platform
	text       string
	secure     bool
	readErr    error
	reads      atomic.Int32
	started    int
	stopped    int
	sink       func()
	startError error
}

func (p *selectionPlatform) ReadSelection() (string, bool, error) {
	p.reads.Add(1)
	return p.text, p.secure, p.readErr
}
func (p *selectionPlatform) StartSelectionWatch(sink func()) error {
	p.started++
	p.sink = sink
	return p.startError
}
func (p *selectionPlatform) StopSelectionWatch() { p.stopped++ }

func TestSelectionWatchSwitchSecureAndURL(t *testing.T) {
	platform := &selectionPlatform{text: "SelectionTerm"}
	app := &App{platform: platform}
	app.prefs = application.DefaultPreferences("linux")
	app.prefs.Selection.Enabled = true
	app.syncSelectionWatch(false, false)
	if platform.started != 0 || platform.reads.Load() != 0 {
		t.Fatal("the watch started without a change")
	}
	app.syncSelectionWatch(false, true)
	if platform.started != 1 || platform.sink == nil {
		t.Fatal("the watch did not start")
	}
	platform.sink()
	waitSelection(t, func() bool {
		offer, ok := app.currentSelection()
		return platform.reads.Load() == 1 && ok && offer.Text == "SelectionTerm"
	})
	app.prefs.Selection.Enabled = false
	app.syncSelectionWatch(true, false)
	platform.sink()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if platform.reads.Load() != 1 {
			t.Fatalf("switch off read the selection: reads=%d", platform.reads.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := app.currentSelection(); platform.stopped != 1 || ok {
		t.Fatal("switch off kept the offer")
	}

	app.prefs.Selection.Enabled = true
	app.selectionOK = true
	app.selectionOffer = application.SelectionOffer{Text: "SelectionTerm"}
	platform.secure = true
	if err := (actions{app}).CopySelection(); domain.CodeOf(err) != domain.CodeSecureFieldBlocked {
		t.Fatalf("secure copy: %v", err)
	}
	var opened string
	app.openSelectionURL = func(raw string) error { opened = raw; return nil }
	app.prefs.Selection.Clients[0].Enabled = true
	platform.secure = false
	if err := (actions{app}).OpenSelection(application.SelectionActionExplain, ""); err != nil {
		t.Fatal(err)
	}
	want, err := app.prefs.Selection.SelectionURL(application.SelectionActionExplain, "SelectionTerm", "")
	if err != nil || opened != want || opened == "" {
		t.Fatalf("opened %s want %s (%v)", opened, want, err)
	}
	opened = ""
	if err := (actions{app}).OpenSelection("missing", ""); !errors.Is(err, application.ErrSelectionAction) || opened != "" {
		t.Fatalf("missing action opened %q: %v", opened, err)
	}
}

func TestSelectionVocabularyReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vocabulary.bin")
	key := bytes.Repeat([]byte{9}, 32)
	store, err := vocabulary.Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	platform := &selectionPlatform{secure: true, text: "secret"}
	app := &App{platform: platform, vocab: store, vocabPath: path}
	app.prefs.Selection.Enabled = true
	app.selectionOK = true
	app.selectionOffer = application.SelectionOffer{Text: "secret"}
	if err := (actions{app}).AddSelectionVocabulary(); domain.CodeOf(err) != domain.CodeSecureFieldBlocked {
		t.Fatalf("secure hotword: %v", err)
	}
	platform.secure = false
	app.selectionOffer.Text = "SelectionTerm"
	if err := (actions{app}).AddSelectionVocabulary(); err != nil {
		t.Fatal(err)
	}
	reopened, err := vocabulary.Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := reopened.List()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.To == "SelectionTerm" && entry.Hotword {
			found = true
		}
		if entry.To == "secret" {
			t.Fatal("secure field was stored")
		}
	}
	if !found {
		t.Fatalf("reopened profile missing hotword: %+v", entries)
	}
}

func waitSelection(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("selection read did not finish")
}
