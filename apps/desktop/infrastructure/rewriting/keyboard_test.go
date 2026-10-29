package rewriting

import (
	"context"
	"testing"

	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/prompt"
)

func TestKeyboardUnavailableNeverProvisions(t *testing.T) {
	s := &Service{}
	_, err := s.Complete(context.Background(), prompt.ChatRequest{})
	if domain.CodeOf(err) != domain.CodeModelMissing || s.Ready() {
		t.Fatalf("unavailable model: %v", err)
	}
}

func TestKeyboardCompletionYieldsToSpeechAndShutdown(t *testing.T) {
	for _, action := range []string{"voice", "rewrite", "disable"} {
		t.Run(action, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := &Service{rankingCancel: cancel}
			switch action {
			case "voice":
				s.SuspendKeyboard(true)
			case "rewrite":
				_, _ = s.Rewrite(context.Background(), ports.RewriteRequest{})
			case "disable":
				s.Disable()
			}
			if ctx.Err() != context.Canceled {
				t.Fatal("low priority work was not cancelled")
			}
			if s.speechJobs != 0 {
				t.Fatal("speech reservation leaked")
			}
			if action == "voice" && !s.keyboardSuspended {
				t.Fatal("voice did not suspend admission")
			}
		})
	}
}
