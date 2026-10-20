// Docs-demo shows the previews of the pages of native UI's docs: the
// website builds it for browsers (website/docs-demo.ts) and shows a demo in
// a frame wherever a page of docs/ui has a comment
//
//	<!-- preview button 120 -->
//
// naming a demo and the frame's height in pixels. It runs on desktops too:
//
//	go run ./website/docs-demo          # every demo, in a sidebar
//	go run ./website/docs-demo button   # the button's
package main

import (
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// demos are the demos by name, the page of docs/ui they preview: "button"
// is docs/ui/button.md's.
var demos = map[string]func(c *ui.Context){}

// demo adds the demo of a page. Demos keep their state in the variables
// their function closes over, as an app keeps its state in a value.
func demo(name string, view func(c *ui.Context)) {
	if _, ok := demos[name]; ok {
		panic("docs-demo: two demos of " + name)
	}
	demos[name] = view
}

func main() {
	name := ""
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	content := index
	if view, ok := demos[name]; ok {
		content = func(c *ui.Context) { preview(c, view) }
	} else if name != "" {
		log.Fatalf("docs-demo: no demo of %q", name)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "MyGo UI",
			Width:   720,
			Height:  480,
			Content: ui.View(content),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// preview shows a demo in the middle of the window, as a page of the docs
// frames it. A demo taking the width stretches itself (AlignSelf(Stretch)).
func preview(c *ui.Context, view func(c *ui.Context)) {
	ui.Column(c).Fill().Center().Gap(12).Padding(20).Children(func() { view(c) })
}

// chosen is the demo the index shows.
var chosen = 0

// index shows every demo, chosen in a sidebar, to try them all at once.
func index(c *ui.Context) {
	names := make([]string, 0, len(demos))
	for name := range demos {
		names = append(names, name)
	}
	slices.Sort(names)
	t := c.Theme()
	ui.Row(c).Fill().Children(func() {
		ui.Scroll(c).Width(200).Background(t.Surface).Children(func() {
			ui.Column(c).Padding(8).Children(func() {
				for i, name := range names {
					b := ui.Button(c.Key(name), name).AlignSelf(ui.Stretch)
					if i == chosen {
						b.Background(t.Accent).TextColor(t.AccentText)
					}
					if b.Clicked() {
						chosen = i
					}
				}
			})
		})
		ui.Column(c).Grow(1).Children(func() {
			if chosen < len(names) {
				ui.Text(c, fmt.Sprintf("%s (docs/ui/%s.md)", names[chosen], names[chosen])).Padding(12).TextColor(t.TextMuted)
				preview(c, demos[names[chosen]])
			}
		})
	})
}
