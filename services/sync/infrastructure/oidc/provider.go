package oidc

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"arion.local/sync/application"
	"arion.local/sync/domain"
)

// Options configures JWT and SET verification. JWKSURL skips discovery when
// set; otherwise the JWKS URI is read from the issuer's OpenID configuration.
type Options struct {
	Issuer      string
	Audience    string
	ClientID    string
	Entitlement string
	JWKSURL     string
	Algorithms  []string
	Leeway      time.Duration
	CacheTTL    time.Duration
	Quota       int64
	Directory   application.Directory
	HTTP        *http.Client
	Now         func() time.Time
}

// Provider verifies access tokens and security events with one JWKS cache.
type Provider struct {
	issuer      string
	audience    string
	clientID    string
	entitlement string
	jwksURL     string
	algs        map[string]struct{}
	leeway      time.Duration
	ttl         time.Duration
	quota       int64
	directory   application.Directory
	client      *http.Client
	now         func() time.Time

	mu          sync.Mutex
	keys        map[string]cachedKey
	fetched     time.Time
	resolvedURL string
}

type cachedKey struct {
	public crypto.PublicKey
	alg    string
}

// New validates options and returns a provider. It does not fetch JWKS until
// a token is presented.
func New(opt Options) (*Provider, error) {
	if opt.Directory == nil || opt.Now == nil || opt.Issuer == "" || opt.Audience == "" || opt.ClientID == "" || opt.Entitlement == "" {
		return nil, errors.New("oidc provider is incomplete")
	}
	if opt.Leeway < 0 || opt.CacheTTL <= 0 {
		return nil, errors.New("oidc timings are invalid")
	}
	algs := make(map[string]struct{}, len(opt.Algorithms))
	for _, alg := range opt.Algorithms {
		if rejectedAlg(alg) {
			return nil, errors.New("jwt alg is not allowed")
		}
		algs[alg] = struct{}{}
	}
	if len(algs) == 0 {
		return nil, errors.New("jwt alg allow-list is empty")
	}
	if err := ValidateFetchURL(opt.Issuer); err != nil {
		return nil, err
	}
	if opt.JWKSURL != "" {
		if err := ValidateFetchURL(opt.JWKSURL); err != nil {
			return nil, err
		}
	}
	client := opt.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second, CheckRedirect: refuseUnsafeRedirect}
	}
	return &Provider{
		issuer:      opt.Issuer,
		audience:    opt.Audience,
		clientID:    opt.ClientID,
		entitlement: opt.Entitlement,
		jwksURL:     opt.JWKSURL,
		algs:        algs,
		leeway:      opt.Leeway,
		ttl:         opt.CacheTTL,
		quota:       opt.Quota,
		directory:   opt.Directory,
		client:      client,
		now:         opt.Now,
		keys:        map[string]cachedKey{},
	}, nil
}

// Mode is "oidc".
func (*Provider) Mode() string { return "oidc" }

type accessClaims struct {
	jwt.RegisteredClaims
	Ent []string `json:"ent"`
}

// Authorize verifies the bearer token and maps (issuer, subject) to a tenant.
func (p *Provider) Authorize(ctx context.Context, header string) (application.Scope, error) {
	raw, err := bearer(header)
	if err != nil {
		return application.Scope{}, err
	}
	claims, err := p.parseAccess(ctx, raw)
	if err != nil {
		return application.Scope{}, err
	}
	if claims.Subject == "" || claims.IssuedAt == nil {
		return application.Scope{}, domain.Fail(domain.CodeUnauthorized)
	}
	if !slices.Contains(claims.Ent, p.entitlement) {
		return application.Scope{}, application.Fail(application.CodeEntitlementRequired)
	}
	access, err := p.directory.Lookup(ctx, p.issuer, claims.Subject)
	if err != nil {
		return application.Scope{}, err
	}
	issued := claims.IssuedAt.Time
	if access.HasSessionRevocation && !issued.After(access.SessionRevokedAt) {
		return application.Scope{}, application.Fail(application.CodeSessionRevoked)
	}
	if access.Purged {
		return application.Scope{}, domain.Fail(domain.CodeUnauthorized)
	}
	if access.Frozen {
		return application.Scope{}, application.Fail(application.CodeTenantFrozen)
	}
	tenant := access.Tenant
	if !access.Linked {
		tenant, err = p.directory.EnsureTenant(ctx, p.issuer, claims.Subject, p.quota, p.now())
		if err != nil {
			return application.Scope{}, err
		}
	}
	return application.Scope{ID: tenant.ID, Quota: tenant.Quota}, nil
}

