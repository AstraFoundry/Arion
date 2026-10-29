//go:build arion_e2e && darwin

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/macos"
)

// Exercises the production adapter against owned AppKit and WebKit editors.
// No message is sent and no input-source or permission setting is changed.
func TestGuardedSystemPaste(t *testing.T) {
	if os.Getenv("ARION_E2E_PASTE") != "1" {
		t.Skip("set ARION_E2E_PASTE=1 to exercise real system paste")
	}
	if (macos.Permissions{}).Accessibility() != macos.AuthGranted {
		t.Fatal("real paste E2E requires existing Accessibility access for the test runner")
	}
	request := startEditorFixture(t)
	request("clipboardSave", nil)
	t.Cleanup(func() { request("clipboardRestore", nil) })
	const sentinel = "Arion owned E2E clipboard"
	request("clipboardSet", map[string]any{"text": sentinel})
	targets := macos.NewTargets()
	id := contracts.Identity{SessionID: "owned-paste", Generation: 1}
	capture := func() contracts.TargetSnapshot {
		t.Helper()
		var snap contracts.TargetSnapshot
		eventually(t, "owned editor focused", func() bool {
			var err error
			snap, err = targets.Capture(context.Background(), id)
			return err == nil && snap.Target != ""
		})
		return snap
	}
	commit := func(ctx context.Context, snap contracts.TargetSnapshot, text string) contracts.CommitResult {
		t.Helper()
		result, err := targets.Commit(ctx, contracts.CommitRequest{Identity: id, Target: snap.Target, Text: text, SourceRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	focus := func(target string) {
		request("focus", map[string]any{"target": target})
		eventually(t, "fixture focus", func() bool {
			s := request("state", nil)
			return s.Active && (target == "web" && s.Web.Focused || s.Focused == target || target == "none" && s.Focused == "")
		})
	}
	checkClipboard := func() {
		t.Helper()
		if !request("clipboardCheck", map[string]any{"text": sentinel}).ClipboardOwned {
			t.Fatal("clipboard sentinel not restored; cleanup will preserve newer clipboard")
		}
	}
	t.Run("unicode-selection-replacement-once", func(t *testing.T) {
		request("set", map[string]any{"text": "prefix OLD suffix", "location": 7, "length": 3})
		focus("A")
		snap := capture()
		text := "中文🙂e\u0301\nsecond line"
		result := commit(context.Background(), snap, text)
		if result.Outcome != contracts.Unknown || result.Failure != "" {
			t.Fatalf("paste result: %+v", result)
		}
		if !within(5*time.Second, func() bool { s := request("state", nil); return s.A.Text == "prefix "+text+" suffix" && s.A.Edits == 1 }) {
			t.Fatalf("native input observation: %+v", request("state", nil))
		}
		if duplicate := commit(context.Background(), snap, text); duplicate.Outcome != contracts.NotApplied {
			t.Fatal("duplicate target reused")
		}
		checkClipboard()
	})
	t.Run("focus-change", func(t *testing.T) {
		request("set", map[string]any{"text": "unchanged", "location": 9})
		focus("A")
		snap := capture()
		focus("B")
		const text = "into B"
		result := commit(context.Background(), snap, text)
		if result.Outcome != contracts.Unknown || result.Failure != "" {
			t.Fatalf("focus change: %+v", result)
		}
		if !within(5*time.Second, func() bool {
			s := request("state", nil)
			return s.A.Text == "unchanged" && s.A.Edits == 0 && s.B.Text == text && s.B.Edits == 1
		}) {
			t.Fatalf("focus change observation: %+v", request("state", nil))
		}
		checkClipboard()
	})
	t.Run("selection-change", func(t *testing.T) {
		request("set", map[string]any{"text": "unchanged", "location": 9})
		focus("A")
		snap := capture()
		request("set", map[string]any{"text": "unchanged", "location": 0})
		const text = "at caret "
		result := commit(context.Background(), snap, text)
		if result.Outcome != contracts.Unknown || result.Failure != "" {
			t.Fatalf("caret move: %+v", result)
		}
		if !within(5*time.Second, func() bool {
			s := request("state", nil)
			return s.A.Text == text+"unchanged" && s.A.Edits == 1
		}) {
			t.Fatalf("caret move observation: %+v", request("state", nil))
		}
		checkClipboard()
	})
	for _, initiallyFocused := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing-caret/initial-focus-%v", initiallyFocused), func(t *testing.T) {
			request("set", map[string]any{"text": "", "location": 0})
			request("set", map[string]any{"target": "B", "text": "", "location": 0})
			focus("none")
			if initiallyFocused {
				focus("A")
			}
			snap := capture()
			focus("none")
			const text = "Processed 中文🙂\nsecond line"
			result := commit(context.Background(), snap, text)
			if result.Outcome != contracts.Copied || result.Failure != "" {
				t.Fatalf("missing caret: %+v", result)
			}
			s := request("state", nil)
			if s.A.Text != "" || s.A.Edits != 0 || s.B.Text != "" || s.B.Edits != 0 {
				t.Fatalf("clipboard-only commit edited a field: %+v", s)
			}
			if !request("clipboardCheck", map[string]any{"text": text}).ClipboardOwned {
				t.Fatal("processed text was not left on the clipboard")
			}
			if duplicate := commit(context.Background(), snap, text); duplicate.Outcome != contracts.NotApplied {
				t.Fatal("clipboard-only target reused")
			}
			request("clipboardSet", map[string]any{"text": sentinel})
		})
	}
	t.Run("readonly-selection", func(t *testing.T) {
		request("set", map[string]any{"text": "Read only transcript", "location": 5, "length": 4})
		focus("readonly")
		snap := capture()
		const text = "Processed result for clipboard"
		if result := commit(context.Background(), snap, text); result.Outcome != contracts.Copied || result.Failure != "" {
			t.Fatalf("readonly selection accepted paste: %+v", result)
		}
		if s := request("state", nil); s.A.Text != "Read only transcript" || s.A.Edits != 0 {
			t.Fatalf("readonly transcript changed: %+v", s)
		}
		if !request("clipboardCheck", map[string]any{"text": text}).ClipboardOwned {
			t.Fatal("readonly result not retained on clipboard")
		}
		focus("A")
		request("clipboardSet", map[string]any{"text": sentinel})
	})
	t.Run("empty-editor", func(t *testing.T) {
		request("set", map[string]any{"text": "", "location": 0})
		focus("A")
		snap := capture()
		const text = "Empty editor receives text"
		if result := commit(context.Background(), snap, text); result.Outcome != contracts.Unknown || result.Failure != "" {
			t.Fatalf("empty editor: %+v", result)
		}
		if !within(5*time.Second, func() bool { s := request("state", nil); return s.A.Text == text && s.A.Edits == 1 }) {
			t.Fatalf("empty editor observation: %+v", request("state", nil))
		}
		checkClipboard()
	})
	for _, boundary := range []string{"cancel", "stale-identity"} {
		t.Run(boundary, func(t *testing.T) {
			request("set", map[string]any{"text": "unchanged", "location": 9})
			focus("A")
			snap := capture()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch boundary {
			case "cancel":
				cancel()
			case "stale-identity":
				id.Generation++
				defer func() { id.Generation-- }()
			}
			result := commit(ctx, snap, "must not appear")
			if result.Outcome != contracts.NotApplied || result.Failure == "" {
				t.Fatalf("boundary allowed paste: %+v", result)
			}
			s := request("state", nil)
			if s.A.Text != "unchanged" || s.A.Edits != 0 {
				t.Fatal("blocked attempt edited original field")
			}
			checkClipboard()
		})
	}
	t.Run("secure-field", func(t *testing.T) {
		focus("secure")
		snap, err := targets.Capture(context.Background(), id)
		if err != nil {
			if domain.CodeOf(err) != domain.CodeSecureFieldBlocked {
				t.Fatal(err)
			}
			return
		}
		if result := commit(context.Background(), snap, "secret"); result.Outcome != contracts.NotApplied || result.Failure != domain.CodeSecureFieldBlocked {
			t.Fatalf("secure result: %+v", result)
		}
		checkClipboard()
	})
	t.Run("webkit-editor", func(t *testing.T) {
		focus("web")
		// Allow WebKit's AX focus notification to reach the system server.
		time.Sleep(150 * time.Millisecond)
		snap := capture()
		const text = "WebKit 中文🙂\nsecond line"
		if result := commit(context.Background(), snap, text); result.Outcome != contracts.Unknown || result.Failure != "" {
			t.Fatalf("web paste: %+v", result)
		}
		if !within(5*time.Second, func() bool { s := request("state", nil); return s.Web.Text == text && s.Web.Edits == 1 }) {
			t.Fatalf("web input observation: %+v", request("state", nil))
		}
		checkClipboard()
	})
	t.Run("chromium-editors", func(t *testing.T) {
		browser := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if _, err := os.Stat(browser); err != nil {
			t.Skip("Chrome is not installed")
		}
		var phase atomic.Int32
		phase.Store(1)
		var mu sync.Mutex
		var observation struct {
			Field   int    `json:"field"`
			Text    string `json:"text"`
			Edits   int    `json:"edits"`
			Focused bool   `json:"focused"`
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/phase":
				fmt.Fprint(w, phase.Load())
			case "/observe":
				mu.Lock()
				defer mu.Unlock()
				if err := json.NewDecoder(r.Body).Decode(&observation); err != nil {
					t.Error(err)
				}
			default:
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprint(w, `<!doctype html><title>Arion owned paste fixture</title><textarea id="plain" style="width:90%;height:100px"></textarea><div id="rich" contenteditable="true" style="border:1px solid;height:100px"></div><script>
let field=0,edits=0;
function report(){let e=field===1?plain:rich;fetch('/observe',{method:'POST',body:JSON.stringify({field,text:field===1?e.value:e.innerText,edits,focused:document.activeElement===e})})}
plain.oninput=rich.oninput=()=>{edits++;report()};
setInterval(async()=>{let next=+(await(await fetch('/phase')).text());if(next!==field){field=next;edits=0;(field===1?plain:rich).focus();report()}},50);
</script>`)
			}
		}))
		defer server.Close()
		command := exec.Command(browser, "--user-data-dir="+t.TempDir(), "--no-first-run", "--no-default-browser-check", "--disable-sync", "--new-window", server.URL)
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
		for field, text := range []string{"Chromium 中文🙂e\u0301\nsecond line", "Chromium rich 中文🙂"} {
			phase.Store(int32(field + 1))
			eventually(t, "isolated Chromium editor focus", func() bool {
				mu.Lock()
				defer mu.Unlock()
				return observation.Field == field+1 && observation.Focused && observation.Edits == 0
			})
			snap := capture()
			if result := commit(context.Background(), snap, text); result.Outcome != contracts.Unknown || result.Failure != "" {
				t.Fatalf("Chromium result: %+v", result)
			}
			eventually(t, "exact Chromium input event", func() bool {
				mu.Lock()
				defer mu.Unlock()
				return observation.Field == field+1 && observation.Text == text && observation.Edits == 1
			})
			checkClipboard()
		}
	})

}
