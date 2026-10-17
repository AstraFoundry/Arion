package account

import (
	"context"
	"errors"
	"sync"
	"time"
)

type stopReason int

const (
	stopNone stopReason = iota
	stopCancel
	stopSignOut
)

// Dependencies are the ports New requires. Neither is dialed by New.
type Dependencies struct {
	Store   RefreshTokenStore
	Browser Browser
	// Now defaults to time.Now. Tests set it.
	Now func() time.Time
}

// Service is safe for concurrent use. One sign-in or refresh runs at a time.
type Service struct {
	cfg     Config
	store   RefreshTokenStore
	browser Browser
	now     func() time.Time

	mu         sync.Mutex
	idle       sync.Cond
	inflight   bool
	cancel     context.CancelFunc
	epoch      uint64
	stopReason stopReason
	session    Session
	hasSession bool

	commitMu sync.Mutex
	// blocked rejects SignIn and Refresh while SignOut owns the store.
	blocked int

	// testAfterPut runs after a refresh token is stored and before the
	// commit is published, so tests can cancel stale work.
	testAfterPut func()
}

// New builds a service. It does not discover, listen, or read the credential store.
func New(cfg Config, deps Dependencies) (*Service, error) {
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	if deps.Store == nil {
		return nil, &Error{Code: CodeStore, Detail: "missing"}
	}
	if deps.Browser == nil {
		return nil, &Error{Code: CodeBrowser, Detail: "missing"}
	}
	svc := &Service{cfg: normalized, store: deps.Store, browser: deps.Browser, now: deps.Now}
	svc.idle.L = &svc.mu
	return svc, nil
}

// Session returns the in-memory session. The refresh token is not included.
func (s *Service) Session() (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession {
		return Session{}, false
	}
	return s.session, true
}

// Stored reports whether the credential store holds a refresh token. It
// reads the store only: no discovery, no request, no refresh. A store that
// cannot be read is an error, never "nothing stored".
func (s *Service) Stored() (bool, error) {
	if s.cfg.Disabled {
		return false, ErrUnsupported
	}
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	token, err := s.store.Get()
	switch {
	case errors.Is(err, ErrNotFound):
		return false, nil
	case errors.Is(err, ErrUnsupported):
		return false, wrap(CodeStore, "unsupported", err)
	case err != nil:
		return false, wrap(CodeStore, "read", err)
	}
	return token != "", nil
}

// Cancel invalidates the in-flight sign-in or refresh. A callback that
// arrives afterwards is not exchanged, and a token already issued is not kept.
func (s *Service) Cancel() { s.stop(stopCancel) }

// SignIn opens the system browser and waits for one loopback callback.
func (s *Service) SignIn(ctx context.Context) (Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	epoch, opCtx, err := s.start(ctx, true)
	if err != nil {
		return Session{}, err
	}
	defer s.finish()

	doc, err := s.discover(opCtx)
	if err != nil {
		return Session{}, s.aborted(ctx, opCtx, err)
	}
	verifier, challenge, err := newPKCE()
	if err != nil {
		return Session{}, wrap(CodeToken, "pkce", err)
	}
	state, err := newState()
	if err != nil {
		return Session{}, wrap(CodeToken, "state", err)
	}
	lb, redirect, err := startLoopback(opCtx)
	if err != nil {
		return Session{}, s.aborted(ctx, opCtx, err)
	}
	defer lb.Close()

	authURL, err := authorizeURL(doc, s.cfg, redirect, state, challenge)
	if err != nil {
		return Session{}, err
	}
	if err := opCtx.Err(); err != nil {
		return Session{}, classifyStop(ctx, opCtx)
	}
	if err := s.browser(authURL); err != nil {
		return Session{}, s.aborted(ctx, opCtx, wrap(CodeBrowser, "open", err))
	}
	if err := opCtx.Err(); err != nil {
		return Session{}, classifyStop(ctx, opCtx)
	}
	cb, err := lb.Wait(opCtx)
	if err != nil {
		return Session{}, classifyStop(ctx, opCtx)
	}
	if err := matchState(state, cb.state); err != nil {
		return Session{}, err
	}
	if cb.error != "" {
		return Session{}, &Error{Code: CodeAuthorization, Detail: cb.error}
	}
	if cb.code == "" {
		return Session{}, &Error{Code: CodeToken, Detail: "missing_code"}
	}
	if err := opCtx.Err(); err != nil {
		return Session{}, classifyStop(ctx, opCtx)
	}
	tok, err := s.exchange(opCtx, doc, redirect, cb.code, verifier)
	if err != nil {
		return Session{}, s.aborted(ctx, opCtx, err)
	}
	session, err := sessionFromTokens(s.cfg.Issuer, tok, s.clock())
	session.Entitlement = s.cfg.Entitlement
	if err != nil {
		return Session{}, err
	}
	s.userInfo(opCtx, doc, &session)
	if err := s.commit(epoch, opCtx, session, tok.RefreshToken, doc.RevocationEndpoint); err != nil {
		return Session{}, err
	}
	return session, nil
}

