//go:build arion_e2e && windows

package e2e

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/windows"
)

func identity() contracts.Identity {
	return contracts.Identity{SessionID: "windows-e2e", Generation: 1}
}

// DSK-A02 on Windows: the production adapter types Unicode text, including
// CJK, into the captured editor exactly once.
func TestWindowsInsertsIntoOwnedEditor(t *testing.T) {
	editor := startWindowsEditor(t)
	editor.focus(t)
	targets := windows.NewTargets(nil, func() windows.Delivery { return windows.DeliveryAutomatic })
	snapshot, err := targets.Capture(context.Background(), identity())
	if err != nil || snapshot.Sensitive {
		t.Fatalf("capture: %+v %v", snapshot, err)
	}
	const text = "Arion 中文 🦊 e\u0301 test"
	result, err := targets.Commit(context.Background(), contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: text})
	if err != nil || result.Outcome == contracts.NotApplied {
		t.Fatalf("commit: %+v %v", result, err)
	}
	if !within(5*time.Second, func() bool { return editor.text() == text }) {
		t.Fatalf("editor text %q after commit %+v", editor.text(), result)
	}
	time.Sleep(300 * time.Millisecond)
	if got := editor.text(); got != text {
		t.Fatalf("editor text %q", got)
	}
	if got, want := int(editor.characters.Load()), len(utf16.Encode([]rune(text))); got != want {
		t.Fatalf("native WM_CHAR count %d, want %d", got, want)
	}
	result, err = targets.Commit(context.Background(), contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: text})
	if err != nil || result.Outcome != contracts.NotApplied || result.Failure != domain.CodeTargetChanged || editor.text() != text {
		t.Fatalf("duplicate commit: %+v %v; text %q", result, err, editor.text())
	}
}

// DSK-05/DSK-A04: stale sessions, cancelled commits and changed selections
// must not deliver text, and a rejected stale caller cannot consume ownership.
func TestWindowsCommitOwnership(t *testing.T) {
	for _, fault := range []string{"stale-generation", "cancel", "selection-change", "secure-field"} {
		t.Run(fault, func(t *testing.T) {
			editor := startWindowsEditor(t)
			editor.setText("prefix replace suffix")
			editor.focus(t)
			editor.selectRange(7, 14)
			if fault == "secure-field" {
				editor.focusControl(t, editor.password)
			}
			targets := windows.NewTargets(nil, func() windows.Delivery { return windows.DeliveryAutomatic })
			snapshot, err := targets.Capture(context.Background(), identity())
			if err != nil {
				t.Fatal(err)
			}
			request := contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: "replacement"}
			ctx := context.Background()
			want := domain.CodeTargetChanged
			switch fault {
			case "stale-generation":
				request.Identity.Generation++
				want = domain.CodeStaleRevision
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				want = domain.CodeCancelled
			case "selection-change":
				editor.selectRange(0, 0)
			case "secure-field":
				want = domain.CodeSecureFieldBlocked
				if !snapshot.Sensitive {
					t.Fatal("password EDIT was not marked sensitive")
				}
			}
			before := editor.changes.Load()
			result, err := targets.Commit(ctx, request)
			if err != nil || result.Outcome != contracts.NotApplied || result.Failure != want {
				t.Fatalf("commit: %+v %v; want %s", result, err, want)
			}
			time.Sleep(100 * time.Millisecond)
			if editor.text() != "prefix replace suffix" || windowText(editor.password) != "" || editor.changes.Load() != before {
				t.Fatalf("refused commit changed an owned field: %q %q", editor.text(), windowText(editor.password))
			}
			if fault == "stale-generation" {
				request.Identity = identity()
				result, err = targets.Commit(context.Background(), request)
				if err != nil || result.Outcome != contracts.Unknown || !within(5*time.Second, func() bool { return editor.text() == "prefix replacement suffix" }) {
					t.Fatalf("valid owner after stale request: %+v %v; text %q", result, err, editor.text())
				}
			}
		})
	}
}

