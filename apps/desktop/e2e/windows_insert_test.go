//go:build arion_e2e && windows

package e2e

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/windows"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procFindWindowExW       = user32.NewProc("FindWindowExW")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procKeybdEvent          = user32.NewProc("keybd_event")
	procSetFocus            = user32.NewProc("SetFocus")
	procAttachThreadInput   = user32.NewProc("AttachThreadInput")
	procGetCurrentThreadId  = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")
)

const (
	wmGetText       = 0x000D
	wmGetTextLength = 0x000E
	vkShift         = 0x10
	keyUp           = 0x0002
)

// notepad is an owned editor: a Notepad process this test starts and kills.
type notepad struct {
	cmd    *exec.Cmd
	window uintptr
	edit   uintptr
}

func startNotepad(t *testing.T) *notepad {
	t.Helper()
	cmd := exec.Command("notepad.exe")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	class, _ := syscall.UTF16PtrFromString("Notepad")
	editClass, _ := syscall.UTF16PtrFromString("Edit")
	n := &notepad{cmd: cmd}
	eventually(t, "notepad window", func() bool {
		var window uintptr
		for {
			window, _, _ = procFindWindowExW.Call(0, window, uintptr(unsafe.Pointer(class)), 0)
			if window == 0 {
				return false
			}
			var pid uint32
			procGetWindowThreadPID.Call(window, uintptr(unsafe.Pointer(&pid)))
			if int(pid) == cmd.Process.Pid {
				n.window = window
				n.edit, _, _ = procFindWindowExW.Call(window, 0, uintptr(unsafe.Pointer(editClass)), 0)
				return n.edit != 0
			}
		}
	})
	return n
}

// focus brings the window forward. Windows only lets a process take the
// foreground after input, so the test taps Shift first (a lone Alt would put
// Notepad's menu bar in keyboard mode and swallow the typed text), then
// focuses the edit control through the window's input thread.
func (n *notepad) focus(t *testing.T) {
	t.Helper()
	procKeybdEvent.Call(vkShift, 0, 0, 0)
	procKeybdEvent.Call(vkShift, 0, keyUp, 0)
	procSetForegroundWindow.Call(n.window)
	thread, _, _ := procGetWindowThreadPID.Call(n.window, 0)
	self, _, _ := procGetCurrentThreadId.Call()
	procAttachThreadInput.Call(self, thread, 1)
	procSetFocus.Call(n.edit)
	procAttachThreadInput.Call(self, thread, 0)
	eventually(t, "notepad foreground", func() bool {
		foreground, _, _ := procGetForegroundWindow.Call()
		return foreground == n.window
	})
}

func (n *notepad) text() string {
	length, _, _ := procSendMessageW.Call(n.edit, wmGetTextLength, 0, 0)
	buffer := make([]uint16, length+1)
	procSendMessageW.Call(n.edit, wmGetText, length+1, uintptr(unsafe.Pointer(&buffer[0])))
	return syscall.UTF16ToString(buffer)
}

func identity() contracts.Identity {
	return contracts.Identity{SessionID: "windows-e2e", Generation: 1}
}

// DSK-A02 on Windows: the production adapter types Unicode text, including
// CJK, into the captured editor exactly once.
func TestWindowsInsertsIntoNotepad(t *testing.T) {
	editor := startNotepad(t)
	editor.focus(t)
	targets := windows.NewTargets(func() windows.Delivery { return windows.DeliveryAutomatic })
	snapshot, err := targets.Capture(context.Background(), identity())
	if err != nil || snapshot.Sensitive {
		t.Fatalf("capture: %+v %v", snapshot, err)
	}
	const text = "Arion 中文 test"
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
}

// DSK-A03 on Windows: a focus change before commit inserts nothing.
func TestWindowsFocusChangeBlocksCommit(t *testing.T) {
	first := startNotepad(t)
	second := startNotepad(t)
	first.focus(t)
	targets := windows.NewTargets(func() windows.Delivery { return windows.DeliveryAutomatic })
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
