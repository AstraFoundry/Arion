package netproxy

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestFirstSupported(t *testing.T) {
	for _, tc := range []struct {
		list []string
		want string
	}{
		{nil, ""},
		{[]string{"direct://"}, ""},
		{[]string{"http://proxy:3128"}, "http://proxy:3128"},
		{[]string{"HTTP://proxy:3128/"}, "http://proxy:3128"},
		{[]string{"https://proxy:443"}, "https://proxy:443"},
		{[]string{"http://user:pw@proxy:3128"}, "http://user:pw@proxy:3128"},
		// GNOME's SOCKS setting: socks5, then socks4a and socks4.
		{[]string{"socks5://s:1080", "socks4a://s:1080", "socks4://s:1080"}, "socks5://s:1080"},
		{[]string{"socks5h://s:1080"}, "socks5h://s:1080"},
		{[]string{"socks://s:1080"}, "socks5://s:1080"},
		{[]string{"socks4://s:1080", "http://p:80"}, "http://p:80"},
		{[]string{"socks4://s:1080", "direct://", "http://p:80"}, ""},
		{[]string{"ftp://f:21", "", "http://"}, ""},
		{[]string{"http://[::1]:8080"}, "http://[::1]:8080"},
	} {
		if got := urlString(firstSupported(tc.list)); got != tc.want {
			t.Errorf("firstSupported(%q) = %q, want %q", tc.list, got, tc.want)
		}
	}
}

func TestWinProxy(t *testing.T) {
	for _, tc := range []struct {
		list, scheme, want string
	}{
		{"", "https", ""},
		{"127.0.0.1:7890", "https", "http://127.0.0.1:7890"},
		{"127.0.0.1:7890", "http", "http://127.0.0.1:7890"},
		{"http://proxy:8080", "https", "http://proxy:8080"},
		{"proxy", "http", "http://proxy"},
		{"http=h:80;https=s:443;socks=k:1080", "http", "http://h:80"},
		{"http=h:80;https=s:443;socks=k:1080", "https", "http://s:443"},
		{"http=h:80;socks=k:1080", "https", "socks5://k:1080"},
		{"ftp=f:21", "https", ""},
		{"HTTPS=s:443 http=h:80", "https", "http://s:443"},
		{"https=https://s:443", "https", "https://s:443"},
		// WinHTTP's answer from a setup script: proxies to try in turn.
		{"a:8080; b:8080", "https", "http://a:8080"},
		{"socks4://k:1080;b:80", "http", "http://b:80"},
	} {
		if got := urlString(winProxy(tc.list, tc.scheme)); got != tc.want {
			t.Errorf("winProxy(%q, %q) = %q, want %q", tc.list, tc.scheme, got, tc.want)
		}
	}
}

func TestWinBypassed(t *testing.T) {
	for _, tc := range []struct {
		list, host string
		want       bool
	}{
		{"", "example.com", false},
		{"<local>", "intranet", true},
		{"<local>", "example.com", false},
		{"<local>", "::1", false},
		{"*.corp.com", "a.b.corp.com", true},
		{"*.corp.com", "corp.com", false},
		{"*.CORP.com", "Wiki.Corp.Com", true},
		{".corp.com", "wiki.corp.com", true},
		{"corp.com", "corp.com", true},
		{"corp.com", "wiki.corp.com", false},
		{"10.*;192.168.*", "10.1.2.3", true},
		{"10.*;192.168.*", "192.168.0.7", true},
		{"10.*;192.168.*", "11.0.0.1", false},
		{"10.0.0.0/8", "10.9.9.9", true},
		{"10.0.0.0/8", "example.com", false},
		{"fe80::/10", "fe80::1", true},
		{"foo.com:8080", "foo.com", true},
		{"*.a.com <local>\t*.b.com", "x.b.com", true},
		{"<-loopback>;*.a.com", "b.com", false},
		{"http://*.a.com", "x.a.com", true},
		{"*", "anything.example", true},
	} {
		if got := winBypassed(tc.list, tc.host); got != tc.want {
			t.Errorf("winBypassed(%q, %q) = %v, want %v", tc.list, tc.host, got, tc.want)
		}
	}
}

func TestWinStatic(t *testing.T) {
	u := func(s string) *url.URL { p, _ := url.Parse(s); return p }
	if got := urlString(winStatic("p:80", "<local>;*.corp.com", u("https://wiki.corp.com/"))); got != "" {
		t.Errorf("bypassed host got proxy %q", got)
	}
	if got := urlString(winStatic("p:80", "<local>;*.corp.com", u("https://example.com/"))); got != "http://p:80" {
		t.Errorf("got %q, want http://p:80", got)
	}
}

