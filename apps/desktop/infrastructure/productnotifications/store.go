package productnotifications

import (
	policy "arion.local/desktop/internal/productnotifications"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Store keeps account hashes and cold action metadata, never message bodies,
// profile names, access tokens or refresh credentials. Save is an atomic rename.
type Store struct{ Path string }

func (s Store) Load() (policy.Ledger, error) {
	empty := policy.Ledger{Schema: 1, Records: map[string]policy.Record{}}
	info, err := os.Lstat(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return empty, policy.ErrStore
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil || len(raw) > 4<<20 {
		return empty, policy.ErrStore
	}
	var l policy.Ledger
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&l) != nil {
		return empty, policy.ErrStore
	}
	var extra any
	if d.Decode(&extra) != io.EOF || policy.ValidateLedger(l) != nil {
		return empty, policy.ErrStore
	}
	return l, nil
}
func (s Store) Save(l policy.Ledger) error {
	if s.Path == "" || policy.ValidateLedger(l) != nil {
		return policy.ErrStore
	}
	raw, err := json.Marshal(l)
	if err != nil || len(raw) > 4<<20 {
		return policy.ErrStore
	}
	if info, err := os.Lstat(s.Path); err == nil && !info.Mode().IsRegular() || err != nil && !errors.Is(err, os.ErrNotExist) {
		return policy.ErrStore
	}
	if os.MkdirAll(filepath.Dir(s.Path), 0700) != nil {
		return policy.ErrStore
	}
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".product-notifications-*")
	if err != nil {
		return policy.ErrStore
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(raw)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil || os.Rename(name, s.Path) != nil {
		return policy.ErrStore
	}
	return nil
}
