//go:build windows

// Package windows implements target capture and insertion with Win32:
// foreground/focus identity, SendInput Unicode events and a guarded
// clipboard paste (ADR 003).
package windows

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	shell32                      = syscall.NewLazyDLL("shell32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetGUIThreadInfo         = user32.NewProc("GetGUIThreadInfo")
	procGetWindowLongW           = user32.NewProc("GetWindowLongW")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procSendInput                = user32.NewProc("SendInput")
	procOpenClipboard            = user32.NewProc("OpenClipboard")
	procCloseClipboard           = user32.NewProc("CloseClipboard")
	procEmptyClipboard           = user32.NewProc("EmptyClipboard")
	procGetClipboardData         = user32.NewProc("GetClipboardData")
	procSetClipboardData         = user32.NewProc("SetClipboardData")
	procGetClipboardSequence     = user32.NewProc("GetClipboardSequenceNumber")
	procGlobalAlloc              = kernel32.NewProc("GlobalAlloc")
	procGlobalLock               = kernel32.NewProc("GlobalLock")
	procGlobalUnlock             = kernel32.NewProc("GlobalUnlock")
	procShellExecuteW            = shell32.NewProc("ShellExecuteW")
)

const (
	gwlStyle         = ^uintptr(15) // GWL_STYLE (-16)
	esPassword       = 0x0020
	inputKeyboard    = 1
	keyeventfUnicode = 0x0004
	keyeventfKeyUp   = 0x0002
	keyeventfNone    = 0
	vkControl        = 0x11
	vkV              = 0x56
	cfUnicodeText    = 13
	gmemMoveable     = 0x0002
)

type guiThreadInfo struct {
	Size        uint32
	Flags       uint32
	Active      uintptr
	Focus       uintptr
	Capture     uintptr
	MenuOwner   uintptr
	MoveSize    uintptr
	Caret       uintptr
	CaretLeft   int32
	CaretTop    int32
	CaretRight  int32
	CaretBottom int32
}

type keyboardInput struct {
	Type    uint32
	_       uint32
	VK      uint16
	Scan    uint16
	Flags   uint32
	Time    uint32
	Extra   uintptr
	Padding uint64
}

// Delivery mirrors the user's insertion preference.
type Delivery string

const (
	DeliveryAutomatic Delivery = "automatic"
	DeliveryClipboard Delivery = "clipboard"
)

type target struct {
	foreground uintptr
	focus      uintptr
	clipboard  bool
	secure     bool
	created    time.Time
}

// Targets implements ports.TargetProvider and ports.InputTarget.
type Targets struct {
	mu       sync.Mutex
	targets  map[contracts.TargetToken]*target
	delivery func() Delivery
	// readScreen gates window text recognition; nil never reads it.
	readScreen func() bool
}

func NewTargets(readScreen func() bool, delivery func() Delivery) *Targets {
	return &Targets{targets: map[contracts.TargetToken]*target{}, delivery: delivery, readScreen: readScreen}
}

func focusOf(foreground uintptr) uintptr {
	thread, _, _ := procGetWindowThreadProcessId.Call(foreground, 0)
	info := guiThreadInfo{Size: uint32(unsafe.Sizeof(guiThreadInfo{}))}
	if ok, _, _ := procGetGUIThreadInfo.Call(thread, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return 0
	}
	return info.Focus
}

func className(hwnd uintptr) string {
	buffer := make([]uint16, 128)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer[:n])
}

func (t *Targets) Capture(_ context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	foreground, _, _ := procGetForegroundWindow.Call()
	if foreground == 0 {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "no foreground window")
	}
	focus := focusOf(foreground)
	secure := false
	if focus != 0 && strings.EqualFold(className(focus), "Edit") {
		style, _, _ := procGetWindowLongW.Call(focus, gwlStyle)
		secure = style&esPassword != 0
	}
	var b [12]byte
	_, _ = rand.Read(b[:])
	token := contracts.TargetToken(hex.EncodeToString(b[:]))
	t.mu.Lock()
	t.targets[token] = &target{foreground: foreground, focus: focus, clipboard: t.delivery() == DeliveryClipboard, secure: secure, created: time.Now()}
	for len(t.targets) > 4 {
		var oldest contracts.TargetToken
		for k, v := range t.targets {
			if oldest == "" || v.created.Before(t.targets[oldest].created) {
				oldest = k
			}
		}
		delete(t.targets, oldest)
	}
	t.mu.Unlock()
	return contracts.TargetSnapshot{Identity: id, Target: token, Sensitive: secure,
		Capabilities: contracts.Capabilities{SingleInsertion: true}}, nil
}

// FocusOwner and Peek: the Windows adapter does not describe targets for
// scene display yet, as it reads no field type or site.
func (t *Targets) FocusOwner() int { return 0 }

func (t *Targets) Peek(context.Context, int) (contracts.TargetSnapshot, error) {
	return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "scene display not offered")
}

func (t *Targets) Capabilities(_ context.Context, token contracts.TargetToken) (contracts.Capabilities, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.targets[token] == nil {
		return contracts.Capabilities{}, domain.Fail(domain.CodeTargetChanged, "unknown target")
	}
	return contracts.Capabilities{SingleInsertion: true}, nil
}

