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

// Gain sets the input gain on a stepped slider, with the speaking distance
// each step suits written under its tick.
func Gain(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	stops := runtime.GainStops
	selected := store.State.Preferences.MicrophoneGain
	value := float64(selected)
	distance := 0
	for _, stop := range stops {
		if stop.DB == selected {
			distance = stop.DistanceCM
		}
	}
	lo, hi := float64(stops[0].DB), float64(stops[len(stops)-1].DB)
	ui.Column(c).Gap(2).Children(func() {
		slider := ui.StepSlider(c.Key("gain"), &value, lo, hi, runtime.GainStep).
			Label(messages.Text(copy.MicGain)).Value(messages.GainStop(selected, distance))
		if slider.Changed() {
			db := int(value)
			store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.MicrophoneGain = db })
		}
		// The slider's ticks sit half a knob in from each end; the labels
		// follow the same geometry so each centers under its tick.
		ui.Box(c).Height(theme.Caption + 6).Draw(func(p *ui.Painter, r ui.Rect) {
			knob := p.Theme().Space(5)
			for i, stop := range stops {
				span := ui.Span{Text: messages.Centimeters(stop.DistanceCM), Size: theme.Caption, Color: tokens.Muted}
				if stop.DB == selected {
					span.Color, span.Weight = tokens.Text, 600
				}
				w, _ := p.MeasureText(0, span)
				x := r.X + knob/2 + (r.W-knob)*float32(i)/float32(len(stops)-1)
				p.RichText(x-w/2, r.Y, 0, span)
			}
		})
	})
}

// GainValue shows the selected gain.
func GainValue(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	ui.Text(c, messages.Decibels(store.State.Preferences.MicrophoneGain)).FontSize(theme.Body).TextColor(tokens.Muted).
		SingleLine().ContentTransition(ui.ContentTransition{})
}

// Meter shows the monitored input level as bars.
func Meter(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	kit.LevelBars(c, store.State.MonitorLevels, 24, 4, 36, 3, tokens.Text)
}
