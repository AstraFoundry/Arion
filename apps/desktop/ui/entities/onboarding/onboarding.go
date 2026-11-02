// Package onboarding holds the first-run step rules.
package onboarding

import "arion.local/desktop/ui/shared/runtime"

// Step identifies one onboarding page. Account lives on Welcome and cloud
// setup lives inside Polish. Dictate and Translate each have a shortcut and
// a practice field. Keyboard setup stays in Settings.
type Step int

const (
	Welcome Step = iota
	Model
	// Polish chooses whether polishing is off, on this device or in the cloud.
	Polish
	// Permissions asks for the microphone and Accessibility together.
	Permissions
	// Dictate sets the polish shortcut and lets the user try it.
	Dictate
	// Translate sets the translation switch, shortcut and a practice field,
	// and completes setup.
	Translate
)

// Steps keeps recognition download ahead of permissions and the two practices.
var Steps = []Step{Welcome, Model, Polish, Permissions, Dictate, Translate}

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
	case Welcome:
		return !s.Account.Busy()
	case Model:
		return s.Model.State == runtime.ModelReady || s.Model.State == runtime.ModelProvisioning || s.Model.State == runtime.ModelLoading
	case Polish:
		if !runtime.PolishOn(s) || s.Preferences.PolishEngine == runtime.PolishCloud {
			return true
		}
		return s.Polish.State == runtime.PolishProvisioning || s.Polish.State == runtime.PolishLoading || s.Polish.State == runtime.PolishReady
	case Permissions:
		return s.Permissions.Microphone == runtime.ConsentGranted && s.Permissions.Accessibility == runtime.ConsentGranted
	case Dictate, Translate:
		return s.Model.State == runtime.ModelReady && s.Hold.Active &&
			s.Permissions.Microphone == runtime.ConsentGranted && s.Permissions.Accessibility == runtime.ConsentGranted && !s.Dictation.Active()
	}
	return true
}

// Skipped omits translation when the platform has no hold-key adapter.
// Granted permissions stay on their page so the user can see the result.
func Skipped(step Step, s *runtime.State) bool {
	return step == Translate && s.Preferences.HoldKey == ""
}

// Progress counts the steps the user will actually see.
func Progress(f Flow, s *runtime.State) (current, total int) {
	for i, step := range Steps {
		if Skipped(step, s) {
			continue
		}
		if i < f.Index {
			current++
		}
		total++
	}
	if total > 0 {
		total--
	}
	return current, total
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
		next := f.Index + delta
		if next < 0 || next >= len(Steps) {
			break
		}
		f.Index = next
	}
	return true
}
