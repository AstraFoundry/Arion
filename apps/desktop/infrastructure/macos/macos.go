//go:build darwin

// Package macos implements permission, target and insertion adapters with
// AVFoundation, Accessibility (AX) and Quartz events.
package macos

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework AVFoundation -framework Carbon
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// AuthStatus mirrors the OS consent state of a capability.
type AuthStatus string

const (
	AuthUndetermined AuthStatus = "undetermined"
	AuthDenied       AuthStatus = "denied"
	AuthRestricted   AuthStatus = "restricted"
	AuthGranted      AuthStatus = "granted"
)

// Permissions reads and requests microphone and accessibility consent.
type Permissions struct{}

func (Permissions) Microphone() AuthStatus {
	switch C.arion_mic_status() {
	case C.ARION_AUTH_GRANTED:
		return AuthGranted
	case C.ARION_AUTH_DENIED:
		return AuthDenied
	case C.ARION_AUTH_RESTRICTED:
		return AuthRestricted
	default:
		return AuthUndetermined
	}
}

func (p Permissions) MicrophoneAuthorized() bool { return p.Microphone() == AuthGranted }

// RequestMicrophone shows the system prompt when consent is undetermined.
func (Permissions) RequestMicrophone() { C.arion_mic_request() }

func (Permissions) Accessibility() AuthStatus {
	if C.arion_ax_trusted(0) == 1 {
		return AuthGranted
	}
	return AuthUndetermined
}

// RequestAccessibility registers Arion in the Accessibility list and shows
// the system prompt that opens System Settings.
func (Permissions) RequestAccessibility() { C.arion_ax_trusted(1) }

// Pane identifies a System Settings privacy pane.
type Pane string

const (
	PaneMicrophone    Pane = "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone"
	PaneAccessibility Pane = "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"
)

func (Permissions) Open(pane Pane) {
	url := C.CString(string(pane))
	defer C.free(unsafe.Pointer(url))
	C.arion_open_url(url)
}

// Route is how a commit reaches the target editor. It is chosen once, at
// capture time, from declared capabilities; a commit never switches routes.
type Route string

const (
	RouteAccessibility Route = "accessibility"
	RouteKeystrokes    Route = "keystrokes"
	RouteClipboard     Route = "clipboard"
)

// Delivery is the user's insertion preference.
type Delivery string

const (
	DeliveryAutomatic Delivery = "automatic"
	DeliveryClipboard Delivery = "clipboard"
)

type target struct {
	element  unsafe.Pointer
	pid      int
	route    Route
	readable bool
	// secure and app gate context reads (DSK-08).
	secure  bool
	app     string
	created time.Time
}

// Targets implements ports.TargetProvider and ports.InputTarget.
type Targets struct {
	mu       sync.Mutex
	targets  map[contracts.TargetToken]*target
	delivery func() Delivery
	// ClipboardRestoreDelay waits for the host to read the pasted text.
	ClipboardRestoreDelay time.Duration
	// ReadSite reports whether Capture may read the host of the focused web
	// page for scene selection; nil never reads it.
	ReadSite func() bool
	// ReadCursor and ReadScreen gate ReadContext's text around the cursor
	// and the focused window's recognized text; nil never reads them.
	ReadCursor func() bool
	ReadScreen func() bool
	// ScreenTimeout bounds window capture.
	ScreenTimeout time.Duration
}

func NewTargets(delivery func() Delivery) *Targets {
	return &Targets{targets: map[contracts.TargetToken]*target{}, delivery: delivery, ClipboardRestoreDelay: 400 * time.Millisecond}
}

const keptTargets = 4

var chromiumBundles = map[string]bool{
	"com.google.Chrome": true, "com.google.Chrome.canary": true, "com.microsoft.edgemac": true, "com.brave.Browser": true,
	"company.thebrowser.Browser": true, "com.operasoftware.Opera": true, "com.vivaldi.Vivaldi": true, "org.chromium.Chromium": true,
}

// usesChromiumEditor reports hosts whose AX text attributes accept writes
// without applying them; those targets use keystroke delivery.
func usesChromiumEditor(bundleID, bundlePath string) bool {
	if chromiumBundles[bundleID] {
		return true
	}
	if bundlePath == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(bundlePath, "Contents", "Frameworks", "Electron Framework.framework"))
	return err == nil
}

func newToken() contracts.TargetToken {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return contracts.TargetToken(hex.EncodeToString(b[:]))
}

