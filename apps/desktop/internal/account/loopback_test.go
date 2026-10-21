package account

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLoopbackServesCallbackPage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lb, redirect, err := startLoopback(ctx, func() []byte { return []byte("<p>page</p>") })
	if err != nil {
		t.Fatal(err)
	}
	defer lb.Close()
	res, err := http.Get(redirect + "?code=c&state=s")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || string(body) != "<p>page</p>" {
		t.Fatalf("status %d body %q", res.StatusCode, body)
	}
	for header, want := range map[string]string{
		"Cache-Control":          "no-store",
		"Content-Type":           "text/html; charset=utf-8",
		"Referrer-Policy":        "no-referrer",
		"X-Content-Type-Options": "nosniff",
	} {
		if got := res.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") || strings.Contains(csp, "script-src") {
		t.Errorf("Content-Security-Policy = %q", csp)
	}
	cb, err := lb.Wait(ctx)
	if err != nil || cb.code != "c" || cb.state != "s" {
		t.Fatalf("callback %+v, %v", cb, err)
	}
}
