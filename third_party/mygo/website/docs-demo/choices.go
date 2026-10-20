package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func init() {
	sync, hidden, reloads := true, false, 0
	mail, calendar, messages := true, false, true
	demo("checkbox", func(c *ui.Context) {
		ui.Column(c).Gap(8).Children(func() {
			ui.Checkbox(c, &sync, "Sync across devices")
			if ui.Checkbox(c, &hidden, "Show hidden files").Changed() {
				reloads++
			}
			ui.CheckboxGroup(c, "Notifications", func() {
				ui.Checkbox(c, &mail, "Mail")
				ui.Checkbox(c, &calendar, "Calendar")
				ui.Checkbox(c, &messages, "Messages")
			})
		})
		status(c, fmt.Sprintf("Reloaded %d times", reloads))
	})

	wifi, bluetooth := true, false
	demo("switch", func(c *ui.Context) {
		ui.Column(c).Width(240).Gap(10).Children(func() {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, "Wi-Fi").Grow(1)
				ui.Switch(c, &wifi).Label("Wi-Fi")
			})
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, "Bluetooth").Grow(1)
				ui.Switch(c, &bluetooth).Label("Bluetooth")
			})
		})
		status(c, fmt.Sprintf("Wi-Fi %s, Bluetooth %s", choicesOnOff(wifi), choicesOnOff(bluetooth)))
	})

	size, align := "Medium", ui.Center
	demo("radio", func(c *ui.Context) {
		ui.Row(c).Gap(48).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				ui.Text(c, "Size").FontSize(12).TextColor(c.Theme().TextMuted)
				ui.RadioGroup(c, func() {
					for _, s := range []string{"Small", "Medium", "Large"} {
						ui.Radio(c, &size, s, s)
					}
				}).Label("Size")
			})
			ui.Column(c).Gap(8).Children(func() {
				ui.Text(c, "Alignment").FontSize(12).TextColor(c.Theme().TextMuted)
				ui.RadioGroup(c, func() {
					ui.Radio(c, &align, ui.Start, "Left")
					ui.Radio(c, &align, ui.Center, "Center")
					ui.Radio(c, &align, ui.End, "Right")
				}).Row().Gap(16).Label("Alignment")
			})
		})
		names := map[ui.Align]string{ui.Start: "left", ui.Center: "center", ui.End: "right"}
		status(c, fmt.Sprintf("%s, aligned %s", size, names[align]))
	})

	plan := "Pro"
	prices := map[string]string{"Free": "$0", "Pro": "$12", "Team": "$30"}
	demo("select", func(c *ui.Context) {
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, "Plan")
			ui.Select(c, &plan, []string{"Free", "Pro", "Team"}).Label("Plan")
		})
		status(c, fmt.Sprintf("%s: %s a month", plan, prices[plan]))
	})

	volume, quality := 35.0, 75.0
	demo("slider", func(c *ui.Context) {
		ui.Slider(c, &volume, 0, 100).Label("Volume").Width(280)
		status(c, fmt.Sprintf("Volume: %.0f", volume))
		ui.StepSlider(c, &quality, 0, 100, 25).Label("Quality").Width(280)
		status(c, fmt.Sprintf("Quality: %.0f%%", quality))
	})

	minPrice, maxPrice := 200.0, 800.0
	demo("range-slider", func(c *ui.Context) {
		ui.RangeSlider(c, &minPrice, &maxPrice, 0, 1000, 10).Label("Price").Width(280)
		ui.Textf(c, "$%.0f – $%.0f", minPrice, maxPrice)
	})

	copies := 2.0
	demo("stepper", func(c *ui.Context) {
		ui.Row(c).Gap(6).Children(func() {
			word := "copies"
			if copies == 1 {
				word = "copy"
			}
			ui.Textf(c, "%.0f %s", copies, word)
			ui.Stepper(c, &copies, 1, 99, 1).Label("Copies")
		})
	})

	stars := 3
	demo("rating", func(c *ui.Context) {
		ui.Rating(c, &stars, 5).Label("Rating")
		if stars == 0 {
			status(c, "Not rated: click a star")
		} else {
			status(c, fmt.Sprintf("Rated %d of 5", stars))
		}
	})
}

// choicesOnOff names a switch's state.
func choicesOnOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}
