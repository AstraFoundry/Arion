package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arion.local/updates/application"
)

type bucket struct {
	objects map[string]string
	fail    bool
}

func (b *bucket) Get(_ context.Context, key string, _ int64) ([]byte, error) {
	if b.fail {
		return nil, errors.New("down")
	}
	if data, ok := b.objects[key]; ok {
		return []byte(data), nil
	}
	return nil, application.ErrNotFound
}

func (b *bucket) PresignGet(key string, _ time.Duration, _ time.Time) (string, error) {
	return "https://bucket.example/" + key + "?signed", nil
}

const sum = "0000000000000000000000000000000000000000000000000000000000000000"

func server(t *testing.T, b *bucket) http.Handler {
	t.Helper()
	svc, err := application.New(application.Options{Bucket: b, PublicURL: "https://api.arion.example", PresignTTL: time.Minute, CacheTTL: time.Nanosecond, Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	return New(Deps{Service: svc, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: time.Now})
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestRoutes(t *testing.T) {
	b := &bucket{objects: map[string]string{
		"channels/beta/update-darwin-arm64.json": `{"version":"0.1.0-alpha.6","url":"https://api.arion.example/v1/updates/a.tar.gz","size":9,"signature":"c2ln","deltas":[{"from":"0.1.0","url":"https://api.arion.example/v1/updates/a.delta","size":4,"signature":"c2ln"}]}`,
		"channels/beta/index.json":               `{"schema":"astrafoundry.release-channel.v1","product":"arion","channel":"beta","updatedAt":"2026-10-20T00:00:00Z","targets":{"darwin-arm64":{"version":"0.1.0-alpha.6","prerelease":true,"publishedAt":"2026-10-20T00:00:00Z","notes":"","installer":{"name":"Arion.0.1.0-alpha.6.arm64.dmg","size":10,"sha256":"` + sum + `"},"update":{"name":"a.tar.gz","size":9,"sha256":"` + sum + `"}}}}`,
	}}
	h := server(t, b)

	rec := get(h, "/v1/updates/update-darwin-arm64.json?channel=beta")
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("beta manifest: %d %s", rec.Code, rec.Body)
	}
	rec = get(h, "/v1/updates/update-darwin-arm64.json")
	var body struct{ Code string }
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusNotFound || body.Code != "NO_RELEASE" {
		t.Fatalf("empty stable channel: %d %s", rec.Code, rec.Body)
	}
	if rec := get(h, "/v1/updates/update-darwin-arm64.json?channel=nightly"); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown channel: %d", rec.Code)
	}
	rec = get(h, "/v1/updates/a.tar.gz")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://bucket.example/files/a.tar.gz?signed" {
		t.Fatalf("file redirect: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	rec = get(h, "/v1/updates/a.delta")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://bucket.example/files/a.delta?signed" {
		t.Fatalf("delta redirect: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	rec = get(h, "/v1/updates/staged.tar.gz")
	var unpublished struct{ Code string }
	_ = json.Unmarshal(rec.Body.Bytes(), &unpublished)
	if rec.Code != http.StatusNotFound || unpublished.Code != "NO_RELEASE" {
		t.Fatalf("unpublished file: %d %s", rec.Code, rec.Body)
	}
	if rec := get(h, "/v1/updates/..%2Fsecret"); rec.Code != http.StatusBadRequest {
		t.Fatalf("traversal: %d", rec.Code)
	}
	rec = get(h, "/v1/downloads/beta/darwin-arm64")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://bucket.example/files/Arion.0.1.0-alpha.6.arm64.dmg?signed" {
		t.Fatalf("download: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if rec := get(h, "/v1/downloads/stable/darwin-arm64"); rec.Code != http.StatusNotFound {
		t.Fatalf("stable download: %d", rec.Code)
	}
	rec = get(h, "/v1/releases/beta")
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("release: %d", rec.Code)
	}

	b.fail = true
	if rec := get(h, "/v1/updates/update-darwin-arm64.json?channel=beta"); rec.Code != http.StatusBadGateway {
		t.Fatalf("storage failure: %d", rec.Code)
	}
}
