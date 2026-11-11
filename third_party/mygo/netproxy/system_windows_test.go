package netproxy

import (
	"net/url"
	"testing"
	"unsafe"
)

// TestStructLayout checks the structures WinHTTP fills against their C
// layout on 64-bit Windows.
func TestStructLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit layouts")
	}
	for _, tc := range []struct {
		name      string
		got, want uintptr
	}{
		{"WINHTTP_CURRENT_USER_IE_PROXY_CONFIG", unsafe.Sizeof(ieProxyConfig{}), 32},
		{"WINHTTP_AUTOPROXY_OPTIONS", unsafe.Sizeof(autoProxyOptions{}), 32},
		{"WINHTTP_AUTOPROXY_OPTIONS.fAutoLogonIfChallenged", unsafe.Offsetof(autoProxyOptions{}.autoLogonIfChallenged), 28},
		{"WINHTTP_PROXY_INFO", unsafe.Sizeof(proxyInfo{}), 24},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// TestSystemWindows asks WinHTTP for the proxy of a URL: whatever the
// machine's settings, the answer is a proxy Go can use or none.
func TestSystemWindows(t *testing.T) {
	if winhttp.get(loadWinHTTP) == nil {
		t.Fatal("WinHTTP did not load")
	}
	p := system(&url.URL{Scheme: "https", Host: "example.com", Path: "/"})
	if p != nil && p.Host == "" {
		t.Errorf("system = %v, a proxy without a host", p)
	}
	t.Logf("system proxy: %v", p)
}
