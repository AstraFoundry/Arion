package usageupload

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	policy "arion.local/desktop/internal/usageupload"
)

type Store struct{ Path string }

const version = 1

func (s Store) Load() (policy.Ledger, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		source := make([]byte, 16)
		if _, err := rand.Read(source); err != nil {
			return policy.Ledger{}, err
		}
		return policy.Ledger{Version: version, Source: hex.EncodeToString(source), Days: map[string]policy.Day{}, Acked: map[string]policy.Day{}}, nil
	}
	if err != nil {
		return policy.Ledger{}, err
	}
	var f policy.Ledger
	if err := json.Unmarshal(data, &f); err != nil || f.Version != version {
		return policy.Ledger{}, policy.ErrStore
	}
	if len(f.Source) != 32 {
		return policy.Ledger{}, policy.ErrStore
	}
	if _, err := hex.DecodeString(f.Source); err != nil {
		return policy.Ledger{}, policy.ErrStore
	}
	for date, day := range f.Days {
		if date != day.Date || !day.Valid() {
			return policy.Ledger{}, policy.ErrStore
		}
	}
	if f.Days == nil {
		f.Days = map[string]policy.Day{}
	}
	if f.Acked == nil {
		f.Acked = map[string]policy.Day{}
	}
	return f, nil
}

func (s Store) Save(f policy.Ledger) error {
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
