package bootstrap

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"arion.local/desktop/internal/account"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

const accountPoll = 5 * time.Second

// accountMaintenance is owned by the main thread. Timers never rotate a
// token themselves; they enter the same generation-guarded action as the UI.
type accountMaintenance struct {
	next    time.Time
	retry   time.Duration
	stopped bool
}

func (m *accountMaintenance) renewed(session account.Session, now time.Time) {
	now = now.Round(0)
	m.retry = 0
	m.next = time.Time{}
	if session.Expiry.IsZero() {
		return
	}
	// Remove monotonic clock readings so deadlines catch up after OS sleep.
	remaining := session.Expiry.Round(0).Sub(now)
	// Reserve 10% of the lifetime, up to one minute, for renewal.
	lead := min(time.Minute, remaining/10)
	m.next = now.Add(max(time.Second, remaining-lead))
}

func (m *accountMaintenance) failed(now time.Time) {
	if m.retry == 0 {
		m.retry = 5 * time.Second
	} else {
		m.retry = min(5*time.Minute, m.retry*2)
	}
	// Jitter avoids synchronized retries when an issuer recovers.
	m.next = now.Round(0).Add(m.retry/2 + time.Duration(rand.Int64N(int64(m.retry/2)+1)))
}

func accountRetryable(err error) bool {
	if account.CodeOf(err) == account.CodeInProgress {
		return true
	}
	if account.CodeOf(err) == account.CodeStore {
		return account.DetailOf(err) == "read" || account.DetailOf(err) == "write"
	}
	detail := account.DetailOf(err)
	if detail == "transport" || detail == "request" || detail == "server_error" || detail == "temporarily_unavailable" {
		return true
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(detail, "http_"))
	return status == 408 || status == 429 || status >= 500 && status <= 599
}

func (a *App) startAccountMaintenance() {
	if a.state.Account.Disabled || a.opts.Offline || a.harness != nil {
		return
	}
	a.accountMaintenance.next = time.Now()
	tick := func() { a.maintainAccount(time.Now()) }
	tick()
	stop := make(chan struct{})
	mygo.App.OnQuit(func() {
		a.accountMaintenance.stopped = true
		close(stop)
	})
	go func() {
		ticker := time.NewTicker(accountPoll)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.post(tick)
			case <-stop:
				return
			}
		}
	}()
}

func (a *App) maintainAccount(now time.Time) {
	m := &a.accountMaintenance
	if a.state.Account.Disabled || a.opts.Offline || a.harness != nil || m.stopped || m.next.IsZero() || now.Before(m.next) || a.state.Account.Busy() || a.state.Account.Credential == arion.CredentialChecking {
		return
	}
	if a.state.Account.Credential == arion.CredentialUnknown && a.state.Account.Phase == arion.AccountSignedOut {
		actions{a}.CheckStoredAccount()
		return
	}
	if a.state.Account.Credential == arion.CredentialAbsent {
		m.next = time.Time{}
		return
	}
	a.refreshAccount(true)
}

func (a *App) finishBackgroundRefresh(session account.Session, err error) {
	if err == nil {
		a.finishAccount(session, nil)
		return
	}
	if !accountRetryable(err) {
		a.accountMaintenance.next = time.Time{}
		a.finishAccount(session, err)
		if a.state.Account.Credential == arion.CredentialUnknown {
			a.state.Account.Credential = arion.CredentialStored
		}
		return
	}
	a.accountMaintenance.failed(time.Now())
	a.state.Account.Credential = arion.CredentialStored
	// A transport failure does not end the account identity. No toast or
	// action is required while maintenance retries in the background.
	if current, ok := a.accounts.Session(); ok {
		a.applySession(current, arion.AccountSignedIn)
	} else {
		a.state.Account.Phase = arion.AccountSignedOut
	}
}
