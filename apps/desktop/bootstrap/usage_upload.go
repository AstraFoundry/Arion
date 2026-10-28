package bootstrap

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"arion.local/core/domain"
	infra "arion.local/desktop/infrastructure/usageupload"
	"arion.local/desktop/internal/account"
	policy "arion.local/desktop/internal/usageupload"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

const usageUploadInterval = 5 * time.Minute

type usageUploadState struct {
	session     account.Session
	ledger      policy.Ledger
	store       infra.Store
	client      *infra.Client
	cancel      context.CancelFunc
	epoch       uint64
	next        time.Time
	retry       time.Duration
	busy        bool
	deniedToken string
}

func (a *App) startUsageUpload() {
	if a.state.Account.Disabled || a.opts.Offline || a.harness != nil {
		return
	}
	stop := make(chan struct{})
	mygo.App.OnQuit(func() { close(stop); a.suspendUsageUpload() })
	go func() {
		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				a.post(func() { a.recordUsageUpload(time.Now(), nil); a.flushUsageUpload(time.Now()) })
			case <-stop:
				return
			}
		}
	}()
}

func (a *App) suspendUsageUpload() {
	u := &a.usageUpload
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}
	u.epoch++
	u.session = account.Session{}
	u.busy = false
}

func (a *App) configureUsageUpload(session account.Session) {
	if a.state.Account.Disabled || a.opts.Offline || a.harness != nil || !session.UsageConsent || session.AccessToken == "" {
		a.suspendUsageUpload()
		return
	}
	u := &a.usageUpload
	if u.session.Issuer == session.Issuer && u.session.Subject == session.Subject && u.session.AccessToken == session.AccessToken {
		return
	}
	a.suspendUsageUpload()
	client, err := infra.NewClient(session.Issuer, a.opts.Account.HTTP)
	if err != nil {
		a.record("usage_upload", policy.ErrPayload)
		return
	}
	key := (policy.Session{Issuer: session.Issuer, Subject: session.Subject}).Key()
	store := infra.Store{Path: filepath.Join(a.opts.DataDir, "usage-upload", key+".json")}
	ledger, err := store.Load()
	if err != nil || (ledger.Account != "" && ledger.Account != key) {
		a.record("usage_upload", policy.ErrStore)
		return
	}
	ledger.Account = key
	u.session, u.client, u.store, u.ledger = session, client, store, ledger
	u.next, u.retry = time.Now(), 0
	a.recordUsageUpload(time.Now(), nil)
	a.flushUsageUpload(time.Now())
}

func (a *App) recordUsageUpload(now time.Time, usage *domain.Usage) {
	u := &a.usageUpload
	if !u.session.UsageConsent || (a.state.Account.Phase != arion.AccountSignedIn && a.state.Account.Phase != arion.AccountRefreshing) {
		return
	}
	if usage == nil && u.ledger.Days[domain.DayKey(now)].Visited {
		return
	}
	if err := u.ledger.Record(now, usage); err != nil {
		a.record("usage_upload", err)
		return
	}
	if err := u.store.Save(u.ledger); err != nil {
		a.record("usage_upload", policy.ErrStore)
	}
}

func (a *App) flushUsageUpload(now time.Time) {
	u := &a.usageUpload
	if u.busy || !u.session.UsageConsent || now.Before(u.next) || u.session.AccessToken == u.deniedToken {
		return
	}
	if !u.session.Expiry.IsZero() && !now.Before(u.session.Expiry) {
		return
	}
	days := u.ledger.Pending()
	if len(days) == 0 {
		u.next = now.Add(usageUploadInterval)
		return
	}
	// The random source identifier must reach disk before any network request.
	if err := u.store.Save(u.ledger); err != nil {
		a.record("usage_upload", policy.ErrStore)
		u.next = now.Add(usageUploadInterval)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel, u.busy = cancel, true
	epoch, client, source, session := u.epoch, u.client, u.ledger.Source, u.session
	go func() {
		defer cancel()
		var acknowledged []policy.Day
		var failed error
		for _, day := range days {
			if ctx.Err() != nil {
				failed = ctx.Err()
				break
			}
			_, failed = client.Report(ctx, policy.Session{Issuer: session.Issuer, Subject: session.Subject, Token: session.AccessToken}, source, day)
			if failed != nil {
				break
			}
			acknowledged = append(acknowledged, day)
		}
		a.post(func() { a.finishUsageUpload(epoch, acknowledged, failed, time.Now()) })
	}()
}

func (a *App) finishUsageUpload(epoch uint64, acknowledged []policy.Day, err error, now time.Time) {
	u := &a.usageUpload
	if epoch != u.epoch {
		return
	}
	u.busy = false
	u.cancel = nil
	if u.ledger.Acked == nil {
		u.ledger.Acked = map[string]policy.Day{}
	}
	// Acknowledge exactly the sent snapshot, not a server aggregate or a newer
	// local value recorded while the request was in flight.
	for _, day := range acknowledged {
		u.ledger.Acked[day.Date] = day
	}
	if saveErr := u.store.Save(u.ledger); saveErr != nil {
		a.record("usage_upload", policy.ErrStore)
	}
	if err == nil {
		u.retry = 0
		u.next = now.Add(usageUploadInterval)
		return
	}
	if errors.Is(err, policy.ErrAuthorization) {
		u.deniedToken = u.session.AccessToken
		u.session.UsageConsent = false
		u.next = now.Add(usageUploadInterval)
		return
	}
	u.retry = min(usageUploadInterval, max(15*time.Second, u.retry*2))
	u.next = now.Add(u.retry)
	a.record("usage_upload", policy.ErrHTTP)
}
