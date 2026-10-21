//go:build !darwin

package wordseg

import (
	"arion.local/core/domain"
	"errors"
)

// Segmenter has no approved linguistic tokenizer on this platform yet.
// Windows also has no readable-field learning adapter (ADR 014).
type Segmenter struct{}

func (Segmenter) Words(string) ([]domain.WordSpan, error) {
	return nil, errors.New("local word segmentation is unsupported on this platform")
}
