//go:build darwin

package netproxy

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

const sample = `<dictionary> {
  ExceptionsList : <array> {
    0 : 192.168.0.0/16
    1 : *.local
    2 : example.internal
  }
  FTPPassive : 1
  HTTPEnable : 1
  HTTPPort : 7890
  HTTPProxy : 127.0.0.1
  HTTPSEnable : 1
  HTTPSPort : 7890
  HTTPSProxy : 127.0.0.1
}`

func TestSystemProxyFromScutil(t *testing.T) {
	s := parseScutil(sample)
	target, _ := url.Parse("https://huggingface.co/onnx-community/x")
	if got := s.For(target); got == nil || got.String() != "http://127.0.0.1:7890" {
		t.Fatalf("huggingface proxy = %v", got)
	}
	for _, direct := range []string{"https://printer.local/", "https://example.internal/a", "http://127.0.0.1:8080/", "http://localhost/"} {
		u, _ := url.Parse(direct)
		if got := s.For(u); got != nil {
			t.Fatalf("%s should bypass the proxy, got %v", direct, got)
		}
	}
}

func TestDisabledProxyIsDirect(t *testing.T) {
	s := parseScutil("<dictionary> {\n  HTTPSEnable : 0\n  HTTPSProxy : 10.0.0.1\n  HTTPSPort : 8080\n}")
	target, _ := url.Parse("https://huggingface.co/")
	if got := s.For(target); got != nil {
		t.Fatalf("disabled proxy used: %v", got)
	}
}

// With no environment variables, requests follow the live system settings.
func TestProxyFallsBackToSystem(t *testing.T) {
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "NO_PROXY", "no_proxy"} {
		t.Setenv(k, "")
	}
	mu.Lock()
	cached, fetched = parseScutil(sample), time.Now()
	mu.Unlock()
	req, _ := http.NewRequest(http.MethodGet, "https://huggingface.co/", nil)
	got, err := Proxy(req)
	if err != nil || got == nil || got.Host != "127.0.0.1:7890" {
		t.Fatalf("got %v, %v", got, err)
	}
}
