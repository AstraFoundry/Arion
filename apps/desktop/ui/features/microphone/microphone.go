// Package microphone chooses the input device and shows its live level.
package microphone

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Picker selects the capture device; the system default is listed first.
func Picker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	names := []string{messages.Text(copy.MicSystem)}
	ids := []string{""}
	selected := messages.Text(copy.MicSystem)
	for _, device := range store.State.Devices {
		names = append(names, device.Name)
		ids = append(ids, device.ID)
		if device.ID == store.State.Preferences.Microphone {
			selected = device.Name
		}
	}
	if ui.Select(c.Key("device"), &selected, names).Label(messages.Text(copy.MicDevice)).Width(240).Changed() {
		for i, name := range names {
			if name == selected {
				id := ids[i]
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Microphone = id })
			}
		}
	}
}

// Meter shows the monitored input level as bars.
func Meter(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	kit.LevelBars(c, store.State.MonitorLevels, 24, 4, 36, 3, tokens.Text)
}
