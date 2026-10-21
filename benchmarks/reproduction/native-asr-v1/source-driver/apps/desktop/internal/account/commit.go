package account

import "context"

func (s *Service) commit(epoch uint64, opCtx context.Context, session Session, refresh, revokeEndpoint string) error {
	s.commitMu.Lock()
	locked := true
	unlock := func() {
		if locked {
			s.commitMu.Unlock()
			locked = false
		}
	}
	defer unlock()

	if s.stale(epoch, opCtx) {
		return ErrSignInCancelled
	}
	if refresh == "" {
		return &Error{Code: CodeToken, Detail: "missing_refresh_token"}
	}
	if err := s.store.Put(refresh); err != nil {
		return wrap(CodeStore, "write", err)
	}
	if s.testAfterPut != nil {
		s.testAfterPut()
	}
	if !s.stale(epoch, opCtx) {
		s.mu.Lock()
		s.session = session
		s.hasSession = true
		s.mu.Unlock()
		return nil
	}
	s.mu.Lock()
	reason := s.stopReason
	s.mu.Unlock()
	if reason == stopSignOut {
		return ErrSignInCancelled
	}
	_ = s.store.Delete()
	unlock()
	if err := s.revoke(context.Background(), revokeEndpoint, refresh); err != nil {
		return &Error{Code: CodeCancelled, Detail: "revoke", err: err}
	}
	return ErrSignInCancelled
}

func (s *Service) stale(epoch uint64, opCtx context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.epoch != epoch || opCtx.Err() != nil
}

func (s *Service) dropIfCurrent(epoch uint64) {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	if s.stale(epoch, context.Background()) {
		return
	}
	_ = s.store.Delete()
	s.mu.Lock()
	s.session = Session{}
	s.hasSession = false
	s.mu.Unlock()
}
