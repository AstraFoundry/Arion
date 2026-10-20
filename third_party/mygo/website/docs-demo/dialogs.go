package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// Icons of the dialogs' demos.
var (
	dialogsShare    = icon(`<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="m8.6 13.5 6.8 4M15.4 6.5l-6.8 4"/>`)
	dialogsBookmark = icon(`<path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>`)
)

func init() {
	name, renaming, draft := "Notes", false, ""
	demo("dialog", func(c *ui.Context) {
		if ui.Button(c, "Rename…").Clicked() {
			draft, renaming = name, true
		}
		ui.Modal(c, &renaming, func() {
			ui.Text(c, "Rename").Bold()
			if ui.TextInput(c, &draft).AutoFocus().Submitted() {
				name, renaming = draft, false
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Cancel").Clicked() {
					renaming = false
				}
				if ui.PrimaryButton(c, "Rename").Clicked() {
					name, renaming = draft, false
				}
			})
		})
		status(c, "The folder is “"+name+"”")
	})

	asking, deleted, canceled := false, false, false
	demo("alert-dialog", func(c *ui.Context) {
		if deleted {
			if ui.Button(c, "Restore “Notes”").Clicked() {
				deleted = false
			}
			status(c, "Deleted “Notes”")
			return
		}
		if ui.Button(c, "Delete").Clicked() {
			asking, canceled = true, false
		}
		switch ui.AlertDialog(c, &asking, "Delete “Notes”?", "You can't undo this.", "Cancel", "Delete") {
		case 1:
			deleted = true
		case 0, ui.AlertDismissed:
			canceled = true
		}
		if canceled {
			status(c, "Canceled")
		}
	})

	// The popover starts open, so that the page shows its panel: the
	// dimensions of a layer, which the layer beside follows.
	open, width, height, radius := true, 160.0, 96.0, 12.0
	demo("popover", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Grow(1).AlignSelf(ui.Stretch).AlignItems(ui.Start).Gap(24).Children(func() {
			dims := ui.Button(c, "").Children(func() {
				ui.Icon(c, dialogsRuler)
				ui.Text(c, "Dimensions").SingleLine()
			})
			if dims.Clicked() {
				open = !open
			}
			ui.Popover(c, dims, &open, func() {
				ui.Column(c).Width(240).Gap(12).Padding(4).Children(func() {
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, "Dimensions").Bold()
						ui.Text(c, "Set the size of the layer.").FontSize(12).TextColor(t.TextMuted)
					})
					ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1), ui.FitContent()).GapX(10).GapY(10).Children(func() {
						field := func(label string, value *float64, lo, hi float64) {
							ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).AlignSelf(ui.Center)
							ui.Slider(c.Key(label), value, lo, hi).Step(2).Label(label).AlignSelf(ui.Center)
							ui.Text(c, fmt.Sprintf("%.0f", *value)).FontSize(12).Width(24).AlignSelf(ui.Center)
						}
						field("Width", &width, 40, 260)
						field("Height", &height, 30, 150)
						field("Radius", &radius, 0, 48)
					})
				})
			})
			ui.Column(c).Grow(1).AlignSelf(ui.Stretch).Center().Children(func() {
				ui.Column(c).Size(float32(width), float32(height)).Radius(float32(radius)).Center().
					Background(t.Accent).TextColor(t.AccentText).Children(func() {
					ui.Text(c, fmt.Sprintf("%.0f × %.0f", width, height)).FontSize(12)
				})
			})
		})
	})

	tipped := ""
	demo("tooltip", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "").Label("Share").Tooltip("Share with others").Children(func() {
				ui.Icon(c, dialogsShare)
			}).Clicked() {
				tipped = "Shared"
			}
			ui.Button(c, "").Label("Bookmark").Tooltip("Add a bookmark").Children(func() { ui.Icon(c, dialogsBookmark) })
			ui.Button(c, "").Label("Settings").Tooltip("Settings").Children(func() { ui.Icon(c, iconGear) })
			b := ui.Button(c, "").Label("Save").Description("Save the note")
			if b.Children(func() { ui.Icon(c, iconSave) }).Clicked() {
				tipped = "Saved"
			}
			ui.TooltipBase(c, b, func(tip ui.Element) {
				tip.AttachTo(b, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, 6)
				tip.Row().Gap(8).Padding(4, 8).Radius(6).Background(t.Accent).TextColor(t.AccentText)
				tip.Children(func() {
					ui.Text(c, "Save the note").FontSize(12)
					ui.Text(c, "⌘S").FontSize(12).Opacity(0.7)
				})
			})
		})
		if tipped == "" {
			status(c, "Rest the pointer on a button")
		} else {
			status(c, tipped)
		}
	})

	notes, uploading := 3, ""
	demo("toast", func(c *ui.Context) {
		dialogsTop(c, func() {
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, "Save").Clicked() {
					c.Toast("Saved")
				}
				if ui.Button(c, "Delete a note").Disabled(notes == 0).Clicked() {
					notes--
					c.ToastAction("Note deleted", "Undo", func() { notes++ })
				}
				if uploading == "" {
					if ui.Button(c, "Upload").Clicked() {
						uploading = c.AddToast(ui.Toast{Title: "Uploading", Description: "report.pdf", Timeout: -1})
					}
				} else if ui.PrimaryButton(c, "Finish upload").Clicked() {
					c.CloseToast(uploading)
					uploading = ""
					c.Toast("Uploaded report.pdf")
				}
			})
			status(c, fmt.Sprintf("%d notes", notes))
		})
	})
}

// dialogsTop builds a demo at the top of the frame, rather than in its
// middle, leaving the room below for what it opens: a popover, toasts.
// dialogsRuler is a ruler's icon, for the dimensions of a layer.
var dialogsRuler = icon(`<path d="M21.3 15.3 8.7 2.7a1 1 0 0 0-1.4 0L2.7 7.3a1 1 0 0 0 0 1.4l12.6 12.6a1 1 0 0 0 1.4 0l4.6-4.6a1 1 0 0 0 0-1.4z"/><path d="m7.5 10.5 2-2M10.5 13.5l2-2M13.5 16.5l2-2"/>`)

func dialogsTop(c *ui.Context, fn func()) {
	ui.Column(c).Grow(1).AlignItems(ui.Center).Gap(12).Children(fn)
}
