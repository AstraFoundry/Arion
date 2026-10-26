package ports

import (
	"arion.local/core/contracts"
	"context"
)

// KeyboardRanker selects only from the exact submitted producer snapshot.
type KeyboardRanker interface {
	RankKeyboard(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error)
}
