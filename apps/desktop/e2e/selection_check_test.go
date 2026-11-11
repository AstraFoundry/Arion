//go:build arion_e2e && (darwin || windows)

package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardos"
	"arion.local/desktop/infrastructure/vocabulary"
	uiapp "arion.local/desktop/ui/app"
	arion "arion.local/desktop/ui/shared/runtime"
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
	// clipboardos.Read is history capture, which refuses Arion's own tagged copies.
	var clipboard string
	mygo.RunOnMain(func() { clipboard = mygo.Clipboard.ReadText() })
	if clipboard != selected {
		t.Fatalf("clipboard %q", clipboard)
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
	got, err := prefs.SelectionURL(application.SelectionActionExplain, "", selected, "")
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

func selectionActionIDs() []string {
	return []string{
		application.SelectionActionTranslate,
		application.SelectionActionExplain,
		application.SelectionActionSummarize,
		application.SelectionActionRewrite,
		application.SelectionActionAsk,
		application.SelectionActionAskChatGPT,
		application.SelectionActionCopy,
		application.SelectionActionVocabulary,
	}
}

func setSelectionEnabled(t *testing.T, on bool) {
	t.Helper()
	app.Update(func(*arion.State) {
		app.Store().Actions.UpdatePreferences(func(p *arion.Preferences) {
			p.Selection.Enabled = on
			if !on {
				return
			}
			clients := append([]application.SelectionClient(nil), p.Selection.Clients...)
			if len(clients) == 0 {
				clients = application.DefaultSelectionPreferences().Clients
			}
			for i := range clients {
				clients[i].Enabled = true
			}
			p.Selection.Clients = clients
		})
	})
}

// exerciseSelectionPopover opens the production popover for selected and runs
// copy and vocabulary through the actions the menu calls. It does not launch a client.
func exerciseSelectionPopover(t *testing.T, selected string) {
	t.Helper()
	setSelectionEnabled(t, true)
	t.Cleanup(func() { setSelectionEnabled(t, false) })
	app.NotifySelectionMouseUp()
	window := app.Windows()["selection"]
	if window == nil || !window.IsVisible() || window.Page() != nil {
		t.Fatal("selection popover was not shown")
	}
	var text string
	var ids []string
	var height int
	app.Update(func(*arion.State) {
		text = app.Store().State.Selection.Text
		height = uiapp.SelectionHeightFor(app.Store())
		for _, action := range app.Store().State.Selection.Actions {
			ids = append(ids, action.ID)
		}
	})
	if text != selected || !sameIDs(ids, selectionActionIDs()) {
		t.Fatalf("popover %q %v", text, ids)
	}
	if width, got := window.ContentSize(); width != uiapp.SelectionWidth || got != height {
		t.Fatalf("popover size %d,%d", width, got)
	}
	var copyErr error
	app.Update(func(*arion.State) { copyErr = app.Store().Actions.CopySelection() })
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	var clipboard string
	mygo.RunOnMain(func() { clipboard = mygo.Clipboard.ReadText() })
	if clipboard != selected {
		t.Fatalf("clipboard %q", clipboard)
	}
	var addErr error
	app.Update(func(*arion.State) { addErr = app.Store().Actions.AddSelectionVocabulary() })
	if addErr != nil {
		t.Fatal(addErr)
	}
	entries, err := app.ReopenedVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	var id string
	found := false
	for _, entry := range entries {
		if entry.To == selected && entry.Hotword {
			found = true
			id = entry.ID
		}
	}
	if !found {
		t.Fatalf("reopened profile missing hotword: %+v", entries)
	}
	t.Cleanup(func() {
		app.Update(func(*arion.State) {
			if id != "" {
				app.Store().Actions.RemoveReplacement(id)
			}
		})
	})
	assertSelectionLogsOmit(t, selected)
}

func assertNoSelectionPopover(t *testing.T) {
	t.Helper()
	app.NotifySelectionMouseUp()
	if window := app.Windows()["selection"]; window != nil && window.IsVisible() {
		t.Fatal("selection popover was shown")
	}
	var text string
	app.Update(func(*arion.State) { text = app.Store().State.Selection.Text })
	if text != "" {
		t.Fatalf("selection offer %q", text)
	}
}

func assertSelectionLogsOmit(t *testing.T, selected string) {
	t.Helper()
	dir := filepath.Join(profileDir, "logs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(selected)) {
			t.Fatalf("selected text was logged in %s", entry.Name())
		}
	}
}

func sameIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
