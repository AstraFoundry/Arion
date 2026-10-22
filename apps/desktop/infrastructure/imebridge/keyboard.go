package imebridge

import (
	"encoding/json"
	"os"
	"path/filepath"

	"arion.local/core/application"
)

// keyboardSettings is the file the input method reads: the preferences, the
// candidate highlight resolved for each appearance, so the palette and its
// text rule live only in application, and the dictation key that titles the
// menu's Voice Input item (IME-06).
type keyboardSettings struct {
	application.KeyboardPreferences
	DictationKey application.HoldKey `json:"dictationKey,omitempty"`
	Highlight    struct {
		Light application.Highlight `json:"light"`
		Dark  application.Highlight `json:"dark"`
	} `json:"highlight"`
}

// WriteKeyboardSettings replaces the file the input method reads its
// keyboard preferences from when a client activates it. The write is atomic,
// so the input method never reads a partial file.
func WriteKeyboardSettings(path string, settings application.KeyboardPreferences, dictationKey application.HoldKey) error {
	file := keyboardSettings{KeyboardPreferences: settings, DictationKey: dictationKey}
	file.Highlight.Light, file.Highlight.Dark = settings.Highlight(false), settings.Highlight(true)
	data, err := json.Marshal(file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".keyboard-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