// Capture snapshots the focused editor. Without Accessibility trust it
// returns PERMISSION_DENIED having read nothing (DSK-A01).
func (t *Targets) Capture(_ context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	var native C.arion_target
	readSite := C.int(0)
	if t.ReadSite != nil && t.ReadSite() {
		readSite = 1
	}
	switch C.arion_capture_target(&native, readSite) {
	case C.ARION_TARGET_UNTRUSTED:
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodePermissionDenied, "accessibility")
	case C.ARION_TARGET_NONE:
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "no frontmost application")
	}
	role, subrole := C.GoString(&native.role[0]), C.GoString(&native.subrole[0])
	bundle, bundlePath := C.GoString(&native.bundle[0]), C.GoString(&native.bundle_path[0])
	secure := native.secure_input == 1 || subrole == "AXSecureTextField"
	known := native.element != nil && role != ""
	tgt := &target{element: native.element, pid: int(native.pid), readable: native.value_readable == 1, secure: secure, app: bundle, created: time.Now()}
	switch {
	case t.delivery() == DeliveryClipboard:
		tgt.route = RouteClipboard
	case native.selected_text_settable == 1 && tgt.readable && !usesChromiumEditor(bundle, bundlePath):
		tgt.route = RouteAccessibility
	default:
		tgt.route = RouteKeystrokes
	}
	token := newToken()
	t.mu.Lock()
	t.targets[token] = tgt
	t.evictLocked()
	t.mu.Unlock()
	return contracts.TargetSnapshot{
		Identity: id, Target: token, Sensitive: secure, Application: bundle,
		Field: fieldKind(role, subrole), Site: C.GoString(&native.site[0]), ApplicationName: C.GoString(&native.app_name[0]),
		Capabilities: contracts.Capabilities{
			// DSK-08: context is readable only from a known, non-secure field.
			ContextRead:     known && !secure && tgt.readable,
			SingleInsertion: true,
		},
	}, nil
}

// FocusOwner returns the process of the frontmost app, read before an
// Arion window takes the focus, for a later Peek.
func (t *Targets) FocusOwner() int { return int(C.arion_frontmost_pid()) }

// Peek describes the focused field of owner for display: app, field type
// and, when ReadSite allows, the page host. It keeps no target and reads no
// field text; Arion itself and unknown processes are TARGET_UNSUPPORTED.
func (t *Targets) Peek(_ context.Context, owner int) (contracts.TargetSnapshot, error) {
	readSite := C.int(0)
	if t.ReadSite != nil && t.ReadSite() {
		readSite = 1
	}
	var native C.arion_target
	switch C.arion_peek_target(C.int(owner), &native, readSite) {
	case C.ARION_TARGET_UNTRUSTED:
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodePermissionDenied, "accessibility")
	case C.ARION_TARGET_NONE:
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "no other application")
	}
	role, subrole := C.GoString(&native.role[0]), C.GoString(&native.subrole[0])
	return contracts.TargetSnapshot{
		Application: C.GoString(&native.bundle[0]), ApplicationName: C.GoString(&native.app_name[0]),
		Field: fieldKind(role, subrole), Site: C.GoString(&native.site[0]),
		Sensitive: native.secure_input == 1 || subrole == "AXSecureTextField",
	}, nil
}

// fieldKind maps AX roles to field types. It follows Chatterfly's context
// builder (search-field subrole is search, text fields and areas are text)
// and also counts editable combo boxes, which web search boxes report.
func fieldKind(role, subrole string) domain.FieldKind {
	switch {
	case subrole == "AXSearchField":
		return domain.FieldSearch
	case role == "AXTextField" || role == "AXTextArea" || role == "AXComboBox":
		return domain.FieldText
	}
	return domain.FieldUnknown
}

func (t *Targets) evictLocked() {
	for len(t.targets) > keptTargets {
		var oldest contracts.TargetToken
		for token, candidate := range t.targets {
			if oldest == "" || candidate.created.Before(t.targets[oldest].created) {
				oldest = token
			}
		}
		C.arion_release(t.targets[oldest].element)
		delete(t.targets, oldest)
	}
}

// RouteOf reports the delivery route chosen for a captured target.
func (t *Targets) RouteOf(token contracts.TargetToken) Route {
	t.mu.Lock()
	defer t.mu.Unlock()
	if tgt := t.targets[token]; tgt != nil {
		return tgt.route
	}
	return ""
}

