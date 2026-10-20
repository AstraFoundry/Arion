package main

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

func init() {
	saved, editing := 0, true
	demo("button", func(c *ui.Context) {
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Cancel").Disabled(!editing).Clicked() {
				editing = false
			}
			if ui.PrimaryButton(c, "").Children(func() {
				ui.Icon(c, iconSave)
				ui.Text(c, "Save").SingleLine()
			}).Clicked() {
				saved++
				editing = true
			}
			if ui.Button(c, "").Label("Delete").Tooltip("Delete").Children(func() { ui.Icon(c, iconTrash) }).Clicked() {
				saved = 0
			}
		})
		switch {
		case !editing:
			status(c, "Canceled: Save again to edit")
		default:
			status(c, fmt.Sprintf("Saved %d times", saved))
		}
	})
}

// Icons of the demos of actions.
var (
	actionsIconDots      = icon(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`)
	actionsIconUnderline = icon(`<path d="M6 4v6a6 6 0 0 0 12 0V4M4 20h16"/>`)
	actionsIconList      = icon(`<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>`)
	actionsIconGrid      = icon(`<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>`)
	actionsIconLock      = icon(`<rect x="4" y="11" width="16" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>`)
	actionsIconPin       = icon(`<path d="M12 17v5M9 10.8V4h6v6.8l2 3.2H7z"/>`)
)

func init() {
	details := false
	demo("link", func(c *ui.Context) {
		ui.Link(c, "Privacy policy", "https://example.com/privacy")
		ui.RichText(c).Children(func() {
			ui.Text(c, "Read ")
			ui.Link(c, "the guide", "https://github.com/egoist/mygo/tree/main/docs/ui")
			ui.Text(c, " to get started.")
		})
		label := "Show details"
		if details {
			label = "Hide details"
		}
		if ui.Link(c, label, "").Clicked() {
			details = !details
		}
		if details {
			status(c, "A link with no URL is a text that is clicked")
		} else {
			status(c, "Links with URLs open a tab")
		}
	})
}

func init() {
	sort, last := "Name", ""
	demo("menu-button", func(c *ui.Context) {
		ui.Row(c).Gap(8).Children(func() {
			ui.MenuButton(c, "Sort by", func(m *ui.Menu) {
				for _, by := range []string{"Name", "Date", "Size"} {
					if m.Item(by).Checked(sort == by).Chosen() {
						sort, last = by, ""
					}
				}
			})
			ui.Button(c, "").Label("More").Tooltip("More").Children(func() { ui.Icon(c, actionsIconDots) }).Menu(func(m *ui.Menu) {
				if m.Item("Duplicate").Chosen() {
					last = "Duplicate"
				}
				if m.Item("Delete").Chosen() {
					last = "Delete"
				}
			})
		})
		if last != "" {
			status(c, "Chose "+last)
		} else {
			status(c, "Sorted by "+sort)
		}
	})
}

// actionsNote is a note of the context menu's demo.
type actionsNote struct {
	name, folder   string
	pinned, locked bool
}

func actionsNotes() []*actionsNote {
	return []*actionsNote{
		{name: "Groceries", folder: "Personal", pinned: true},
		{name: "Meeting notes", folder: "Work"},
		{name: "Passwords", folder: "Personal", locked: true},
	}
}

func init() {
	notes := actionsNotes()
	folders := []string{"Personal", "Work", "Archive"}
	last := "Right-click a note for its menu"
	demo("context-menu", func(c *ui.Context) {
		t := c.Theme()
		if len(notes) == 0 {
			if ui.Button(c, "Restore the notes").Clicked() {
				notes, last = actionsNotes(), "Restored the notes"
			}
			status(c, last)
			return
		}
		deleted, moved, moveTo := -1, -1, ""
		ui.Column(c).Width(320).Border(1, t.Border).Radius(8).Padding(4).Children(func() {
			for i, note := range notes {
				row := ui.Row(c.Key(note)).Height(32).PaddingX(10).Gap(8).Radius(6)
				if row.Hovered() {
					row.Background(t.SurfaceHover)
				}
				row.ContextMenu(func(m *ui.Menu) {
					if m.Item("Rename").Shortcut(0, ui.KeyF2).Chosen() {
						last = "Chose Rename for " + note.name
					}
					if m.Item("Pinned").Checked(note.pinned).Chosen() {
						note.pinned = !note.pinned
						last = "Pinned " + note.name
						if !note.pinned {
							last = "Unpinned " + note.name
						}
					}
					m.Submenu("Move to", func(m *ui.Menu) {
						for _, f := range folders {
							if m.Item(f).Checked(f == note.folder).Chosen() {
								moveTo, moved = f, i
							}
						}
					})
					m.Separator()
					if m.Item("Delete").Disabled(note.locked).Chosen() {
						deleted = i
					}
				})
				row.Children(func() {
					ui.Icon(c, iconFile)
					ui.Text(c, note.name).SingleLine().Grow(1)
					if note.locked {
						ui.Icon(c, actionsIconLock).TextColor(t.TextMuted)
					}
					if note.pinned {
						ui.Icon(c, actionsIconPin).TextColor(t.Accent)
					}
					ui.Text(c, note.folder).SingleLine().FontSize(12).TextColor(t.TextMuted)
				})
			}
		})
		// Changes that take a note out of the loop, once it is done.
		if moved >= 0 {
			notes[moved].folder = moveTo
			last = "Moved " + notes[moved].name + " to " + moveTo
		}
		if deleted >= 0 {
			last = "Deleted " + notes[deleted].name
			notes = append(notes[:deleted:deleted], notes[deleted+1:]...)
		}
		status(c, last)
	})
}

func init() {
	inspector, bold, italic, underline := false, true, false, false
	demo("toggle", func(c *ui.Context) {
		ui.Row(c).Gap(12).Children(func() {
			ui.Toggle(c, &inspector, "Inspector")
			ui.ToggleGroup(c, func() {
				ui.Toggle(c, &bold, "").Label("Bold").Tooltip("Bold").Children(func() { ui.Icon(c, iconBold) })
				ui.Toggle(c, &italic, "").Label("Italic").Tooltip("Italic").Children(func() { ui.Icon(c, iconItalic) })
				ui.Toggle(c, &underline, "").Label("Underline").Tooltip("Underline").Children(func() { ui.Icon(c, actionsIconUnderline) })
			}).Label("Style")
		})
		text := ui.Text(c, "The quick brown fox").FontSize(18)
		if bold {
			text.Bold()
		}
		if italic {
			text.Italic()
		}
		if underline {
			text.Underline()
		}
		if inspector {
			status(c, "Inspector shown")
		} else {
			status(c, "Inspector hidden")
		}
	})
}

func init() {
	view := 0
	files := []string{"Notes", "Photos", "Music", "Drafts"}
	demo("segmented", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(12).Children(func() {
			ui.Segmented(c, &view, "List", "Grid").Label("View")
			seg := ui.SegmentedBase(c, &view, 2)
			seg.Track.Padding(3).Radius(8).Background(t.Surface).Children(func() {
				for i, ic := range []*ui.SVG{actionsIconList, actionsIconGrid} {
					s := seg.Segment(i).Padding(4, 10).Radius(6).Label([]string{"List", "Grid"}[i])
					if i == view {
						s.Background(t.Background)
					}
					s.Children(func() { ui.Icon(c, ic) })
				}
			})
		})
		switch view {
		case 0:
			ui.Column(c).Width(260).Height(116).Gap(2).Children(func() {
				for _, name := range files {
					ui.Row(c).Height(26).PaddingX(8).Gap(8).Radius(6).Background(t.Surface).Children(func() {
						ui.Icon(c, iconFolder)
						ui.Text(c, name).SingleLine()
					})
				}
			})
		case 1:
			ui.Row(c).Width(260).Height(116).Gap(8).AlignItems(ui.Start).Children(func() {
				for _, name := range files {
					ui.Column(c).Size(59, 72).Center().Gap(6).Radius(8).Background(t.Surface).Children(func() {
						ui.Icon(c, iconFolder)
						ui.Text(c, name).SingleLine().FontSize(12)
					})
				}
			})
		}
	})
}

func init() {
	router := ui.NewRouter("/1")
	bold, italic, shared := false, false, 0
	demo("toolbar", func(c *ui.Context) {
		t := c.Theme()
		ui.Column(c).Width(460).Border(1, t.Border).Radius(8).Clip().Children(func() {
			ui.Toolbar(c, func() {
				ui.BackButton(c, router)
				ui.ForwardButton(c, router)
				ui.ToggleGroup(c, func() {
					ui.Toggle(c, &bold, "Bold")
					ui.Toggle(c, &italic, "Italic")
				})
				ui.Spacer(c)
				if ui.Button(c, "Share").Clicked() {
					shared++
				}
			}).Label("Format")
			ui.Divider(c)
			router.View(c, func(r *ui.Route) {
				if !r.Match("/{n}") {
					return
				}
				n, _ := strconv.Atoi(r.Param("n"))
				ui.Column(c).Padding(12, 16).Gap(8).Children(func() {
					text := ui.Textf(c, "Chapter %d: the fox jumps over the lazy dog.", n)
					if bold {
						text.Bold()
					}
					if italic {
						text.Italic()
					}
					ui.Link(c, "Next chapter", strconv.Itoa(n+1))
				})
			}).Height(72)
		})
		if shared > 0 {
			status(c, fmt.Sprintf("Shared %d times", shared))
		} else {
			status(c, "Next chapter, then Back")
		}
	})
}
