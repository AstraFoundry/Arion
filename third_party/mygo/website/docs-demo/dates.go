package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// datesHex is a color in hex, as the color picker shows it.
func datesHex(c ui.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func init() {
	due := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	demo("calendar", func(c *ui.Context) {
		ui.Calendar(c, &due).Label("Due date")
		status(c, "Due "+due.Format("Monday, January 2, 2006"))
	})

	birthday := time.Date(1990, 4, 12, 0, 0, 0, 0, time.UTC)
	meeting := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	demo("date-input", func(c *ui.Context) {
		// The fields go at the top, leaving room for the calendar below them.
		ui.Column(c).Grow(1).Gap(12).Children(func() {
			ui.Row(c).Gap(24).Children(func() {
				ui.Field(c, "Birthday", func() {
					ui.DateInput(c, &birthday)
				})
				ui.Field(c, "Meeting", func() {
					ui.Row(c).Gap(8).Children(func() {
						ui.DateInput(c, &meeting).Label("Date")
						ui.TimeInput(c, &meeting).Label("Time")
					})
				})
			})
			status(c, "Born on "+birthday.Format("January 2, 2006")+", meeting on "+meeting.Format("Mon, Jan 2 at 15:04"))
		})
	})

	alarm := time.Date(2026, 10, 15, 7, 30, 0, 0, time.UTC)
	demo("time-input", func(c *ui.Context) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconBell).TextColor(c.Theme().TextMuted)
			ui.TimeInput(c, &alarm).Label("Alarm")
		})
		status(c, "The alarm rings at "+alarm.Format("3:04 PM"))
	})

	tint, background := ui.Hex("#E11D48"), ui.Hex("#FDF2F8")
	demo("color-picker", func(c *ui.Context) {
		// The well goes at the top, leaving room for its picker below it.
		ui.Row(c).Grow(1).Gap(24).AlignItems(ui.Start).Children(func() {
			ui.ColorPicker(c, &background).Width(240).Label("Background")
			ui.Column(c).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Field(c, "Tint", func() {
					ui.ColorWell(c, &tint)
				})
				ui.Column(c).Width(180).Padding(16).Gap(4).Radius(10).Background(background).
					BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
					ui.Text(c, "Hello").FontSize(20).Bold().TextColor(tint)
					ui.Text(c, "Tint on background").FontSize(12).TextColor(tint)
				})
				status(c, datesHex(tint)+" on "+datesHex(background))
			})
		})
	})
}
