//go:build arion_e2e && windows

package e2e

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/windows"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
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

// DSK-05/DSK-A04: stale sessions, cancelled commits and secure fields
// must not deliver text, and a rejected stale caller cannot consume ownership.
// A caret move still inserts at the new caret (DSK-A03).
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
				result, err := targets.Commit(ctx, request)
				if err != nil || result.Outcome != contracts.Unknown || result.Failure != "" {
					t.Fatalf("caret move: %+v %v", result, err)
				}
				if !within(5*time.Second, func() bool { return editor.text() == "replacementprefix replace suffix" }) {
					t.Fatalf("text %q", editor.text())
				}
				return
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
	t.Logf("owned-window UI Automation total capture time: %.2f ms", read.ScreenCaptureMs)
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

// DSK-A03 on Windows: a focus change before commit inserts into the focused editor.
func TestWindowsFocusChangeInsertsIntoFocusedEditor(t *testing.T) {
	first := startWindowsEditor(t)
	second := startWindowsEditor(t)
	first.focus(t)
	targets := windows.NewTargets(nil, func() windows.Delivery { return windows.DeliveryAutomatic })
	snapshot, err := targets.Capture(context.Background(), identity())
	if err != nil {
		t.Fatal(err)
	}
	second.focus(t)
	const text = "into the focused editor"
	result, err := targets.Commit(context.Background(), contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: text})
	if err != nil || result.Outcome != contracts.Unknown || result.Failure != "" {
		t.Fatalf("commit after focus change: %+v %v", result, err)
	}
	if !within(5*time.Second, func() bool { return first.text() == "" && second.text() == text }) {
		t.Fatalf("text after focus change: %q %q", first.text(), second.text())
	}
}

// DSK-A03: a focused button has no insertion point. Copy the processed text
// without sending keys, for both an absent initial caret and a lost caret.
func TestWindowsCopiesWithoutEditableFocus(t *testing.T) {
	for _, delivery := range []windows.Delivery{windows.DeliveryAutomatic, windows.DeliveryClipboard} {
		for _, initial := range []string{"button", "editor"} {
			for _, readOnly := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/readonly-%v", delivery, initial, readOnly), func(t *testing.T) {
					editor := startWindowsEditor(t)
					noneditable := editor.button
					if readOnly {
						procSendMessageW.Call(editor.edit, 0x00CF, 1, 0) // EM_SETREADONLY
						noneditable = editor.edit
					}
					editor.focusControl(t, noneditable)
					if initial == "editor" {
						editor.focus(t)
					}
					targets := windows.NewTargets(nil, func() windows.Delivery { return delivery })
					snapshot, err := targets.Capture(context.Background(), identity())
					if err != nil {
						t.Fatal(err)
					}
					editor.focusControl(t, noneditable)
					previous, err := mygo.Clipboard.Read()
					if err != nil {
						t.Fatal(err)
					}
					const text = "Processed 中文🙂\nsecond line"
					t.Cleanup(func() {
						if mygo.Clipboard.ReadText() == text {
							if err := mygo.Clipboard.Write(previous); err != nil {
								t.Error(err)
							}
						}
					})
					request := contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: text}
					result, err := targets.Commit(context.Background(), request)
					if err != nil || result.Outcome != contracts.Copied || result.Failure != "" {
						t.Fatalf("noneditable focus: %+v %v", result, err)
					}
					if got := mygo.Clipboard.ReadText(); got != text {
						t.Fatalf("clipboard text %q, want %q", got, text)
					}
					if editor.text() != "" || editor.characters.Load() != 0 || editor.changes.Load() != 0 {
						t.Fatal("clipboard-only delivery edited a field")
					}
					if duplicate, _ := targets.Commit(context.Background(), request); duplicate.Outcome != contracts.NotApplied {
						t.Fatal("clipboard-only target reused")
					}
				})
			}
		}
	}
}

func TestWindowsPasteRestoresClipboard(t *testing.T) {
	for _, original := range []string{"", "Arion owned clipboard sentinel"} {
		t.Run(fmt.Sprintf("original-text-%v", original != ""), func(t *testing.T) {
			editor := startWindowsEditor(t)
			editor.focus(t)
			previous, err := mygo.Clipboard.Read()
			if err != nil {
				t.Fatal(err)
			}
			const text = "Paste into the editor"
			t.Cleanup(func() {
				if current := mygo.Clipboard.ReadText(); current == original || current == text {
					if err := mygo.Clipboard.Write(previous); err != nil {
						t.Error(err)
					}
				}
			})
			var contents transfer.Data
			if original != "" {
				contents = transfer.TextData(original)
			}
			if err := mygo.Clipboard.Write(contents); err != nil {
				t.Fatal(err)
			}
			targets := windows.NewTargets(nil, func() windows.Delivery { return windows.DeliveryClipboard })
			snapshot, err := targets.Capture(context.Background(), identity())
			if err != nil {
				t.Fatal(err)
			}
			result, err := targets.Commit(context.Background(), contracts.CommitRequest{Identity: identity(), Target: snapshot.Target, SourceRevision: 1, Text: text})
			if err != nil || result.Outcome != contracts.Unknown || result.Failure != "" {
				t.Fatalf("paste result: %+v %v", result, err)
			}
			if !within(5*time.Second, func() bool { return editor.text() == text }) {
				t.Fatalf("editor text %q, want %q", editor.text(), text)
			}
			if got := mygo.Clipboard.ReadText(); got != original {
				t.Fatalf("clipboard text %q, want original %q", got, original)
			}
			if original == "" && len(mygo.Clipboard.Formats()) != 0 {
				t.Fatal("paste left clipboard formats on the originally empty clipboard")
			}
		})
	}
}
