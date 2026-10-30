package bootstrap

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardhistory"
	"arion.local/desktop/infrastructure/settings"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

func clipboardFixture(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	repo, err := clipboardhistory.Open(filepath.Join(dir, "clipboard.bin"), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	p := application.DefaultPreferences(runtime.GOOS)
	return &App{prefs: p, prefStore: settings.Store{Path: filepath.Join(dir, "preferences.json")}, state: arion.State{Preferences: p}, clipboard: &clipboardController{service: application.NewClipboard(repo)}, queueReady: make(chan struct{}, 1)}
}
func finishClipboardWork(t *testing.T, a *App) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for a.state.Clipboard.Busy || a.state.Clipboard.Using {
		select {
		case <-a.queueReady:
			a.queueMu.Lock()
			batch := a.queue
			a.queue = nil
			a.queueMu.Unlock()
			for _, fn := range batch {
				fn()
			}
		case <-deadline:
			t.Fatal("clipboard work did not finish")
		}
	}
}

func TestClipboardLimitEditProtectsPinsAndOtherPreferences(t *testing.T) {
	a := clipboardFixture(t)
	l := a.state.Preferences.Clipboard.Limits
	items, err := a.clipboard.service.Capture(domain.ClipTextPayload(strings.Repeat("x", (1<<20)+1)), l, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	items, err = a.clipboard.service.Pin(items[0].ID, true, l)
	if err != nil {
		t.Fatal(err)
	}
	a.state.Clipboard.Items = items
	tight := l
	tight.Bytes = 1 << 20
	tight.ItemBytes = 1 << 20
	actions{a}.SetClipboardLimits(tight)
	if a.state.Clipboard.Error != copy.ClipboardCapacity || a.state.Preferences.Clipboard.Limits != l {
		t.Fatal("invalid limit destroyed pinned policy")
	}
	l.Count = 10
	actions{a}.SetClipboardLimits(l)
	// Another setting changes while the history worker finishes.
	a.prefs.Sounds = false
	a.state.Preferences.Sounds = false
	finishClipboardWork(t, a)
	if a.state.Preferences.Sounds || a.state.Preferences.Clipboard.Limits.Count != 10 {
		t.Fatal("late clipboard work replaced unrelated preferences")
	}
}

func TestClipboardInvalidReferenceCannotPaste(t *testing.T) {
	a := clipboardFixture(t)
	payload := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{filepath.Join(t.TempDir(), "missing")}}
	a.state.Clipboard.Items = []domain.Clip{{ID: payload.ID(), Payload: payload, Copied: time.Now()}}
	actions{a}.PasteClipboard(payload.ID())
	finishClipboardWork(t, a)
	if a.state.Clipboard.Error != copy.ClipboardUnavailable || a.resourceJobs.Load() != 0 {
		t.Fatal("missing reference was not rejected")
	}
}

func TestClipboardClosingDuringValidationCancelsPaste(t *testing.T) {
	a := clipboardFixture(t)
	p := domain.ClipTextPayload("Cancelled selection")
	a.state.Clipboard.Items = []domain.Clip{{ID: p.ID(), Payload: p, Copied: time.Now()}}
	actions{a}.PasteClipboard(p.ID())
	a.clipboard.windowSeq++
	finishClipboardWork(t, a)
	if a.state.Clipboard.Error != "" || a.clipboard.delivering || a.resourceJobs.Load() != 0 {
		t.Fatal("closed picker attempted a paste")
	}
}

func TestClipboardRecorderPhysicalKeyAccelerators(t *testing.T) {
	for _, test := range []struct {
		key  application.HoldKey
		want string
	}{
		{"super+shift+v", "super+shift+v"},
		{"ctrl+page_up", "ctrl+PageUp"},
		{"alt+bracket_left", "alt+["},
		{"ctrl+semicolon", "ctrl+;"},
		{"shift+backslash", "shift+\\"},
		{"f9", "f9"},
	} {
		if got := clipboardAccelerator(test.key); got != test.want {
			t.Fatalf("%q mapped to %q, want %q", test.key, got, test.want)
		}
	}
}

func TestClipboardShortcutConflictsRejectVoiceAssignments(t *testing.T) {
	a := clipboardFixture(t)
	a.prefs.Clipboard.Shortcut = "ctrl+shift+k"
	a.state.Preferences = a.prefs
	before := a.prefs
	actions{a}.SetHoldKey("ctrl+shift+k")
	actions{a}.SetTranslateKey("ctrl+shift+k")
	if a.prefs.HoldKey != before.HoldKey || a.prefs.TranslateKey != before.TranslateKey || a.prefs.Clipboard != before.Clipboard {
		t.Fatal("clipboard shortcut was assigned to a voice action")
	}
}

