//go:build windows

package windows

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"arion.local/desktop/infrastructure/focusclass"
)

var (
	historyOLE            = syscall.NewLazyDLL("ole32.dll")
	historyCreate         = historyOLE.NewProc("CoCreateInstance")
	historyInit           = historyOLE.NewProc("CoInitializeEx")
	historyUninit         = historyOLE.NewProc("CoUninitialize")
	historyVariantClear   = syscall.NewLazyDLL("oleaut32.dll").NewProc("VariantClear")
	historyClientToScreen = user32.NewProc("ClientToScreen")
	historyMonitor        = user32.NewProc("MonitorFromRect")
	historyDPI            = syscall.NewLazyDLL("shcore.dll").NewProc("GetDpiForMonitor")
	historyClass          = syscall.GUID{Data1: 0xe22ad333, Data2: 0xb25f, Data3: 0x460c, Data4: [8]byte{0x83, 0xd0, 0x05, 0x81, 0x10, 0x73, 0x95, 0xc9}}
	historyInterface      = syscall.GUID{Data1: 0x34723aff, Data2: 0x0c9d, Data3: 0x49d0, Data4: [8]byte{0x98, 0x96, 0x7a, 0xb5, 0x2d, 0xf8, 0xcd, 0x8a}}
)

// Method slots follow Microsoft's UIAutomationClient.h. Only metadata is
// requested: process, password, read-only state, focus identity and geometry.
type historyCOM struct{ table *[64]uintptr }

//go:uintptrescapes
func (o *historyCOM) call(slot int, args ...uintptr) uintptr {
	all := append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)
	result, _, _ := syscall.SyscallN(o.table[slot], all...)
	runtime.KeepAlive(o)
	return result
}
func (o *historyCOM) release() {
	if o != nil {
		o.call(2)
	}
}
func historyOK(hr uintptr) bool { return int32(hr) >= 0 }

// VARIANT is 24 bytes on Windows x64. VT_BOOL is 11 and carries a VARIANT_BOOL.
func rangeReadOnly(r *historyCOM) (readOnly, known bool) {
	var attribute struct {
		Kind     uint16
		Reserved [3]uint16
		Data     [2]uint64
	}
	ok := historyOK(r.call(9, 40015, uintptr(unsafe.Pointer(&attribute)))) // GetAttributeValue(IsReadOnly)
	vt := attribute.Kind
	readonly := int16(attribute.Data[0])
	historyVariantClear.Call(uintptr(unsafe.Pointer(&attribute)))
	if !ok || vt != 11 {
		return false, false
	}
	return readonly != 0, true
}

func textPatternReadOnly(text *historyCOM) (readOnly, known bool) {
	var document *historyCOM
	if !historyOK(text.call(7, uintptr(unsafe.Pointer(&document)))) || document == nil { // DocumentRange
		document.release()
		return false, false
	}
	defer document.release()
	return rangeReadOnly(document)
}

func parentElement(client, element *historyCOM) *historyCOM {
	var walker *historyCOM
	if !historyOK(client.call(14, uintptr(unsafe.Pointer(&walker)))) || walker == nil { // ControlViewWalker
		walker.release()
		return nil
	}
	defer walker.release()
	var parent *historyCOM
	if !historyOK(walker.call(3, uintptr(unsafe.Pointer(element)), uintptr(unsafe.Pointer(&parent)))) || parent == nil {
		parent.release()
		return nil
	}
	return parent
}

// caretSelectionEditable reports whether a text selection at the caret is
// writable. The focused composer is often a group with no text pattern; the
// pattern then belongs to an ancestor document whose own range is read-only.
// A failed query on the focused element is uncertain. A failed query further
// up the tree adds no signal.
func caretSelectionEditable(client, element *historyCOM) (editable, uncertain bool) {
	current := element
	owned := false
	defer func() {
		if owned && current != nil {
			current.release()
		}
	}()
	for depth := 0; depth < 6 && current != nil; depth++ {
		text, known := historyPattern(current, 10014) // TextPattern
		if !known {
			return false, depth == 0
		}
		if text != nil {
			editable, uncertain = selectionNotReadOnly(text)
			text.release()
			return editable, uncertain
		}
		parent := parentElement(client, current)
		if owned {
			current.release()
		}
		current = parent
		owned = true
	}
	return false, false
}

