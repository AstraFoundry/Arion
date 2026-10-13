// Package permissions shows and requests OS consent for one capability.
package permissions

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
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

// Status shows the consent as a colored label.
func Status(c *ui.Context, store runtime.Store, tokens theme.Tokens, kind Kind) {
	text, color, icon := copy.StatusUndetermined, tokens.Muted, icons.Info
	switch consent(store, kind) {
	case runtime.ConsentGranted:
		text, color, icon = copy.StatusGranted, tokens.Success, icons.CircleCheck
	case runtime.ConsentDenied, runtime.ConsentRestricted:
		text, color, icon = copy.StatusDenied, tokens.Danger, icons.TriangleAlert
	}
	ui.Row(c).Gap(5).AlignItems(ui.Center).TextColor(color).Children(func() {
		ui.Icon(c, icon).FontSize(13)
		ui.Text(c, text).FontSize(theme.Caption).FontWeight(500).SingleLine().ContentTransition(ui.ContentTransition{})
	})
}

// Request offers the next step: ask, or open System Settings once denied.
// Nothing is shown once consent is granted.
func Request(c *ui.Context, store runtime.Store, kind Kind, label string, primary bool) {
	state := consent(store, kind)
	if state == runtime.ConsentGranted {
		return
	}
	// macOS never prompts again after a denial: the switch is in System
	// Settings from then on.
	denied := state == runtime.ConsentDenied || state == runtime.ConsentRestricted
	if denied {
		label = copy.OpenSettings
	}
	var button ui.Element
	if primary {
		button = ui.PrimaryButton(c.Key(kind), label)
	} else {
		button = ui.Button(c.Key(kind), label)
	}
	if button.Clicked() {
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
