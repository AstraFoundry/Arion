package dictation

import (
	"testing"
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

// The note follows the Done outcome, so it is shown only after the dictation
// completed and only for a bounded time.
func TestPolishNoteShownAfterTheOutcome(t *testing.T) {
	start := time.Now()
	d := runtime.Dictation{State: runtime.StateCompleted, PolishNote: runtime.PolishRateLimited, Changed: start}
	for _, tc := range []struct {
		at   time.Duration
		want bool
	}{
		{0, false}, {DoneVisible - time.Millisecond, false}, {DoneVisible, true},
		{DoneVisible + PolishNoteVisible - time.Millisecond, true}, {DoneVisible + PolishNoteVisible, false},
	} {
		if got := PolishNoteShown(d, start.Add(tc.at)); got != tc.want {
			t.Errorf("at %v: shown %v, want %v", tc.at, got, tc.want)
		}
	}
	d.Failure = runtime.CodeTargetChanged
	if PolishNoteShown(d, start.Add(DoneVisible)) {
		t.Error("a failed dictation shows its failure, not the note")
	}
	d.Failure, d.PolishNote = "", ""
	if PolishNoteShown(d, start.Add(DoneVisible)) {
		t.Error("no note recorded")
	}
}

func TestPolishNoteWording(t *testing.T) {
	for _, tc := range []struct {
		detail, provider string
		allowed          bool
		title            copy.Key
	}{
		{runtime.PolishReasoningRequired, runtime.CloudOpenRouter, false, copy.PolishNoteReasoningTitle},
		{runtime.PolishReasoningRequired, runtime.CloudOpenRouter, true, copy.PolishNoteRefusedTitle},
		{runtime.PolishCredentialRejected, runtime.CloudOpenAI, false, copy.PolishNoteCredentialTitle},
		{runtime.PolishCredentialRejected, runtime.CloudChatGPT, false, copy.PolishNoteSignInTitle},
		{runtime.PolishRateLimited, runtime.CloudOpenAI, false, copy.PolishNoteRateTitle},
		{runtime.PolishQuotaExhausted, runtime.CloudOpenAI, false, copy.PolishNoteQuotaTitle},
		{runtime.PolishModelUnavailable, runtime.CloudOpenAI, false, copy.PolishNoteOutageTitle},
		{runtime.PolishModelNotFound, runtime.CloudOpenAI, false, copy.PolishNoteNotFoundTitle},
		{runtime.PolishTimeout, runtime.CloudOpenAI, false, copy.PolishNoteTimeoutTitle},
	} {
		got, ok := PolishNoteOf(tc.detail, tc.provider, tc.allowed)
		if !ok || got.Title != tc.title || got.Hint == "" {
			t.Errorf("%s/%s/%v: %+v ok=%v", tc.detail, tc.provider, tc.allowed, got, ok)
		}
	}
	if _, ok := PolishNoteOf("unknown", runtime.CloudOpenAI, false); ok {
		t.Error("an unknown category has no wording")
	}
}
