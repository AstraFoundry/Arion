package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLimiterForgetsIdleIdentities(t *testing.T) {
	l := &limiter{limit: 1, hits: map[string][]time.Time{}}
	start := time.Unix(1_700_000_000, 0)
	for i := range 1000 {
		if !l.allow(fmt.Sprintf("device-%d", i), start) {
			t.Fatalf("first request of device-%d was limited", i)
		}
	}
	if l.allow("device-0", start.Add(time.Second)) {
		t.Fatal("second request inside the window was allowed")
	}
	if !l.allow("device-new", start.Add(61*time.Second)) {
		t.Fatal("request after the window was limited")
	}
	if len(l.hits) != 1 {
		t.Fatalf("limiter retained %d identities, want only the active one", len(l.hits))
	}
}

func TestInvalidDeviceHeaderIsRejectedBeforeLimiting(t *testing.T) {
	env := newNoop(t, 1<<20, time.Hour, 100)
	for _, id := range []string{strings.Repeat("a", 129), "device id", "device/../x"} {
		rec := env.raw(http.MethodGet, "/v1/sync/pull", "", id, env.at, "c2ln", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("device %q: %d %s", id, rec.Code, rec.Body.Bytes())
		}
	}
}
