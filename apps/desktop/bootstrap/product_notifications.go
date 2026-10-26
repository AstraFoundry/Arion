package bootstrap

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"arion.local/desktop/infrastructure/browser"
	"arion.local/desktop/infrastructure/diagnostics"
	infra "arion.local/desktop/infrastructure/productnotifications"
	policy "arion.local/desktop/internal/productnotifications"
	"arion.local/desktop/ui/pages/settings"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

type productNotificationTransport struct{ a *App }

func (t productNotificationTransport) Show(ctx context.Context, m policy.Message, id string) error {
	allowed := false
	mygo.RunOnMain(func() {
		allowed = ctx.Err() == nil && t.a.productNotificationsAvailable() && t.a.state.UpdateNotifications.Permission == arion.UpdateNotificationsAllowed && !t.a.updateBusy() && !t.a.updateApplying()
	})
	if !allowed {
		return policy.ErrDeferred
	}
	options := mygo.NotificationOptions{ID: id, Title: m.Title, Body: m.Body, Silent: true, Group: "product"}
	for i, b := range m.Buttons {
		options.Actions = append(options.Actions, mygo.NotificationAction{ID: "button-" + strconv.Itoa(i), Title: b.Label})
	}
	err := mygo.NewNotification(options).Show()
	if ctx.Err() != nil {
		mygo.RemoveNotification(id)
		return ctx.Err()
	}
	if err != nil {
		t.a.post(t.a.refreshUpdatePermission)
	}
	return err
}
func (t productNotificationTransport) Remove(id string) { mygo.RemoveNotification(id) }

func (a *App) productNotificationsAvailable() bool {
	return a.center.Standard() && !a.opts.Offline && a.harness == nil
}

func (a *App) startProductNotifications() {
	if !a.productNotificationsAvailable() {
		return
	}
	client, err := infra.NewClient(a.center.Issuer, a.opts.Account.HTTP)
	if err == nil {
		a.productNotifications, err = policy.New(client, infra.Store{Path: filepath.Join(a.opts.DataDir, "product-notifications.json")}, productNotificationTransport{a}, browser.Open, func(status policy.Status) {
			a.post(func() {
				if a.accounts == nil {
					return
				}
				current, ok := a.accounts.Session()
				if !ok || (policy.Session{Issuer: current.Issuer, Subject: current.Subject}).Key() != status.AccountKey {
					return
				}
				if status.Error != nil {
					a.record("product_notification", status.Error)
				} else if status.Uncertain {
					a.recordEntry(diagnostics.Entry{Event: "product_notification", Outcome: "unknown"})
				}
			})
		})
	}
	if err != nil {
		a.record("product_notification", err)
		return
	}
	a.configureProductNotifications()
	stop := make(chan struct{})
	mygo.App.OnQuit(func() { close(stop); a.productNotifications.Stop() })
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.post(a.pollProductNotifications)
			case <-stop:
				return
			}
		}
	}()
}
func (a *App) configureProductNotifications() {
	if a.productNotifications == nil {
		return
	}
	var session policy.Session
	if a.accounts != nil && a.state.Account.Phase != arion.AccountSigningOut && a.state.Account.Phase != arion.AccountSigningIn {
		if current, ok := a.accounts.Session(); ok {
			session = policy.Session{Issuer: current.Issuer, Subject: current.Subject, Token: current.AccessToken}
		}
	}
	available := a.productNotificationsAvailable()
	safe := !a.updateBusy() && !a.updateApplying()
	a.productNotifications.Configure(session, available, a.state.UpdateNotifications.Permission == arion.UpdateNotificationsAllowed, safe)
	permission := a.state.UpdateNotifications.Permission
	if available && session.Valid() && a.state.Account.Phase == arion.AccountSignedIn && safe && (permission == "" || permission == arion.UpdateNotificationsUnknown) {
		a.requestSystemNotificationPermission()
	}
	if permission != "" && permission != arion.UpdateNotificationsUnknown && permission != arion.UpdateNotificationsAllowed {
		a.pendingProductActions = nil
	}
	if session.Valid() && permission == arion.UpdateNotificationsAllowed {
		pending := a.pendingProductActions
		a.pendingProductActions = nil
		for _, action := range pending {
			a.productNotificationAction(action[0], action[1])
		}
	}
}
func (a *App) pollProductNotifications() {
	a.configureProductNotifications()
	if a.productNotifications != nil {
		go func() { _ = a.productNotifications.Poll(context.Background(), time.Now()) }()
	}
}
func (a *App) suspendProductNotifications() {
	a.pendingProductActions = nil
	if a.productNotifications != nil {
		a.productNotifications.Configure(policy.Session{}, false, false, false)
	}
}
func (a *App) productNotificationAction(id, action string) {
	if !strings.HasPrefix(id, policy.NativePrefix) || len(id) != len(policy.NativePrefix)+64 || action != "" && action != "button-0" && action != "button-1" {
		return
	}
	if !a.productNotificationsAvailable() {
		return
	}
	// Permission discovery and account restoration may finish in either order
	// during cold launch. Retain only bounded owned identifiers until both agree.
	if permission := a.state.UpdateNotifications.Permission; permission == "" || permission == arion.UpdateNotificationsUnknown {
		if len(a.pendingProductActions) < 8 {
			a.pendingProductActions = append(a.pendingProductActions, [2]string{id, action})
		}
		return
	}
	if a.state.UpdateNotifications.Permission != arion.UpdateNotificationsAllowed {
		return
	}
	if a.productNotifications == nil || a.accounts == nil {
		if len(a.pendingProductActions) < 8 {
			a.pendingProductActions = append(a.pendingProductActions, [2]string{id, action})
		}
		return
	}
	if _, ok := a.accounts.Session(); !ok {
		// Cold clicks wait only for the account restoration already underway. A
		// signed-out profile never opens a previous account's URL.
		if a.state.Account.Credential == arion.CredentialStored || a.state.Account.Credential == arion.CredentialChecking || a.state.Account.Phase == arion.AccountRefreshing {
			if len(a.pendingProductActions) < 8 {
				a.pendingProductActions = append(a.pendingProductActions, [2]string{id, action})
			}
		}
		return
	}
	generation := a.accountGeneration()
	go func() {
		err := a.productNotifications.Activate(context.Background(), id, action, time.Now())
		a.post(func() { a.finishProductNotificationAction(generation, action, err) })
	}()
}

func (a *App) finishProductNotificationAction(generation uint64, action string, err error) {
	if !a.accountLive(generation) || a.state.Account.Phase != arion.AccountSignedIn || !a.productNotificationsAvailable() || a.state.UpdateNotifications.Permission != arion.UpdateNotificationsAllowed {
		return
	}
	if err != nil {
		if !errors.Is(err, policy.ErrStale) {
			a.record("product_notification", err)
		}
		return
	}
	if action == "" {
		a.state.Focus = arion.SettingsFocus{Section: settings.Account, Seq: a.state.Focus.Seq + 1}
		a.openSettings()
	}
}
