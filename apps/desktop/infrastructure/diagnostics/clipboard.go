package diagnostics

// ClipboardTargetDiagnostic contains only fixed query outcomes and flags.
// It excludes target identities, roles, titles, paths and clipboard/editor data.
type ClipboardTargetDiagnostic struct {
	Focus        string `json:"focus"`
	Window       string `json:"window"`
	Verification string `json:"verification"`
	Trusted      bool   `json:"trusted"`
	Secure       bool   `json:"secure"`
	Geometry     bool   `json:"geometry"`
	// Older development logs contain these fields. Keep export compatibility
	// without querying or emitting a second focus scope in new captures.
	SystemFocus        string `json:"systemFocus,omitempty"`
	SystemOwnerMatches bool   `json:"systemOwnerMatches,omitempty"`
}

func (d ClipboardTargetDiagnostic) valid() bool {
	queries := []string{d.Focus, d.Window}
	if d.SystemFocus != "" {
		queries = append(queries, d.SystemFocus)
	}
	for _, query := range queries {
		switch query {
		case "not_queried", "available", "empty", "absent", "unsupported", "cannot_complete", "api_disabled", "failed":
		default:
			return false
		}
	}
	switch d.Verification {
	case "ready", "not_checked", "no_destination", "permission_denied", "secure_input", "owner_changed", "missing_focus", "window_unavailable", "window_changed", "focus_unavailable", "focus_changed", "verification_failed":
		return true
	default:
		return false
	}
}
