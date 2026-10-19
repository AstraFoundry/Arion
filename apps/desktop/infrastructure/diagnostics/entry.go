// Package diagnostics stores bounded, content-free local support logs.
package diagnostics

import (
	"arion.local/models"
	"context"
	"encoding/hex"
	"errors"
	"math"
	"regexp"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// Entry deliberately has no message, error object or arbitrary attribute map.
// Strings are checked against closed sets before writing and again on export.
type Entry struct {
	QueueDepth    int    `json:"queueDepth,omitempty"`
	QueueOverflow bool   `json:"queueOverflow,omitempty"`
	Run           string `json:"run,omitempty"`
	ModelID       string `json:"modelId,omitempty"`
	Capability    string `json:"capability,omitempty"`
	Phase         string `json:"phase,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Reads         int    `json:"reads,omitempty"`
	Failures      int    `json:"failures,omitempty"`

	At         time.Time           `json:"at"`
	Event      string              `json:"event"`
	Session    string              `json:"session,omitempty"`
	Generation uint64              `json:"generation,omitempty"`
	Sequence   uint64              `json:"sequence,omitempty"`
	State      domain.SessionState `json:"state,omitempty"`
	Outcome    string              `json:"outcome,omitempty"`
	Code       domain.ErrorCode    `json:"code,omitempty"`
	DurationMs float64             `json:"durationMs,omitempty"`
	Count      int                 `json:"count,omitempty"`
}

var modelIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,95}$`)

var diagnosticCatalog = sync.OnceValues(models.Load)

var errInvalidEntry = errors.New("invalid diagnostic entry")

func (e Entry) valid() bool {
	switch e.Event {
	case "app_started", "app_stopped", "preferences_repaired", "preferences_save", "usage_load", "usage_save", "vocabulary_key", "vocabulary_load", "vocabulary_save", "vocabulary_recall", "edit_learning", "e2e_listen", "ime_listen", "ime_source_state", "ime_install_started", "ime_install_failed", "ime_install_pending", "ime_install_completed", "ime_settings_open", "ime_remove", "consent_guide", "shortcut_register", "hold_register", "permissions_changed", "devices_list", "model_status", "rewrite_status", "model_activate", "update_check", "update_install", "export_started", "export_completed", "export_failed", "export_cancelled", "events_dropped", "session_state", "session_error", "correction_ready", "commit_result", "first_draft", "target_capture", "recognizer_create", "audio_start", "acoustic_finalize", "context_read", "rewrite_preview", "rewrite_final", "target_commit", "segment_queue", "rewrite_validation", "dictation_toggle", "dictation_cancel", "power_suspend", "screen_locked", "hold_start", "hold_stop", "hold_cancel":
	default:
		return false
	}
	for _, id := range []string{e.Run, e.Session} {
		if id != "" {
			b, err := hex.DecodeString(id)
			if err != nil || len(b) != 8 {
				return false
			}
		}
	}
	if e.ModelID != "" && !modelIDPattern.MatchString(e.ModelID) {
		return false
	}

	switch e.State {
	case "", domain.Idle, domain.Preparing, domain.Capturing, domain.Finalizing, domain.Reviewing, domain.Committing, domain.Completed, domain.Cancelled, domain.Failed:
	default:
		return false
	}
	switch e.Outcome {
	case "", "absent", "pending", "installed", "selected", "learned", "cleared", "not_found", "outside", "too_large", "rewrite", "no_pair", "markup", "length", "new_number", "unsupported_content", "negation_dropped", "final_value_dropped", "content_dropped", "uncertainty_dropped", "context_copied", "ok", "failed", "cancelled", "unselected", "provisioning", "loading", "ready", "unsupported", "off", "downloading", "verifying", "extracting", "granted", "denied", "restricted", "undetermined", "accepted", "timeout", "unavailable", "preview", "applied", "not_applied", "unknown", "unchanged", "empty", "truncated", "too_long", "meaning_changed", "unsafe", "invalid", "rejected":
	default:
		return false
	}
	switch e.Capability {
	case "", "microphone", "accessibility", "input_monitoring", "screen_recording", "speech", "rewrite":
	default:
		return false
	}
	switch e.Phase {
	case "", "downloading", "verifying", "extracting", "ready":
	default:
		return false
	}
	switch e.Reason {
	case "", "idle", "focus_lost", "cleared", "next_dictation", "max_time", "abandoned", "no_baseline":
	default:
		return false
	}
	return e.QueueDepth >= 0 && e.Reads >= 0 && e.Failures >= 0 && validCode(e.Code) && !e.At.IsZero() && !math.IsNaN(e.DurationMs) && !math.IsInf(e.DurationMs, 0) && e.DurationMs >= 0 && e.Count >= 0
}

func validCode(code domain.ErrorCode) bool {
	switch code {
	case "", "IO_FAILED", domain.CodePermissionDenied, domain.CodeModelMissing, domain.CodeModelUnsupported, domain.CodeAudioDiscontinuity, domain.CodeTargetChanged, domain.CodeTargetUnsupported, domain.CodeStaleRevision, domain.CodeCancelled, domain.CodeInsertionUncertain, domain.CodeSyncUnauthorized, domain.CodeSyncConflict, domain.CodeContractUnsupported, domain.CodeDeviceUnavailable, domain.CodeInvalidTransition, domain.CodeSessionBusy, domain.CodeNoSpeech, domain.CodeArtifactUnverified, domain.CodeProvisioningFailed, domain.CodeSecureFieldBlocked, domain.CodeEvaluationUnavailable, domain.CodeRewriteRejected, domain.CodeRewriteTimeout:
		return true
	}
	return false
}

// Known models are checked at ingestion. Historical catalog IDs remain readable
// after an update removes a bundle; exports still enforce the bounded ID syntax.
func knownModel(id string) bool {
	if id == "" {
		return true
	}
	catalog, err := diagnosticCatalog()
	if err != nil {
		return false
	}
	_, bundle := catalog.Bundle(id)
	_, artifact := catalog.Artifact(id)
	_, rewriter := catalog.Rewriter(id)
	return bundle || artifact || rewriter
}

// Failure never serializes an error's message or unrecognized code.
func Failure(err error) domain.ErrorCode {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return domain.CodeCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.CodeRewriteTimeout
	}
	code := domain.CodeOf(err)
	if code != "" && validCode(code) {
		return code
	}
	return "IO_FAILED"
}

