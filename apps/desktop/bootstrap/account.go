package bootstrap

import (
	"context"

	"arion.local/desktop/internal/account"
	arion "arion.local/desktop/ui/shared/runtime"
)

// Account returns the sign-in service for the composition root. The access
// token is available from Session. Views read State.Account instead.
func (a *App) Account() *account.Service { return a.accounts }

func (x actions) SignIn() {
	a := x.a
	ctx, seq := a.beginAccount()
	a.post(func() {
		if !a.accountLive(seq) {
			return
		}
		a.state.Account.Phase = arion.AccountSigningIn
		a.state.Account.Credential = arion.CredentialUnknown
		a.state.Account.ErrorCode = ""
		a.state.Account.ErrorDetail = ""
	})
	go func() {
		if !a.accountReady(seq, ctx) {
			return
		}
		session, err := a.accounts.SignIn(ctx)
		a.post(func() {
			if !a.accountLive(seq) {
				return
			}
			a.finishAccount(session, err)
		})
	}()
}

func (x actions) CancelSignIn() {
	a := x.a
	_, seq := a.beginAccount()
	a.accounts.Cancel()
	a.post(func() {
		if !a.accountLive(seq) {
			return
		}
		a.state.Account.ErrorCode = string(account.CodeCancelled)
		a.state.Account.ErrorDetail = ""
		a.state.Account.Credential = arion.CredentialUnknown
		if current, ok := a.accounts.Session(); ok {
			a.applySession(current, arion.AccountSignedIn)
			return
		}
		a.state.Account.Phase = arion.AccountSignedOut
	})
}

func (x actions) RefreshAccount() {
	a := x.a
	ctx, seq := a.beginAccount()
	a.post(func() {
		if !a.accountLive(seq) {
			return
		}
		a.state.Account.Phase = arion.AccountRefreshing
		a.state.Account.Credential = arion.CredentialUnknown
		a.state.Account.ErrorCode = ""
		a.state.Account.ErrorDetail = ""
	})
	go func() {
		if !a.accountReady(seq, ctx) {
			return
		}
		session, err := a.accounts.Refresh(ctx)
		a.post(func() {
			if !a.accountLive(seq) {
				return
			}
			a.finishAccount(session, err)
		})
	}()
}

func (x actions) SignOut() {
	a := x.a
	go func() {
		var seq uint64
		result, err := a.accounts.SignOut(context.Background(), func() {
			a.accountMu.Lock()
			a.accountSeq++
			seq = a.accountSeq
			cancel := a.accountCancel
			a.accountCancel = nil
			a.accountMu.Unlock()
			if cancel != nil {
				cancel()
			}
			a.post(func() {
				if !a.accountLive(seq) {
					return
				}
				a.state.Account.Phase = arion.AccountSigningOut
			})
		})
		a.post(func() {
			if !a.accountLive(seq) {
				return
			}
			a.finishSignOut(result, err)
		})
	}()
}

func (x actions) JoinHostedSyncWaitlist() {
	a := x.a
	a.post(func() {
		a.state.Account.Waitlist.Phase = arion.AccountWaitlistJoining
		a.state.Account.ErrorCode = ""
		a.state.Account.ErrorDetail = ""
	})
	go func() {
		result, err := a.accounts.JoinHostedSyncWaitlist(context.Background())
		a.post(func() { a.finishWaitlist(result, err) })
	}()
}

func (x actions) CheckStoredAccount() {
	a := x.a
	a.post(func() {
		current := &a.state.Account
		if current.Phase != arion.AccountSignedOut || current.Credential != arion.CredentialUnknown {
			return
		}
		current.Credential = arion.CredentialChecking
		seq := a.accountGeneration()
		go func() {
			stored, err := a.accounts.Stored()
			a.post(func() { a.finishCredential(seq, stored, err) })
		}()
	})
}

// finishCredential records a credential-store read unless a sign-in,
// refresh, cancel or sign-out happened since; the view reads again then.
func (a *App) finishCredential(seq uint64, stored bool, err error) {
	current := &a.state.Account
	if current.Credential != arion.CredentialChecking {
		return
	}
	switch {
	case !a.accountLive(seq):
		current.Credential = arion.CredentialUnknown
	case err != nil:
		current.Credential = arion.CredentialUnreadable
		current.ErrorCode = string(account.CodeOf(err))
		current.ErrorDetail = account.DetailOf(err)
	case stored:
		current.Credential = arion.CredentialStored
	default:
		current.Credential = arion.CredentialAbsent
	}
}

