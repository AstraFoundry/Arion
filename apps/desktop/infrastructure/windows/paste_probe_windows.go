//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	oleaut32          = syscall.NewLazyDLL("oleaut32.dll")
	procSysStringLen  = oleaut32.NewProc("SysStringLen")
	procSysFreeString = oleaut32.NewProc("SysFreeString")
)

const (
	wmGetTextLength = 0x000E
	smtoAbortIfHung = 0x0002
)

// fieldProbe measures the length of the field focused at paste time. Only
// a length is kept: UI Automation text is freed as soon as it is counted.
// A field that cannot be measured never reports that the paste landed.
type fieldProbe struct {
	measure func() (int, bool)
	before  int
	known   bool
	close   func()
}

// openFieldProbe runs before the chord. A standard edit or rich edit
// answers WM_GETTEXTLENGTH. Other controls are read through the focused UI
// Automation element when it belongs to pid and is an editable,
// non-password field. The calling goroutine stays on one COM thread until
// close.
func openFieldProbe(pid uint32, focus uintptr, standardEdit bool) *fieldProbe {
	probe := &fieldProbe{close: func() {}}
	switch {
	case standardEdit && focus != 0:
		probe.measure = func() (int, bool) { return windowTextLength(focus) }
	default:
		probe.measure, probe.close = automationLength(pid)
	}
	if probe.measure != nil {
		probe.before, probe.known = probe.measure()
	}
	return probe
}

func (p *fieldProbe) landed() bool {
	if !p.known {
		return false
	}
	n, ok := p.measure()
	return ok && n != p.before
}

func windowTextLength(hwnd uintptr) (int, bool) {
	var length uintptr
	completed, _, _ := procSendMessageTimeoutW.Call(hwnd, wmGetTextLength, 0, 0, smtoAbortIfHung, 100, uintptr(unsafe.Pointer(&length)))
	return int(length), completed != 0
}

// automationLength returns a measure of the focused element's value
// (ValuePattern) or document text (TextPattern), in UTF-16 units.
func automationLength(pid uint32) (func() (int, bool), func()) {
	runtime.LockOSThread()
	hr, _, _ := historyInit.Call(0, 0)
	initialized := historyOK(hr)
	if !initialized && uint32(hr) != 0x80010106 {
		runtime.UnlockOSThread()
		return nil, func() {}
	}
	var input *historyInput
	var pattern *historyCOM
	release := func() {
		pattern.release()
		if input != nil {
			input.close()
		}
		if initialized {
			historyUninit.Call()
		}
		runtime.UnlockOSThread()
	}
	input, ok := readHistoryInput(pid)
	if !ok || !input.editable || input.secure {
		release()
		return nil, func() {}
	}
	if value, known := historyPattern(input.element, 10002); known && value != nil { // ValuePattern
		pattern = value
		return func() (int, bool) {
			return bstrLength(func(out uintptr) uintptr { return pattern.call(4, out) }) // CurrentValue
		}, release
	}
	text, known := historyPattern(input.element, 10014) // TextPattern
	if !known || text == nil {
		release()
		return nil, func() {}
	}
	pattern = text
	return func() (int, bool) {
		var document *historyCOM
		if !historyOK(pattern.call(7, uintptr(unsafe.Pointer(&document)))) || document == nil { // DocumentRange
			document.release()
			return 0, false
		}
		defer document.release()
		return bstrLength(func(out uintptr) uintptr { return document.call(12, ^uintptr(0), out) }) // GetText(-1)
	}, release
}

// bstrLength calls a COM getter with a BSTR out-pointer, counts the string
// and frees it.
func bstrLength(call func(out uintptr) uintptr) (int, bool) {
	bstr := new(uintptr)
	ok := historyOK(call(uintptr(unsafe.Pointer(bstr))))
	runtime.KeepAlive(bstr)
	if !ok || *bstr == 0 {
		return 0, ok
	}
	defer procSysFreeString.Call(*bstr)
	n, _, _ := procSysStringLen.Call(*bstr)
	return int(n), true
}
