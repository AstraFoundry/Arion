// Package model projects recognition-bundle readiness for the UI.
package model

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

// Fraction is download progress between 0 and 1, or -1 when unknown.
func Fraction(m runtime.Model) float64 {
	if m.Total <= 0 {
		return -1
	}
	return min(float64(m.Done)/float64(m.Total), 1)
}

// PhaseLabel names what provisioning is doing now.
func PhaseLabel(m runtime.Model) copy.Key {
	switch m.State {
	case runtime.ModelReady:
		return copy.ModelVerified
	case runtime.ModelLoading:
		return copy.ModelLoading
	case runtime.ModelProvisioning:
		switch m.Phase {
		case runtime.PhaseVerifying:
			return copy.ModelVerifying
		case runtime.PhaseExtracting:
			return copy.ModelUnpacking
		}
		return copy.ModelDownloading
	}
	return ""
}

// Busy reports work in progress.
func Busy(m runtime.Model) bool {
	return m.State == runtime.ModelProvisioning || m.State == runtime.ModelLoading
}

// Recommended returns the automatic choice among evaluated bundles.
func Recommended(m runtime.Model) (runtime.Bundle, bool) {
	for _, b := range m.Bundles {
		if b.Recommended {
			return b, true
		}
	}
	return runtime.Bundle{}, false
}
