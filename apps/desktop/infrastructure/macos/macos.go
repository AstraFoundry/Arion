//go:build darwin

// Package macos implements permission, target and insertion adapters with
// AVFoundation, Accessibility (AX) and Quartz events.
package macos

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework AVFoundation -framework Carbon -framework Security
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
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

// Route names the only shortcut delivery transport on macOS.
type Route string

const RouteClipboard Route = "clipboard"

type target struct {
	contextReadable bool
	element         unsafe.Pointer
	selection       unsafe.Pointer
	identity        contracts.Identity
	pid             int
	route           Route
	readable        bool
	// secure and app gate context reads (DSK-08).
	secure  bool
	app     string
	created time.Time
	// follow keeps the element after Commit for edit learning (ADR 014).
	follow bool
}

// followed is a committed field kept readable for edit learning. Its own
// lock serializes reads with the release, so a slow host never holds the
// Targets lock that capture and commit need.
type followed struct {
	mu      sync.Mutex
	element unsafe.Pointer
	pid     int
}

// keptFollowed bounds followed fields; the tracker follows one at a time.
const keptFollowed = 4

// Targets implements ports.TargetProvider and ports.InputTarget.
type Targets struct {
	mu       sync.Mutex
	targets  map[contracts.TargetToken]*target
	followed map[contracts.TargetToken]*followed
	// ClipboardRestoreDelay is a consumption grace period, not a host acknowledgement.
	ClipboardRestoreDelay time.Duration
	// ReadSite reports whether Capture may read the host of the focused web
	// page for scene selection; nil never reads it.
	ReadSite func() bool
	// ReadCursor and ReadScreen gate ReadContext's text around the cursor
	// and visible Accessibility window text; nil never reads them.
	ReadCursor func() bool
	ReadScreen func() bool
	// ScreenTimeout bounds Accessibility window traversal.
	ScreenTimeout time.Duration
}

func NewTargets() *Targets {
	return &Targets{targets: map[contracts.TargetToken]*target{}, followed: map[contracts.TargetToken]*followed{},
		ClipboardRestoreDelay: 400 * time.Millisecond}
}

const keptTargets = 4

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
	}
	role, subrole := C.GoString(&native.role[0]), C.GoString(&native.subrole[0])
	bundle := C.GoString(&native.bundle[0])
	secure := native.secure_input == 1 || subrole == "AXSecureTextField"
	known := native.element != nil && fieldKind(role, subrole) != domain.FieldUnknown
	tgt := &target{element: native.element, selection: native.selection, identity: id, pid: int(native.pid), readable: native.value_readable == 1, secure: secure, app: bundle, created: time.Now()}
	tgt.contextReadable = known && !secure && tgt.readable
	tgt.route = RouteClipboard
	token := newToken()
	t.mu.Lock()
	t.targets[token] = tgt
	t.evictLocked()
	t.mu.Unlock()
	return contracts.TargetSnapshot{
		Identity: id, Target: token, Sensitive: secure, Application: bundle,
		Field: fieldKind(role, subrole), Site: C.GoString(&native.site[0]), ApplicationName: C.GoString(&native.app_name[0]),
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
		C.arion_release(t.targets[oldest].selection)
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

// ReadContext reads the context the user allows for a captured target:
// text around its cursor and visible Accessibility text in its window.
// Secure fields and denied apps read nothing (DSK-08). Unavailable window
// text is an explicit failure; screenshots and OCR are never substituted.
func (t *Targets) ReadContext(ctx context.Context, id contracts.Identity, token contracts.TargetToken) (contracts.ContextSnapshot, error) {
	snap := contracts.ContextSnapshot{Identity: id, Target: token}
	t.mu.Lock()
	tgt := t.targets[token]
	if tgt == nil || tgt.identity != id {
		t.mu.Unlock()
		return snap, domain.Fail(domain.CodeTargetChanged, "context target changed")
	}
	if !tgt.contextReadable || domain.ContextDenied(tgt.app) {
		t.mu.Unlock()
		return snap, nil
	}
	// Retain this read's element so eviction can proceed without waiting on AX.
	element, pid := C.arion_retain(tgt.element), tgt.pid
	t.mu.Unlock()
	defer C.arion_release(element)
	if ctx.Err() != nil {
		return snap, domain.Fail(domain.CodeCancelled, "context cancelled")
	}
	if C.arion_focus_matches(element, C.int(pid)) != 1 {
		return snap, domain.Fail(domain.CodeTargetChanged, "context target changed")
	}
	snap.SafeToRead = true
	if t.ReadCursor != nil && t.ReadCursor() {
		snap.Before, snap.After, _ = cursorContext(element, domain.MaxCursorContext)
	}
	if t.ReadScreen != nil && t.ReadScreen() {
		timeout := t.ScreenTimeout
		if timeout <= 0 {
			timeout = 1500 * time.Millisecond
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < timeout {
			timeout = time.Until(deadline)
		}
		screen, err := ReadWindowText(element, pid, timeout)
		if err != nil {
			return snap, err
		}
		snap.Screen, snap.ScreenCaptureMs = screen.Text, screen.CaptureMs
	}
	if ctx.Err() != nil {
		return snap, domain.Fail(domain.CodeCancelled, "context cancelled")
	}
	if C.arion_focus_matches(element, C.int(pid)) != 1 {
		return snap, domain.Fail(domain.CodeTargetChanged, "context target changed")
	}
	return snap, nil
}

// Commit inserts once at the caret focused now. A caret move, a switch to
// another editable field, or an edit in the captured field still inserts
// there. A secure field is refused. With no editable focus, the text is left
// on the system clipboard (DSK-06).
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
	releaseElement := true
	defer func() {
		C.arion_release(tgt.selection)
		if releaseElement {
			C.arion_release(tgt.element)
		}
	}()
	// A capture already marked secure never follows the caret or the
	// clipboard. The snapshot is enough, so this returns before Accessibility
	// or pasteboard access.
	if tgt.secure {
		result.Failure = domain.CodeSecureFieldBlocked
		return result, nil
	}
	paste, secure, noInput := adoptLiveCursor(tgt)
	if secure {
		result.Failure = domain.CodeSecureFieldBlocked
		return result, nil
	}
	if !paste {
		copied, err := publishClipboard(ctx, request.Identity, request.Text)
		copied.NoInput = noInput && copied.Outcome == contracts.Copied
		return copied, err
	}
	if tgt.follow {
		t.keepFollowed(request.Target, tgt)
		releaseElement = false
	}
	pasted, err := commitPaste(ctx, request.Identity, request.Text, t.ClipboardRestoreDelay, nativePasteTarget{tgt})
	if pasted.Outcome == contracts.NotApplied && pasted.Failure == domain.CodeTargetChanged {
		return publishClipboard(ctx, request.Identity, request.Text)
	}
	return pasted, err
}

// adoptLiveCursor points tgt at the editable field focused now.
// paste is false when nothing editable is focused. secure is set when the
// focused field must not receive text.
func adoptLiveCursor(tgt *target) (paste bool, secure bool, noInput bool) {
	var native C.arion_target
	if C.arion_capture_target(&native, 0) != C.ARION_TARGET_OK {
		return false, false, false
	}
	role, subrole := C.GoString(&native.role[0]), C.GoString(&native.subrole[0])
	liveSecure := native.secure_input == 1 || subrole == "AXSecureTextField"
	ownProcess := int(native.pid) == os.Getpid()
	editable := native.element != nil && !ownProcess && (native.editable == 1 && fieldKind(role, subrole) != domain.FieldUnknown)
	if !editable || liveSecure {
		C.arion_release(native.element)
		C.arion_release(native.selection)
		return false, liveSecure && native.element != nil && !ownProcess, !liveSecure && !ownProcess && (native.no_input == 1 || (native.selection == nil && fieldKind(role, subrole) == domain.FieldUnknown && role != ""))
	}
	if tgt.element == nil || C.arion_focus_matches(tgt.element, C.int(tgt.pid)) != 1 {
		tgt.follow = false
	}
	C.arion_release(tgt.element)
	C.arion_release(tgt.selection)
	tgt.element = native.element
	tgt.selection = native.selection
	tgt.pid = int(native.pid)
	tgt.secure = false
	tgt.readable = native.value_readable == 1
	return true, false, false
}

// Follow marks a captured target to stay readable after Commit. Secure,
// denied and unreadable fields are refused and never read (DSK-08).
func (t *Targets) Follow(token contracts.TargetToken) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	tgt := t.targets[token]
	if tgt == nil || tgt.secure || !tgt.readable || tgt.element == nil || domain.ContextDenied(tgt.app) {
		return false
	}
	tgt.follow = true
	return true
}

