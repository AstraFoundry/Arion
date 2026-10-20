package domain

import (
	"encoding/json"
	"errors"
)

// ErrInvalidManifest means a stored update manifest is not one an app can
// install.
var ErrInvalidManifest = errors.New("update manifest is invalid")

// MaxManifestSize is the MyGo updater's limit on a manifest.
const MaxManifestSize = 1 << 20

// CheckManifest validates the fields the MyGo updater requires. The
// manifest itself is served byte for byte: its archive signature is
// verified by the app, never by this service.
func CheckManifest(data []byte) error {
	if len(data) > MaxManifestSize {
		return ErrInvalidManifest
	}
	var manifest struct {
		Version   string `json:"version"`
		URL       string `json:"url"`
		Size      int64  `json:"size"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ErrInvalidManifest
	}
	if manifest.Version == "" || manifest.URL == "" || manifest.Signature == "" || manifest.Size <= 0 {
		return ErrInvalidManifest
	}
	return nil
}
