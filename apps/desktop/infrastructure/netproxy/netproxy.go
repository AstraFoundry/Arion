// Package netproxy chooses the proxy for outbound downloads: explicit
// environment variables first, then the operating system's proxy settings.
// Apps started from the Finder or Dock receive no shell environment, so
// without the system settings they would bypass a proxy the user set up.
package netproxy

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Settings are the system's manual HTTP/HTTPS proxies and bypass list.
type Settings struct {
	HTTP, HTTPS *url.URL
	Exceptions  []string
}

var (
	mu      sync.Mutex
	cached  Settings
	fetched time.Time
)

// cacheFor bounds how stale system settings may be; users switch proxies
// without restarting the app.
const cacheFor = 30 * time.Second

// Proxy satisfies http.Transport.Proxy.
func Proxy(req *http.Request) (*url.URL, error) {
	if u, err := http.ProxyFromEnvironment(req); u != nil || err != nil || hasEnvironment() {
		return u, err
	}
	mu.Lock()
	if time.Since(fetched) > cacheFor {
		cached, fetched = system(), time.Now()
	}
	s := cached
	mu.Unlock()
	return s.For(req.URL), nil
}

func hasEnvironment() bool {
	for _, key := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		if strings.TrimSpace(getenv(key)) != "" {
			return true
		}
	}
	return false
}

// For returns the proxy for target, or nil to connect directly.
func (s Settings) For(target *url.URL) *url.URL {
	host := strings.ToLower(target.Hostname())
	if host == "localhost" || net.ParseIP(host).IsLoopback() {
		return nil
	}
	for _, exception := range s.Exceptions {
		e := strings.ToLower(strings.TrimSpace(exception))
		e = strings.TrimPrefix(e, "*")
		if e == "" {
			continue
		}
		if host == strings.TrimPrefix(e, ".") || (strings.HasPrefix(e, ".") && strings.HasSuffix(host, e)) {
			return nil
		}
	}
	if target.Scheme == "https" {
		return s.HTTPS
	}
	return s.HTTP
}
