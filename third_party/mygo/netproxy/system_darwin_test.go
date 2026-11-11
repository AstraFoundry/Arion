//go:build darwin && (amd64 || arm64)

package netproxy

import (
	"net/url"
	"runtime"
	"slices"
	"testing"
)

// TestSystemDarwin asks CFNetwork for the proxy of a URL: whatever the
// machine's settings, the answer is a proxy Go can use or none.
func TestSystemDarwin(t *testing.T) {
	if cf() == nil {
		t.Fatal("CFNetwork did not load")
	}
	p := system(&url.URL{Scheme: "https", Host: "example.com", Path: "/"})
	if p != nil && p.Host == "" {
		t.Errorf("system = %v, a proxy without a host", p)
	}
	t.Logf("system proxy: %v", p)
}

// TestPACScript runs a setup script on this thread's run loop and reads
// the proxies it chose.
func TestPACScript(t *testing.T) {
	a := cf()
	if a == nil {
		t.Fatal("CFNetwork did not load")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	script := a.stringCreate(0, `function FindProxyForURL(url, host) { return "PROXY p.example:8080; SOCKS s.example:1080; DIRECT"; }`, cfStringEncodingUTF8)
	defer a.release(script)
	s := a.stringCreate(0, "https://example.com/", cfStringEncodingUTF8)
	defer a.release(s)
	target := a.urlCreate(0, s, 0)
	defer a.release(target)
	got := a.runPAC(script, target, true)
	want := []string{"http://p.example:8080", "socks5://s.example:1080", "direct://"}
	if !slices.Equal(got, want) {
		t.Errorf("runPAC = %q, want %q", got, want)
	}
}
