//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/domain"
)

var (
	selectionOLE               = syscall.NewLazyDLL("oleaut32.dll")
	selectionSysStringLen      = selectionOLE.NewProc("SysStringLen")
	selectionSysFreeString     = selectionOLE.NewProc("SysFreeString")
	selectionSafeArrayAccess   = selectionOLE.NewProc("SafeArrayAccessData")
	selectionSafeArrayUnaccess = selectionOLE.NewProc("SafeArrayUnaccessData")
	selectionSafeArrayLBound   = selectionOLE.NewProc("SafeArrayGetLBound")
	selectionSafeArrayUBound   = selectionOLE.NewProc("SafeArrayGetUBound")
	selectionSafeArrayDestroy  = selectionOLE.NewProc("SafeArrayDestroy")
)

// ReadSelectedText returns the focused control's UI Automation selection.
// A password control returns no text. This is not the visible-range script.
func ReadSelectedText() (string, bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := historyInit.Call(0, 0)
	if historyOK(hr) {
		defer historyUninit.Call()
	} else if uint32(hr) != 0x80010106 {
		return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	client, element, err := selectionClient()
	if err != nil {
		return "", false, err
	}
	defer client.release()
	if element == nil {
		return "", false, nil
	}
	defer element.release()
	return selectionFromElement(client, element)
}

func selectionClient() (*historyCOM, *historyCOM, error) {
	var client *historyCOM
	hr, _, _ := historyCreate.Call(uintptr(unsafe.Pointer(&historyClass)), 0, 1, uintptr(unsafe.Pointer(&historyInterface)), uintptr(unsafe.Pointer(&client)))
	if !historyOK(hr) || client == nil {
		client.release()
		return nil, nil, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	if !historyOK(client.call(61, 200)) || !historyOK(client.call(63, 200)) {
		client.release()
		return nil, nil, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	var element *historyCOM
	if !historyOK(client.call(8, uintptr(unsafe.Pointer(&element)))) {
		element.release()
		client.release()
		return nil, nil, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	return client, element, nil
}

func selectionFromElement(client, element *historyCOM) (string, bool, error) {
	current := element
	owned := false
	defer func() {
		if owned && current != nil {
			current.release()
		}
	}()
	for depth := 0; depth < 6 && current != nil; depth++ {
		secure, err := elementPassword(current)
		if err != nil {
			return "", false, err
		}
		if secure {
			return "", true, nil
		}
		text, known := historyPattern(current, 10014)
		if !known {
			if depth == 0 {
				return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
			}
			return "", false, nil
		}
		if text != nil {
			value, secureRange, readErr := textSelection(text)
			text.release()
			return value, secureRange, readErr
		}
		parent := parentElement(client, current)
		if owned {
			current.release()
		}
		current = parent
		owned = true
	}
	return "", false, nil
}

func elementPassword(element *historyCOM) (bool, error) {
	var password int32
	if !historyOK(element.call(35, uintptr(unsafe.Pointer(&password)))) {
		return false, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	return password != 0, nil
}

// SelectedTextAnchor is the focused selection's screen rectangle in DIPs.
// It reads geometry only. A secure field and a missing rectangle return ok false.
func SelectedTextAnchor() (x, y, width, height int, ok bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := historyInit.Call(0, 0)
	if historyOK(hr) {
		defer historyUninit.Call()
	} else if uint32(hr) != 0x80010106 {
		return 0, 0, 0, 0, false
	}
	client, element, err := selectionClient()
	if err != nil {
		return 0, 0, 0, 0, false
	}
	defer client.release()
	if element == nil {
		return 0, 0, 0, 0, false
	}
	defer element.release()
	return anchorFromElement(client, element)
}

func anchorFromElement(client, element *historyCOM) (int, int, int, int, bool) {
	current := element
	owned := false
	defer func() {
		if owned && current != nil {
			current.release()
		}
	}()
	for depth := 0; depth < 6 && current != nil; depth++ {
		secure, err := elementPassword(current)
		if err != nil || secure {
			return 0, 0, 0, 0, false
		}
		text, known := historyPattern(current, 10014)
		if !known || text == nil {
			parent := parentElement(client, current)
			if owned {
				current.release()
			}
			current = parent
			owned = true
			continue
		}
		rng, rangeErr := selectionRange(text)
		text.release()
		if rangeErr != nil || rng == nil {
			return 0, 0, 0, 0, false
		}
		defer rng.release()
		return rangeBounds(rng)
	}
	return 0, 0, 0, 0, false
}

func selectionRange(text *historyCOM) (*historyCOM, error) {
	var ranges *historyCOM
	if !historyOK(text.call(5, uintptr(unsafe.Pointer(&ranges)))) {
		ranges.release()
		return nil, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	if ranges == nil {
		return nil, nil
	}
	defer ranges.release()
	var length int32
	if !historyOK(ranges.call(3, uintptr(unsafe.Pointer(&length)))) {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	if length <= 0 {
		return nil, nil
	}
	var rng *historyCOM
	if !historyOK(ranges.call(4, 0, uintptr(unsafe.Pointer(&rng)))) || rng == nil {
		rng.release()
		return nil, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	return rng, nil
}

func textSelection(text *historyCOM) (string, bool, error) {
	rng, err := selectionRange(text)
	if err != nil || rng == nil {
		return "", false, err
	}
	defer rng.release()
	var bstr *uint16
	if !historyOK(rng.call(12, ^uintptr(0), uintptr(unsafe.Pointer(&bstr)))) {
		freeBSTR(bstr)
		return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	return bstrString(bstr), false, nil
}

// rangeBounds reads left, top, width, height in physical pixels. It does not read text.
func rangeBounds(rng *historyCOM) (int, int, int, int, bool) {
	var arr uintptr
	if !historyOK(rng.call(10, uintptr(unsafe.Pointer(&arr)))) || arr == 0 {
		if arr != 0 {
			selectionSafeArrayDestroy.Call(arr)
		}
		return 0, 0, 0, 0, false
	}
	defer selectionSafeArrayDestroy.Call(arr)
	var lower, upper int32
	if hr, _, _ := selectionSafeArrayLBound.Call(arr, 1, uintptr(unsafe.Pointer(&lower))); int32(hr) < 0 {
		return 0, 0, 0, 0, false
	}
	if hr, _, _ := selectionSafeArrayUBound.Call(arr, 1, uintptr(unsafe.Pointer(&upper))); int32(hr) < 0 || int(upper-lower+1) < 4 {
		return 0, 0, 0, 0, false
	}
	var data *float64
	if hr, _, _ := selectionSafeArrayAccess.Call(arr, uintptr(unsafe.Pointer(&data))); int32(hr) < 0 || data == nil {
		return 0, 0, 0, 0, false
	}
	defer selectionSafeArrayUnaccess.Call(arr)
	values := unsafe.Slice(data, 4)
	left, top, width, height := values[0], values[1], values[2], values[3]
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	return historyDIP(historyRect{Left: int32(left), Top: int32(top), Right: int32(left + width), Bottom: int32(top + height)})
}

func bstrString(bstr *uint16) string {
	if bstr == nil {
		return ""
	}
	defer freeBSTR(bstr)
	count, _, _ := selectionSysStringLen.Call(uintptr(unsafe.Pointer(bstr)))
	if count == 0 {
		return ""
	}
	chars := unsafe.Slice(bstr, count)
	return string(utf16.Decode(chars))
}

func freeBSTR(bstr *uint16) {
	if bstr != nil {
		selectionSysFreeString.Call(uintptr(unsafe.Pointer(bstr)))
	}
}
