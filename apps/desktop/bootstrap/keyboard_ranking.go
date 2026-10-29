package bootstrap

import (
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/ranking"
)

type keyboardRankingReceiver interface {
	ConfigureKeyboardRanking(ports.KeyboardRanker) error
}

func (a *App) configureKeyboardRanking() {
	receiver, ok := a.ime.(keyboardRankingReceiver)
	if !ok {
		return
	}
	var ranker ports.KeyboardRanker
	if a.state.Preferences.Keyboard.NeuralRanking {
		ranker = &ranking.Ranker{Chat: a.rewriter}
	}
	a.record("keyboard_ranking_configuration", receiver.ConfigureKeyboardRanking(ranker))
}
