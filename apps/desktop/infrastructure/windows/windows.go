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
	"arion.local/core/domain/ports"
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
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procSendInput                = user32.NewProc("SendInput")
	procOpenClipboard            = user32.NewProc("OpenClipboard")
	procCloseClipboard           = user32.NewProc("CloseClipboard")
	procEmptyClipboard           = user32.NewProc("EmptyClipboard")
	procGetClipboardData         = user32.NewProc("GetClipboardData")
	procSetClipboardData         = user32.NewProc("SetClipboardData")
	procGetClipboardSequence     = user32.NewProc("GetClipboardSequenceNumber")
	procRegisterClipboardFormatW = user32.NewProc("RegisterClipboardFormatW")
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
	vkControl        = 0x11
	vkV              = 0x56
	cfUnicodeText    = 13
	gmemMoveable     = 0x0002
	gmemZeroInit     = 0x0040
)

// clipboardHistoryExclusion is the registered clipboard format whose presence
// keeps Windows clipboard history and cloud clipboard sync from recording
// the content (https://learn.microsoft.com/windows/win32/dataxchg/clipboard-formats#cloud-clipboard-and-clipboard-history-formats).
const clipboardHistoryExclusion = "ExcludeClipboardContentFromMonitorProcessing"

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
	identity   contracts.Identity
	foreground uintptr
	focus      uintptr
	clipboard  bool
	secure     bool
	readable   bool
	selection  bool
	start, end uint32
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
	secure, readable := false, false
	var start, end uint32
	selection := false
	if focus != 0 && strings.EqualFold(className(focus), "Edit") {
		style, _, _ := procGetWindowLongW.Call(focus, gwlStyle)
		secure = style&esPassword != 0
		readable = !secure
		start, end, selection = editSelection(focus)
	}
	var b [12]byte
	_, _ = rand.Read(b[:])
	token := contracts.TargetToken(hex.EncodeToString(b[:]))
	t.mu.Lock()
	t.targets[token] = &target{identity: id, foreground: foreground, focus: focus, clipboard: t.delivery() == DeliveryClipboard, secure: secure, readable: readable, selection: selection, start: start, end: end, created: time.Now()}
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
	return contracts.TargetSnapshot{Identity: id, Target: token, Sensitive: secure}, nil
}

// FocusOwner and Peek: the Windows adapter does not describe targets for
// scene display yet, as it reads no field type or site.
func (t *Targets) FocusOwner() int { return 0 }

func (t *Targets) Peek(context.Context, int) (contracts.TargetSnapshot, error) {
	return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "scene display not offered")
}

// Follow refuses: the Windows adapter reads no field text, so edits are
// not learned on Windows yet (ADR 014).
func (t *Targets) Follow(contracts.TargetToken) bool { return false }

func (t *Targets) ReadField(context.Context, contracts.TargetToken) (ports.FieldRead, error) {
	return ports.FieldRead{}, domain.Fail(domain.CodeTargetUnsupported, "field reading not offered")
}

func (t *Targets) Release(contracts.TargetToken) {}