func selectionNotReadOnly(text *historyCOM) (editable, uncertain bool) {
	var ranges *historyCOM
	if !historyOK(text.call(5, uintptr(unsafe.Pointer(&ranges)))) { // GetSelection
		ranges.release()
		return false, true
	}
	if ranges == nil {
		return false, false
	}
	defer ranges.release()
	var length int32
	if !historyOK(ranges.call(3, uintptr(unsafe.Pointer(&length)))) { // Length
		return false, true
	}
	if length <= 0 {
		return false, false
	}
	var rng *historyCOM
	if !historyOK(ranges.call(4, 0, uintptr(unsafe.Pointer(&rng)))) || rng == nil { // GetElement(0)
		rng.release()
		return false, true
	}
	defer rng.release()
	readOnly, known := rangeReadOnly(rng)
	if !known {
		return false, false
	}
	return !readOnly, false
}

type historyRect struct{ Left, Top, Right, Bottom int32 }
type historyInput struct {
	client, element  *historyCOM
	bounds           historyRect
	editable, secure bool
}

func (i *historyInput) close() { i.element.release(); i.client.release() }

// Only an explicit unsupported-pattern result establishes absence. Provider
// timeout, permission denial and stale elements remain unknown capabilities.
func historyPattern(element *historyCOM, id uintptr) (*historyCOM, bool) {
	var pattern *historyCOM
	hr := element.call(16, id, uintptr(unsafe.Pointer(&pattern)))
	if !historyOK(hr) {
		pattern.release()
		return nil, uint32(hr) == 0x80040204 || uint32(hr) == 0x80004002
	}
	return pattern, true
}

func readHistoryInput(pid uint32) (*historyInput, bool) {
	i := &historyInput{}
	hr, _, _ := historyCreate.Call(uintptr(unsafe.Pointer(&historyClass)), 0, 1, uintptr(unsafe.Pointer(&historyInterface)), uintptr(unsafe.Pointer(&i.client)))
	if !historyOK(hr) || i.client == nil {
		return nil, false
	}
	success := false
	defer func() {
		if !success {
			i.close()
		}
	}()
	// IUIAutomation2 bounds each connection and provider transaction.
	if !historyOK(i.client.call(61, 200)) || !historyOK(i.client.call(63, 200)) {
		return nil, false
	}
	if !historyOK(i.client.call(8, uintptr(unsafe.Pointer(&i.element)))) || i.element == nil {
		return nil, false
	}
	var owner uint32
	if !historyOK(i.element.call(20, uintptr(unsafe.Pointer(&owner)))) || owner != pid {
		return nil, false
	}
	var password int32
	if !historyOK(i.element.call(35, uintptr(unsafe.Pointer(&password)))) {
		return nil, false
	}
	i.secure = password != 0
	if i.secure {
		success = true
		return i, true
	}
	var kind int32
	if !historyOK(i.element.call(21, uintptr(unsafe.Pointer(&kind)))) {
		return nil, false
	}
	obs := focusclass.WinObservation{ControlType: kind}
	value, known := historyPattern(i.element, 10002)
	if !known {
		return nil, false
	}
	if value != nil {
		var readonly int32
		ok := historyOK(value.call(5, uintptr(unsafe.Pointer(&readonly))))
		value.release()
		if !ok {
			return nil, false
		}
		obs.ValuePattern = true
		obs.ValueReadOnly = readonly != 0
	} else {
		text, known := historyPattern(i.element, 10014)
		if !known {
			return nil, false
		}
		if text != nil {
			readOnly, knownAttr := textPatternReadOnly(text)
			text.release()
			obs.TextPattern = true
			obs.TextReadOnly = readOnly
			obs.TextReadOnlyKnown = knownAttr
		}
		// TextEditPattern and the caret are extra positive signals. A failed
		// query must not erase a text pattern that already answered.
		edit, editKnown := historyPattern(i.element, 10032) // TextEditPattern
		if editKnown && edit != nil {
			edit.release()
			obs.TextEditPattern = true
		}
		if focusclass.WindowsCaretHost(kind) && !obs.TextEditPattern {
			caret, uncertain := caretSelectionEditable(i.client, i.element)
			obs.CaretEditable = caret
			if !(obs.TextPattern && obs.TextReadOnlyKnown) {
				obs.CaretUncertain = uncertain
			}
		}
	}
	decision := focusclass.DecideWindows(obs)
	if !decision.Known {
		return nil, false
	}
	i.editable = decision.Editable
	i.secure = decision.Secure
	// Bounds place the history picker. A missing rectangle does not change
	// whether the focused control can accept text.
	if i.editable && !historyOK(i.element.call(43, uintptr(unsafe.Pointer(&i.bounds)))) {
		i.bounds = historyRect{}
	}
	success = true
	return i, true
}

