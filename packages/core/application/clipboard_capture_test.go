package application

import "testing"

func TestClipboardCaptureBoundariesAndInternalRevisions(t *testing.T) {
	var s ClipboardCapture
	if s.Pending(10, false) {
		t.Fatal("captured before opt in")
	}
	s.Configure(true, 10)
	if s.Pending(10, false) {
		t.Fatal("imported pre-enable clipboard")
	}
	if !s.Pending(11, false) {
		t.Fatal("missed user copy")
	}
	s.Accept(11)
	if s.Pending(11, false) {
		t.Fatal("duplicated revision")
	}
	if s.Pending(12, true) {
		t.Fatal("captured internal stage")
	}
	if s.Pending(13, true) {
		t.Fatal("captured internal restore")
	}
	// Identical text copied again still has a distinct external revision.
	if !s.Pending(14, false) {
		t.Fatal("missed subsequent user copy")
	}
	s.Accept(14)
	s.Configure(false, 14)
	if s.Pending(15, false) {
		t.Fatal("captured while paused")
	}
	s.Configure(true, 16)
	if s.Pending(16, false) {
		t.Fatal("imported paused copy")
	}
	if !s.Pending(17, false) {
		t.Fatal("missed first resumed copy")
	}
	// An unstable OS read is not accepted, so a stable retry remains eligible.
	if !s.Pending(17, false) {
		t.Fatal("unstable read lost")
	}
}
