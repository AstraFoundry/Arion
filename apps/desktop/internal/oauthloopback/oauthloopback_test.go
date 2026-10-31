package oauthloopback

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestStartServesOneCallbackOnItsPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, redirect, err := Start(ctx, "127.0.0.1:0", "/auth/callback", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !strings.HasPrefix(redirect, "http://127.0.0.1:") || !strings.HasSuffix(redirect, "/auth/callback") {
		t.Fatalf("redirect %q", redirect)
	}
	if res, err := http.Get(strings.TrimSuffix(redirect, "/auth/callback") + "/other"); err != nil || res.StatusCode != http.StatusNotFound {
		t.Fatalf("other path: %v %v", res, err)
	}
	res, err := http.Get(redirect + "?code=c&state=s")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	cb, err := s.Wait(ctx)
	if err != nil || cb != (Callback{State: "s", Code: "c"}) {
		t.Fatalf("callback %+v %v", cb, err)
	}
	// A busy fixed port is an error, not another port.
	addr := strings.TrimPrefix(strings.TrimSuffix(redirect, "/auth/callback"), "http://")
	if _, _, err := Start(ctx, addr, "/auth/callback", nil, ""); err == nil {
		t.Fatal("second listener on a busy port")
	}
}

func TestMismatchedStateDoesNotConsumeTheCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, redirect, err := Start(ctx, "127.0.0.1:0", "/auth/callback", nil, "expected-state")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := http.Get(redirect + "?code=stolen&state=forged")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("forged callback status %d", res.StatusCode)
	}
	res, err = http.Get(redirect + "?code=real&state=expected-state")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	cb, err := s.Wait(ctx)
	if err != nil || cb != (Callback{State: "expected-state", Code: "real"}) {
		t.Fatalf("callback %+v %v", cb, err)
	}
}

func TestMatchState(t *testing.T) {
	if !MatchState("a", "a") || MatchState("a", "b") || MatchState("", "") {
		t.Fatal("state comparison")
	}
}
