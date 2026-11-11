// Loading shows the 29 loading indicators of the loading plugin, after
// loading.dev, in native UI, with their options: size, color, easing, the
// ends of their strokes, and a pause.
//
//	go run ./examples/loading
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/loading"
	"github.com/egoist/mygo/ui"
)

// gallery is what the window shows: the options chosen.
type gallery struct {
	size, color, easing, cap int
	paused                   bool
}

var sizes = [...]float32{16, 24, 32, 48}

func (g *gallery) view(c *ui.Context) {
	t := c.Theme()
	o := loading.Options{
		Size:   sizes[g.size],
		Color:  [...]ui.Color{t.Text, t.Accent, t.TextMuted}[g.color],
		Easing: loading.Easing(g.easing),
		Cap:    loading.Cap(g.cap),
		Paused: g.paused,
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Wrap().Gap(12).Padding(16).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &g.size, "16", "24", "32", "48").Label("Size")
			ui.Segmented(c, &g.color, "Text", "Accent", "Muted").Label("Color")
			ui.Segmented(c, &g.easing, "Linear", "Ease in out", "Stacked").Label("Easing")
			ui.Segmented(c, &g.cap, "Round", "Flat").Label("Cap")
			ui.Checkbox(c, &g.paused, "Paused")
		})
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Row(c).Wrap().Gap(12).Padding(0, 16, 16).Children(func() {
				for _, s := range loading.All() {
					ui.Column(c).Width(132).Height(116).Center().Gap(14).Radius(t.Radius).
						Background(t.Surface).Border(1, t.Border).Children(func() {
						ui.Column(c).Height(48).Center().Children(func() { s.New(c, o).Label(s.Name) })
						ui.Text(c, s.Name).FontSize(12).TextColor(t.TextMuted)
					})
				}
			})
		})
	})
}

func main() {
	g := &gallery{size: 1}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Loading",
			Width:   780,
			Height:  640,
			Content: ui.View(g.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
