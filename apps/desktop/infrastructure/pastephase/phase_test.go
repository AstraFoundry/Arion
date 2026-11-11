package pastephase

import (
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func TestRestorePhasesStayDistinct(t *testing.T) {
	cases := []struct {
		owns, hadText, wrote bool
		want                 Phase
	}{
		{true, true, true, Restored},
		{true, false, false, Restored},
		{true, true, false, RestoreFailed},
		{false, true, true, Left},
		{false, false, false, Left},
	}
	for _, tc := range cases {
		if got := Restore(tc.owns, tc.hadText, tc.wrote); got != tc.want {
			t.Fatalf("owns=%v hadText=%v wrote=%v: got %s, want %s", tc.owns, tc.hadText, tc.wrote, got, tc.want)
		}
	}
	if RestoreFailed == Restored || RestoreFailed == Left || RestoreFailed == Posted || RestoreFailed == Copied {
		t.Fatal("restore-failed collapsed into another phase")
	}
}

func TestFailedRestoreIsNotASuccessfulPaste(t *testing.T) {
	posted := contracts.CommitResult{Outcome: contracts.Unknown}
	NoteRestoreFailure(&posted, RestoreFailed)
	if posted.Outcome != contracts.Unknown || posted.Failure != domain.CodeClipboardRestoreFailed {
		t.Fatalf("posted restore failure: %+v", posted)
	}
	clean := contracts.CommitResult{Outcome: contracts.Unknown}
	NoteRestoreFailure(&clean, Restored)
	NoteRestoreFailure(&clean, Left)
	if clean.Outcome != contracts.Unknown || clean.Failure != "" {
		t.Fatalf("restore or left looked like a failure: %+v", clean)
	}
	moved := contracts.CommitResult{Outcome: contracts.NotApplied, Failure: domain.CodeTargetChanged}
	NoteRestoreFailure(&moved, Left)
	if moved.Failure != domain.CodeTargetChanged {
		t.Fatalf("leaving the clipboard erased the destination change: %+v", moved)
	}
	NoteRestoreFailure(&moved, RestoreFailed)
	if moved.Outcome != contracts.NotApplied || moved.Failure != domain.CodeClipboardRestoreFailed {
		t.Fatalf("failed restore looked successful: %+v", moved)
	}
}
