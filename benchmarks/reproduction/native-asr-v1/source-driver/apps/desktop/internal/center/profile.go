// Package center owns the immutable public center profile compiled into a package.
package center

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
)

//go:embed default.json
var defaultJSON []byte

type Profile struct {
	URL         string   `json:"url"`
	AuthMode    string   `json:"authMode"`
	Issuer      string   `json:"issuer"`
	ClientID    string   `json:"clientID"`
	Resource    string   `json:"resource"`
	Scopes      []string `json:"scopes"`
	Entitlement string   `json:"entitlement"`
}

// Load reads compiled bytes only. Runtime environment variables cannot retarget a package.
func Load() (Profile, error) { return Parse(embeddedProfile) }

func Parse(data []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	if err := p.Validate(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (p Profile) Validate() error {
	if !validURL(p.URL) {
		return errors.New("invalid center URL")
	}
	switch p.AuthMode {
	case "device":
		if p.Issuer != "" || p.ClientID != "" || p.Resource != "" || len(p.Scopes) != 0 || p.Entitlement != "" {
			return errors.New("device center cannot contain OIDC settings")
		}
	case "oidc":
		if !validURL(p.Issuer) || !validURL(p.Resource) || strings.TrimSpace(p.ClientID) == "" || strings.TrimSpace(p.Entitlement) == "" || len(p.Scopes) == 0 {
			return errors.New("incomplete center OIDC profile")
		}
		openid := false
		for _, scope := range p.Scopes {
			if scope == "openid" {
				openid = true
			}
			if strings.TrimSpace(scope) != scope || scope == "" || strings.ContainsAny(scope, " \t\r\n") {
				return errors.New("invalid OIDC scope")
			}
		}
		if !openid {
			return errors.New("OIDC scopes require openid")
		}
	default:
		return errors.New("invalid center authentication mode")
	}
	return nil
}

func validURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || strings.Contains(raw, "?") || strings.Contains(raw, "#") {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "http" && (u.Hostname() == "localhost" || ip != nil && ip.IsLoopback())
}

func (p Profile) Fingerprint() string {
	data, _ := json.Marshal(p)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (p Profile) Standard() bool {
	var standard Profile
	if err := json.Unmarshal(defaultJSON, &standard); err != nil {
		panic(err)
	}
	return p.Fingerprint() == standard.Fingerprint()
}
