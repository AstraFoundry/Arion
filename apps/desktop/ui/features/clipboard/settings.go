package clipboard

import (
	"slices"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Settings reveals collection options using the same disclosure as dictation.
// The page supplies the shared shortcut feature through this composition slot.
func Settings(c *ui.Context, store runtime.Store, tokens theme.Tokens, shortcut func()) {
	messages := store.Copy()
	prefs := store.State.Preferences.Clipboard
	kit.Card(c, tokens).Children(func() {
		kit.TipRow(c, tokens, messages.Text(copy.ClipboardEnabled), messages.Text(copy.ClipboardPrivacy), func() {
			on := prefs.Enabled
			if ui.Switch(c.Key("enabled"), &on).Label(messages.Text(copy.ClipboardEnabled)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Clipboard.Enabled = on })
			}
		})
		kit.Disclosure(c.Key("clipboard-details"), prefs.Enabled, func() {
			kit.Divider(c, tokens)
			shortcut()
			kit.Divider(c, tokens)
			count, days := prefs.Limits.Count, prefs.Limits.Days
			kit.Row(c, tokens, messages.Text(copy.ClipboardCount), "", func() {
				if kit.Select(c.Key("count"), &count, retentionOptions(count, 100, 200, 500, 1000), messages.ClipboardEntries, nil).
					Label(messages.Text(copy.ClipboardCount)).Disabled(store.State.Clipboard.Busy).Changed() {
					limits := prefs.Limits
					limits.Count = count
					store.Actions.SetClipboardLimits(limits)
				}
			})
			kit.Divider(c, tokens)
			kit.TipRow(c, tokens, messages.Text(copy.ClipboardDays), messages.Text(copy.ClipboardLimitsHint), func() {
				if kit.Select(c.Key("days"), &days, retentionOptions(days, 1, 7, 14, 30, 90), messages.ClipboardDays, nil).
					Label(messages.Text(copy.ClipboardDays)).Disabled(store.State.Clipboard.Busy).Changed() {
					limits := prefs.Limits
					limits.Days = days
					store.Actions.SetClipboardLimits(limits)
				}
			})
		})
	})
	Status(c, store, tokens)
}

func retentionOptions(current int, options ...int) []int {
	if !slices.Contains(options, current) {
		options = append(options, current)
		slices.Sort(options)
	}
	return options
}
