package bootstrap

import (
	"arion.local/desktop/internal/account"
	"arion.local/desktop/internal/center"
	"reflect"
	"testing"
)

func TestStandardCenterMatchesAccountDefaults(t *testing.T) {
	p, err := center.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg := centerAccountConfig(p, account.Config{})
	if p.Standard() && (cfg.Issuer != account.DefaultIssuer || cfg.ClientID != account.DefaultClientID || cfg.Resource != account.DefaultResource || cfg.Entitlement != account.HostedSyncEntitlement || !reflect.DeepEqual(cfg.Scopes, account.DefaultScopes())) {
		t.Fatal("standard profile drifted from account defaults")
	}
}

func TestCustomCenterCannotInheritOfficialAccountRoutes(t *testing.T) {
	device := center.Profile{URL: "https://sync.example.org", AuthMode: "device"}
	cfg := centerAccountConfig(device, account.Config{Issuer: account.DefaultIssuer})
	if !cfg.Disabled || !cfg.DisableWaitlist || cfg.Issuer != "" {
		t.Fatal("device center inherited account flow")
	}
	p := center.Profile{URL: "https://sync.example.org", AuthMode: "oidc", Issuer: "https://login.example.org", ClientID: "custom", Resource: "https://audience.example.org", Scopes: []string{"openid"}, Entitlement: "custom.sync"}
	cfg = centerAccountConfig(p, account.Config{Issuer: account.DefaultIssuer})
	if cfg.Disabled || !cfg.DisableWaitlist || cfg.Issuer != p.Issuer || cfg.Resource != p.Resource || cfg.Entitlement != p.Entitlement {
		t.Fatal("custom OIDC identity was not applied")
	}
}