func (a *App) accountGeneration() uint64 {
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	return a.accountSeq
}

// beginAccount starts a sign-in or refresh generation and aborts the previous
// browser wait. Sign-out does not call this; it stops the service first so a
// stored refresh token is revoked on the sign-out result.
func (a *App) beginAccount() (context.Context, uint64) {
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	a.accountSeq++
	if a.accountCancel != nil {
		a.accountCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.accountCancel = cancel
	return ctx, a.accountSeq
}

func (a *App) accountLive(seq uint64) bool {
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	return seq == a.accountSeq
}

func (a *App) accountReady(seq uint64, ctx context.Context) bool {
	a.accountMu.Lock()
	defer a.accountMu.Unlock()
	return seq == a.accountSeq && ctx.Err() == nil
}

func (a *App) finishAccount(session account.Session, err error) {
	a.state.Account.Credential = arion.CredentialUnknown
	if err != nil {
		a.state.Account.ErrorCode = string(account.CodeOf(err))
		a.state.Account.ErrorDetail = account.DetailOf(err)
		if current, ok := a.accounts.Session(); ok {
			a.applySession(current, arion.AccountSignedIn)
			return
		}
		a.state.Account.Phase = arion.AccountSignedOut
		a.clearAccountProfile()
		return
	}
	a.applySession(session, arion.AccountSignedIn)
	a.state.Account.ErrorCode = ""
	a.state.Account.ErrorDetail = ""
	a.state.Account.Waitlist = arion.AccountWaitlist{}
	a.state.Account.SignOut = arion.AccountSignOut{}
}

func (a *App) applySession(session account.Session, phase arion.AccountPhase) {
	a.state.Account.Phase = phase
	a.state.Account.Subject = session.Subject
	a.state.Account.Name = session.Profile.Name
	a.state.Account.Email = session.Profile.Email
	a.state.Account.Picture = session.Profile.Picture
	a.state.Account.Locale = session.Profile.Locale
	a.state.Account.HostedSync = session.HasHostedSync()
}

func (a *App) clearAccountProfile() {
	a.state.Account.Subject = ""
	a.state.Account.Name = ""
	a.state.Account.Email = ""
	a.state.Account.Picture = ""
	a.state.Account.Locale = ""
	a.state.Account.HostedSync = false
}

func (a *App) finishSignOut(result account.SignOutResult, err error) {
	detail := ""
	if result.RevokeError != nil {
		detail = account.DetailOf(result.RevokeError)
	}
	signOut := arion.AccountSignOut{
		Attempted:           true,
		RefreshTokenDeleted: result.RefreshTokenDeleted,
		RevokeAttempted:     result.RevokeAttempted,
		Revoked:             result.Revoked,
		RevokeDetail:        detail,
	}
	if current, ok := a.accounts.Session(); ok {
		a.applySession(current, arion.AccountSignedIn)
		a.state.Account.SignOut = signOut
		a.state.Account.ErrorCode = string(account.CodeOf(err))
		a.state.Account.ErrorDetail = account.DetailOf(err)
		return
	}
	a.state.Account = arion.Account{Phase: arion.AccountSignedOut, Issuer: a.state.Account.Issuer, SignOut: signOut, Disabled: a.state.Account.Disabled, WaitlistDisabled: a.state.Account.WaitlistDisabled}
	if err != nil {
		a.state.Account.ErrorCode = string(account.CodeOf(err))
		a.state.Account.ErrorDetail = account.DetailOf(err)
	}
}

func (a *App) finishWaitlist(result account.WaitlistResult, err error) {
	phase := arion.AccountWaitlistPhase(result.Status)
	code := result.Code
	if err != nil {
		a.state.Account.ErrorCode = string(account.CodeOf(err))
		a.state.Account.ErrorDetail = account.DetailOf(err)
		if phase == "" {
			phase = arion.AccountWaitlistError
		}
		if code == "" {
			code = string(account.CodeOf(err))
		}
	}
	a.state.Account.Waitlist = arion.AccountWaitlist{
		Phase:      phase,
		EntryID:    result.EntryID,
		Code:       code,
		HTTPStatus: result.HTTPStatus,
	}
}