// ReadContext reads visible UI Automation text from the captured foreground window.
// Text around the cursor is not read on Windows.
func (t *Targets) ReadContext(ctx context.Context, id contracts.Identity, token contracts.TargetToken) (contracts.ContextSnapshot, error) {
	snap := contracts.ContextSnapshot{Identity: id, Target: token}
	t.mu.Lock()
	tgt := t.targets[token]
	t.mu.Unlock()
	if tgt == nil {
		return snap, domain.Fail(domain.CodeTargetChanged, "unknown target")
	}
	if tgt.identity != id {
		return snap, domain.Fail(domain.CodeTargetChanged, "context identity changed")
	}
	if ctx.Err() != nil {
		return snap, domain.Fail(domain.CodeCancelled, "context cancelled")
	}
	if !tgt.readable || tgt.secure || tgt.focus == 0 {
		return snap, nil
	}
	if foreground, _, _ := procGetForegroundWindow.Call(); foreground != tgt.foreground || focusOf(foreground) != tgt.focus {
		return snap, domain.Fail(domain.CodeTargetChanged, "context target changed")
	}
	// A standard edit can become a password field after capture. Other control
	// classes have no verified safety contract and are never read.
	style, _, _ := procGetWindowLongW.Call(tgt.focus, gwlStyle)
	if !strings.EqualFold(className(tgt.focus), "Edit") || style&esPassword != 0 {
		return snap, nil
	}
	snap.SafeToRead = true
	if t.readScreen != nil && t.readScreen() {
		start := time.Now()
		if foreground, _, _ := procGetForegroundWindow.Call(); foreground != tgt.foreground || focusOf(foreground) != tgt.focus {
			return snap, domain.Fail(domain.CodeTargetChanged, "context target changed")
		}
		text, err := readWindowText(ctx, tgt.foreground, func() error {
			foreground, _, _ := procGetForegroundWindow.Call()
			if foreground != tgt.foreground || focusOf(foreground) != tgt.focus {
				return domain.Fail(domain.CodeTargetChanged, "context target changed during initialization")
			}
			if !strings.EqualFold(className(tgt.focus), "Edit") {
				return domain.Fail(domain.CodeTargetChanged, "context field safety changed during initialization")
			}
			style, _, _ := procGetWindowLongW.Call(tgt.focus, gwlStyle)
			if style&esPassword != 0 {
				return domain.Fail(domain.CodeSecureFieldBlocked, "context field became secure")
			}
			return nil
		})
		if ctx.Err() != nil {
			return snap, domain.Fail(domain.CodeCancelled, "context cancelled")
		}
		if err != nil {
			return snap, err
		}
		if foreground, _, _ := procGetForegroundWindow.Call(); foreground != tgt.foreground || focusOf(foreground) != tgt.focus {
			return snap, domain.Fail(domain.CodeTargetChanged, "context target changed")
		}
		snap.Screen, snap.ScreenCaptureMs = text, float64(time.Since(start).Microseconds())/1000
	}
	return snap, nil
}

func (t *Targets) Commit(ctx context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	result := contracts.CommitResult{Identity: request.Identity, Outcome: contracts.NotApplied}
	t.mu.Lock()
	tgt := t.targets[request.Target]
	if tgt != nil && tgt.identity != request.Identity {
		t.mu.Unlock()
		result.Failure = domain.CodeStaleRevision
		return result, nil
	}
	delete(t.targets, request.Target)
	t.mu.Unlock()
	if tgt == nil {
		result.Failure = domain.CodeTargetChanged
		return result, nil
	}
	if ctx.Err() != nil {
		result.Failure = domain.CodeCancelled
		return result, nil
	}
	// A capture already marked secure never follows the caret or the
	// clipboard. That decision uses the snapshot, so an invalid handle
	// returns before any OS input.
	if tgt.secure {
		result.Failure = domain.CodeSecureFieldBlocked
		return result, nil
	}
	foreground, _, _ := procGetForegroundWindow.Call()
	focus := uintptr(0)
	if foreground != 0 {
		focus = focusOf(foreground)
	}
	if foreground == 0 || focus == 0 {
		if !publishClipboard(request.Text) {
			result.Failure = domain.CodeClipboardUnavailable
			return result, nil
		}
		result.Outcome = contracts.Copied
		return result, nil
	}
	if strings.EqualFold(className(focus), "Edit") {
		style, _, _ := procGetWindowLongW.Call(focus, gwlStyle)
		if style&esPassword != 0 {
			result.Failure = domain.CodeSecureFieldBlocked
			return result, nil
		}
	}
	tgt.foreground, tgt.focus = foreground, focus
	if ctx.Err() != nil {
		result.Failure = domain.CodeCancelled
		return result, nil
	}
	if tgt.clipboard {
		if !paste(request.Text) {
			result.Failure = domain.CodeTargetUnsupported
			return result, nil
		}
	} else {
		posted, failure := typeText(ctx, singleLine(request.Text), func() bool {
			foreground, _, _ := procGetForegroundWindow.Call()
			return foreground == tgt.foreground && focusOf(foreground) == tgt.focus
		})
		if !posted && failure == domain.CodeTargetChanged {
			if !publishClipboard(request.Text) {
				result.Failure = domain.CodeClipboardUnavailable
				return result, nil
			}
			result.Outcome = contracts.Copied
			return result, nil
		}
		if posted {
			result.Outcome = contracts.Unknown
		}
		result.Failure = failure
		return result, nil
	}
	result.Outcome = contracts.Unknown
	return result, nil
}

