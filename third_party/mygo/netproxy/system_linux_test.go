//go:build linux && (amd64 || arm64)

package netproxy

import (
	"net/url"
	"testing"
)

// TestSystemLinux asks GIO, when it is installed, for the proxy of a URL:
// whatever the machine's settings, the answer is a proxy Go can use or
// none.
func TestSystemLinux(t *testing.T) {
	if gio.get(loadGIO) == nil {
		t.Log("no GIO: requests go direct")
	}
	p := system(&url.URL{Scheme: "https", Host: "example.com", Path: "/"})
	if p != nil && p.Host == "" {
		t.Errorf("system = %v, a proxy without a host", p)
	}
	t.Logf("system proxy: %v", p)
}
