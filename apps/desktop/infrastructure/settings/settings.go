// Package settings persists user preferences as JSON in the per-user
// configuration directory, written atomically.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"arion.local/core/application"
)

// Store reads and writes one preferences file.
type Store struct{ Path string }

// DefaultPath is ~/Library/Application Support/Arion/preferences.json on
// macOS and %AppData%\Arion\preferences.json on Windows.
func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Arion", "preferences.json"), nil
}

// Load returns stored preferences, defaults on first run, and the names of
// fields that had to be repaired.
func (s Store) Load() (application.Preferences, []string, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return application.DefaultPreferences(), nil, nil
	}
	if err != nil {
		return application.Preferences{}, nil, err
	}
	prefs := application.DefaultPreferences()
	if err := json.Unmarshal(data, &prefs); err != nil {
		return application.Preferences{}, nil, err
	}
	normalized, repaired := prefs.Normalize()
	return normalized, repaired, nil
}

// Save writes through a temporary file and rename.
func (s Store) Save(prefs application.Preferences) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