type setClaims struct {
	jwt.RegisteredClaims
	SubID struct {
		Format string `json:"format"`
		Iss    string `json:"iss"`
		Sub    string `json:"sub"`
	} `json:"sub_id"`
	Events map[string]json.RawMessage `json:"events"`
}

// Verify checks an RFC 8417 SET and returns the event types it carries.
func (p *Provider) Verify(ctx context.Context, token string) (application.SecurityEvent, error) {
	header, err := peekHeader(token)
	if err != nil {
		return application.SecurityEvent{}, application.SET("invalid_request", "event rejected")
	}
	if header.Typ != "secevent+jwt" {
		return application.SecurityEvent{}, application.SET("invalid_request", "type rejected")
	}
	if rejectedAlg(header.Alg) || header.Kid == "" {
		return application.SecurityEvent{}, application.SET("invalid_key", "key rejected")
	}
	if _, ok := p.algs[header.Alg]; !ok {
		return application.SecurityEvent{}, application.SET("invalid_key", "key rejected")
	}
	key, err := p.publicKey(ctx, header.Kid, header.Alg)
	if err != nil {
		return application.SecurityEvent{}, application.SET("invalid_key", "key rejected")
	}
	claims := &setClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{header.Alg}),
		jwt.WithIssuer(p.issuer),
		jwt.WithAudience(p.clientID),
		jwt.WithLeeway(p.leeway),
		jwt.WithTimeFunc(p.now),
	)
	if _, err := parser.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != header.Alg {
			return nil, application.SET("invalid_key", "key rejected")
		}
		return key, nil
	}); err != nil {
		return application.SecurityEvent{}, classifySET(err)
	}
	if claims.ID == "" || claims.IssuedAt == nil || len(claims.Events) == 0 {
		return application.SecurityEvent{}, application.SET("invalid_request", "event rejected")
	}
	if claims.SubID.Format != "iss_sub" || claims.SubID.Iss != p.issuer || claims.SubID.Sub == "" {
		return application.SecurityEvent{}, application.SET("invalid_request", "subject rejected")
	}
	types := make([]string, 0, len(claims.Events))
	for kind := range claims.Events {
		types = append(types, kind)
	}
	slices.Sort(types)
	return application.SecurityEvent{
		JTI:     claims.ID,
		Issued:  claims.IssuedAt.Time,
		Issuer:  claims.SubID.Iss,
		Subject: claims.SubID.Sub,
		Types:   types,
	}, nil
}

func (p *Provider) parseAccess(ctx context.Context, token string) (*accessClaims, error) {
	header, err := peekHeader(token)
	if err != nil || header.Kid == "" {
		return nil, domain.Fail(domain.CodeUnauthorized)
	}
	if header.Typ != "at+jwt" || rejectedAlg(header.Alg) {
		return nil, domain.Fail(domain.CodeUnauthorized)
	}
	if _, ok := p.algs[header.Alg]; !ok {
		return nil, domain.Fail(domain.CodeUnauthorized)
	}
	key, err := p.publicKey(ctx, header.Kid, header.Alg)
	if err != nil {
		return nil, domain.Fail(domain.CodeUnauthorized)
	}
	claims := &accessClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{header.Alg}),
		jwt.WithIssuer(p.issuer),
		jwt.WithAudience(p.audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(p.leeway),
		jwt.WithTimeFunc(p.now),
	)
	if _, err := parser.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != header.Alg {
			return nil, domain.Fail(domain.CodeUnauthorized)
		}
		return key, nil
	}); err != nil {
		return nil, domain.Fail(domain.CodeUnauthorized)
	}
	return claims, nil
}

func classifySET(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return application.SET("invalid_audience", "audience rejected")
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return application.SET("invalid_issuer", "issuer rejected")
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return application.SET("invalid_key", "key rejected")
	case errors.Is(err, errUnknownKey):
		return application.SET("invalid_key", "key rejected")
	default:
		var set *application.SETError
		if errors.As(err, &set) {
			return set
		}
		return application.SET("invalid_request", "event rejected")
	}
}

func rejectedAlg(alg string) bool {
	return alg == "" || strings.EqualFold(alg, "none") || strings.HasPrefix(strings.ToUpper(alg), "HS")
}