// EM_GETSEL reports UTF-16 offsets for a standard EDIT. Bound the cross-process
// call so a hung editor cannot block cancellation indefinitely.
func editSelection(hwnd uintptr) (start, end uint32, ok bool) {
	var result uintptr
	completed, _, _ := procSendMessageTimeoutW.Call(hwnd, 0x00B0, uintptr(unsafe.Pointer(&start)), uintptr(unsafe.Pointer(&end)), 0x0002, 100, uintptr(unsafe.Pointer(&result)))
	return start, end, completed != 0
}

// memory converts a Win32 memory handle returned as uintptr by a lazy proc
// call into a pointer; the memory is owned by the global allocator, not Go.
func memory(address uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&address)) }

func singleLine(text string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text)), " ")
}

func send(inputs []keyboardInput) bool {
	return sendCount(inputs) == len(inputs)
}

func sendCount(inputs []keyboardInput) int {
	if len(inputs) == 0 {
		return 0
	}
	sent, _, _ := procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	return int(sent)
}

// typeText sends UTF-16 units as Unicode key events in small batches.
// Once any input is posted, cancellation or a partial send cannot claim that
// nothing was applied: Windows supplies no editor acknowledgement.
func typeText(ctx context.Context, text string, focused func() bool) (posted bool, failure domain.ErrorCode) {
	units := utf16.Encode([]rune(text))
	for start := 0; start < len(units); start += 16 {
		if ctx.Err() != nil {
			return posted, domain.CodeCancelled
		}
		if !focused() {
			return posted, domain.CodeTargetChanged
		}
		var batch []keyboardInput
		for _, unit := range units[start:min(start+16, len(units))] {
			batch = append(batch,
				keyboardInput{Type: inputKeyboard, Scan: unit, Flags: keyeventfUnicode},
				keyboardInput{Type: inputKeyboard, Scan: unit, Flags: keyeventfUnicode | keyeventfKeyUp})
		}
		sent := sendCount(batch)
		posted = posted || sent > 0
		if sent != len(batch) {
			return posted, domain.CodeTargetUnsupported
		}
		time.Sleep(4 * time.Millisecond)
	}
	return posted, ""
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

// excludeFromClipboardHistory marks the content written in the current
// clipboard session. The caller holds the clipboard open.
func excludeFromClipboardHistory() bool {
	name, err := syscall.UTF16PtrFromString(clipboardHistoryExclusion)
	if err != nil {
		return false
	}
	format, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))
	if format == 0 {
		return false
	}
	handle, _, _ := procGlobalAlloc.Call(gmemMoveable|gmemZeroInit, 4)
	if handle == 0 {
		return false
	}
	ok, _, _ := procSetClipboardData.Call(format, handle)
	return ok != 0
}

// publishClipboard leaves text on the clipboard and does not paste it.
func publishClipboard(text string) bool {
	if text == "" {
		return false
	}
	if ok, _, _ := procOpenClipboard.Call(0); ok == 0 {
		return false
	}
	written := writeClipboardText(text)
	procCloseClipboard.Call()
	return written
}

// paste writes the text, sends Ctrl+V and restores the previous text only
// if no other program changed the clipboard meanwhile (DSK-A05). Only the
// Unicode text format is preserved on Windows. The transient text is
// excluded from clipboard history and cloud clipboard; when that mark cannot
// be set, the previous text is put back and the paste fails.
func paste(text string) bool {
	if ok, _, _ := procOpenClipboard.Call(0); ok == 0 {
		return false
	}
	previous, hadText := readClipboardText()
	written := writeClipboardText(text)
	if written && !excludeFromClipboardHistory() {
		procEmptyClipboard.Call()
		if hadText {
			writeClipboardText(previous)
		}
		written = false
	}
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