func TestClipboardDictationRecordsOnlyPositiveNoInputCopies(t *testing.T) {
	for _, test := range []struct {
		name             string
		enabled, noInput bool
		outcome          contracts.CommitOutcome
		failure          domain.ErrorCode
		want             int
	}{
		{"no-input-copy", true, true, contracts.Copied, "", 1},
		{"input-insertion", true, false, contracts.Unknown, "", 0},
		{"focus-change-copy", true, false, contracts.Copied, "", 0},
		{"permission-failure", true, true, contracts.NotApplied, domain.CodePermissionDenied, 0},
		{"copy-failure", true, true, contracts.Copied, domain.CodeClipboardUnavailable, 0},
		{"collection-off", false, true, contracts.Copied, "", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := clipboardFixture(t)
			a.state.Preferences.Clipboard.Enabled = test.enabled
			a.recordClipboardDictation(contracts.Event{Envelope: contracts.Envelope{Kind: contracts.CommitResultReady}, NoInput: test.noInput, Outcome: test.outcome, Error: test.failure, Text: "Completed dictation"})
			finishClipboardWork(t, a)
			if len(a.state.Clipboard.Items) != test.want {
				t.Fatalf("history size %d want %d", len(a.state.Clipboard.Items), test.want)
			}
		})
	}
}

func TestClipboardDictationWaitsForCurrentMutation(t *testing.T) {
	a := clipboardFixture(t)
	a.state.Preferences.Clipboard.Enabled = true
	limits := a.prefs.Clipboard.Limits
	a.clipboardWork(func() ([]domain.Clip, error) {
		return a.clipboard.service.Capture(domain.ClipTextPayload("User copy"), limits, time.Now())
	}, copy.ClipboardSaveFailed)
	for _, text := range []string{"First dictation", "Second dictation"} {
		a.recordClipboardDictation(contracts.Event{Envelope: contracts.Envelope{Kind: contracts.CommitResultReady}, Outcome: contracts.Copied, NoInput: true, Text: text})
	}
	finishClipboardWork(t, a)
	if len(a.state.Clipboard.Items) != 3 || len(a.clipboard.pendingDictations) != 0 {
		t.Fatal("busy history lost dictation results")
	}
}

func TestClipboardPlacementStaysBesideInputAndInsideWorkArea(t *testing.T) {
	for _, test := range []struct {
		name         string
		anchor, area mygo.Rectangle
		want         mygo.Rectangle
	}{
		{"below", mygo.Rectangle{X: 100, Y: 100, Width: 2, Height: 20}, mygo.Rectangle{Width: 1000, Height: 800}, mygo.Rectangle{X: 100, Y: 124, Width: 360, Height: 360}},
		{"flip-and-clamp-right", mygo.Rectangle{X: 950, Y: 750, Width: 2, Height: 20}, mygo.Rectangle{Width: 1000, Height: 800}, mygo.Rectangle{X: 640, Y: 386, Width: 360, Height: 360}},
		{"negative-origin", mygo.Rectangle{X: -1000, Y: 100, Width: 2, Height: 20}, mygo.Rectangle{X: -1200, Width: 1200, Height: 800}, mygo.Rectangle{X: -1000, Y: 124, Width: 360, Height: 360}},
		{"small-work-area", mygo.Rectangle{X: 10, Y: 10, Width: 2, Height: 20}, mygo.Rectangle{Width: 200, Height: 100}, mygo.Rectangle{Width: 200, Height: 100}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := clipboardBounds(test.anchor, test.area, 360, 360); got != test.want {
				t.Fatalf("%+v want %+v", got, test.want)
			}
		})
	}
}

func TestClipboardLegacyFolderPinsReturnToCountRetention(t *testing.T) {
	a := clipboardFixture(t)
	old := time.Now().Add(-100 * 24 * time.Hour)
	payload := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{t.TempDir()}}
	limits := a.prefs.Clipboard.Limits
	items, err := a.clipboard.service.Capture(payload, limits, old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.clipboard.service.Pin(items[0].ID, true, limits); err != nil {
		t.Fatal(err)
	}
	kept, err := refreshClipboard(a.clipboard.service, limits)
	if err != nil || len(kept) != 1 || kept[0].Pinned {
		t.Fatal("hidden unsupported pin remained protected")
	}
	limits.Count = 1
	kept, err = a.clipboard.service.Capture(domain.ClipTextPayload("new ordinary copy"), limits, time.Now())
	if err != nil || len(kept) != 1 || kept[0].Payload.Text != "new ordinary copy" {
		t.Fatal("hidden unsupported reference did not follow the count limit")
	}
}

func TestClipboardLegacyFoldersStayOutOfPublishedItems(t *testing.T) {
	a := clipboardFixture(t)
	folder := t.TempDir()
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	limits := a.prefs.Clipboard.Limits
	for _, path := range []string{folder, file} {
		if _, err := a.clipboard.service.Capture(domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}, limits, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	a.clipboardWork(func() ([]domain.Clip, error) { return refreshClipboard(a.clipboard.service, limits) }, copy.ClipboardLoadFailed)
	finishClipboardWork(t, a)
	if len(a.state.Clipboard.Items) != 1 || a.state.Clipboard.Items[0].Payload.Files[0] != file {
		t.Fatal("legacy folder was published or ordinary file was hidden")
	}
}

func TestClipboardClosingDuringValidationCancelsCopy(t *testing.T) {
	a := clipboardFixture(t)
	p := domain.ClipTextPayload("Cancelled copy")
	a.state.Clipboard.Items = []domain.Clip{{ID: p.ID(), Payload: p}}
	actions{a}.CopyClipboard(p.ID())
	a.clipboard.windowSeq++
	finishClipboardWork(t, a)
	if a.state.Clipboard.Error != "" || a.resourceJobs.Load() != 0 {
		t.Fatal("closed picker attempted a clipboard write")
	}
}
