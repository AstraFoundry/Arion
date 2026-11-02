package ports

import "context"

// KeyboardScorer returns comparable mean log probabilities in input order.
// Texts are provisional alternatives; scoring never inserts or learns them.
type KeyboardScorer interface {
	ScoreKeyboard(context.Context, string, string, []string) ([]float64, error)
}
