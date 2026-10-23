package diagnostics

import (
	"encoding/hex"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
)

// InputSourceDiagnostic contains only Arion-owned identity and native flags.
type InputSourceDiagnostic struct {
	ID              string `json:"id"`
	Found           bool   `json:"found"`
	Enabled         bool   `json:"enabled"`
	Selected        bool   `json:"selected"`
	Selectable      bool   `json:"selectable"`
	IdentityMatches bool   `json:"identityMatches"`
}

// InputMethodDiagnostic deliberately excludes foreign source IDs and paths.
type InputMethodDiagnostic struct {
	MaintenancePID   int                   `json:"maintenancePid,omitempty"`
	MaintenanceState string                `json:"maintenanceState,omitempty"`
	Operation        string                `json:"operation"`
	Step             string                `json:"step"`
	RequestedID      string                `json:"requestedId,omitempty"`
	ProcessID        int                   `json:"processId"`
	Installed        bool                  `json:"installed"`
	Mode             InputSourceDiagnostic `json:"mode"`
	Parent           InputSourceDiagnostic `json:"parent"`
	Current          string                `json:"current"`
	OSStatus         *int32                `json:"osStatus,omitempty"`
	HostBuild        string                `json:"hostBuild,omitempty"`
	BuildCode        domain.ErrorCode      `json:"buildCode,omitempty"`
	BundleCode       domain.ErrorCode      `json:"bundleCode,omitempty"`
}

func (d InputMethodDiagnostic) valid() bool {
	switch d.Operation {
	case "query", "register", "enable", "select", "switch_away", "install", "remove", "host_launch", "host_stop", "host_ready", "disable", "verify_removal":
	default:
		return false
	}
	if d.MaintenancePID < 0 {
		return false
	}
	switch d.MaintenanceState {
	case "", "unknown", "available", "selected", "unavailable":
	default:
		return false
	}
	switch d.Step {
	case "before", "after", "observed":
	default:
		return false
	}
	if d.RequestedID != "" && d.RequestedID != imebridge.SourceID && d.RequestedID != imebridge.BundleID {
		return false
	}
	if d.ProcessID <= 0 || d.Mode.ID != imebridge.SourceID || d.Parent.ID != imebridge.BundleID {
		return false
	}
	switch d.Current {
	case "unavailable", "arion_mode", "arion_parent", "other":
	default:
		return false
	}
	if d.HostBuild != "" {
		decoded, err := hex.DecodeString(d.HostBuild)
		if err != nil || len(decoded) != 32 {
			return false
		}
	}
	return validCode(d.BuildCode) && validCode(d.BundleCode)
}
