//go:build darwin

package mygo

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/update"
)

// TestPreparedUpdateNativeLifecycle builds and runs real native fixture bundles.
// It is opt-in because AppKit requires a logged-in macOS desktop session.
func TestPreparedUpdateNativeLifecycle(t *testing.T) {
	if os.Getenv("MYGO_UPDATER_E2E") != "1" {
		t.Skip("set MYGO_UPDATER_E2E=1 in a macOS desktop session")
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString(pub)
	var mu sync.Mutex
	var manifest update.Manifest
	var archive []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/update.json":
			json.NewEncoder(w).Encode(manifest)
		case "/app.tar.gz":
			w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	build := func(name, version string) string {
		t.Helper()
		bundle := filepath.Join(root, name, "Updater Fixture.app")
		executable := filepath.Join(bundle, "Contents", "MacOS", "fixture")
		if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
			t.Fatal(err)
		}
		resource := filepath.Join(bundle, "Contents", "Resources")
		if err := os.MkdirAll(resource, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(resource, "version"), []byte(version), 0o644); err != nil {
			t.Fatal(err)
		}
		plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>si.astrafoundry.arion.updater-fixture</string><key>CFBundleName</key><string>Updater Fixture</string><key>CFBundleExecutable</key><string>fixture</string><key>CFBundlePackageType</key><string>APPL</string><key>CFBundleVersion</key><string>%s</string><key>CFBundleShortVersionString</key><string>%s</string><key>LSUIElement</key><true/></dict></plist>`, version, version)
		if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
			t.Fatal(err)
		}
		flags := "-X github.com/egoist/mygo.production=1 -X github.com/egoist/mygo.packageVersion=" + version + " -X github.com/egoist/mygo.packageUpdateFeed=" + server.URL + "/update.json -X github.com/egoist/mygo.packageUpdateKey=" + key
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "-ldflags", flags, "-o", executable, "./internal/updaterfixture")
		cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build fixture: %v\n%s", err, out)
		}
		sign := exec.CommandContext(ctx, "/usr/bin/codesign", "--force", "--deep", "--sign", "-", bundle)
		if out, err := sign.CombinedOutput(); err != nil {
			t.Fatalf("sign fixture: %v\n%s", err, out)
		}
		return bundle
	}
	oldBundle := build("installed", "1.0.0")
	newBundle := build("release", "2.0.0")
	var compressed bytes.Buffer
	if err := update.WriteArchive(&compressed, filepath.Dir(newBundle), []string{filepath.Base(newBundle)}); err != nil {
		t.Fatal(err)
	}
	archive = compressed.Bytes()
	digest := sha256.Sum256(archive)
	manifest = update.Manifest{Version: "2.0.0", URL: server.URL + "/app.tar.gz", Size: int64(len(archive)), Signature: update.Sign(priv, digest[:])}
	state := filepath.Join(root, "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(bundle, action string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "Contents", "MacOS", "fixture"))
		cmd.Env = append(os.Environ(), "MYGO_UPDATER_FIXTURE_DIR="+state, "MYGO_UPDATER_FIXTURE_ACTION="+action)
		// Relaunch inherits these files; a real file avoids a pipe waiting for the
		// detached child after the parent has already completed its native quit.
		output, err := os.Create(filepath.Join(state, action+".log"))
		if err != nil {
			t.Fatal(err)
		}
		defer output.Close()
		cmd.Stdout, cmd.Stderr = output, output
		if err := cmd.Run(); err != nil {
			b, _ := os.ReadFile(output.Name())
			t.Fatalf("native fixture %s: %v\n%s", action, err, b)
		}
		if b, err := os.ReadFile(filepath.Join(state, "failure.json")); err == nil {
			t.Fatalf("native fixture failure: %s", b)
		}
	}
	observe := func(event, version string) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for {
			b, err := os.ReadFile(filepath.Join(state, event+".json"))
			if err == nil {
				var result struct{ Version, Resource string }
				if err := json.Unmarshal(b, &result); err != nil {
					t.Fatal(err)
				}
				if result.Version != version || result.Resource != version {
					t.Fatalf("%s version/resource mismatch: %s", event, b)
				}
				t.Logf("native %s observation: %s", event, b)
				return
			}
			if b, err := os.ReadFile(filepath.Join(state, "failure.json")); err == nil {
				t.Fatalf("native fixture failure: %s", b)
			}
			if time.Now().After(deadline) {
				t.Fatalf("no native %s observation: %v", event, err)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	run(oldBundle, "prepare")
	observe("prepared", "1.0.0")
	oldResource := filepath.Join(oldBundle, "Contents", "Resources", "version")
	if b, err := os.ReadFile(oldResource); err != nil || string(b) != "1.0.0" {
		t.Fatalf("prepare changed installed resources: %q %v", b, err)
	}
	run(oldBundle, "apply")
	observe("applied", "1.0.0")
	observe("relaunched", "2.0.0")
	if b, err := os.ReadFile(oldResource); err != nil || string(b) != "2.0.0" {
		t.Fatalf("apply did not replace installed resources: %q %v", b, err)
	}
	// A separate original bundle receives an invalid signature from the same
	// local feed. It must remain executable with its original resource bytes.
	rejectedBundle := build("signature-rejected", "1.0.0")
	mu.Lock()
	manifest.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	mu.Unlock()
	run(rejectedBundle, "bad-signature")
	observe("rejected", "1.0.0")
	b, err := os.ReadFile(filepath.Join(state, "rejected.json"))
	if err != nil || !strings.Contains(string(b), "not signed") {
		t.Fatalf("wrong rejection: %s %v", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(rejectedBundle, "Contents", "Resources", "version")); err != nil || string(b) != "1.0.0" {
		t.Fatalf("signature failure replaced resources: %q %v", b, err)
	}
}
