package dictation

import (
	"testing"
	"time"

	"arion.local/desktop/ui/shared/runtime"
)

// Finishing splits into recognition, the rewrite organizing the text, and
// the decided text awaiting insertion.
func TestPhaseOfSeparatesTheRewrite(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		d    runtime.Dictation
		want Phase
	}{
		{runtime.Dictation{State: runtime.StateFinalizing}, Transcribing},
		{runtime.Dictation{State: runtime.StateFinalizing, Rewriting: true}, Polishing},
		{runtime.Dictation{State: runtime.StateReviewing, Rewriting: true}, Reviewing},
		{runtime.Dictation{State: runtime.StateCommitting}, Inserting},
	} {
		if got := PhaseOf(tc.d, now); got != tc.want {
			t.Errorf("PhaseOf(%+v) = %v, want %v", tc.d, got, tc.want)
		}
	}
}
