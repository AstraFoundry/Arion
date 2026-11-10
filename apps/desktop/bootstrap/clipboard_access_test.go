package bootstrap

import (
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// CLP-A08: disabling hides stored files and rejects stale delivery IDs.
func TestClipboardFilesDisablePreservesStorageAndBlocksDelivery(t *testing.T) {
	a := clipboardFixture(t)
	file := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{filepath.Join(t.TempDir(), "missing.txt")}}
	limits := a.prefs.Clipboard.Limits
	for _, payload := range []domain.ClipPayload{file, domain.ClipTextPayload("kept text")} {
		items, err := a.clipboard.service.Capture(payload, limits, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if payload.Kind == domain.ClipFiles {
			items, err = a.clipboard.service.Pin(file.ID(), true, limits)
			if err != nil {
				t.Fatal(err)
			}
		}
		a.state.Clipboard.Items = items
	}
	a.setClipboardFiles(false)
	finishClipboardWork(t, a)
	if len(a.state.Clipboard.Items) != 1 || a.state.Clipboard.Items[0].Payload.Kind != domain.ClipText {
		t.Fatal("file still visible")
	}
	if _, ok := a.clipboardItem(file.ID()); ok {
		t.Fatal("stale file ID remains actionable")
	}
	actions{a}.CopyClipboard(file.ID())
	actions{a}.PasteClipboard(file.ID())
	if a.state.Clipboard.Using {
		t.Fatal("disabled file delivery started")
	}
	kept, err := a.clipboard.service.Refresh(limits)
	if err != nil || len(kept) != 2 {
		t.Fatal("turning off deleted encrypted records", err)
	}
	prefs, _, err := a.prefStore.Load()
	if err != nil || prefs.Clipboard.FilesEnabled {
		t.Fatal("disabled preference not saved", err)
	}
}

func TestClipboardFilesConfirmationAndFailedSave(t *testing.T) {
	a := clipboardFixture(t)
	a.prefs.Clipboard.FilesEnabled = false
	a.state.Preferences = a.prefs
	actions{a}.ConfirmClipboardFiles()
	if a.state.Preferences.Clipboard.FilesEnabled {
		t.Fatal("enabled without confirmation")
	}
	a.state.Clipboard.FileConfirmation = true
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	original := a.prefStore.Path
	a.prefStore.Path = filepath.Join(file, "preferences.json")
	actions{a}.ConfirmClipboardFiles()
	if a.state.Preferences.Clipboard.FilesEnabled || !a.state.Clipboard.FileConfirmation || a.state.Clipboard.Error != copy.ClipboardPreferencesSaveFailed {
		t.Fatal("failed save enabled recording or lost explanation")
	}
	a.clipboardWork(func() ([]domain.Clip, error) { return a.clipboard.service.Refresh(a.prefs.Clipboard.Limits) }, copy.ClipboardLoadFailed)
	finishClipboardWork(t, a)
	if a.state.Clipboard.Error != copy.ClipboardPreferencesSaveFailed || !a.state.Clipboard.FileConfirmation {
		t.Fatal("unrelated history refresh cleared preference failure")
	}
	a.prefStore.Path = original
	actions{a}.ConfirmClipboardFiles()
	finishClipboardWork(t, a)
	prefs, _, err := a.prefStore.Load()
	if err != nil || !prefs.Clipboard.FilesEnabled {
		t.Fatal("confirmed opt-in not persisted", err)
	}
	if !a.state.Preferences.Clipboard.FilesEnabled || a.state.Clipboard.FileConfirmation {
		t.Fatal("confirmed opt-in not applied")
	}
}

// CLP-A08: changing mode after metadata preparation must rebuild publication.
func TestClipboardFilesEnabledDuringPublicationRefreshesMetadata(t *testing.T) {
	a := clipboardFixture(t)
	path := filepath.Join(t.TempDir(), "available.txt")
	if err := os.WriteFile(path, []byte("owned fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	limits := a.prefs.Clipboard.Limits
	var items []domain.Clip
	for _, payload := range []domain.ClipPayload{
		{Kind: domain.ClipFiles, Files: []string{t.TempDir()}},
		{Kind: domain.ClipFiles, Files: []string{path}},
		domain.ClipTextPayload("kept text"),
	} {
		var err error
		items, err = a.clipboard.service.Capture(payload, limits, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	a.prefs.Clipboard.FilesEnabled = false
	a.state.Preferences = a.prefs
	a.clipboardWork(func() ([]domain.Clip, error) { return items, nil }, copy.ClipboardLoadFailed)
	select {
	case <-a.queueReady:
	case <-time.After(5 * time.Second):
		t.Fatal("publication not queued")
	}
	// The first result now has no file metadata, but has not reached the UI.
	a.setClipboardFiles(true)
	select {
	case a.queueReady <- struct{}{}:
	default:
	}
	finishClipboardWork(t, a)
	if len(a.state.Clipboard.Items) != 2 {
		t.Fatal("stale result exposed a folder or lost an ordinary file")
	}
	for _, item := range a.state.Clipboard.Items {
		if item.Payload.Kind == domain.ClipFiles {
			meta := a.state.Clipboard.Meta[item.ID]
			if len(meta.Files) != 1 || !meta.Files[0].Available {
				t.Fatal("file published without rebuilt metadata")
			}
		}
	}
}

func TestClipboardStoragePermissionsKeepStorageError(t *testing.T) {
	for _, failure := range []copy.Key{copy.ClipboardLoadFailed, copy.ClipboardSaveFailed} {
		a := clipboardFixture(t)
		a.clipboardWork(func() ([]domain.Clip, error) {
			return nil, &os.PathError{Op: "open", Path: filepath.Join(t.TempDir(), "clipboard.bin"), Err: os.ErrPermission}
		}, failure)
		finishClipboardWork(t, a)
		if a.state.Clipboard.Error != failure {
			t.Fatal("history storage error became source permission guidance")
		}
	}
}
