package account

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultIssuer is the Astra Foundry issuer. Relying parties key accounts
	// on (issuer, subject).
	DefaultIssuer = "https://astrafoundry.si"
	// DefaultClientID is the public desktop client.
	DefaultClientID = "arion-desktop"
	// DefaultResource is the sync API audience (RFC 8707 resource).
	DefaultResource = "https://api.arion.astrafoundry.si"
	// HostedSyncEntitlement is the feature key the access token's ent claim
	// carries when hosted sync is granted. Decoding it is for the UI; the
	// sync server is the authority.
	HostedSyncEntitlement = "arion.sync.hosted"
	// HostedSyncWaitlistKey is the catalog key whose id JoinHostedSyncWaitlist
	// resolves. The id itself is not a constant.
	HostedSyncWaitlistKey = "hosted-sync"
	// CatalogProduct is the catalog slug that lists that waitlist.
	CatalogProduct = "arion"
	// DefaultSignInTimeout bounds one browser round trip.
	DefaultSignInTimeout = 5 * time.Minute
	callbackPath         = "/callback"
)

// DefaultScopes are the scopes requested from the issuer.
func DefaultScopes() []string {
	return []string{"openid", "profile", "offline_access", "arion.sync", "waitlist:join"}
}

// Config is read once by New. The zero value selects the product defaults
// and does not contact the network.
type Config struct {
	// Issuer is the OIDC issuer. Empty selects DefaultIssuer.
	Issuer string
	// ClientID is the public client id. Empty selects DefaultClientID.
	ClientID string
	// Scopes are requested in order. Empty selects DefaultScopes.
	Scopes []string
	// Resource is sent as the RFC 8707 resource parameter. Empty selects
	// DefaultResource.
	Resource string
	// WaitlistID overrides catalog lookup. Empty resolves the id of
	// HostedSyncWaitlistKey from the issuer's Arion catalog when the user
	// joins.
	WaitlistID string
	// SignInTimeout bounds the browser wait. Zero selects DefaultSignInTimeout.
	// A negative value waits only on the caller's context.
	SignInTimeout time.Duration
	// HTTP is used for discovery, the token endpoint, revocation, catalog,
	// and waitlist calls. Nil selects a client that does not follow redirects.
	HTTP *http.Client
}

func (c Config) normalized() (Config, error) {
	if c.Issuer == "" {
		c.Issuer = DefaultIssuer
	}
	c.Issuer = strings.TrimRight(c.Issuer, "/")
	if err := checkHTTPURL(c.Issuer); err != nil {
		return Config{}, &Error{Code: CodeDiscovery, Detail: "issuer", err: err}
	}
	if c.ClientID == "" {
		c.ClientID = DefaultClientID
	}
	if c.Resource == "" {
		c.Resource = DefaultResource
	}
	if err := checkHTTPURL(c.Resource); err != nil {
		return Config{}, &Error{Code: CodeDiscovery, Detail: "resource", err: err}
	}
	if len(c.Scopes) == 0 {
		c.Scopes = DefaultScopes()
	} else {
		c.Scopes = append([]string(nil), c.Scopes...)
	}
	if c.SignInTimeout == 0 {
		c.SignInTimeout = DefaultSignInTimeout
	}
	if c.WaitlistID != "" && !validWaitlistID(c.WaitlistID) {
		return Config{}, &Error{Code: CodeWaitlist, Detail: "waitlist_id"}
	}
	c.HTTP = clientWithoutRedirects(c.HTTP)
	return c, nil
}

func checkHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return errBadURL
	}
	return nil
}

func clientWithoutRedirects(client *http.Client) *http.Client {
	if client == nil {
		transport := http.DefaultTransport
		if base, ok := transport.(*http.Transport); ok {
			transport = base.Clone()
		}
		return &http.Client{Timeout: 30 * time.Second, Transport: transport, CheckRedirect: refuseRedirect}
	}
	clone := *client
	clone.CheckRedirect = refuseRedirect
	return &clone
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
