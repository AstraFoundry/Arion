// Embed-webview shows web pages in a window of native UI: a sidebar of
// tabs, each a web view, under a toolbar whose menu, select, tooltip and
// dialog show over the page, as the badge over its corner does. The notes
// page calls Go and hears from it through the bridge.
//
//	go run ./examples/embed-webview
package main

import (
	"context"
	"embed"
	"log"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// frontend is the notes page, which the app serves: a page loaded from a
// URL reloads, unlike one loaded with LoadHTML.
//
//go:embed notes.html
var frontend embed.FS

// Notes is bound for the pages: Like counts a like and tells every page.
type Notes struct{ app *app }

// Liked tells pages how many likes there are.
var Liked = mygo.NewEvent[int]("liked")

func (n *Notes) Like(ctx context.Context) int {
	likes := int(n.app.likes.Add(1))
	Liked.Broadcast(likes)
	// The native badge shows them too.
	n.app.win.Invalidate()
	return likes
}

type tab struct {
	id, title string
	view      *mygo.WebView
}

type app struct {
	win    *mygo.Window
	tabs   []*tab
	tab    string
	menu   bool
	dialog bool
	zoom   string
	likes  atomic.Int64
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	shown := a.tabs[0]
	for _, tb := range a.tabs {
		if tb.id == a.tab {
			shown = tb
		}
	}
	page := shown.view.Page()
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Sidebar(c, &a.tab, func() {
			for _, tb := range a.tabs {
				ui.SidebarItem(c, tb.id, nil, tb.title)
			}
		}).Width(180)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "‹").Label("Back").Disabled(!page.CanGoBack()).Clicked() {
					page.GoBack()
				}
				if ui.Button(c, "›").Label("Forward").Disabled(!page.CanGoForward()).Clicked() {
					page.GoForward()
				}
				if ui.Button(c, "Reload").Clicked() {
					page.Reload()
				}
				menu := ui.Button(c, "Menu ▾")
				if menu.Clicked() {
					a.menu = !a.menu
				}
				// The popover shows over the page.
				ui.Popover(c, menu, &a.menu, func() {
					for _, item := range []string{"Like the note", "Open the dialog", "Show a toast"} {
						entry := ui.Row(c.Key(item)).Padding(6, 12).Radius(5).Width(200)
						if entry.Hovered() {
							entry.Background(t.Accent).TextColor(t.AccentText)
						}
						if entry.Clicked() {
							a.menu = false
							switch item {
							case "Like the note":
								go (&Notes{a}).Like(context.Background())
							case "Open the dialog":
								a.dialog = true
							default:
								c.Toast("Toasts show over the page too")
							}
						}
						entry.Children(func() { ui.Text(c, item) })
					}
				})
				if ui.Select(c, &a.zoom, []string{"75%", "100%", "125%", "150%"}).Width(100).Changed() {
					zoom := map[string]float64{"75%": 0.75, "100%": 1, "125%": 1.25, "150%": 1.5}[a.zoom]
					for _, tb := range a.tabs {
						tb.view.Page().SetZoomFactor(zoom)
					}
				}
				ui.Button(c, "?").Label("Help").Tooltip("Web views show under what MyGo paints after them.")
				ui.Row(c).Grow(1)
				ui.Textf(c, "%d likes", a.likes.Load()).TextColor(t.TextMuted)
			})
			// The page of the tab shown, rounded, with a badge over its
			// corner; the others keep their pages.
			ui.Column(c).Grow(1).Margin(0, 12, 12, 12).Children(func() {
				ui.WebView(c, shown.view).Grow(1).Radius(12).Border(1, t.Border)
				ui.Row(c).Absolute().Top(12).Right(16).Padding(4, 10).Radius(12).Background(t.Accent).
					Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 60)).Children(func() {
					ui.Textf(c, "♥ %d", a.likes.Load()).TextColor(t.AccentText).FontSize(12).Bold()
				})
			})
		})
	})
	ui.Modal(c, &a.dialog, func() {
		ui.Text(c, "A dialog over the page").FontSize(18).Bold()
		ui.Text(c, "Its backdrop takes the pointer: the page under it does not.").TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.PrimaryButton(c, "Done").Clicked() {
				a.dialog = false
			}
		})
	})
}

func main() {
	a := &app{tab: "notes", zoom: "100%"}
	mygo.SetFrontend(frontend)
	mygo.Bind(&Notes{a})
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:    "Web views",
			Width:    1000,
			Height:   680,
			MinWidth: 560,
			Content:  ui.View(a.view),
		})
		notes, err := a.win.NewWebView(mygo.WebViewOptions{URL: "/notes.html"})
		if err != nil {
			log.Fatal(err)
		}
		web, err := a.win.NewWebView(mygo.WebViewOptions{URL: "https://example.com"})
		if err != nil {
			log.Fatal(err)
		}
		a.tabs = []*tab{{"notes", "Notes", notes}, {"web", "example.com", web}}
		// Navigation changes what Back and Forward can do.
		for _, tb := range a.tabs {
			tb.view.Page().OnDidNavigate(func(string) { a.win.Invalidate() })
		}
		a.win.Invalidate()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
