//go:build darwin

package macos

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// Serialize the shared system pasteboard across all target-provider instances.
// Waiting for ownership remains cancellable and never changes the clipboard.
var pasteGate = make(chan struct{}, 1)

type pasteTarget interface {
	validate() domain.ErrorCode
	prepare(string) pasteAttempt
}

type pasteAttempt interface {
	stage() domain.ErrorCode
	post() domain.ErrorCode
	finish() bool
}

// publishClipboard leaves text on the general pasteboard. It does not paste
// and does not restore the previous contents.
func publishClipboard(ctx context.Context, id contracts.Identity, text string) (contracts.CommitResult, error) {
	result := contracts.CommitResult{Identity: id, Outcome: contracts.NotApplied}
	if text == "" {
		result.Failure = domain.CodeNoSpeech
		return result, nil
	}
	if strings.IndexByte(text, 0) >= 0 || !utf8.ValidString(text) {
		result.Failure = domain.CodeTargetUnsupported
		return result, nil
	}
	select {
	case pasteGate <- struct{}{}:
		defer func() { <-pasteGate }()
	case <-ctx.Done():
		result.Failure = domain.CodeCancelled
		return result, nil
	}
	if ctx.Err() != nil {
		result.Failure = domain.CodeCancelled
		return result, nil
	}
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	if C.arion_clipboard_publish(value) != 1 {
		result.Failure = domain.CodeClipboardUnavailable
		return result, nil
	}
	result.Outcome = contracts.Copied
	return result, nil
}

func commitPaste(ctx context.Context, id contracts.Identity, text string, hold time.Duration, target pasteTarget) (result contracts.CommitResult, err error) {
	result = contracts.CommitResult{Identity: id, Outcome: contracts.NotApplied}
	if text == "" {
		result.Failure = domain.CodeNoSpeech
		return result, nil
	}
	if strings.IndexByte(text, 0) >= 0 || !utf8.ValidString(text) {
		result.Failure = domain.CodeTargetUnsupported
		return result, nil
	}
	select {
	case pasteGate <- struct{}{}:
		defer func() { <-pasteGate }()
	case <-ctx.Done():
		result.Failure = domain.CodeCancelled
		return result, nil
	}
	validate := func() domain.ErrorCode {
		if ctx.Err() != nil {
			return domain.CodeCancelled
		}
		return target.validate()
	}
	if result.Failure = validate(); result.Failure != "" {
		return result, nil
	}
	tx := target.prepare(text)
	if tx == nil {
		result.Failure = domain.CodeClipboardUnavailable
		return result, nil
	}
	defer func() {
		if !tx.finish() {
			result.Failure = domain.CodeClipboardRestoreFailed
		}
	}()
	if result.Failure = validate(); result.Failure != "" {
		return result, nil
	}
	if result.Failure = tx.stage(); result.Failure != "" {
		return result, nil
	}
	if result.Failure = validate(); result.Failure != "" {
		return result, nil
	}
	if result.Failure = tx.post(); result.Failure != "" {
		return result, nil
	}
	result.Outcome = contracts.Unknown
	// A hold period gives the host time to consume the board. It is not an
	// acknowledgement. Cancellation after posting cannot undo the attempt
	// or shorten clipboard retention; cleanup must still run exactly once.
	if hold > 0 {
		time.Sleep(hold)
	}
	return result, nil
}

type nativePasteTarget struct{ target *target }

func (t nativePasteTarget) validate() domain.ErrorCode {
	if C.arion_focus_matches(t.target.element, C.int(t.target.pid)) != 1 ||
		C.arion_selection_matches(t.target.element, t.target.selection) != 1 {
		return domain.CodeTargetChanged
	}
	if C.arion_secure_input() == 1 {
		return domain.CodeSecureFieldBlocked
	}
	return ""
}

func (t nativePasteTarget) prepare(text string) pasteAttempt {
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	tx := C.arion_paste_begin(value)
	if tx == nil {
		return nil
	}
	return &nativePasteAttempt{transaction: tx, target: t.target}
}

type nativePasteAttempt struct {
	transaction unsafe.Pointer
	target      *target
}

func pasteFailure(status C.int) domain.ErrorCode {
	switch status {
	case C.ARION_PASTE_OK:
		return ""
	case C.ARION_PASTE_TARGET_CHANGED:
		return domain.CodeTargetChanged
	case C.ARION_PASTE_SECURE:
		return domain.CodeSecureFieldBlocked
	default:
		return domain.CodeClipboardUnavailable
	}
}

func (t *nativePasteAttempt) stage() domain.ErrorCode {
	return pasteFailure(C.arion_paste_stage(t.transaction))
}
func (t *nativePasteAttempt) post() domain.ErrorCode {
	return pasteFailure(C.arion_paste_post(t.transaction, t.target.element, t.target.selection, C.int(t.target.pid)))
}
func (t *nativePasteAttempt) finish() bool {
	return C.arion_paste_finish(t.transaction) >= 0
}
