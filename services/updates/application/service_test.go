package application

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"arion.local/updates/domain"
)

type fakeBucket struct {
	objects map[string]string
	gets    int
	fail    error
}

func (b *fakeBucket) Get(_ context.Context, key string, _ int64) ([]byte, error) {
	b.gets++
	if b.fail != nil {
		return nil, b.fail
	}
	data, ok := b.objects[key]
	if !ok {
		return nil, ErrNotFound
	}
	return []byte(data), nil
}

func (b *fakeBucket) PresignGet(key string, ttl time.Duration, _ time.Time) (string, error) {
	return "https://bucket.example/" + key + "?ttl=" + ttl.String(), nil
}

const sum = "0000000000000000000000000000000000000000000000000000000000000000"

const betaIndex = `{"schema":"astrafoundry.release-channel.v1","product":"arion","channel":"beta","updatedAt":"2026-10-20T00:00:00Z","targets":{"darwin-arm64":{"version":"0.1.0-alpha.6","prerelease":true,"publishedAt":"2026-10-20T00:00:00Z","notes":"- Faster","installer":{"name":"Arion.0.1.0-alpha.6.arm64.dmg","size":10,"sha256":"` + sum + `"},"update":{"name":"arion-0.1.0-alpha.6-darwin-arm64.tar.gz","size":9,"sha256":"` + sum + `"}}}}`

func newService(t *testing.T, bucket *fakeBucket, now *time.Time) *Service {
	t.Helper()
	svc, err := New(Options{Bucket: bucket, PublicURL: "https://api.arion.example", PresignTTL: 5 * time.Minute, CacheTTL: time.Minute, Now: func() time.Time { return *now }})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestManifestChannelsAndCache(t *testing.T) {
	now := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	bucket := &fakeBucket{objects: map[string]string{
		"channels/beta/update-darwin-arm64.json": `{"version":"0.1.0-alpha.6","url":"https://api.arion.example/v1/updates/a.tar.gz","size":9,"signature":"c2ln"}`,
	}}
	svc := newService(t, bucket, &now)
	if _, err := svc.Manifest(context.Background(), domain.Beta, "darwin-arm64"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Manifest(context.Background(), domain.Stable, "darwin-arm64"); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("empty stable channel: %v", err)
	}
	bucket.objects["channels/stable/update-darwin-arm64.json"] = bucket.objects["channels/beta/update-darwin-arm64.json"]
	if _, err := svc.Manifest(context.Background(), domain.Stable, "darwin-arm64"); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("a cached miss expired early: %v", err)
	}
	now = now.Add(time.Minute)
	if _, err := svc.Manifest(context.Background(), domain.Stable, "darwin-arm64"); err != nil {
		t.Fatalf("publication not visible after the cache TTL: %v", err)
	}
	gets := bucket.gets
	bucket.fail = errors.New("bucket unavailable")
	now = now.Add(time.Minute)
	if _, err := svc.Manifest(context.Background(), domain.Beta, "darwin-arm64"); err == nil || errors.Is(err, ErrNoRelease) {
		t.Fatalf("a storage failure must not read as an empty channel: %v", err)
	}
	if bucket.gets != gets+1 {
		t.Fatalf("expected one bucket read, got %d", bucket.gets-gets)
	}
}

func TestReleaseAndDownloads(t *testing.T) {
	now := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	bucket := &fakeBucket{objects: map[string]string{"channels/beta/index.json": betaIndex}}
	svc := newService(t, bucket, &now)
	release, err := svc.Release(context.Background(), domain.Beta)
	if err != nil {
		t.Fatal(err)
	}
	installer := release.Targets["darwin-arm64"].Installer
	if installer.URL != "https://api.arion.example/v1/updates/Arion.0.1.0-alpha.6.arm64.dmg" || installer.SHA256 != sum {
		t.Fatalf("installer = %+v", installer)
	}
	empty, err := svc.Release(context.Background(), domain.Stable)
	if err != nil || len(empty.Targets) != 0 {
		t.Fatalf("empty stable = %+v, %v", empty, err)
	}
	url, err := svc.InstallerURL(context.Background(), domain.Beta, "darwin-arm64")
	if err != nil || url != "https://bucket.example/files/Arion.0.1.0-alpha.6.arm64.dmg?ttl=5m0s" {
		t.Fatalf("installer url = %q, %v", url, err)
	}
	if _, err := svc.InstallerURL(context.Background(), domain.Beta, "windows-amd64"); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("missing target: %v", err)
	}
	if _, err := svc.FileURL(context.Background(), "../secret"); !errors.Is(err, domain.ErrInvalidFile) {
		t.Fatalf("traversal: %v", err)
	}
}

func TestContractFixture(t *testing.T) {
	data, err := os.ReadFile("../../../packages/contracts/releases/channel-index-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.ParseIndex(data, domain.Beta); err != nil {
		t.Fatalf("contract fixture rejected: %v", err)
	}
}
