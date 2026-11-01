// Package onboarding holds the first-run step rules.
package onboarding

import "arion.local/desktop/ui/shared/runtime"

// Step identifies one onboarding page. Optional features live in Settings.
type Step int

const (
	Welcome Step = iota
	Model
	Microphone
	Accessibility
	Try
	Finished
)

// Steps keeps download preparation ahead of permissions and live practice.
var Steps = []Step{Welcome, Model, Microphone, Accessibility, Try, Finished}

// Flow is the user's position; Direction drives the page slide.
type Flow struct {
	Index     int
	Direction int
}

func (f Flow) Step() Step { return Steps[f.Index] }

// Ready permits background provisioning during setup, but never reports
// completion until dictation can run. Practice itself may be skipped.
func Ready(step Step, s *runtime.State) bool {
	switch step {
	case Model:
		return s.Model.State == runtime.ModelReady || s.Model.State == runtime.ModelProvisioning || s.Model.State == runtime.ModelLoading
	case Microphone:
		return s.Permissions.Microphone == runtime.ConsentGranted
	case Accessibility:
		return s.Permissions.Accessibility == runtime.ConsentGranted
	case Try, Finished:
		return s.Model.State == runtime.ModelReady && s.Hold.Active &&
			s.Permissions.Microphone == runtime.ConsentGranted && s.Permissions.Accessibility == runtime.ConsentGranted && !s.Dictation.Active()
	}
	return true
}

// Skipped omits permissions already granted in either direction.
func Skipped(step Step, s *runtime.State) bool {
	return (step == Microphone || step == Accessibility) && Ready(step, s)
}

// Progress uses stable milestones, independent of optional settings.
func Progress(f Flow, _ *runtime.State) (current, total int) {
	return f.Index, len(Steps) - 1
}

// Move advances through applicable steps without bypassing readiness.
func (f *Flow) Move(delta int, s *runtime.State) bool {
	if delta > 0 {
		if f.Index == len(Steps)-1 || !Ready(f.Step(), s) {
			return false
		}
		delta = 1
	} else {
		if f.Index == 0 {
			return false
		}
		delta = -1
	}
	f.Index += delta
	f.Direction = delta
	for Skipped(f.Step(), s) {
		f.Index += delta
	}
	return true
}
