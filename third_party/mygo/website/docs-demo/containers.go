package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Icons of the containers' demos.
var (
	containersChevron = icon(`<path d="m9 18 6-6-6-6"/>`)
	containersTerm    = icon(`<path d="m4 17 6-6-6-6M12 19h8"/>`)
)

// containersTiles are the colors of the grid demo's tiles.
var containersTiles = []string{"#f97316", "#eab308", "#22c55e", "#14b8a6", "#3b82f6", "#8b5cf6", "#ec4899", "#ef4444"}

func init() {
	containersScroll()
	containersGrid()
	containersSplit()
	containersCollapsible()
	containersAccordion()
	containersForm()
}

// containersScroll is a log that follows its end, unless the user scrolled
// up from it.
func containersScroll() {
	lines := []string{}
	for i := range 12 {
		lines = append(lines, fmt.Sprintf("Built package %d of 12", i+1))
	}
	var log ui.ScrollState
	demo("scroll", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Add a line").Clicked() {
				lines = append(lines, fmt.Sprintf("Line %d", len(lines)+1))
			}
			if ui.Button(c, "Top").Clicked() {
				log.Y = 0
			}
			if ui.Button(c, "End").Clicked() {
				log.Y = math.MaxFloat32
			}
		})
		following := log.Y >= log.MaxY
		if following {
			log.Y = math.MaxFloat32
		}
		ui.Scroll(c).TrackScroll(&log).AlignSelf(ui.Stretch).Height(150).Padding(10, 12).Gap(4).
			Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
			for i, line := range lines {
				ui.Text(c.Key(i), line).Font("monospace").FontSize(12)
			}
		})
		if following {
			status(c, "At the end: new lines scroll into view")
		} else {
			status(c, fmt.Sprintf("Scrolled to %.0f of %.0f DIPs: new lines wait below", log.Y, log.MaxY))
		}
	})
}

// containersGrid lays out tiles in equal columns, and labels beside
// controls in a column as wide as its content.
func containersGrid() {
	tiles, columns := 6, 3
	email, bio := "ada@example.com", ""
	demo("grid", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(32).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Width(260).Gap(10).Children(func() {
				ui.Grid(c).Columns(columns).Gap(8).Children(func() {
					for i := range tiles {
						ui.Column(c.Key(i)).Height(36).Radius(8).Center().
							Background(ui.Hex(containersTiles[i%len(containersTiles)])).Children(func() {
							ui.Text(c, fmt.Sprint(i+1)).Bold().TextColor(ui.RGB(255, 255, 255))
						})
					}
				})
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "Add").Disabled(tiles >= 8).Clicked() {
						tiles++
					}
					if ui.Button(c, "Remove").Disabled(tiles <= 1).Clicked() {
						tiles--
					}
					if ui.Button(c, fmt.Sprintf("Columns: %d", columns)).Clicked() {
						columns = 2 + (columns-1)%3
					}
				})
			})
			ui.Grid(c).Width(300).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(16).GapY(8).Children(func() {
				ui.Text(c, "Profile").Bold().ColumnSpan(-1)
				ui.Text(c, "Email").TextColor(t.TextMuted).Padding(7, 0, 0)
				ui.TextInput(c, &email)
				ui.Text(c, "Bio").TextColor(t.TextMuted).Padding(7, 0, 0)
				ui.TextArea(c, &bio).Placeholder("A line or two").Lines(2, 4)
			})
		})
	})
}

