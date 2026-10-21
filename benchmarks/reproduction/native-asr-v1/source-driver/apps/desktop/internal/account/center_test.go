package account

import (
	"context"
	"errors"
	"testing"
)

type forbiddenCenterStore struct{}

func (forbiddenCenterStore) Put(string) error     { panic("unexpected credential write") }
func (forbiddenCenterStore) Get() (string, error) { panic("unexpected credential read") }
func (forbiddenCenterStore) Delete() error        { panic("unexpected credential deletion") }

func TestDeviceCenterDoesNotAccessAccounts(t *testing.T) {
	svc, err := New(Config{Disabled: true}, Dependencies{Store: forbiddenCenterStore{}, Browser: func(string) error { panic("unexpected browser") }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Stored(); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := svc.SignIn(context.Background()); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(context.Background()); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := svc.SignOut(context.Background(), nil); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := svc.JoinHostedSyncWaitlist(context.Background()); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestCustomAccountPolicies(t *testing.T) {
	svc, err := New(Config{DisableWaitlist: true, Entitlement: "custom.sync"}, Dependencies{Store: forbiddenCenterStore{}, Browser: func(string) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.JoinHostedSyncWaitlist(context.Background()); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if (Session{Entitlement: "custom.sync", Entitlements: []string{HostedSyncEntitlement}}).HasHostedSync() {
		t.Fatal("accepted unrelated official entitlement")
	}
	if !(Session{Entitlement: "custom.sync", Entitlements: []string{"custom.sync"}}).HasHostedSync() {
		t.Fatal("missing custom entitlement")
	}
}
