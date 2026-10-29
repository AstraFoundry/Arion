package domain

// Session is the dictation aggregate. It owns the state machine, monotonic
// draft revisions and the exactly-once commit claim for one generation.
// It is not safe for concurrent use; the application serializes access.
type Session struct {
	id         string
	generation uint64
	mode       InputMode
	state      SessionState
	revision   uint64
	committed  bool
	failure    ErrorCode
}

// NewSession starts an idle session. A new dictation always creates a new
// session; terminal sessions never revive (CTR-A02).
func NewSession(id string, generation uint64, mode InputMode) (*Session, error) {
	if id == "" || generation == 0 {
		return nil, Fail(CodeContractUnsupported, "session identity required")
	}
	if mode != ShortcutMode && mode != SystemIMEMode {
		return nil, Fail(CodeContractUnsupported, "unknown input mode")
	}
	return &Session{id: id, generation: generation, mode: mode, state: Idle}, nil
}

func (s *Session) ID() string                    { return s.id }
func (s *Session) Generation() uint64            { return s.generation }
func (s *Session) Mode() InputMode               { return s.mode }
func (s *Session) State() SessionState           { return s.state }
func (s *Session) Revision() uint64              { return s.revision }
func (s *Session) Failure() ErrorCode            { return s.failure }
func (s *Session) Terminal() bool                { return IsTerminal(s.state) }
func (s *Session) Owns(id string, g uint64) bool { return s.id == id && s.generation == g }

// IsTerminal reports states that can never transition again.
func IsTerminal(state SessionState) bool {
	return state == Completed || state == Cancelled || state == Failed
}

// Advance moves along the documented lifecycle (CTR-A01).
func (s *Session) Advance(to SessionState) error {
	if to == Failed {
		return Fail(CodeInvalidTransition, "use Fail with a code")
	}
	next, err := Transition(s.state, to)
	if err != nil {
		return Fail(CodeInvalidTransition, err.Error())
	}
	s.state = next
	return nil
}

// FailWith ends an active session with an explicit code.
func (s *Session) FailWith(code ErrorCode) error {
	if !CanTransition(s.state, Failed) {
		return Fail(CodeInvalidTransition, string(s.state)+" cannot fail")
	}
	s.state, s.failure = Failed, code
	return nil
}

// AcceptRevision admits a strictly newer draft revision while the session is
// still producing drafts. Older or duplicate revisions are stale (SPH-A01).
func (s *Session) AcceptRevision(revision uint64) error {
	if s.state != Capturing && s.state != Finalizing {
		return Fail(CodeStaleRevision, "session no longer accepts drafts")
	}
	if revision <= s.revision {
		return Fail(CodeStaleRevision, "revision is not newer")
	}
	s.revision = revision
	return nil
}

// ClaimCommit grants the single commit attempt for the reviewed revision.
// A second claim, a claim for another revision or a claim outside review is
// rejected; an uncertain insertion can therefore never be retried (CTR-A06).
func (s *Session) ClaimCommit(sourceRevision uint64) error {
	if s.committed {
		return Fail(CodeInsertionUncertain, "commit already attempted")
	}
	if s.state != Reviewing {
		return Fail(CodeInvalidTransition, "commit requires review")
	}
	if sourceRevision != s.revision {
		return Fail(CodeStaleRevision, "commit revision differs from reviewed draft")
	}
	if err := s.Advance(Committing); err != nil {
		return err
	}
	s.committed = true
	return nil
}
