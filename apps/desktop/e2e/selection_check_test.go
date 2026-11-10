//go:build arion_e2e && (darwin || windows)

package e2e

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardos"
	"arion.local/desktop/infrastructure/vocabulary"
	"github.com/egoist/mygo"
)

func assertSelectionCopyHotwordURLAndSwitch(t *testing.T, selected string, reading application.SelectionReading) {
	t.Helper()
	if reading.Secure || strings.TrimSpace(reading.Text) == "" {
		t.Fatalf("reader did not return the selection: %+v", reading)
	}
	if reading.Text != selected {
		t.Fatalf("reader text %q", reading.Text)
	}
	var wrote string
	if err := application.CopySelection(reading, func(text string) error {
		var writeErr error
		mygo.RunOnMain(func() {
			_, writeErr = clipboardos.Write(domain.ClipTextPayload(text))
		})
		wrote = text
		return writeErr
	}); err != nil || wrote != selected {
		t.Fatalf("copy %q %v", wrote, err)
	}
	var payload domain.ClipPayload
	var readErr error
	mygo.RunOnMain(func() {
		payload, readErr = clipboardos.Read(1<<20, false)
	})
	if readErr != nil || payload.Text != selected {
		t.Fatalf("clipboard %q %v", payload.Text, readErr)
	}

	path := filepath.Join(t.TempDir(), "vocabulary.bin")
	key := bytes.Repeat([]byte{9}, 32)
	store, err := vocabulary.Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.AddSelectionHotword(reading, func(text string) error {
		_, addErr := store.AddHotword(text)
		return addErr
	}); err != nil {
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
		if entry.To == selected && entry.Hotword {
			found = true
		}
	}
	if !found {
		t.Fatalf("reopened profile missing hotword: %+v", entries)
	}

	prefs := application.DefaultSelectionPreferences()
	prefs.Enabled = true
	prefs.Clients[0].Enabled = true
	got, err := prefs.SelectionURL(application.SelectionActionExplain, selected, "")
	filled, fillErr := application.FillSelectionTemplate(application.SelectionTemplateClaude, strings.ReplaceAll(application.SelectionPromptExplain, "{{selection}}", selected))
	if err != nil || fillErr != nil || got != filled || !strings.HasPrefix(got, "claude://claude.ai/new?q=") {
		t.Fatalf("url %s %v %v", got, err, fillErr)
	}

	reads := 0
	_, ok, err := application.OfferAfterMouseUp(application.DefaultSelectionPreferences(), func() (application.SelectionReading, error) {
		reads++
		return reading, nil
	})
	if ok || err != nil || reads != 0 {
		t.Fatalf("switch off offered a selection: %v %v reads=%d", ok, err, reads)
	}
}

func assertSecureSelectionRefused(t *testing.T, reading application.SelectionReading) {
	t.Helper()
	if !reading.Secure || reading.Text != "" {
		t.Fatalf("secure read %+v", reading)
	}
	wrote := false
	if err := application.CopySelection(reading, func(string) error { wrote = true; return nil }); domain.CodeOf(err) != domain.CodeSecureFieldBlocked || wrote {
		t.Fatalf("secure copy wrote=%v %v", wrote, err)
	}
	path := filepath.Join(t.TempDir(), "vocabulary.bin")
	store, err := vocabulary.Open(path, bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	added := false
	if err := application.AddSelectionHotword(reading, func(string) error { added = true; return nil }); domain.CodeOf(err) != domain.CodeSecureFieldBlocked || added {
		t.Fatalf("secure hotword added=%v %v", added, err)
	}
	prefs := application.DefaultSelectionPreferences()
	prefs.Enabled = true
	_, ok, err := application.OfferAfterMouseUp(prefs, func() (application.SelectionReading, error) { return reading, nil })
	if ok || err != nil {
		t.Fatal("secure field produced an offer")
	}
	entries, err := store.List()
	if err != nil || len(entries) != 0 {
		t.Fatalf("secure field stored %+v %v", entries, err)
	}
}
