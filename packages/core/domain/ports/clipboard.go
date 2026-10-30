package ports

import "arion.local/core/domain"

// ClipboardRepository persists one authenticated snapshot. Failures must leave
// the last committed snapshot intact; no plaintext resources are written.
type ClipboardRepository interface {
	Load() ([]domain.Clip, error)
	Save([]domain.Clip) error
}
