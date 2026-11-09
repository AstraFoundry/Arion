// Package dictation projects the live session into what the UI shows.
package dictation

import (
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

// Phase is the overlay's visual state.
type Phase int

const (
	Hidden Phase = iota
	Preparing
	Listening
	Transcribing
	// Polishing is the final rewrite organizing the recognized text.
	Polishing
	// Reviewing holds the final text, decided, before it is inserted.
	Reviewing
	Inserting
	Done
	Failed
	Cancelled
)

// How long outcomes stay visible before the overlay leaves.
const (
	DoneVisible      = 1100 * time.Millisecond
	FailedVisible    = 3600 * time.Millisecond
	CancelledVisible = 700 * time.Millisecond
	// LearnedVisible keeps a learned entry's undo on screen, as long as
	// Typeless shows its "added to dictionary" alert.
	LearnedVisible = 5 * time.Second
)

// LearnedShown reports whether the learned notice is still on screen; a
// running session hides it.
func LearnedShown(l runtime.Learned, d runtime.Dictation, now time.Time) bool {
	return l.ID != "" && !Active(d) && now.Sub(l.At) < LearnedVisible
}

// PolishNoteVisible is how long the note stays after the outcome leaves.
const PolishNoteVisible = 3200 * time.Millisecond

// PolishNoteShown reports whether the note for a cloud rewrite that fell back
// to local cleanup is on screen. It follows the Done outcome, because the
// dictation itself succeeded.
func PolishNoteShown(d runtime.Dictation, now time.Time) bool {
	if d.PolishNote == "" || d.State != runtime.StateCompleted || d.Failure != "" {
		return false
	}
	since := now.Sub(d.Changed)
	return since >= DoneVisible && since < DoneVisible+PolishNoteVisible
}

// PolishNoteOf explains a cloud rewrite that fell back to local cleanup. The
// provider picks key or sign-in wording, and the reasoning switch decides
// whether allowing reasoning is still a next step.
func PolishNoteOf(detail, provider string, reasoningAllowed bool) (Notice, bool) {
	switch detail {
	case runtime.PolishReasoningRequired:
		if reasoningAllowed {
			return Notice{copy.PolishNoteRefusedTitle, copy.PolishNoteRefusedHint}, true
		}
		return Notice{copy.PolishNoteReasoningTitle, copy.PolishNoteReasoningHint}, true
	case runtime.PolishRateLimited:
		return Notice{copy.PolishNoteRateTitle, copy.PolishNoteRateHint}, true
	case runtime.PolishQuotaExhausted:
		return Notice{copy.PolishNoteQuotaTitle, copy.PolishNoteQuotaHint}, true
	case runtime.PolishCredentialRejected:
		if provider == runtime.CloudChatGPT {
			return Notice{copy.PolishNoteSignInTitle, copy.PolishNoteSignInHint}, true
		}
		return Notice{copy.PolishNoteCredentialTitle, copy.PolishNoteCredentialHint}, true
	case runtime.PolishModelUnavailable:
		return Notice{copy.PolishNoteOutageTitle, copy.PolishNoteOutageHint}, true
	case runtime.PolishModelNotFound:
		return Notice{copy.PolishNoteNotFoundTitle, copy.PolishNoteNotFoundHint}, true
	case runtime.PolishTimeout:
		return Notice{copy.PolishNoteTimeoutTitle, copy.PolishNoteTimeoutHint}, true
	}
	return Notice{}, false
}

// PolishNoteRemaining is how long until the note's visibility changes.
func PolishNoteRemaining(d runtime.Dictation, now time.Time) time.Duration {
	if d.PolishNote == "" || d.State != runtime.StateCompleted || d.Failure != "" {
		return 0
	}
	return max(DoneVisible+PolishNoteVisible-now.Sub(d.Changed), 0)
}

// PhaseOf maps the session state to an overlay phase at time now.
func PhaseOf(d runtime.Dictation, now time.Time) Phase {
	since := now.Sub(d.Changed)
	switch d.State {
	case runtime.StatePreparing:
		return Preparing
	case runtime.StateCapturing:
		return Listening
	case runtime.StateFinalizing:
		if d.Rewriting {
			return Polishing
		}
		return Transcribing
	case runtime.StateReviewing:
		return Reviewing
	case runtime.StateCommitting:
		return Inserting
	case runtime.StateCompleted:
		if d.Failure != "" {
			if since < FailedVisible {
				return Failed
			}
			return Hidden
		}
		if since < DoneVisible {
			return Done
		}
	case runtime.StateFailed:
		if since < FailedVisible {
			return Failed
		}
	case runtime.StateCancelled:
		if since < CancelledVisible {
			return Cancelled
		}
	}
	return Hidden
}

// Remaining is how long the current outcome stays, for scheduling a frame.
func Remaining(d runtime.Dictation, now time.Time) time.Duration {
	visible := map[runtime.SessionState]time.Duration{
		runtime.StateCompleted: DoneVisible, runtime.StateFailed: FailedVisible, runtime.StateCancelled: CancelledVisible,
	}[d.State]
	if d.State == runtime.StateCompleted && d.Failure != "" {
		visible = FailedVisible
	}
	if visible == 0 {
		return 0
	}
	return max(visible-now.Sub(d.Changed), 0)
}

// Active reports a session that holds the microphone or pending work.
func Active(d runtime.Dictation) bool { return d.Active() }

// Notice explains a failure with its next step.
type Notice struct {
	Title copy.Key
	Hint  copy.Key
}

// NoticeOf explains the session failure. A permission failure names the
// permission that is actually missing.
func NoticeOf(d runtime.Dictation, permissions runtime.Permissions) Notice {
	switch d.Failure {
	case runtime.CodePermissionDenied:
		switch {
		case d.FailureDetail == runtime.DetailMicrophone:
			return Notice{copy.FailMicTitle, copy.FailMicHint}
		case d.FailureDetail == runtime.DetailAccessibility:
			return Notice{copy.FailAXTitle, copy.FailAXHint}
		case permissions.Microphone != runtime.ConsentGranted:
			return Notice{copy.FailMicTitle, copy.FailMicHint}
		}
		return Notice{copy.FailAXTitle, copy.FailAXHint}
	case runtime.CodeArtifactUnverified:
		if d.FailureDetail == runtime.DetailIMEClientOutdated {
			return Notice{copy.FailIMEProtocolTitle, copy.FailIMEProtocolHint}
		}
		if d.FailureDetail == runtime.DetailIMEArtifactUnverified {
			return Notice{copy.FailIMEArtifactTitle, copy.FailIMEArtifactHint}
		}
		return Notice{copy.FailModelTitle, copy.FailModelHint}
	case runtime.CodeContractUnsupported:
		if d.FailureDetail == runtime.DetailIMEProtocolUnsupported {
			return Notice{copy.FailIMEProtocolTitle, copy.FailIMEProtocolHint}
		}
	case runtime.CodeModelMissing, runtime.CodeModelUnsupported:
		if d.FailureDetail == runtime.DetailTranslationModel || d.Translating {
			if d.Failure == runtime.CodeModelMissing {
				return Notice{copy.FailTranslateModelTitle, copy.FailTranslateModelHint}
			}
			return Notice{copy.FailTranslateTitle, copy.FailTranslateHint}
		}
		return Notice{copy.FailModelTitle, copy.FailModelHint}
	case runtime.CodeRewriteTimeout:
		return Notice{copy.FailTranslateTimeoutTitle, copy.FailTranslateHint}
	case runtime.CodeRewriteRejected:
		return Notice{copy.FailTranslateTitle, copy.FailTranslateHint}
	case runtime.CodeAudioDiscontinuity:
		return Notice{copy.FailDiscontinuityTitle, copy.FailDiscontinuityHint}
	case runtime.CodeDeviceUnavailable:
		return Notice{copy.FailDeviceTitle, copy.FailDeviceHint}
	case runtime.CodeTargetChanged:
		return Notice{copy.FailTargetTitle, copy.FailTargetHint}
	case runtime.CodeTargetUnsupported:
		switch d.FailureDetail {
		case runtime.DetailInputMethod:
			return Notice{copy.FailIMETitle, copy.FailIMEHint}
		case runtime.DetailIMEClient:
			return Notice{copy.FailUnsupportedTitle, copy.FailIMEClientHint}
		case runtime.DetailIMEUpdating:
			return Notice{copy.FailIMEUpdatingTitle, copy.FailIMEUpdatingHint}
		}
		return Notice{copy.FailUnsupportedTitle, copy.FailTargetHint}
	case runtime.CodeClipboardUnavailable:
		return Notice{copy.FailClipboardTitle, copy.FailClipboardHint}
	case runtime.CodeClipboardRestoreFailed:
		return Notice{copy.FailClipboardRestoreTitle, copy.FailClipboardRestoreHint}
	case runtime.CodeSecureFieldBlocked:
		return Notice{copy.FailSecureTitle, copy.FailSecureHint}
	case runtime.CodeNoSpeech:
		return Notice{copy.FailNoSpeechTitle, copy.FailNoSpeechHint}
	case runtime.CodeSessionBusy:
		return Notice{copy.FailBusyTitle, ""}
	}
	return Notice{copy.FailGenericTitle, ""}
}

// ShortcutCommitPasted is the pill's paste mark for one shortcut commit.
// Both Windows delivery preferences post one system paste, the same as
// macOS. A posted paste is unknown and labeled pasted (ADR 003). Copied
// and not-applied commits are not that label.
func ShortcutCommitPasted(mode runtime.SessionMode, outcome runtime.CommitOutcome) bool {
	return mode == runtime.ModeShortcut && outcome != runtime.OutcomeNotApplied && outcome != runtime.OutcomeCopied
}

// OutcomeLabel names a completed commit honestly: only a verified
// insertion is "Inserted".
func OutcomeLabel(d runtime.Dictation, pasted bool) copy.Key {
	switch {
	case d.Outcome == runtime.OutcomeCopied:
		return copy.PillCopied
	case d.Outcome == runtime.OutcomeApplied:
		return copy.PillInserted
	case pasted:
		return copy.PillPasted
	default:
		return copy.PillTyped
	}
}

// Text is what the pill shows while a session runs.
func Text(d runtime.Dictation) string {
	if d.Final != "" {
		return d.Final
	}
	return d.Draft
}
