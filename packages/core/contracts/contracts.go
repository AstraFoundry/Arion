// Package contracts defines portable boundary data, not a ready IPC encoding.
package contracts

import "arion.local/core/domain"

const Version = 1

type Identity struct {
	SessionID  string `json:"session_id"`
	Generation uint64 `json:"generation"`
}

type EventKind string

const (
	SessionStateChanged EventKind = "session_state"
	AudioDiscontinuity  EventKind = "audio_discontinuity"
	DraftReplaced       EventKind = "draft_replaced"
	CorrectionReady     EventKind = "correction_ready"
	CommitResultReady   EventKind = "commit_result"
	ErrorOccurred       EventKind = "error"
	// RewriteStarted marks that the final rewrite is running on the
	// recognized text; it carries no text.
	RewriteStarted EventKind = "rewrite_started"
	// AudioLevel carries a content-free input level for meters.
	AudioLevel EventKind = "audio_level"
)

type Envelope struct {
	Version int `json:"version"`
	Identity
	Sequence uint64    `json:"sequence"`
	Kind     EventKind `json:"kind"`
	Revision uint64    `json:"revision,omitempty"`
}

// Frame samples are interleaved float32 PCM in the adapter-declared format.
type Frame struct {
	Identity
	Sequence      uint64
	SampleOffset  uint64
	SampleRate    uint32
	Channels      uint16
	Samples       []float32
	Discontinuity bool
}

type Draft struct {
	Identity
	Revision   uint64             `json:"revision"`
	Text       string             `json:"text"`
	Final      bool               `json:"final"`
	Candidates []domain.Candidate `json:"candidates"`
	// Segments are recognizer segments closed at an endpoint; they no longer
	// change. Tail is the open segment. Text joins both.
	Segments []string `json:"segments,omitempty"`
	Tail     string   `json:"tail,omitempty"`
}

type TargetToken string

// ContextSnapshot is the bounded field and window text of one captured target.
// Before and After are the field's text around the selection. Screen is
// visible text read from the captured window. All stay in session RAM.
type ContextSnapshot struct {
	Identity
	Target     TargetToken
	SafeToRead bool
	Before     string
	After      string
	Screen     string
	// ScreenCaptureMs measures context acquisition, including Accessibility.
	// It carries no text.
	ScreenCaptureMs float64
}

type CommitRequest struct {
	Identity
	Target         TargetToken
	SourceRevision uint64
	Text           string
}

type CommitOutcome string

const (
	Applied    CommitOutcome = "applied"
	NotApplied CommitOutcome = "not_applied"
	Unknown    CommitOutcome = "unknown"
	// Copied means the text was left on the system clipboard instead of
	// inserted. NoInput distinguishes positive absence from delivery fallbacks.
	// It is not an insertion and is not retried.
	Copied CommitOutcome = "copied"
)

type CommitResult struct {
	Identity
	// NoInput is true only when the adapter positively observed no editable input.
	NoInput bool
	Outcome CommitOutcome
	// Failure explains a not_applied outcome with a stable code.
	Failure domain.ErrorCode
}

// TargetSnapshot is captured when dictation starts. The token is opaque;
// native handles stay inside the OS adapter (DSK-05).
type TargetSnapshot struct {
	Identity
	Target TargetToken
	// Sensitive is true for secure fields and fields whose safety the
	// adapter cannot establish (DSK-08).
	Sensitive bool
	// Application is the platform identity of the focused app, used only for
	// local policy such as the context denylist. It is never logged.
	Application string
	// ApplicationName is the app's display name, shown only in Arion's own
	// interface. It is never logged.
	ApplicationName string
	// Field is the focused field's type, from its platform role.
	Field domain.FieldKind
	// Site is the host of the web page owning the focused field, read only
	// when the user lets the scene follow the app. It is never logged.
	Site string
}

// SceneHints are the snapshot's inputs to scene selection.
func (t TargetSnapshot) SceneHints() domain.SceneHints {
	return domain.SceneHints{App: t.Application, Field: t.Field, Site: t.Site}
}

// Event is the typed application event delivered to UI adapters. Text lives
// only in session RAM and is never logged (PRV-05).
type Event struct {
	Envelope
	State domain.SessionState `json:"state,omitempty"`
	Text  string              `json:"text,omitempty"`
	Raw   string              `json:"raw,omitempty"`
	// Source is the rule-corrected text a rewrite started from; Rewrite is
	// the rewrite outcome (a domain.RewriteVerdict or a RewriteOutcome) and
	// empty when no rewrite ran. Text is what review shows and commit inserts.
	Source  string `json:"source,omitempty"`
	Rewrite string `json:"rewrite,omitempty"`
	// Scene is the formatting scene the rewrite was asked for.
	Scene   string           `json:"scene,omitempty"`
	Level   float32          `json:"level,omitempty"`
	Outcome CommitOutcome    `json:"outcome,omitempty"`
	NoInput bool             `json:"no_input,omitempty"`
	Error   domain.ErrorCode `json:"error,omitempty"`
	// Detail is a content-free diagnostic category such as "microphone" or
	// "accessibility" for PERMISSION_DENIED.
	Detail string `json:"detail,omitempty"`
	// AtMs is the monotonic time since the session started.
	AtMs float64 `json:"at_ms"`
}
