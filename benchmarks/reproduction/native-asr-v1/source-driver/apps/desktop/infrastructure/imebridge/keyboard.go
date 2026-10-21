package imebridge

import (
	"encoding/json"
	"os"
	"path/filepath"

	"arion.local/core/application"
)

// WriteKeyboardSettings replaces the file the input method reads its
// keyboard preferences from when a client activates it. The write is atomic,
// so the input method never reads a partial file.
func WriteKeyboardSettings(path string, settings application.KeyboardPreferences) error {
	data, err := json.Marshal(settings)
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