// clearEnv unsets the variables that hand the decision to the
// environment.
func clearEnv(t *testing.T) {
	for _, name := range envNames {
		t.Setenv(name, "")
	}
}

// fakeSystem replaces the system's settings with proxy, counting the
// questions, and the clock with a fake one.
func fakeSystem(t *testing.T, proxy string) (asked *int, clock *time.Time) {
	asked, clock = new(int), new(time.Time)
	*clock = time.Unix(1000, 0)
	saved, savedNow := systemProxy, now
	systemProxy = func(target *url.URL) *url.URL {
		*asked++
		if target.Path != "/" || target.RawQuery != "" {
			t.Errorf("system asked about %s, want scheme and host only", target)
		}
		p, _ := url.Parse(proxy)
		return p
	}
	now = func() time.Time { return *clock }
	cache.Lock()
	cache.m = nil
	cache.Unlock()
	t.Cleanup(func() { systemProxy, now = saved, savedNow })
	return asked, clock
}

func proxyFor(t *testing.T, rawURL string) string {
	t.Helper()
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	return urlString(p)
}

func TestProxyLoopbackIsDirect(t *testing.T) {
	clearEnv(t)
	asked, _ := fakeSystem(t, "http://proxy:3128")
	for _, u := range []string{
		"http://localhost:5173/", "http://LocalHost/", "http://app.localhost/",
		"http://127.0.0.1:8080/", "http://127.4.5.6/", "http://[::1]:3000/",
	} {
		if got := proxyFor(t, u); got != "" {
			t.Errorf("Proxy(%s) = %q, want direct", u, got)
		}
	}
	if *asked != 0 {
		t.Errorf("system asked %d times about loopback hosts", *asked)
	}
	t.Setenv("HTTPS_PROXY", "http://env:1")
	if got := proxyFor(t, "https://localhost/"); got != "" {
		t.Errorf("Proxy(localhost) with HTTPS_PROXY = %q, want direct", got)
	}
}

func TestProxyEnvironmentWins(t *testing.T) {
	saved := envProxy
	t.Cleanup(func() { envProxy = saved })
	env, _ := url.Parse("http://env:8080")
	envProxy = func(*http.Request) (*url.URL, error) { return env, nil }
	for _, name := range envNames {
		t.Run(name, func(t *testing.T) {
			clearEnv(t)
			asked, _ := fakeSystem(t, "http://system:3128")
			t.Setenv(name, "x")
			if got := proxyFor(t, "https://example.com/"); got != "http://env:8080" {
				t.Errorf("Proxy = %q, want the environment's", got)
			}
			if *asked != 0 {
				t.Errorf("system asked %d times", *asked)
			}
		})
	}
}

func TestProxySystemCached(t *testing.T) {
	clearEnv(t)
	asked, clock := fakeSystem(t, "http://system:3128")
	for range 3 {
		if got := proxyFor(t, "https://example.com/a?b=c"); got != "http://system:3128" {
			t.Fatalf("Proxy = %q, want the system's", got)
		}
	}
	if *asked != 1 {
		t.Errorf("system asked %d times, want 1", *asked)
	}
	proxyFor(t, "http://example.com/")
	proxyFor(t, "https://example.com:8443/")
	if *asked != 3 {
		t.Errorf("system asked %d times for 3 schemes and hosts, want 3", *asked)
	}
	*clock = clock.Add(ttl)
	proxyFor(t, "https://example.com/")
	if *asked != 4 {
		t.Errorf("system asked %d times once the answer expired, want 4", *asked)
	}

	// The answer is a copy: changing it changes no other.
	req, _ := http.NewRequest("GET", "https://example.com/", nil)
	p, _ := Proxy(req)
	p.Host = "changed"
	if got := proxyFor(t, "https://example.com/"); got != "http://system:3128" {
		t.Errorf("Proxy after changing its answer = %q", got)
	}
}

func TestTransport(t *testing.T) {
	tr := Transport()
	if tr != Transport() {
		t.Error("Transport is not shared")
	}
	if tr.Proxy == nil || tr == http.DefaultTransport {
		t.Error("Transport is not a clone of http.DefaultTransport with Proxy")
	}
}

func urlString(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}
