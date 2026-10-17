// Package onboarding holds the first-run step rules.
package onboarding

import "arion.local/desktop/ui/shared/runtime"

// Step identifies one onboarding page.
type Step int

const (
	Welcome Step = iota
	Microphone
	Accessibility
	Model
	Shortcut
	Try
	// Sync offers continuing locally, the default, or signing in to set up
	// sync (SYNC-10). It follows the first dictation so an account never
	// stands between the user and dictating.
	Sync
	Finished
)

// Steps is the page order.
var Steps = []Step{Welcome, Microphone, Accessibility, Model, Shortcut, Try, Sync, Finished}

// Flow is the user's position; Direction drives the page slide.
type Flow struct {
	Index     int
	Direction int
}

func (f Flow) Step() Step { return Steps[f.Index] }

// Ready reports whether a step's requirement is met. Permissions and the
// model are real gates: dictation cannot work without them.
func Ready(step Step, s *runtime.State) bool {
	switch step {
	case Microphone:
		return s.Permissions.Microphone == runtime.ConsentGranted
	case Accessibility:
		return s.Permissions.Accessibility == runtime.ConsentGranted
	case Model:
		// Setup continues in the background; the user may move on once the
		// download started, and the try step waits for readiness.
		return s.Model.State == runtime.ModelReady || s.Model.State == runtime.ModelProvisioning || s.Model.State == runtime.ModelLoading
	case Shortcut:
		return s.Shortcut.Registered
	case Sync:
		// A sign-in in flight is cancelled or finished before moving on.
		return !s.Account.Busy()
	}
	return true
}

// Next advances when the current step is ready.
func (f *Flow) Next(s *runtime.State) bool {
	if f.Index >= len(Steps)-1 || !Ready(f.Step(), s) {
		return false
	}
	f.Index++
	f.Direction = 1
	return true
}

// Back returns to the previous step.
func (f *Flow) Back() {
	if f.Index > 0 {
		f.Index--
		f.Direction = -1
	}
}

// Progress counts the steps between welcome and finish, for the indicator.
func Progress(f Flow) (current, total int) {
	return f.Index, len(Steps) - 1
}

// Move advances through supported setup steps in either direction.
func (f *Flow) Move(delta int, s *runtime.State) bool {
	moved := false
	if delta > 0 {
		moved = f.Next(s)
	} else if f.Index > 0 {
		f.Back()
		moved = true
	}
	if moved && f.Step() == Sync && s.Account.Disabled {
		if delta > 0 {
			f.Next(s)
		} else {
			f.Back()
		}
	}
	return moved
}
