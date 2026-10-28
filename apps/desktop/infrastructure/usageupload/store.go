package usageupload

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	policy "arion.local/desktop/internal/usageupload"
)

type Store struct{ Path string }

type file struct {
	Version int                   `json:"version"`
	Account string                `json:"account"`
	Visited []string              `json:"visited"`
	Acked   map[string]policy.Day `json:"acked"`
}

const version = 1

func (s Store) Load() (file, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return file{Version: version, Acked: map[string]policy.Day{}}, nil
	}
	if err != nil {
		return file{}, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil || f.Version != version {
		return file{}, policy.ErrStore
	}
	if f.Acked == nil {
		f.Acked = map[string]policy.Day{}
	}
	return f, nil
}

func (s Store) Save(f file) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	f.Version = version
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
