//go:build windows

package windows

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"arion.local/core/domain"
)

const (
	whMouseLL   = 14
	wmLButtonUp = 0x0202
)

var selectionMouse struct {
	sync.Mutex
	sink   func()
	thread uintptr
	done   chan struct{}
}

// selectionMouseProc observes left mouse-up and always forwards the event.
// It is not the dictation keyboard hook, and it does not read field text.
var selectionMouseProc = syscall.NewCallback(func(code int, wparam, lparam uintptr) uintptr {
	if code >= 0 && wparam == wmLButtonUp {
		selectionMouse.Lock()
		sink := selectionMouse.sink
		selectionMouse.Unlock()
		if sink != nil {
			sink()
		}
	}
	next, _, _ := procCallNextHookEx.Call(0, uintptr(code), wparam, lparam)
	return next
})

// StartSelectionWatch observes left mouse-up. sink runs on the hook thread
// and must not block.
func StartSelectionWatch(sink func()) error {
	StopSelectionWatch()
	selectionMouse.Lock()
	selectionMouse.sink = sink
	selectionMouse.Unlock()
	started := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(done)
		var message msg
		procPeekMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, 0)
		module, _, _ := procGetModuleHandleW.Call(0)
		hook, _, _ := procSetWindowsHookExW.Call(whMouseLL, selectionMouseProc, module, 0)
		if hook == 0 {
			started <- domain.Fail(domain.CodeDeviceUnavailable, "mouse hook")
			return
		}
		defer procUnhookWindowsHookEx.Call(hook)
		thread, _, _ := procGetCurrentThreadId.Call()
		selectionMouse.Lock()
		selectionMouse.thread, selectionMouse.done = thread, done
		selectionMouse.Unlock()
		started <- nil
		for {
			if result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0); int32(result) <= 0 {
				return
			}
		}
	}()
	if err := <-started; err != nil {
		selectionMouse.Lock()
		selectionMouse.sink = nil
		selectionMouse.Unlock()
		return err
	}
	return nil
}

// StopSelectionWatch removes the mouse-up hook.
func StopSelectionWatch() {
	selectionMouse.Lock()
	thread, done := selectionMouse.thread, selectionMouse.done
	selectionMouse.sink, selectionMouse.thread, selectionMouse.done = nil, 0, nil
	selectionMouse.Unlock()
	if thread == 0 {
		return
	}
	procPostThreadMessageW.Call(thread, wmQuit, 0, 0)
	<-done
}
