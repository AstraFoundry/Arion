// Package pastephase names one system-paste transaction on macOS and Windows.
// A failed restore is its own phase and is not a successful paste.
package pastephase

import (
	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// Phase is where a paste transaction is. The names are the transaction,
// not collapsed return codes.
type Phase string

const (
	Prepared      Phase = "prepared"
	Staged        Phase = "staged"
	Posted        Phase = "posted"
	Restored      Phase = "restored"
	RestoreFailed Phase = "restore-failed"
	Left          Phase = "left"
	Copied        Phase = "copied"
)

// Restore classifies clipboard cleanup after a posted paste.
// owns is false when another owner changed the clipboard: that clipboard
// is left alone. hadText is false when the previous board was empty, so
// clearing it is a restore. A failed write of previous text is not a restore.
func Restore(owns, hadText, wrote bool) Phase {
	if !owns {
		return Left
	}
	if hadText && !wrote {
		return RestoreFailed
	}
	return Restored
}

// NoteRestoreFailure records a failed restore on the commit result.
// An already posted paste stays an uncertain outcome, and the failure is
// what keeps that outcome from looking like a clean paste. Restored and
// left do not replace an earlier failure, so a destination change can
// still fall back to a copy.
func NoteRestoreFailure(result *contracts.CommitResult, phase Phase) {
	if phase == RestoreFailed {
		result.Failure = domain.CodeClipboardRestoreFailed
	}
}