func (t *Targets) Capabilities(_ context.Context, token contracts.TargetToken) (contracts.Capabilities, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.targets[token] == nil {
		return contracts.Capabilities{}, domain.Fail(domain.CodeTargetChanged, "unknown target")
	}
	return contracts.Capabilities{SingleInsertion: true}, nil
}

// ReadContext is not offered: Arion does not read foreign field content in
// this release, so the capability stays declared unsupported.
// ReadContext reads the context the user allows for a captured target:
// text around its cursor and the recognized text of its window. Secure
// fields and denied apps read nothing (DSK-08); each source that fails is
// left empty, so the rewrite proceeds with whatever was read.
func (t *Targets) ReadContext(_ context.Context, id contracts.Identity, token contracts.TargetToken) (contracts.ContextSnapshot, error) {
	snap := contracts.ContextSnapshot{Identity: id, Target: token}
	t.mu.Lock()
	tgt := t.targets[token]
	if tgt == nil {
		t.mu.Unlock()
		return snap, domain.Fail(domain.CodeTargetChanged, "unknown target")
	}
	if tgt.secure || domain.ContextDenied(tgt.app) {
		t.mu.Unlock()
		return snap, nil
	}
	snap.SafeToRead = true
	if t.ReadCursor != nil && t.ReadCursor() && tgt.readable && tgt.element != nil {
		// The element stays retained while the lock prevents its eviction.
		snap.Before, snap.After, _ = cursorContext(tgt.element, domain.MaxCursorContext)
	}
	pid := tgt.pid
	t.mu.Unlock()
	if t.ReadScreen != nil && t.ReadScreen() {
		timeout := t.ScreenTimeout
		if timeout <= 0 {
			timeout = 1500 * time.Millisecond
		}
		if screen, err := RecognizeWindow(pid, OCRAccurate, timeout); err == nil {
			snap.Screen, snap.ScreenCaptureMs, snap.ScreenOCRMs = screen.Text, screen.CaptureMs, screen.OCRMs
		}
	}
	return snap, nil
}

// Commit revalidates focus and inserts once through the captured route.
func (t *Targets) Commit(_ context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	result := contracts.CommitResult{Identity: request.Identity, Outcome: contracts.NotApplied}
	t.mu.Lock()
	tgt := t.targets[request.Target]
	delete(t.targets, request.Target)
	t.mu.Unlock()
	if tgt == nil {
		result.Failure = domain.CodeTargetChanged
		return result, nil
	}
	defer C.arion_release(tgt.element)
	// DSK-A03: a focus change means the user must act; nothing is inserted.
	if C.arion_focus_matches(tgt.element, C.int(tgt.pid)) != 1 {
		result.Failure = domain.CodeTargetChanged
		return result, nil
	}
	if C.arion_secure_input() == 1 {
		result.Failure = domain.CodeSecureFieldBlocked
		return result, nil
	}
	text := request.Text
	before := int64(-1)
	if tgt.readable {
		before = int64(C.arion_value_length(tgt.element))
	}
	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))
	switch tgt.route {
	case RouteAccessibility:
		if C.arion_ax_insert(tgt.element, ctext) != 0 {
			result.Failure = domain.CodeTargetUnsupported
			return result, nil
		}
	case RouteKeystrokes:
		typed := C.CString(singleLine(text))
		defer C.free(unsafe.Pointer(typed))
		if C.arion_type_unicode(typed) != 0 {
			result.Failure = domain.CodeTargetUnsupported
			return result, nil
		}
	case RouteClipboard:
		owned := C.arion_paste(ctext)
		time.Sleep(t.ClipboardRestoreDelay)
		C.arion_restore_clipboard(owned)
	}
	result.Outcome = contracts.Unknown
	if before >= 0 {
		// Keystroke delivery is asynchronous; give the host a moment.
		deadline := time.Now().Add(300 * time.Millisecond)
		for {
			after := int64(C.arion_value_length(tgt.element))
			if after > before {
				result.Outcome = contracts.Applied
				break
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(15 * time.Millisecond)
		}
	}
	return result, nil
}

// singleLine replaces line breaks: synthesized Return could send a message
// or run a terminal command, which Arion never does.
func singleLine(text string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text)), " ")
}

// UTF16Length counts host string units for AX range conversion (CTR-A05).
func UTF16Length(text string) int { return len(utf16.Encode([]rune(text))) }
