//go:build arion_e2e && darwin

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/macos"
)

// SCN-A04: the macOS target adapter reads the focused web page's host from
// real browsers, and reads nothing when scene selection is off. It needs
// Accessibility access for the process running the test, granted by a
// person, so it runs only with ARION_E2E_SITE=1.
func TestCaptureReadsWebSite(t *testing.T) {
	if os.Getenv("ARION_E2E_SITE") != "1" {
		t.Skip("set ARION_E2E_SITE=1 after granting Accessibility to the test runner")
	}
	if (macos.Permissions{}).Accessibility() != macos.AuthGranted {
		t.Fatal("ARION_E2E_SITE=1 requires Accessibility access for the test runner")
	}
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><title>Arion site fixture</title><textarea autofocus></textarea>`)
	}))
	defer page.Close()
	browsers := map[string]string{"Safari": "/Applications/Safari.app", "Chrome": "/Applications/Google Chrome.app"}
	for name, path := range browsers {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(path); err != nil {
				t.Skipf("%s is not installed", name)
			}
			if err := exec.Command("open", "-a", path, page.URL).Run(); err != nil {
				t.Fatal(err)
			}
			targets := macos.NewTargets()
			read := true
			targets.ReadSite = func() bool { return read }
			var snap contracts.TargetSnapshot
			deadline := time.Now().Add(15 * time.Second)
			for time.Now().Before(deadline) {
				var err error
				snap, err = targets.Capture(context.Background(), contracts.Identity{SessionID: "site", Generation: 1})
				if err == nil && snap.Site == "127.0.0.1" {
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
			if snap.Site != "127.0.0.1" || snap.Field != domain.FieldText {
				t.Fatalf("captured app=%q field=%q site=%q", snap.Application, snap.Field, snap.Site)
			}
			if route := targets.RouteOf(snap.Target); route != macos.RouteClipboard {
				t.Fatalf("web field route %q, want clipboard", route)
			}
			read = false
			off, err := targets.Capture(context.Background(), contracts.Identity{SessionID: "site", Generation: 2})
			if err != nil || off.Site != "" {
				t.Fatalf("site read with scene selection off: %q (%v)", off.Site, err)
			}
			if route := targets.RouteOf(off.Target); route != macos.RouteClipboard {
				t.Fatalf("web field route with scene selection off %q, want clipboard", route)
			}
		})
	}
}
