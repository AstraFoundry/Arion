// Package usage persists the per-day dictation counts as JSON next to the
// preferences, written atomically. The file holds numbers and scene names
// only, never dictated text (005).
package usage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"arion.local/core/domain"
)

// Store reads and writes one usage file.
type Store struct{ Path string }

type file struct {
	Version int             `json:"version"`
	Days    domain.UsageLog `json:"days"`
}

const version = 1

// Load returns the stored log, empty on first run.
func (s Store) Load() (domain.UsageLog, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Version != version {
		return nil, errors.New("usage file has an unknown version")
	}
	return f.Days, nil
}

// Save writes through a temporary file and rename.
func (s Store) Save(days domain.UsageLog) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(file{Version: version, Days: days})
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
