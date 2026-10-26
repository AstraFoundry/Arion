//go:build arion_e2e && windows

package e2e

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procKeybdEvent          = user32.NewProc("keybd_event")
	procSetFocus            = user32.NewProc("SetFocus")
	procAttachThreadInput   = user32.NewProc("AttachThreadInput")
	procGetCurrentThreadId  = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId")
	procRegisterClassW      = user32.NewProc("RegisterClassW")
	procUnregisterClassW    = user32.NewProc("UnregisterClassW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	editorSequence          atomic.Uint32
)

const (
	wmGetText       = 0x000D
	wmGetTextLength = 0x000E
	vkShift         = 0x10
	keyUp           = 0x0002
	emSetSel        = 0x00B1
)

type editorWindowClass struct {
	Style               uint32
	WindowProc          uintptr
	ClassExtra          int32
	WindowExtra         int32
	Instance            uintptr
	Icon                uintptr
	Cursor              uintptr
	Background          uintptr
	MenuName, ClassName *uint16
}

type editorMessage struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	X, Y           int32
	Private        uint32
}

// windowsEditor owns an actual standard Win32 EDIT, a password EDIT, and a
// noneditable control. Its dedicated OS thread pumps real input messages.
// It avoids dependence on packaged Notepad's version, process model or tabs.
type windowsEditor struct {
	window, edit, password, button uintptr
	changes                        atomic.Int32
	characters                     atomic.Int32
	done                           chan struct{}
}

func startWindowsEditor(t *testing.T) *windowsEditor {
	t.Helper()
	editor := &windowsEditor{done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(editor.done)
		name, _ := syscall.UTF16PtrFromString(fmt.Sprintf("ArionOwnedEditor%d", editorSequence.Add(1)))
		class := editorWindowClass{ClassName: name, WindowProc: syscall.NewCallback(func(hwnd, msg, wparam, lparam uintptr) uintptr {
			if msg == 0x0002 { // WM_DESTROY
				procPostQuitMessage.Call(0)
			}
			if msg == 0x0111 && wparam>>16 == 0x0300 { // WM_COMMAND / EN_CHANGE
				editor.changes.Add(1)
			}
			result, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
			return result
		})}
		if atom, _, err := procRegisterClassW.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
			ready <- fmt.Errorf("register owned editor: %w", err)
			return
		}
		defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(name)), 0)
		title, _ := syscall.UTF16PtrFromString("Arion owned editor fixture")
		editor.window, _, _ = procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(title)), 0x00CF0000, 100, 100, 640, 360, 0, 0, 0, 0)
		editClass, _ := syscall.UTF16PtrFromString("Edit")
		editor.edit, _, _ = procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(editClass)), 0, 0x50810044, 10, 10, 600, 180, editor.window, 1, 0, 0)
		editor.password, _, _ = procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(editClass)), 0, 0x50810020, 10, 200, 600, 30, editor.window, 2, 0, 0)
		buttonClass, _ := syscall.UTF16PtrFromString("Button")
		editor.button, _, _ = procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(buttonClass)), 0, 0x50010000, 10, 240, 120, 30, editor.window, 3, 0, 0)
		if editor.window == 0 || editor.edit == 0 || editor.password == 0 || editor.button == 0 {
			if editor.window != 0 {
				user32.NewProc("DestroyWindow").Call(editor.window)
			}
			ready <- fmt.Errorf("create owned editor controls")
			return
		}
		procShowWindow.Call(editor.window, 5)
		ready <- nil
		var msg editorMessage
		for {
			result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if result == 0 || int32(result) == -1 {
				return
			}
			if msg.Window == editor.edit && msg.Message == 0x0102 { // WM_CHAR
				editor.characters.Add(1)
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("owned editor creation timed out")
	}
	t.Cleanup(func() {
		procPostMessageW.Call(editor.window, 0x0010, 0, 0) // WM_CLOSE
		select {
		case <-editor.done:
		case <-time.After(5 * time.Second):
			t.Error("owned editor did not close")
		}
	})
	return editor
}

func (editor *windowsEditor) focus(t *testing.T) { editor.focusControl(t, editor.edit) }

func (editor *windowsEditor) focusControl(t *testing.T, control uintptr) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	self, _, _ := procGetCurrentThreadId.Call()
	eventually(t, "owned editor foreground", func() bool {
		foreground, _, _ := procGetForegroundWindow.Call()
		if foreground == editor.window {
			return true
		}
		procKeybdEvent.Call(vkShift, 0, 0, 0)
		procKeybdEvent.Call(vkShift, 0, keyUp, 0)
		current, _, _ := procGetWindowThreadPID.Call(foreground, 0)
		procAttachThreadInput.Call(self, current, 1)
		procSetForegroundWindow.Call(editor.window)
		procAttachThreadInput.Call(self, current, 0)
		foreground, _, _ = procGetForegroundWindow.Call()
		return foreground == editor.window
	})
	thread, _, _ := procGetWindowThreadPID.Call(editor.window, 0)
	procAttachThreadInput.Call(self, thread, 1)
	procSetFocus.Call(control)
	procAttachThreadInput.Call(self, thread, 0)
	// Assert actual GUI-thread focus rather than assuming SetFocus succeeded.
	eventually(t, "owned editor focus", func() bool {
		info := struct {
			Size, Flags                                        uint32
			Active, Focus, Capture, MenuOwner, MoveSize, Caret uintptr
			Left, Top, Right, Bottom                           int32
		}{Size: 72}
		info.Size = uint32(unsafe.Sizeof(info))
		ok, _, _ := user32.NewProc("GetGUIThreadInfo").Call(thread, uintptr(unsafe.Pointer(&info)))
		return ok != 0 && info.Focus == control
	})
}

func (editor *windowsEditor) text() string { return windowText(editor.edit) }

func windowText(control uintptr) string {
	length, _, _ := procSendMessageW.Call(control, wmGetTextLength, 0, 0)
	buffer := make([]uint16, length+1)
	procSendMessageW.Call(control, wmGetText, length+1, uintptr(unsafe.Pointer(&buffer[0])))
	return syscall.UTF16ToString(buffer)
}

func (editor *windowsEditor) setText(text string) {
	value, _ := syscall.UTF16PtrFromString(text)
	procSetWindowTextW.Call(editor.edit, uintptr(unsafe.Pointer(value)))
}

func (editor *windowsEditor) selectRange(start, end uintptr) {
	procSendMessageW.Call(editor.edit, emSetSel, start, end)
}
