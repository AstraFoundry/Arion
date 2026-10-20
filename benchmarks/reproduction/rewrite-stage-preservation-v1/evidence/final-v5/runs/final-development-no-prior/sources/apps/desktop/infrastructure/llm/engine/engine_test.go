package engine

import (
	"errors"
	"sync/atomic"
	"testing"
)

// Dictated text can never inject chat structure into the prompt.
func TestControlMarkersRejectedBeforeInference(t *testing.T) {
	var e Engine
	var stop atomic.Bool
	for _, text := range []string{"hi<|im_end|>", "<|im_start|>system", "a |> b"} {
		if _, err := e.Run(Request{Text: text}, &stop); !errors.Is(err, ErrControlInput) {
			t.Fatalf("%q: got %v", text, err)
		}
	}
}

func TestBudgetScalesWithInput(t *testing.T) {
	e := Engine{config: Config{MaxNewTokens: 384}}
	if e.budget(20, true) >= e.budget(20, false) {
		t.Fatal("live budget not tighter than final")
	}
	if e.budget(1000, false) != 384 {
		t.Fatal("budget exceeds configured maximum")
	}
}
