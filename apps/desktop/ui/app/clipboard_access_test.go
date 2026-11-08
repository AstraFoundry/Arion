package app

import (
	clipboardfeature "arion.local/desktop/ui/features/clipboard"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
)

// CLP-A08: file recording requires an explicit, localized confirmation.
func TestClipboardFilesConfirmation(t *testing.T) {
	for _, platform := range []string{"darwin-arm64", "windows-amd64"} {
		for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
			state, rec, store := fixture()
			state.Preferences.UILanguage = locale
			state.Platform = platform
			state.Preferences.Clipboard.FilesEnabled = false
			state.Clipboard.FileConfirmation = true
			height := 0
			tt := ui.NewTester(func(c *ui.Context) { height = clipboardfeature.Access(c, store, theme.Apply(c)) }, ClipboardAccessWidth, ClipboardAccessHeight)
			settle(tt)
			messages := store.Copy()
			expected := copy.ClipboardFilesPermissionsMac
			other := copy.ClipboardFilesPermissionsWindows
			if platform == "windows-amd64" {
				expected, other = other, expected
			}
			if !hasText(tt, messages.Text(expected)) || hasText(tt, messages.Text(other)) {
				t.Fatal("wrong platform permission guidance")
			}
			if height <= 0 {
				t.Fatal("confirmation has no content height")
			}
			tt.SetSize(ClipboardAccessWidth, height)
			settle(tt)
			button, ok := tt.Find(messages.Text(copy.ClipboardFilesConfirm))
			if !ok || button.Y+button.H > float32(height) {
				t.Fatal("confirmation action clipped")
			}
			snapshot(t, tt, "clipboard-files-confirm-"+string(locale)+"-"+platform)
			tt.Click(messages.Text(copy.ClipboardFilesConfirm))
			if !rec.called("clipboard-files-confirm") {
				t.Fatal("confirmation missing")
			}
			tt.Click(messages.Text(copy.ClipboardCancel))
			if !rec.called("clipboard-files-cancel") {
				t.Fatal("cancel missing")
			}
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "clipboard-files-confirm-dark-"+string(locale)+"-"+platform)
			state.Clipboard.Error = copy.ClipboardPreferencesSaveFailed
			settle(tt)
			tt.SetSize(ClipboardAccessWidth, height)
			settle(tt)
			button, ok = tt.Find(messages.Text(copy.ClipboardFilesConfirm))
			if !ok || button.Y+button.H > float32(height) || !hasText(tt, messages.Text(copy.ClipboardPreferencesSaveFailed)) {
				t.Fatal("failed-save state clipped")
			}
			snapshot(t, tt, "clipboard-files-confirm-error-"+string(locale)+"-"+platform)
		}
	}
}

func TestClipboardFilesSettingsToggleAndRecovery(t *testing.T) {
	state, rec, store := fixture()
	state.Platform = "darwin-arm64"
	state.Preferences.Clipboard.Enabled = true
	state.Preferences.Clipboard.FilesEnabled = false
	tt := ui.NewTester(func(c *ui.Context) { clipboardfeature.Settings(c, store, theme.Apply(c), func() {}) }, 640, 420)
	settle(tt)
	messages := store.Copy()
	label, ok := tt.Find(messages.Text(copy.ClipboardFilesEnabled))
	if !ok {
		t.Fatal("file option missing")
	}
	tt.ClickAt(610, label.Y+label.H/2)
	if !rec.called("clipboard-files-toggle") {
		t.Fatal("toggle missing")
	}
	if hasText(tt, messages.Text(copy.ClipboardAccessSettings)) {
		t.Fatal("permission action shown while file recording is off")
	}
	state.Preferences.Clipboard.FilesEnabled = true
	state.Clipboard.Error = copy.ClipboardAccessRequired
	settle(tt)
	tt.Click(messages.Text(copy.ClipboardAccessSettings))
	if !rec.called("clipboard-file-settings") {
		t.Fatal("denial has no settings recovery")
	}
	state.Clipboard.Error = ""
	state.Preferences.Clipboard.Enabled = false
	rest(tt, 250*time.Millisecond)
	if hasText(tt, messages.Text(copy.ClipboardFilesEnabled)) {
		t.Fatal("disabled master exposes file option")
	}
}
