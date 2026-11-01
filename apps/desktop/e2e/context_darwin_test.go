//go:build arion_e2e && darwin

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/macos"
)

// CTX-A04: on a real browser window the target adapter reads the text
// before the cursor and the window's recognized text, and neither with its
// gates closed. Needs Accessibility for the process
// running the test, granted by a person, so it runs only with
// ARION_E2E_CONTEXT=1.
func TestReadContextFromWindow(t *testing.T) {
	if os.Getenv("ARION_E2E_CONTEXT") != "1" {
		t.Skip("set ARION_E2E_CONTEXT=1 after granting Accessibility to the test runner")
	}
	permissions := macos.Permissions{}
	if permissions.Accessibility() != macos.AuthGranted {
		t.Fatal("ARION_E2E_CONTEXT=1 requires Accessibility access for the test runner")
	}
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><title>Arion context fixture</title><h1 style="font:32px sans-serif">Kubernetes migration review</h1>`+
			`<textarea id="t" style="font:20px sans-serif" autofocus>Meeting with Siobhan about chong g y xia tomorrow</textarea>`+
			`<script>const t=document.getElementById('t');t.focus();t.selectionStart="Meeting with Siobhan about ".length;t.selectionEnd=t.selectionStart+"chong g y xia".length</script>`)
	}))
	defer page.Close()
	if err := exec.Command("open", "-a", "/Applications/Safari.app", page.URL).Run(); err != nil {
		t.Fatal(err)
	}
	targets := macos.NewTargets()
	open := true
	targets.ReadCursor = func() bool { return open }
	targets.ReadScreen = func() bool { return open }
	var snap contracts.ContextSnapshot
	var lastCode domain.ErrorCode
	var browser bool
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		target, err := targets.Capture(context.Background(), contracts.Identity{SessionID: "ctx", Generation: 1})
		if err == nil {
			browser = target.Application == "com.apple.Safari"
			if !browser {
				targets.Discard(target.Target)
				time.Sleep(300 * time.Millisecond)
				continue
			}
			snap, err = targets.ReadContext(context.Background(), target.Identity, target.Target)
			lastCode = domain.CodeOf(err)
			targets.Discard(target.Target)
			if strings.Contains(snap.Before, "Siobhan") && strings.Contains(snap.Screen, "Kubernetes") && snap.Selected == "chong g y xia" && snap.After == " tomorrow" {
				break
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if !strings.Contains(snap.Before, "Siobhan") || !strings.Contains(snap.Screen, "Kubernetes") || snap.Selected != "chong g y xia" || snap.After != " tomorrow" {
		t.Fatalf("fixture context unavailable: browser=%v safe=%v code=%s beforeMatch=%v selectedMatch=%v afterMatch=%v screenMatch=%v", browser, snap.SafeToRead, lastCode, strings.Contains(snap.Before, "Siobhan"), snap.Selected == "chong g y xia", snap.After == " tomorrow", strings.Contains(snap.Screen, "Kubernetes"))
	}
	t.Logf("capture %.0f ms", snap.ScreenCaptureMs)
	open = false
	target, err := targets.Capture(context.Background(), contracts.Identity{SessionID: "ctx", Generation: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer targets.Discard(target.Target)
	closed, _ := targets.ReadContext(context.Background(), target.Identity, target.Target)
	if closed.Before != "" || closed.Screen != "" || closed.Selected != "" {
		t.Fatal("context read with its gates closed")
	}
}
