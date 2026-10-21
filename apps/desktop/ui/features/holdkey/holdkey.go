// Package holdkey chooses the dictation key, the only global activation,
// and explains why it is not listening.
package holdkey

import (
	platform "runtime"

	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Picker selects the hold key among this platform's keys.
func Picker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	choices := runtime.HoldKeysFor(platform.GOOS)
	labels := make([]string, len(choices))
	selected := holdname.Name(messages, store.State.Preferences.HoldKey)
	for i, key := range choices {
		labels[i] = holdname.Name(messages, key)
	}
	if ui.Select(c.Key("hold"), &selected, labels).Changed() {
		for i, label := range labels {
			if label == selected {
				store.Actions.SetHoldKey(choices[i])
			}
		}
	}
}

// HasNotes reports whether Notes has anything to say.
func HasNotes(store runtime.Store) bool {
	return !store.State.Hold.Active || store.State.Preferences.HoldKey == runtime.HoldFn
}

// Notes explain what the chosen key still needs.
func Notes(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	hold := store.State.Hold
	ui.Column(c).Gap(8).Children(func() {
		switch {
		case hold.Failure == runtime.CodePermissionDenied:
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, messages.Text(copy.HoldKeyNeedsMonitoring)).FontSize(theme.Caption).TextColor(tokens.Danger).Grow(1)
				if kit.Button(c.Key("monitoring"), tokens, kit.Outline, messages.Text(copy.Allow), true).Clicked() {
					store.Actions.RequestInputMonitoring()
				}
			})
		case !hold.Active:
			ui.Text(c, messages.Text(copy.HoldKeyUnavailable)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
		if store.State.Preferences.HoldKey == runtime.HoldFn {
			ui.Text(c, messages.Text(copy.HoldKeyFnHint)).FontSize(theme.Caption).TextColor(tokens.Muted)
		}
	})
}