// containersSplit is a sidebar beside an editor above a console.
func containersSplit() {
	sidebar, editor := float32(180), float32(120)
	files := []string{"main.go", "app.go", "view.go", "go.mod"}
	open := 0
	demo("split", func(c *ui.Context) {
		t := c.Theme()
		ui.Split(c, &sidebar, func() {
			ui.Column(c).Fill().Padding(8).Gap(2).Background(t.Surface).Children(func() {
				for i, name := range files {
					row := ui.Row(c.Key(name)).Gap(6).Padding(5, 8).Radius(6).Children(func() {
						ui.Icon(c, iconFile)
						ui.Text(c, name).SingleLine()
					})
					if i == open {
						row.Background(t.Selection)
					}
					if row.Clicked() {
						open = i
					}
				}
			})
		}, func() {
			ui.SplitVertical(c, &editor, func() {
				ui.Column(c).Fill().Padding(12).Gap(6).Children(func() {
					ui.Text(c, files[open]).Bold()
					ui.Text(c, "package main\n\nfunc main() {}").Font("monospace").FontSize(12).TextColor(t.TextMuted)
				})
			}, func() {
				ui.Row(c).Fill().AlignItems(ui.Start).Padding(10, 12).Gap(6).Background(t.Surface).Children(func() {
					ui.Icon(c, containersTerm).TextColor(t.TextMuted)
					ui.Text(c, "go run .").Font("monospace").FontSize(12)
				})
			}).Fill()
		}).AlignSelf(ui.Stretch).Height(200).Border(1, t.Border).Radius(8).Clip()
		status(c, fmt.Sprintf("Sidebar %.0f DIPs wide, editor %.0f DIPs high", sidebar, editor))
	})
}

// containersCollapsible shows a disclosure, and one of its own look
// without a look of its own.
func containersCollapsible() {
	advanced, verbose, experimental := true, false, true
	details := false
	demo("collapsible", func(c *ui.Context) {
		t := c.Theme()
		ui.Column(c).Width(320).Gap(10).Children(func() {
			ui.Collapsible(c, "Advanced", &advanced, func() {
				ui.Checkbox(c, &verbose, "Verbose logging")
				ui.Checkbox(c, &experimental, "Experimental features")
			})
			p := ui.CollapsibleBase(c, &details)
			p.Trigger.Gap(6).Children(func() {
				ui.Icon(c, containersChevron).Rotate(90 * p.Progress())
				ui.Text(c, "Details")
			})
			p.Panel(func() {
				ui.Text(c, "CollapsibleBase grows its panel with Progress, for a look of your own.").
					TextColor(t.TextMuted).Padding(4, 0, 0, 22)
			})
		})
	})
}

// containersAccordion keeps one section open at a time.
func containersAccordion() {
	section := 0
	name, sync, analytics, crashes, beta := "Ada's Mac", true, false, true, false
	sections := []struct {
		title string
		build func(c *ui.Context)
	}{
		{"General", func(c *ui.Context) {
			ui.Field(c, "Device name", func() { ui.TextInput(c, &name) })
			ui.Checkbox(c, &sync, "Sync settings")
		}},
		{"Privacy", func(c *ui.Context) {
			ui.Checkbox(c, &analytics, "Share analytics")
			ui.Checkbox(c, &crashes, "Send crash reports")
		}},
		{"Advanced", func(c *ui.Context) {
			ui.Checkbox(c, &beta, "Join the beta channel")
		}},
	}
	demo("accordion", func(c *ui.Context) {
		ui.Accordion(c, func() {
			for i, s := range sections {
				open := section == i
				if ui.AccordionItem(c.Key(s.title), s.title, &open, func() { s.build(c) }).Changed() {
					section = -1
					if open {
						section = i
					}
				}
			}
		}).Width(380)
		if section < 0 {
			status(c, "Every section closed")
		} else {
			status(c, sections[section].title+" open: opening another closes it")
		}
	})
}

// containersForm lines labels up beside their controls, with a fieldset.
func containersForm() {
	name, email, plan := "Ada Lovelace", "ada", "Pro"
	plans := []string{"Free", "Pro", "Team"}
	mail, sound := true, false
	demo("form", func(c *ui.Context) {
		var invalid string
		if email != "" && !strings.Contains(email, "@") {
			invalid = "Enter an email address."
		}
		ui.Column(c).Width(400).Children(func() {
			ui.Form(c, func() {
				ui.Field(c, "Name", func() { ui.TextInput(c, &name) })
				ui.Field(c, "Email", func() { ui.TextInput(c, &email) }).Description("For receipts.").Error(invalid)
				ui.Field(c, "Plan", func() { ui.Select(c, &plan, plans) })
				ui.Fieldset(c, "Notifications", func() {
					ui.Field(c, "Email", func() { ui.Checkbox(c, &mail, "Weekly summary") })
					ui.Field(c, "Sound", func() { ui.Switch(c, &sound) })
				})
			})
		})
	})
}
