// Package clipboardhistory implements the encrypted local clipboard repository.
package clipboardhistory

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/png"
	"os"
	"path/filepath"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/encryptedfile"
)

const maxSnapshotBytes = 720 << 20

type Store struct{ file *encryptedfile.File }

func Open(path string, key []byte) (*Store, error) {
	f, err := encryptedfile.New(path, key, "arion-clipboard-v1", maxSnapshotBytes)
	if err != nil {
		return nil, err
	}
	return &Store{file: f}, nil
}

type snapshot struct {
	Version int           `json:"version"`
	Items   []domain.Clip `json:"items"`
}

func (s *Store) Load() ([]domain.Clip, error) {
	b, err := s.file.Read()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snap snapshot
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&snap); err != nil {
		return nil, err
	}
	if snap.Version != 1 {
		return nil, domain.ErrClipInvalid
	}
	seen := map[string]bool{}
	for _, c := range snap.Items {
		if err := c.Payload.Validate(); err != nil {
			return nil, err
		}
		if c.ID != c.Payload.ID() || seen[c.ID] || c.Copied.IsZero() {
			return nil, domain.ErrClipInvalid
		}
		if err := ValidateResource(c.Payload); err != nil {
			return nil, err
		}
		seen[c.ID] = true
	}
	return snap.Items, nil
}

func (s *Store) Save(items []domain.Clip) error {
	b, err := json.Marshal(snapshot{Version: 1, Items: items})
	if err != nil {
		return err
	}
	return s.file.Write(b)
}

// ValidateResource checks structure without requiring references to exist.
// PNG dimensions are bounded before preview allocation.
func ValidateResource(p domain.ClipPayload) error {
	if p.Kind == domain.ClipFiles {
		for _, path := range p.Files {
			if !filepath.IsAbs(path) {
				return domain.ErrClipInvalid
			}
		}
	}
	if p.Kind == domain.ClipImage {
		c, err := png.DecodeConfig(bytes.NewReader(p.PNG))
		if err != nil {
			return domain.ErrClipInvalid
		}
		if c.Width <= 0 || c.Height <= 0 || int64(c.Width)*int64(c.Height) > 40_000_000 {
			return domain.ErrClipOversize
		}
	}
	return nil
}
