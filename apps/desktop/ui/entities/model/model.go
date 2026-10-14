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

// FailureKey explains a provisioning or loading failure code.
func FailureKey(code runtime.ErrorCode) copy.Key {
	switch code {
	case runtime.CodeArtifactUnverified:
		return copy.ModelFailChecksum
	case runtime.CodeProvisioningFailed:
		return copy.ModelFailNetwork
	}
	return copy.FailModelTitle
}

// PolishFraction is rewrite-model download progress, or -1 when unknown.
func PolishFraction(p runtime.Polish) float64 {
	if p.Total <= 0 || p.Phase != runtime.PhaseDownloading {
		return -1
	}
	return min(float64(p.Done)/float64(p.Total), 1)
}

// PolishPhaseLabel names what rewrite-model provisioning is doing now.
func PolishPhaseLabel(p runtime.Polish) copy.Key {
	switch p.State {
	case runtime.PolishReady:
		return copy.PolishReady
	case runtime.PolishLoading:
		return copy.PolishLoading
	case runtime.PolishProvisioning:
		if p.Phase == runtime.PhaseVerifying {
			return copy.ModelVerifying
		}
		return copy.ModelDownloading
	}
	return ""
}

// names gives catalog bundles plain names; catalog identifiers and engine
// names never reach the interface.
var names = map[string]copy.Key{
	"xasr-160-xasr-final":       copy.ModelNameAccurate,
	"xasr-480-xasr-final":       copy.ModelNameAccurateSteady,
	"xasr-160":                  copy.ModelNameFast,
	"xasr-480":                  copy.ModelNameSteady,
	"zipformer-2023":            copy.ModelNameLight,
	"xasr-160-sensevoice-final": copy.ModelNameAlternate,
	"xasr-160-qwen3asr-final":   copy.ModelNameDeep,
}

// NameKey is a bundle's user-facing name.
func NameKey(id string) copy.Key {
	if key, ok := names[id]; ok {
		return key
	}
	return copy.ModelNameGeneric
}