func bearer(header string) (string, error) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.Contains(token, " ") {
		return "", domain.Fail(domain.CodeUnauthorized)
	}
	return token, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

func peekHeader(token string) (jwtHeader, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, errors.New("malformed jwt")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, err
	}
	var header jwtHeader
	if err := json.Unmarshal(raw, &header); err != nil {
		return jwtHeader{}, err
	}
	return header, nil
}

var errUnknownKey = errors.New("unknown jwk")

func (p *Provider) publicKey(ctx context.Context, kid, alg string) (crypto.PublicKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if key, ok := p.cached(kid, alg); ok && p.now().Sub(p.fetched) < p.ttl {
		return key, nil
	}
	if err := p.refresh(ctx); err != nil {
		return nil, err
	}
	if key, ok := p.cached(kid, alg); ok {
		return key, nil
	}
	return nil, errUnknownKey
}

func (p *Provider) cached(kid, alg string) (crypto.PublicKey, bool) {
	key, ok := p.keys[kid]
	if !ok || (key.alg != "" && key.alg != alg) {
		return nil, false
	}
	return key.public, true
}

func (p *Provider) refresh(ctx context.Context) error {
	url, err := p.resolveURL(ctx)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks status %d", response.StatusCode)
	}
	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&doc); err != nil {
		return err
	}
	keys := make(map[string]cachedKey, len(doc.Keys))
	for _, key := range doc.Keys {
		if key.Kid == "" || (key.Use != "" && key.Use != "sig") {
			continue
		}
		public, err := key.public()
		if err != nil {
			continue
		}
		keys[key.Kid] = cachedKey{public: public, alg: key.Alg}
	}
	p.keys = keys
	p.fetched = p.now()
	return nil
}

func (p *Provider) resolveURL(ctx context.Context) (string, error) {
	if p.jwksURL != "" {
		return p.jwksURL, nil
	}
	if p.resolvedURL != "" {
		return p.resolvedURL, nil
	}
	endpoint := strings.TrimRight(p.issuer, "/") + "/.well-known/openid-configuration"
	if err := ValidateFetchURL(endpoint); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discovery status %d", response.StatusCode)
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&doc); err != nil {
		return "", err
	}
	if doc.JWKSURI == "" {
		return "", errors.New("discovery response has no jwks_uri")
	}
	if err := ValidateFetchURL(doc.JWKSURI); err != nil {
		return "", err
	}
	p.resolvedURL = doc.JWKSURI
	return doc.JWKSURI, nil
}

// ValidateFetchURL accepts the issuer and JWKS URLs the relay may request.
// Link-local addresses and the cloud metadata name are rejected. Loopback and
// ordinary hosts stay available for tests and self-hosted issuers. An
// operator-configured issuer can still name a public host; this stops a
// discovery document from pointing the relay at instance metadata.
func ValidateFetchURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" {
		return errors.New("jwks url rejected")
	}
	switch parsed.Scheme {
	case "https", "http":
	default:
		return errors.New("jwks url rejected")
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "metadata.google.internal") {
		return errors.New("jwks url rejected")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return errors.New("jwks url rejected")
		}
	}
	return nil
}

func refuseUnsafeRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return errors.New("jwks url rejected")
	}
	return ValidateFetchURL(req.URL.String())
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	Use string `json:"use"`
	X   string `json:"x"`
	Y   string `json:"y"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (k jwk) public() (crypto.PublicKey, error) {
	switch k.Kty {
	case "OKP":
		if k.Crv != "Ed25519" {
			return nil, errors.New("unsupported curve")
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, errors.New("invalid ed25519 key")
		}
		return ed25519.PublicKey(raw), nil
	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
		e := new(big.Int).SetBytes(eBytes).Int64()
		if e < 2 || e > int64(^uint(0)>>1) {
			return nil, errors.New("invalid rsa exponent")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(e)}, nil
	case "EC":
		curve, err := curveFor(k.Crv)
		if err != nil {
			return nil, err
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{Curve: curve, X: new(big.Int).SetBytes(xBytes), Y: new(big.Int).SetBytes(yBytes)}, nil
	default:
		return nil, errors.New("unsupported key type")
	}
}

func curveFor(name string) (elliptic.Curve, error) {
	switch name {
	case "P-256":
		return elliptic.P256(), nil
	case "P-384":
		return elliptic.P384(), nil
	case "P-521":
		return elliptic.P521(), nil
	default:
		return nil, errors.New("unsupported curve")
	}
}