// Observe implements the domain diagnostic port. Core cannot send content.
func (r *Recorder) Observe(d ports.Diagnostic) {
	r.Record(Entry{Event: string(d.Stage), Session: d.Identity.SessionID, Generation: d.Identity.Generation,
		DurationMs: float64(d.Duration.Microseconds()) / 1000, QueueDepth: d.QueueDepth, QueueOverflow: d.QueueOverflow, Outcome: string(d.Verdict), Code: FailureCode(d.Failure)})
}

func FailureCode(code domain.ErrorCode) domain.ErrorCode {
	if validCode(code) {
		return code
	}
	return domain.CodeContractUnsupported
}

// Session projects only safe fields; text, raw, source, detail, target and scene
// never enter the recorder. Drafts are sampled once per session, audio ignored.
func (r *Recorder) Session(e contracts.Event) {
	name := ""
	switch e.Kind {
	case contracts.DraftReplaced:
		r.mu.Lock()
		first := r.draftSession == e.SessionID && !r.drafted
		if first {
			r.drafted = true
		}
		r.mu.Unlock()
		if !first {
			return
		}
		name = "first_draft"
	case contracts.SessionStateChanged:
		if e.State == domain.Preparing {
			r.mu.Lock()
			r.draftSession, r.drafted = e.SessionID, false
			r.mu.Unlock()
		}
		name = "session_state"
	case contracts.ErrorOccurred:
		name = "session_error"
	case contracts.CorrectionReady:
		name = "correction_ready"
	case contracts.CommitResultReady:
		name = "commit_result"
	default:
		return
	}
	outcome := string(e.Outcome)
	if e.Kind == contracts.CorrectionReady {
		switch e.Rewrite {
		case "", "accepted", "timeout", "unavailable", "empty", "truncated", "markup", "length", "new_number", "unsupported_content", "negation_dropped", "final_value_dropped", "content_dropped", "uncertainty_dropped", "context_copied":
			outcome = e.Rewrite
		default:
			outcome = "rejected"
		}
	}
	r.Record(Entry{Event: name, Session: e.SessionID, Generation: e.Generation, Sequence: e.Sequence,
		State: e.State, Outcome: outcome, Code: FailureCode(e.Error), DurationMs: e.AtMs})
}
