// Package holdkey chooses the hold-to-talk key and explains why a chosen
// key is not listening.
package holdkey

import (
	platform "runtime"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

var names = map[runtime.HoldKey]copy.Key{
	runtime.HoldOff:          copy.HoldKeyOff,
	runtime.HoldFn:           copy.HoldKeyFn,
	runtime.HoldRightOption:  copy.HoldKeyRightOption,
	runtime.HoldRightCommand: copy.HoldKeyRightCommand,
	runtime.HoldRightAlt:     copy.HoldKeyRightAlt,
	runtime.HoldRightCtrl:    copy.HoldKeyRightCtrl,
}

// Name is the key's display name.
func Name(messages copy.Catalog, key runtime.HoldKey) string { return messages.Text(names[key]) }

// Choices are Off followed by this platform's keys.
func Choices() []runtime.HoldKey {
	return append([]runtime.HoldKey{runtime.HoldOff}, runtime.HoldKeysFor(platform.GOOS)...)
}

// Picker selects the hold key.
func Picker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	choices := Choices()
	labels := make([]string, len(choices))
	selected := Name(messages, store.State.Preferences.HoldKey)
	for i, key := range choices {
		labels[i] = Name(messages, key)
	}
	if ui.Select(c.Key("hold"), &selected, labels).Changed() {
		for i, label := range labels {
			if label == selected {
				store.Actions.SetHoldKey(choices[i])
			}
		}
	}
}

// Notes explain what the chosen key still needs, under the picker's row.
func Notes(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	key := store.State.Preferences.HoldKey
	if key == runtime.HoldOff {
		return
	}
	shortcut := store.State.Shortcut
	ui.Column(c).Padding(0, 16, 12, 16).Gap(8).Children(func() {
		switch {
		case shortcut.HoldFailure == runtime.CodePermissionDenied:
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, messages.Text(copy.HoldKeyNeedsMonitoring)).FontSize(theme.Caption).TextColor(tokens.Danger).Grow(1)
				if kit.Button(c.Key("monitoring"), tokens, kit.Outline, messages.Text(copy.Allow), true).Clicked() {
					store.Actions.RequestInputMonitoring()
				}
			})
		case !shortcut.HoldActive:
			ui.Text(c, messages.Text(copy.HoldKeyUnavailable)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
		if key == runtime.HoldFn {
			ui.Text(c, messages.Text(copy.HoldKeyFnHint)).FontSize(theme.Caption).TextColor(tokens.Muted)
		}
	})
}