// Refresh loads the stored refresh token, rotates it, and replaces the
// in-memory access token.
func (s *Service) Refresh(ctx context.Context) (Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	epoch, opCtx, err := s.start(ctx, false)
	if err != nil {
		return Session{}, err
	}
	defer s.finish()

	s.commitMu.Lock()
	current, err := s.store.Get()
	s.commitMu.Unlock()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Session{}, &Error{Code: CodeRefresh, Detail: "not_found", err: ErrNotFound}
		}
		return Session{}, wrap(CodeStore, "read", err)
	}
	doc, err := s.discover(opCtx)
	if err != nil {
		return Session{}, s.aborted(ctx, opCtx, err)
	}
	tok, err := s.refreshToken(opCtx, doc, current)
	if err != nil {
		if opCtx.Err() != nil {
			return Session{}, classifyStop(ctx, opCtx)
		}
		if DetailOf(err) == "invalid_grant" {
			s.dropIfCurrent(epoch)
		}
		return Session{}, &Error{Code: CodeRefresh, Detail: DetailOf(err), err: err}
	}
	next := tok.RefreshToken
	if next == "" {
		next = current
	}
	session, err := sessionFromTokens(s.cfg.Issuer, tok, s.clock())
	session.Entitlement = s.cfg.Entitlement
	if err != nil {
		return Session{}, err
	}
	s.userInfo(opCtx, doc, &session)
	if err := s.commit(epoch, opCtx, session, next, doc.RevocationEndpoint); err != nil {
		return Session{}, err
	}
	return session, nil
}

// SignOut revokes the refresh token when one is stored, then deletes it.
// Revocation failure is returned and recorded on SignOutResult after the
// local item is deleted. Local encryption keys are not a part of this store.
//
// armed, when set, runs after the in-flight sign-in is marked as sign-out
// and before this method waits for it. The composition root cancels the
// sign-in context there so a sign-in that has not entered yet stops, while
// a token already stored is left for this method to revoke.
func (s *Service) SignOut(ctx context.Context, armed func()) (SignOutResult, error) {
	if s.cfg.Disabled {
		return SignOutResult{}, ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	s.blocked++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.blocked--
		s.mu.Unlock()
	}()
	s.stop(stopSignOut)
	if armed != nil {
		armed()
	}
	s.waitIdle()

	var result SignOutResult
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	token, err := s.store.Get()
	if err != nil && !errors.Is(err, ErrNotFound) {
		return result, wrap(CodeStore, "read", err)
	}
	if token != "" {
		result.RevokeAttempted = true
		doc, discoverErr := s.discover(ctx)
		if discoverErr != nil {
			result.RevokeError = &Error{Code: CodeRevoke, Detail: "discovery", err: discoverErr}
		} else if revErr := s.revoke(ctx, doc.RevocationEndpoint, token); revErr != nil {
			result.RevokeError = revErr
		} else {
			result.Revoked = true
		}
	}
	delErr := s.store.Delete()
	if delErr == nil {
		result.RefreshTokenDeleted = true
		s.mu.Lock()
		s.session = Session{}
		s.hasSession = false
		s.mu.Unlock()
	}
	return result, errors.Join(result.RevokeError, storeErr(delErr))
}

func (s *Service) snapshotForJoin() (token string, entitled, busy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inflight || s.blocked > 0 {
		return "", false, true
	}
	if !s.hasSession {
		return "", false, false
	}
	return s.session.AccessToken, s.session.HasHostedSync(), false
}

func (s *Service) start(ctx context.Context, browserWait bool) (uint64, context.Context, error) {
	if s.cfg.Disabled {
		return 0, nil, ErrUnsupported
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked > 0 || ctx.Err() != nil {
		return 0, nil, ErrSignInCancelled
	}
	if s.inflight {
		return 0, nil, ErrSignInInProgress
	}
	opCtx, cancel := context.WithCancel(ctx)
	if browserWait && s.cfg.SignInTimeout > 0 {
		var stop context.CancelFunc
		opCtx, stop = context.WithTimeout(opCtx, s.cfg.SignInTimeout)
		parent := cancel
		cancel = func() { stop(); parent() }
	}
	s.inflight = true
	s.cancel = cancel
	s.stopReason = stopNone
	return s.epoch, opCtx, nil
}

func (s *Service) finish() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.inflight = false
	s.idle.Broadcast()
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) stop(reason stopReason) {
	s.mu.Lock()
	s.epoch++
	s.stopReason = reason
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) waitIdle() {
	s.mu.Lock()
	for s.inflight {
		s.idle.Wait()
	}
	s.mu.Unlock()
}

func (s *Service) aborted(parent, op context.Context, err error) error {
	if op.Err() != nil {
		return classifyStop(parent, op)
	}
	return err
}

func classifyStop(parent, op context.Context) error {
	if errors.Is(op.Err(), context.DeadlineExceeded) && parent.Err() == nil {
		return ErrSignInTimeout
	}
	return ErrSignInCancelled
}

func storeErr(err error) error {
	if err == nil {
		return nil
	}
	return wrap(CodeStore, "delete", err)
}
