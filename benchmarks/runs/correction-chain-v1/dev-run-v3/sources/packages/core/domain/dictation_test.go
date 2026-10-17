package domain

import (
	"errors"
	"testing"
)

func reviewing(t *testing.T) *Session {
	t.Helper()
	s, err := NewSession("s1", 1, ShortcutMode)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []SessionState{Preparing, Capturing} {
		if err := s.Advance(state); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AcceptRevision(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Advance(Finalizing); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptRevision(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Advance(Reviewing); err != nil {
		t.Fatal(err)
	}
	return s
}

// SPH-A01 / CTR-A04: older or duplicate revisions are ignored.
func TestStaleRevisionRejected(t *testing.T) {
	s, _ := NewSession("s1", 1, ShortcutMode)
	_ = s.Advance(Preparing)
	_ = s.Advance(Capturing)
	if err := s.AcceptRevision(3); err != nil {
		t.Fatal(err)
	}
	for _, stale := range []uint64{3, 2, 0} {
		if err := s.AcceptRevision(stale); CodeOf(err) != CodeStaleRevision {
			t.Fatalf("revision %d accepted after 3: %v", stale, err)
		}
	}
}

// CTR-A06 / DSK-A02: the reviewed revision commits exactly once.
func TestCommitClaimedExactlyOnce(t *testing.T) {
	s := reviewing(t)
	if err := s.ClaimCommit(1); CodeOf(err) != CodeStaleRevision {
		t.Fatalf("older revision claimed commit: %v", err)
	}
	if err := s.ClaimCommit(2); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimCommit(2); CodeOf(err) != CodeInsertionUncertain {
		t.Fatalf("second commit claim allowed: %v", err)
	}
	if err := s.Advance(Completed); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptRevision(9); CodeOf(err) != CodeStaleRevision {
		t.Fatal("completed session accepted a draft")
	}
}

// DSK-A04 / SPH-A02: a cancelled session accepts neither drafts nor commits.
func TestCancelledSessionRejectsLateWork(t *testing.T) {
	s := reviewing(t)
	if err := s.Advance(Cancelled); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptRevision(5); CodeOf(err) != CodeStaleRevision {
		t.Fatal("late draft accepted after cancel")
	}
	if err := s.ClaimCommit(2); err == nil {
		t.Fatal("cancelled session claimed commit")
	}
	if err := s.FailWith(CodeNoSpeech); err == nil {
		t.Fatal("terminal session failed again")
	}
}

func TestSessionIdentityValidation(t *testing.T) {
	if _, err := NewSession("", 1, ShortcutMode); err == nil {
		t.Fatal("empty id accepted")
	}
	if _, err := NewSession("a", 0, ShortcutMode); err == nil {
		t.Fatal("zero generation accepted")
	}
	if _, err := NewSession("a", 1, "voice"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	s, _ := NewSession("a", 4, SystemIMEMode)
	if !s.Owns("a", 4) || s.Owns("a", 3) || s.Owns("b", 4) {
		t.Fatal("ownership check wrong")
	}
	if !errors.Is(Fail(CodeNoSpeech, "x"), Fail(CodeNoSpeech, "")) {
		t.Fatal("errors.Is must match by code")
	}
}