// ReadContext recognizes the captured foreground window's text when allowed.
// Text around the cursor is not read on Windows.
func (t *Targets) ReadContext(ctx context.Context, id contracts.Identity, token contracts.TargetToken) (contracts.ContextSnapshot, error) {
	snap := contracts.ContextSnapshot{Identity: id, Target: token}
	t.mu.Lock()
	tgt := t.targets[token]
	t.mu.Unlock()
	if tgt == nil {
		return snap, domain.Fail(domain.CodeTargetChanged, "unknown target")
	}
	if tgt.secure {
		return snap, nil
	}
	snap.SafeToRead = true
	if t.readScreen != nil && t.readScreen() {
		ctx, cancel := context.WithTimeout(ctx, screenTimeout)
		defer cancel()
		start := time.Now()
		if text, err := recognizeWindow(ctx, tgt.foreground); err == nil {
			snap.Screen, snap.ScreenOCRMs = text, float64(time.Since(start).Microseconds())/1000
		}
	}
	return snap, nil
}

func (t *Targets) Commit(_ context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	result := contracts.CommitResult{Identity: request.Identity, Outcome: contracts.NotApplied}
	t.mu.Lock()
	tgt := t.targets[request.Target]
	delete(t.targets, request.Target)
	t.mu.Unlock()
	foreground, _, _ := procGetForegroundWindow.Call()
	if tgt == nil || foreground != tgt.foreground || (tgt.focus != 0 && focusOf(foreground) != tgt.focus) {
		result.Failure = domain.CodeTargetChanged
		return result, nil
	}
	if tgt.clipboard {
		if !paste(request.Text) {
			result.Failure = domain.CodeTargetUnsupported
			return result, nil
		}
	} else if !typeText(singleLine(request.Text)) {
		result.Failure = domain.CodeTargetUnsupported
		return result, nil
	}
	result.Outcome = contracts.Unknown
	return result, nil
}

// memory converts a Win32 memory handle returned as uintptr by a lazy proc
// call into a pointer; the memory is owned by the global allocator, not Go.
func memory(address uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&address)) }

func singleLine(text string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text)), " ")
}

func send(inputs []keyboardInput) bool {
	if len(inputs) == 0 {
		return true
	}
	sent, _, _ := procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	return int(sent) == len(inputs)
}

// typeText sends UTF-16 units as Unicode key events in small batches.
func typeText(text string) bool {
	units := utf16.Encode([]rune(text))
	for start := 0; start < len(units); start += 16 {
		var batch []keyboardInput
		for _, unit := range units[start:min(start+16, len(units))] {
			batch = append(batch,
				keyboardInput{Type: inputKeyboard, Scan: unit, Flags: keyeventfUnicode},
				keyboardInput{Type: inputKeyboard, Scan: unit, Flags: keyeventfUnicode | keyeventfKeyUp})
		}
		if !send(batch) {
			return false
		}
		time.Sleep(4 * time.Millisecond)
	}
	return true
}

func readClipboardText() (string, bool) {
	handle, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", false
	}
	pointer, _, _ := procGlobalLock.Call(handle)
	if pointer == 0 {
		return "", false
	}
	defer procGlobalUnlock.Call(handle)
	var units []uint16
	for p := memory(pointer); ; p = unsafe.Add(p, 2) {
		unit := *(*uint16)(p)
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units)), true
}

func writeClipboardText(text string) bool {
	units := append(utf16.Encode([]rune(text)), 0)
	handle, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(units)*2))
	if handle == 0 {
		return false
	}
	pointer, _, _ := procGlobalLock.Call(handle)
	copy(unsafe.Slice((*uint16)(memory(pointer)), len(units)), units)
	procGlobalUnlock.Call(handle)
	procEmptyClipboard.Call()
	ok, _, _ := procSetClipboardData.Call(cfUnicodeText, handle)
	return ok != 0
}

// paste writes the text, sends Ctrl+V and restores the previous text only
// if no other program changed the clipboard meanwhile (DSK-A05). Only the
// Unicode text format is preserved on Windows.
func paste(text string) bool {
	if ok, _, _ := procOpenClipboard.Call(0); ok == 0 {
		return false
	}
	previous, hadText := readClipboardText()
	written := writeClipboardText(text)
	procCloseClipboard.Call()
	if !written {
		return false
	}
	owned, _, _ := procGetClipboardSequence.Call()
	send([]keyboardInput{
		{Type: inputKeyboard, VK: vkControl}, {Type: inputKeyboard, VK: vkV},
		{Type: inputKeyboard, VK: vkV, Flags: keyeventfKeyUp}, {Type: inputKeyboard, VK: vkControl, Flags: keyeventfKeyUp},
	})
	time.Sleep(400 * time.Millisecond)
	if current, _, _ := procGetClipboardSequence.Call(); current == owned && hadText {
		if ok, _, _ := procOpenClipboard.Call(0); ok != 0 {
			writeClipboardText(previous)
			procCloseClipboard.Call()
		}
	}
	return true
}

// OpenSettings opens a Windows Settings page such as ms-settings:privacy-microphone.
func OpenSettings(uri string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(uri)
	procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
}
