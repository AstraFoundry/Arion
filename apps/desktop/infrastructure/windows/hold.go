//go:build windows

package windows

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
)

var (
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
)

const (
	whKeyboardLL = 13
	wmQuit       = 0x0012
)

type kbdllHookStruct struct {
	VKCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      [2]int32
	Private uint32
}

var hold struct {
	sync.Mutex
	interpreter *holdkey.Windows
	sink        func(application.HoldKey, holdkey.Event)
	thread      uintptr
	done        chan struct{}
}

// hookProc observes only: it always passes the event on. Windows removes a
// low-level hook whose callback is slow, so the sink must not block.
var hookProc = syscall.NewCallback(func(code int, message, lparam uintptr) uintptr {
	if code >= 0 {
		info := (*kbdllHookStruct)(memory(lparam))
		hold.Lock()
		interpreter, sink := hold.interpreter, hold.sink
		var key application.HoldKey
		var event holdkey.Event
		ok := false
		if interpreter != nil {
			key, event, ok = interpreter.InterpretKey(message, info.VKCode, info.ScanCode)
		}
		hold.Unlock()
		if ok {
			sink(key, event)
		}
	}
	next, _, _ := procCallNextHookEx.Call(0, uintptr(code), message, lparam)
	return next
})

// StartHold observes key with a low-level keyboard hook on a dedicated
// thread. sink runs on that thread and must not block.
func StartHold(key application.HoldKey, sink func(holdkey.Event)) error {
	return StartHoldKeys([]application.HoldKey{key}, func(_ application.HoldKey, e holdkey.Event) {
		sink(e)
	})
}

// StartHoldKeys observes keys with a low-level keyboard hook on a dedicated
// thread. sink runs on that thread and must not block.
func StartHoldKeys(keys []application.HoldKey, sink func(application.HoldKey, holdkey.Event)) error {
	interpreter, ok := holdkey.NewWindows(keys...)
	if !ok {
		return domain.Fail(domain.CodeTargetUnsupported, "hold key not available on Windows")
	}
	StopHold()
	hold.Lock()
	hold.interpreter, hold.sink = interpreter, sink
	hold.Unlock()
	started := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(done)
		module, _, _ := procGetModuleHandleW.Call(0)
		hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, hookProc, module, 0)
		if hook == 0 {
			started <- domain.Fail(domain.CodeDeviceUnavailable, "keyboard hook: "+err.Error())
			return
		}
		defer procUnhookWindowsHookEx.Call(hook)
		thread, _, _ := procGetCurrentThreadId.Call()
		hold.Lock()
		hold.thread, hold.done = thread, done
		hold.Unlock()
		started <- nil
		var m msg
		for {
			if r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0); int32(r) <= 0 {
				return
			}
		}
	}()
	if err := <-started; err != nil {
		hold.Lock()
		hold.interpreter, hold.sink = nil, nil
		hold.Unlock()
		return err
	}
	return nil
}

// StopHold removes the hook and ends its thread.
func StopHold() {
	hold.Lock()
	thread, done := hold.thread, hold.done
	hold.interpreter, hold.sink, hold.thread, hold.done = nil, nil, 0, nil
	hold.Unlock()
	if thread == 0 {
		return
	}
	procPostThreadMessageW.Call(thread, wmQuit, 0, 0)
	<-done
}
