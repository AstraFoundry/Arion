// Package onboarding holds the first-run step rules.
package onboarding

import "arion.local/desktop/ui/shared/runtime"

// Step identifies one onboarding page.
type Step int

const (
	Welcome Step = iota
	// Account offers signing in or creating an AstraFoundry account, through
	// Google, GitHub or email in the browser, or continuing locally, the
	// default (SYNC-10). Dictation never waits on it.
	Account
	Microphone
	Accessibility
	InputMethod
	Model
	// Polish chooses whether polishing is off, on this device or in the
	// cloud.
	Polish
	// Cloud connects the cloud model; only shown when the cloud is chosen.
	Cloud
	// HoldKey confirms the dictation key listens, which on macOS needs
	// Input Monitoring, and lets the user try it.
	HoldKey
	// Translate introduces voice translation and chooses its target
	// language.
	Translate
	// TranslateKey chooses the translation key, which never collides with
	// the dictation key, and lets the user try it.
	TranslateKey
	Try
	Finished
)

// Steps is the page order.
var Steps = []Step{Welcome, Account, Microphone, Accessibility, InputMethod, Model, Polish, Cloud, HoldKey, Translate, TranslateKey, Try, Finished}

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
	case InputMethod:
		return !s.InputMethodInstalling && !s.InputMethodActivating && s.InputMethodError == "" && (s.InputMethod == runtime.InputMethodSelected || s.InputMethod == runtime.InputMethodUnsupported)
	case Model:
		// Setup continues in the background; the user may move on once the
		// download started, and the try step waits for readiness.
		return s.Model.State == runtime.ModelReady || s.Model.State == runtime.ModelProvisioning || s.Model.State == runtime.ModelLoading
	case HoldKey:
		return s.Hold.Active
	case Polish:
		if !runtime.PolishOn(s) || s.Preferences.PolishEngine == runtime.PolishCloud {
			return true
		}
		return s.Polish.State == runtime.PolishProvisioning || s.Polish.State == runtime.PolishLoading || s.Polish.State == runtime.PolishReady
	case Account:
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

// Skipped reports a step that does not apply to this setup.
func Skipped(step Step, s *runtime.State) bool {
	switch step {
	case Account:
		return s.Account.Disabled
	case InputMethod:
		return s.InputMethod == runtime.InputMethodUnsupported
	case Cloud:
		return !s.Preferences.Polish || s.Preferences.PolishEngine != runtime.PolishCloud
	case Translate, TranslateKey:
		// Translation needs the polishing model, and a platform without a
		// hold key adapter has no translation key.
		return !runtime.PolishOn(s) || s.Preferences.HoldKey == ""
	}
	return false
}

// Progress counts the steps between welcome and finish that apply, for the
// indicator.
func Progress(f Flow, s *runtime.State) (current, total int) {
	for i, step := range Steps {
		if i == 0 || Skipped(step, s) {
			continue
		}
		total++
		if i <= f.Index {
			current++
		}
	}
	return current, total
}

// Move advances through supported setup steps in either direction.
func (f *Flow) Move(delta int, s *runtime.State) bool {
	if f.Step() == InputMethod && s.InputMethodInstalling {
		return false
	}
	moved := false
	if delta > 0 {
		moved = f.Next(s)
	} else if f.Index > 0 {
		f.Back()
		moved = true
	}
	for moved && Skipped(f.Step(), s) {
		if delta > 0 {
			moved = f.Next(s)
		} else {
			f.Back()
		}
	}
	return moved
}

// SkipInputMethod leaves optional keyboard setup for Settings without changing
// the user's dictation mode. An in-flight installation must finish first.
func (f *Flow) SkipInputMethod(s *runtime.State) bool {
	if f.Step() != InputMethod || s.InputMethodInstalling {
		return false
	}
	f.Index++
	f.Direction = 1
	return true
}