// DSK-A04: cancellation during SendInput delivery stops later batches and
// preserves the unknown outcome of input that was already posted.
func TestWindowsCancellationDuringDelivery(t *testing.T) {
	editor := startWindowsEditor(t)
	editor.focus(t)
	targets := windows.NewTargets(nil, func() windows.Delivery { return windows.DeliveryAutomatic })
	snapshot, err := targets.Capture(context.Background(), identity())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make(chan contracts.CommitResult, 1)
	go func() {
		result, _ := targets.Commit(ctx, contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: strings.Repeat("x", 6000)})
		results <- result
	}()
	eventually(t, "first input batch delivered", func() bool { return editor.characters.Load() > 0 })
	cancel()
	select {
	case result := <-results:
		if result.Outcome != contracts.Unknown || result.Failure != domain.CodeCancelled {
			t.Fatalf("cancel during delivery: %+v", result)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled delivery did not stop")
	}
	if count := len(editor.text()); count == 0 || count >= 6000 {
		t.Fatalf("cancelled delivery inserted %d of 6000 characters", count)
	}
	// SendInput acknowledges posting, not dispatch. Allow the bounded already
	// posted batch to drain before checking that delivery remains quiescent.
	last, quietSince := editor.characters.Load(), time.Now()
	eventually(t, "posted input drained", func() bool {
		current := editor.characters.Load()
		if current != last {
			last, quietSince = current, time.Now()
		}
		return time.Since(quietSince) >= 100*time.Millisecond && int(current) == len(editor.text())
	})
	count := editor.characters.Load()
	time.Sleep(100 * time.Millisecond)
	if got := editor.characters.Load(); got != count {
		t.Fatalf("late input after cancellation: %d -> %d", count, got)
	}
}

// CTX-A04/PRV-A05: UI Automation reads only an owned visible window, respects
// the source switch, and excludes password/unknown fields before extraction.
func TestWindowsContextFromOwnedWindow(t *testing.T) {
	editor := startWindowsEditor(t)
	const visible = "Arion visible context canary 中文"
	const secret = "Arion password canary must stay private"
	editor.setText(visible)
	password, _ := syscall.UTF16PtrFromString(secret)
	procSetWindowTextW.Call(editor.password, uintptr(unsafe.Pointer(password)))
	editor.focus(t)
	enabled := true
	targets := windows.NewTargets(func() bool { return enabled }, func() windows.Delivery { return windows.DeliveryAutomatic })
	snapshot, err := targets.Capture(context.Background(), identity())
	if err != nil {
		t.Fatal(err)
	}
	read, err := targets.ReadContext(context.Background(), identity(), snapshot.Target)
	if err != nil || !read.SafeToRead || !strings.Contains(read.Screen, visible) || strings.Contains(read.Screen, secret) {
		t.Fatalf("visible context: %+v %v", read, err)
	}
	enabled = false
	read, err = targets.ReadContext(context.Background(), identity(), snapshot.Target)
	if err != nil || !read.SafeToRead || read.Screen != "" || read.Before != "" || read.After != "" {
		t.Fatalf("disabled screen context: %+v %v", read, err)
	}
	enabled = true
	for _, control := range []uintptr{editor.password, editor.button} {
		editor.focusControl(t, control)
		unsafeTarget, err := targets.Capture(context.Background(), identity())
		if err != nil {
			t.Fatal(err)
		}
		read, err = targets.ReadContext(context.Background(), identity(), unsafeTarget.Target)
		if err != nil || read.SafeToRead || read.Screen != "" || read.Before != "" || read.After != "" {
			t.Fatalf("unsafe context: %+v %v", read, err)
		}
	}
	read, err = targets.ReadContext(context.Background(), identity(), snapshot.Target)
	if err == nil || read.Screen != "" {
		t.Fatalf("changed-focus context: %+v %v", read, err)
	}
}

// DSK-A03 on Windows: a focus change before commit inserts nothing.
func TestWindowsFocusChangeBlocksCommit(t *testing.T) {
	first := startWindowsEditor(t)
	second := startWindowsEditor(t)
	first.focus(t)
	targets := windows.NewTargets(nil, func() windows.Delivery { return windows.DeliveryAutomatic })
	snapshot, err := targets.Capture(context.Background(), identity())
	if err != nil {
		t.Fatal(err)
	}
	second.focus(t)
	result, err := targets.Commit(context.Background(), contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: "should not appear"})
	if err != nil || result.Outcome != contracts.NotApplied || result.Failure != domain.CodeTargetChanged {
		t.Fatalf("commit after focus change: %+v %v", result, err)
	}
	time.Sleep(300 * time.Millisecond)
	if first.text() != "" || second.text() != "" {
		t.Fatalf("text inserted after focus change: %q %q", first.text(), second.text())
	}
}