func (t *HistoryTarget) Anchor() (int, int, int, int, bool) {
	if t == nil || t.input == nil || !t.input.editable || t.input.secure {
		return 0, 0, 0, 0, false
	}
	r := t.input.bounds
	thread, _, _ := procGetWindowThreadProcessId.Call(t.window, 0)
	info := guiThreadInfo{Size: uint32(unsafe.Sizeof(guiThreadInfo{}))}
	if ok, _, _ := procGetGUIThreadInfo.Call(thread, uintptr(unsafe.Pointer(&info))); ok != 0 && info.Focus == t.focus && info.Caret != 0 && info.CaretBottom > info.CaretTop {
		top := struct{ X, Y int32 }{info.CaretLeft, info.CaretTop}
		bottom := struct{ X, Y int32 }{info.CaretRight, info.CaretBottom}
		first, _, _ := historyClientToScreen.Call(info.Caret, uintptr(unsafe.Pointer(&top)))
		second, _, _ := historyClientToScreen.Call(info.Caret, uintptr(unsafe.Pointer(&bottom)))
		if first != 0 && second != 0 {
			r = historyRect{top.X, top.Y, bottom.X, bottom.Y}
		}
	}
	if r.Right < r.Left || r.Bottom <= r.Top {
		return 0, 0, 0, 0, false
	}
	monitor, _, _ := historyMonitor.Call(uintptr(unsafe.Pointer(&r)), 2)
	var dx, dy uint32
	hr, _, _ := historyDPI.Call(monitor, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy)))
	if !historyOK(hr) || dx == 0 || dy == 0 {
		return 0, 0, 0, 0, false
	}
	return int(r.Left) * 96 / int(dx), int(r.Top) * 96 / int(dy), int(r.Right-r.Left) * 96 / int(dx), int(r.Bottom-r.Top) * 96 / int(dy), true
}

// Dictation runs on a worker. Give its metadata query a stable COM apartment;
// the history picker instead uses the existing main-thread apartment.
func dictationInput(pid uint32) (editable, secure, known bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := historyInit.Call(0, 0)
	if historyOK(hr) {
		defer historyUninit.Call()
	} else if uint32(hr) != 0x80010106 {
		return false, false, false
	}
	input, ok := readHistoryInput(pid)
	if !ok {
		return false, false, false
	}
	defer input.close()
	return input.editable, input.secure, true
}

func (t *Targets) NoInput() bool {
	window, _, _ := procGetForegroundWindow.Call()
	if window == 0 {
		return false
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(window, uintptr(unsafe.Pointer(&pid)))
	// Do not synchronously query this UI thread's own accessibility provider.
	if pid == uint32(os.Getpid()) {
		return false
	}
	editable, secure, known := dictationInput(pid)
	current, _, _ := procGetForegroundWindow.Call()
	return current == window && known && !editable && !secure
}
