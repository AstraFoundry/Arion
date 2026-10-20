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

// PhaseOf maps the session state to an overlay phase at time now.
func PhaseOf(d runtime.Dictation, now time.Time) Phase {
	since := now.Sub(d.Changed)
	switch d.State {
	case runtime.StatePreparing:
		return Preparing
	case runtime.StateCapturing:
		return Listening
	case runtime.StateFinalizing, runtime.StateReviewing:
		return Transcribing
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
func Active(d runtime.Dictation) bool {
	switch d.State {
	case runtime.StatePreparing, runtime.StateCapturing, runtime.StateFinalizing, runtime.StateReviewing, runtime.StateCommitting:
		return true
	}
	return false
}

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
	case runtime.CodeModelMissing, runtime.CodeModelUnsupported, runtime.CodeArtifactUnverified:
		return Notice{copy.FailModelTitle, copy.FailModelHint}
	case runtime.CodeAudioDiscontinuity:
		return Notice{copy.FailDiscontinuityTitle, copy.FailDiscontinuityHint}
	case runtime.CodeDeviceUnavailable:
		return Notice{copy.FailDeviceTitle, copy.FailDeviceHint}
	case runtime.CodeTargetChanged:
		return Notice{copy.FailTargetTitle, copy.FailTargetHint}
	case runtime.CodeTargetUnsupported:
		if d.FailureDetail == runtime.DetailInputMethod {
			return Notice{copy.FailIMETitle, copy.FailIMEHint}
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

// OutcomeLabel names a completed commit honestly: only a verified
// insertion is "Inserted".
func OutcomeLabel(d runtime.Dictation, pasted bool) copy.Key {
	switch {
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