func (t *Targets) keepFollowed(token contracts.TargetToken, tgt *target) {
	t.mu.Lock()
	t.followed[token] = &followed{element: tgt.element, pid: tgt.pid}
	var evict []*followed
	for other, f := range t.followed {
		if len(t.followed)-len(evict) <= keptFollowed {
			break
		}
		if other != token {
			evict = append(evict, f)
			delete(t.followed, other)
		}
	}
	t.mu.Unlock()
	for _, f := range evict {
		f.release()
	}
}

// ReadField reads a bounded excerpt around the followed field's cursor and
// whether it still has focus. Each AX call has its messaging timeout.
func (t *Targets) ReadField(_ context.Context, token contracts.TargetToken) (ports.FieldRead, error) {
	t.mu.Lock()
	f := t.followed[token]
	t.mu.Unlock()
	if f == nil {
		return ports.FieldRead{}, domain.Fail(domain.CodeTargetChanged, "field not followed")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.element == nil {
		return ports.FieldRead{}, domain.Fail(domain.CodeTargetChanged, "field released")
	}
	if C.arion_secure_input() == 1 {
		return ports.FieldRead{}, domain.Fail(domain.CodeSecureFieldBlocked, "secure input")
	}
	read := ports.FieldRead{Focused: C.arion_focus_matches(f.element, C.int(f.pid)) == 1}
	before, after, ok := cursorContext(f.element, domain.MaxEditRunes+domain.MaxCursorContext)
	if !ok {
		return read, domain.Fail(domain.CodeTargetUnsupported, "field unreadable")
	}
	read.Text = before + after
	return read, nil
}

// Release stops following a field and frees its element.
func (t *Targets) Release(token contracts.TargetToken) {
	t.mu.Lock()
	f := t.followed[token]
	delete(t.followed, token)
	t.mu.Unlock()
	if f != nil {
		f.release()
	}
}

func (f *followed) release() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.element != nil {
		C.arion_release(f.element)
		f.element = nil
	}
}

// UTF16Length counts host string units for AX range conversion (CTR-A05).
func UTF16Length(text string) int { return len(utf16.Encode([]rune(text))) }

// Discard releases a captured target that was used only for context.
func (t *Targets) Discard(token contracts.TargetToken) {
	t.mu.Lock()
	target := t.targets[token]
	delete(t.targets, token)
	t.mu.Unlock()
	if target != nil {
		C.arion_release(target.element)
		C.arion_release(target.selection)
	}
}
