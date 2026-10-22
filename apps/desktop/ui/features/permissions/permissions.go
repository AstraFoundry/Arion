// Package permissions shows and requests OS consent for one capability,
// and guides the user through System Settings lists that Arion cannot
// prompt for.
package permissions

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Kind is a capability that needs consent.
type Kind int

const (
	Microphone Kind = iota
	Accessibility
)

func consent(store runtime.Store, kind Kind) runtime.Consent {
	if kind == Microphone {
		return store.State.Permissions.Microphone
	}
	return store.State.Permissions.Accessibility
}

// Granted reports consent for a capability.
func Granted(store runtime.Store, kind Kind) bool {
	return consent(store, kind) == runtime.ConsentGranted
}

// Missing reports a capability still without consent.
func Missing(store runtime.Store) bool {
	return !Granted(store, Microphone) || !Granted(store, Accessibility)
}

// Status shows the consent as a colored label.
func Status(c *ui.Context, store runtime.Store, tokens theme.Tokens, kind Kind) {
	messages := store.Copy()
	text, color, icon := messages.Text(copy.StatusUndetermined), tokens.Muted, icons.Info
	switch consent(store, kind) {
	case runtime.ConsentGranted:
		text, color, icon = messages.Text(copy.StatusGranted), tokens.Success, icons.CircleCheck
	case runtime.ConsentDenied, runtime.ConsentRestricted:
		text, color, icon = messages.Text(copy.StatusDenied), tokens.Danger, icons.TriangleAlert
	}
	kit.Status(c, tokens, icon, color, text)
}

// Request offers the next step: ask, or open System Settings once denied.
// Nothing is shown once consent is granted.
func Request(c *ui.Context, store runtime.Store, tokens theme.Tokens, kind Kind, label string, primary bool) {
	messages := store.Copy()
	state := consent(store, kind)
	if state == runtime.ConsentGranted {
		return
	}
	// macOS never prompts again after a denial: the switch is in System
	// Settings from then on.
	denied := state == runtime.ConsentDenied || state == runtime.ConsentRestricted
	if denied {
		label = messages.Text(copy.OpenSettings)
	}
	variant := kit.Outline
	if primary {
		variant = kit.Primary
	}
	if kit.Button(c.Key(kind), tokens, variant, label, true).Clicked() {
		switch {
		case kind == Microphone && denied:
			store.Actions.OpenMicrophoneSettings()
		case kind == Microphone:
			store.Actions.RequestMicrophone()
		case denied:
			store.Actions.OpenAccessibilitySettings()
		default:
			store.Actions.RequestAccessibility()
		}
	}
}
