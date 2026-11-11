// SF Symbols shows every SF Symbol the running macOS has, drawn by the
// system through the sfsymbols plugin, in native UI, as SwiftUI's
// Image(systemName:) shows them, with their modifiers: size, color,
// weight, the filled variant, and whether each fills its box or keeps its
// natural size. A search narrows them by name. The grid builds only the
// symbols that show, so thousands scroll as fast as a few. On another OS,
// which has no SF Symbols, it shows a few names with their fallback.
//
//	go run ./examples/sf-symbols
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/sfsymbols"
	"github.com/egoist/mygo/ui"
)

// fewNames are what the window shows where the system lists no symbols,
// the last of them one that no macOS has.
var fewNames = []string{
	"gearshape", "magnifyingglass", "xmark", "checkmark", "plus", "minus",
	"trash", "pencil", "terminal", "globe", "cpu", "keyboard",
	"sidebar.leading", "sidebar.trailing", "rectangle.split.2x1", "rectangle.on.rectangle",
	"arrow.triangle.branch", "doc.text", "note.text", "sparkles",
	"server.rack", "display", "laptopcomputer", "no.such.symbol",
}

// missing is the fallback of a symbol the system has not: a dashed square.
var missing = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-dasharray="3 3"><rect x="3" y="3" width="18" height="18" rx="3"/></svg>`))

// gallery is what the window shows: every name, those the search keeps,
// and the options chosen.
type gallery struct {
	all, shown          []string
	query, filtered     string
	grid                ui.GridState
	size, color, weight int
	fill, natural       bool
}

var (
	weights = [...]sfsymbols.Weight{sfsymbols.Light, sfsymbols.Regular, sfsymbols.Semibold, sfsymbols.Black}
	sizes   = [...]float32{16, 24, 32, 48}
)

func newGallery(names []string) *gallery {
	if len(names) == 0 {
		names = fewNames
	}
	return &gallery{all: names, shown: names, size: 1, weight: 1}
}

// filter keeps the names that hold every word of the query.
func (g *gallery) filter() {
	if g.query == g.filtered {
		return
	}
	g.filtered = g.query
	words := strings.Fields(strings.ToLower(g.query))
	if len(words) == 0 {
		g.shown = g.all
		return
	}
	g.shown = nil
	for _, n := range g.all {
		keep := true
		for _, w := range words {
			if !strings.Contains(n, w) {
				keep = false
				break
			}
		}
		if keep {
			g.shown = append(g.shown, n)
		}
	}
}

func (g *gallery) view(c *ui.Context) {
	t := c.Theme()
	g.filter()
	size := sizes[g.size]
	col := [...]ui.Color{t.Text, t.Accent, t.TextMuted}[g.color]
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Wrap().Gap(12).Padding(16).AlignItems(ui.Center).Children(func() {
			ui.SearchField(c, &g.query).Label("Search symbols").Width(220)
			ui.Segmented(c, &g.size, "16", "24", "32", "48").Label("Size")
			ui.Segmented(c, &g.color, "Text", "Accent", "Muted").Label("Color")
			ui.Segmented(c, &g.weight, "Light", "Regular", "Semibold", "Black").Label("Weight")
			ui.Checkbox(c, &g.fill, "Fill")
			ui.Checkbox(c, &g.natural, "Natural size")
			ui.Text(c, fmt.Sprintf("%d of %d symbols", len(g.shown), len(g.all))).TextColor(t.TextMuted)
			if !sfsymbols.Available() {
				ui.Text(c, "This system has no SF Symbols").TextColor(t.TextMuted)
			}
		})
		ui.GridView(c, &g.grid, len(g.shown), 132, 100, func(i int) {
			name := g.shown[i]
			ui.Column(c).Grow(1).Center().Gap(10).Padding(6).Radius(t.Radius).
				Background(t.Surface).Border(1, t.Border).Tooltip(name).Children(func() {
				ui.Column(c).Height(48).Center().Children(func() {
					s := sfsymbols.Symbol(c, name).Size(size).Color(col).Weight(weights[g.weight]).Fallback(missing)
					if g.fill {
						s.Fill()
					}
					if g.natural {
						s.Natural()
					}
				})
				// The cell's width, with long names ending in an ellipsis: the
				// tooltip shows them whole.
				ui.Text(c, name).FontSize(11).TextColor(t.TextMuted).
					AlignSelf(ui.Stretch).TextAlign(ui.Center).MaxLines(1)
			})
		}).Grow(1).Gap(10).Padding(0, 16, 16)
	})
}

func main() {
	g := newGallery(sfsymbols.Names())
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "SF Symbols",
			Width:   1000,
			Height:  720,
			Content: ui.View(g.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
