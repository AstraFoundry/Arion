package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Icons of the navigation demos.
var (
	navInbox = icon(`<path d="M22 12h-6l-2 3h-4l-2-3H2"/><path d="M5.5 5.1 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.5-6.9A2 2 0 0 0 16.8 4H7.2a2 2 0 0 0-1.7 1.1z"/>`)
	navSent  = icon(`<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>`)
	navDraft = icon(`<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>`)
	navTag   = icon(`<path d="M12.6 2.6A2 2 0 0 0 11.2 2H4a2 2 0 0 0-2 2v7.2a2 2 0 0 0 .6 1.4l8.7 8.7a2.4 2.4 0 0 0 3.4 0l6.6-6.6a2.4 2.4 0 0 0 0-3.4z"/><circle cx="7.5" cy="7.5" r=".5"/>`)
)

// navFrame is a bordered box taking the preview's width, which holds a
// demo's window-like part: a sidebar and its page, a router's pages.
func navFrame(c *ui.Context, height float32) ui.Element {
	t := c.Theme()
	return ui.Column(c).AlignSelf(ui.Stretch).Height(height).Border(1, t.Border).Radius(10).Clip()
}

func init() {
	tab := 0
	notify, sounds := true, false
	dark, compact := false, true
	logging := false
	demo("tabs", func(c *ui.Context) {
		ui.Column(c).AlignSelf(ui.Stretch).Gap(12).Children(func() {
			ui.Tabs(c, &tab, "General", "Appearance", "Advanced")
			ui.Column(c).Gap(10).Padding(0, 4).Height(64).Children(func() {
				switch tab {
				case 0:
					ui.Checkbox(c, &notify, "Show notifications")
					ui.Checkbox(c, &sounds, "Play sounds")
				case 1:
					ui.Checkbox(c, &dark, "Dark sidebar")
					ui.Checkbox(c, &compact, "Compact rows")
				case 2:
					ui.Checkbox(c, &logging, "Write a debug log")
					status(c, "Changes apply after a restart")
				}
			})
		})
	})

	mailbox, mailboxesShown, unread := "inbox", true, 3
	demo("sidebar", func(c *ui.Context) {
		t := c.Theme()
		navFrame(c, 240).Row().AlignItems(ui.Stretch).Children(func() {
			ui.Sidebar(c, &mailbox, func() {
				ui.SidebarSection(c, "Mailboxes", &mailboxesShown, func() {
					ui.SidebarItem(c, "inbox", navInbox, "Inbox").Children(func() {
						if unread > 0 {
							ui.Badge(c, fmt.Sprint(unread))
						}
					})
					ui.SidebarItem(c, "drafts", navDraft, "Drafts")
					ui.SidebarItem(c, "sent", navSent, "Sent")
				})
				ui.SidebarSection(c, "Tags", nil, func() {
					ui.SidebarItem(c, "work", navTag, "Work")
					ui.SidebarItem(c, "family", navTag, "Family")
				})
			}).Width(200)
			ui.Column(c).Grow(1).Padding(16).Gap(8).AlignItems(ui.Start).Background(t.Background).Children(func() {
				names := map[string]string{"inbox": "Inbox", "drafts": "Drafts", "sent": "Sent", "work": "Work", "family": "Family"}
				ui.Text(c, names[mailbox]).FontSize(16).Bold()
				if mailbox == "inbox" {
					if unread > 0 {
						status(c, fmt.Sprintf("%d unread messages", unread))
						if ui.Button(c, "Mark all as read").Clicked() {
							unread = 0
						}
					} else {
						status(c, "No unread messages")
					}
				} else {
					status(c, "No messages")
				}
			})
		})
	})

	path := []string{"Macintosh HD", "Users", "ada", "Documents"}
	chosen := 0
	folders := map[string][]string{
		"Macintosh HD": {"Applications", "Library", "Users"},
		"Users":        {"ada", "Shared"},
		"ada":          {"Desktop", "Documents", "Downloads"},
		"Documents":    {"Notes", "Projects"},
		"Projects":     {"MyGo", "Website"},
	}
	demo("breadcrumbs", func(c *ui.Context) {
		if ui.Breadcrumbs(c, path, &chosen).Label("Path").Changed() {
			path = path[:chosen+1]
		}
		ui.Row(c).Gap(8).Wrap().Justify(ui.Center).Children(func() {
			subfolders := folders[path[len(path)-1]]
			for _, name := range subfolders {
				if ui.Button(c.Key(name), "").Children(func() {
					ui.Icon(c, iconFolder)
					ui.Text(c, name).SingleLine()
				}).Clicked() {
					path = append(path[:len(path):len(path)], name)
				}
			}
			if len(subfolders) == 0 {
				status(c, "No folders")
			}
		})
		status(c, "/"+strings.Join(path[1:], "/"))
	})

	router := ui.NewRouter("/")
	demo("history-buttons", func(c *ui.Context) {
		t := c.Theme()
		navFrame(c, 200).Children(func() {
			ui.Toolbar(c, func() {
				ui.BackButton(c, router)
				ui.ForwardButton(c, router)
				ui.Text(c, router.Title()).Bold()
			}).Padding(6, 8)
			ui.Divider(c)
			router.View(c, func(r *ui.Route) {
				ui.Column(c).Grow(1).Padding(16).Gap(10).Background(t.Background).Children(func() {
					switch {
					case r.Match("/"):
						r.Title("Home")
						status(c, "Open a page, then go back and forward")
						ui.Link(c, "Notes", "/notes")
						ui.Link(c, "Files", "/files")
					case r.Match("/notes"):
						r.Title("Notes")
						ui.Link(c, "Groceries", "/notes/groceries")
						ui.Link(c, "Ideas", "/notes/ideas")
					case r.Match("/notes/{id}"):
						r.Title("Note")
						ui.Text(c, "The note “"+r.Param("id")+"”")
					case r.Match("/files"):
						r.Title("Files")
						ui.Link(c, "Report.pdf", "/files/report")
					default:
						r.Title("File")
						ui.Text(c, "A file")
					}
				})
			})
		})
	})
}
