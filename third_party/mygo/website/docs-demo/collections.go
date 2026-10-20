package main

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Icons of the collections' demos.
var (
	collectionsIconImage = icon(`<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.1-3.1a2 2 0 0 0-2.8 0L6 21"/>`)
	collectionsIconCode  = icon(`<path d="m16 18 6-6-6-6M8 6l-6 6 6 6"/>`)
)

// collectionsFile is a file of the demos' lists and tables.
type collectionsFile struct {
	id   int
	name string
	kind string
	size int // in bytes
}

// collectionsFiles returns the demos' files, new each time, for a demo to
// keep its own and change them.
func collectionsFiles() []*collectionsFile {
	names := []string{
		"Annual Report.pdf", "Budget 2026.xlsx", "Beach.jpg", "Cover Letter.docx",
		"Design Review.key", "Expenses.csv", "Family.jpg", "Flight Tickets.pdf",
		"Garden Plan.png", "Groceries.txt", "Holiday Video.mov", "Invoice 1042.pdf",
		"Lecture Notes.md", "main.go", "Meeting Notes.txt", "Mountains.jpg",
		"Podcast Episode.mp3", "Quarterly Plan.pages", "README.md", "Recipes.pdf",
		"Resume.pdf", "Road Trip.png", "Screenshot.png", "Slides.key",
		"Sunset.jpg", "Taxes 2025.pdf", "Team Photo.jpg", "Wedding Speech.docx",
	}
	kinds := map[string]string{
		".pdf": "PDF document", ".xlsx": "Spreadsheet", ".csv": "CSV text", ".jpg": "JPEG image",
		".png": "PNG image", ".docx": "Word document", ".pages": "Pages document", ".key": "Keynote",
		".txt": "Plain text", ".md": "Markdown", ".go": "Go source", ".mov": "QuickTime movie",
		".mp3": "MP3 audio",
	}
	files := make([]*collectionsFile, len(names))
	for i, name := range names {
		// Sizes that look like files', the same at every run.
		size := (i*7919+1237)%9000*137 + 900
		if k := path.Ext(name); k == ".mov" || k == ".mp3" {
			size *= 40
		}
		files[i] = &collectionsFile{id: i, name: name, kind: kinds[path.Ext(name)], size: size}
	}
	return files
}

// collectionsSize formats a size in bytes as Finder does.
func collectionsSize(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%d KB", n/1000)
	}
	return fmt.Sprintf("%d bytes", n)
}

// collectionsFileIcon is the icon of a file by its name.
func collectionsFileIcon(name string) *ui.SVG {
	switch path.Ext(name) {
	case ".jpg", ".png":
		return collectionsIconImage
	case ".go":
		return collectionsIconCode
	}
	return iconFile
}

// collectionsMove moves the items at rows, in order, before the item at
// to, as a Reorder asks.
func collectionsMove[T any](items []T, rows []int, to int) []T {
	moved := make([]T, 0, len(rows))
	rest := make([]T, 0, len(items))
	before := 0
	for i, it := range items {
		if slices.Contains(rows, i) {
			moved = append(moved, it)
			if i < to {
				before++
			}
			continue
		}
		rest = append(rest, it)
	}
	at := to - before
	return slices.Concat(rest[:at], moved, rest[at:])
}

func init() {
	// List: files to choose several of, by their IDs, and delete.
	files := collectionsFiles()
	var (
		chosen ui.Selection[int]
		state  ui.ListState
		row    = 2
		opened string
	)
	chosen.Add(files[row].id)
	demo("list", func(c *ui.Context) {
		t := c.Theme()
		state.Key = func(i int) any { return files[i].id }
		state.Label = func(i int) string { return files[i].name }
		state.Selected = &row
		state.Selection = &chosen
		list := ui.List(c, &state, len(files), func(i int) {
			f := files[i]
			ui.Row(c).Gap(8).Padding(6, 10).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, collectionsFileIcon(f.name)).Size(16, 16)
				ui.Text(c, f.name).SingleLine().Grow(1)
				muted := t.TextMuted
				if chosen.Has(f.id) {
					muted = t.AccentText.Alpha(0.75)
				}
				ui.Text(c, collectionsSize(f.size)).FontSize(12).TextColor(muted)
			})
		}).AlignSelf(ui.Stretch).Height(196).Padding(4).Border(1, t.Border).Radius(8).Children(func() {
			if len(files) == 0 {
				ui.Text(c, "No files").TextColor(t.TextMuted).Padding(12)
			}
		})
		if list.Submitted() && row >= 0 && row < len(files) {
			opened = files[row].name
		}
		ui.Row(c).Gap(12).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			text := fmt.Sprintf("%d files, %d chosen: Shift-click or Cmd-click to choose several", len(files), chosen.Len())
			if opened != "" {
				text = "Opened " + opened
			}
			status(c, text)
			ui.Spacer(c)
			if ui.Button(c, "Restore").Disabled(len(files) == len(collectionsFiles())).Clicked() {
				files, opened = collectionsFiles(), ""
				chosen.Clear()
				row = -1
			}
			if ui.Button(c, fmt.Sprintf("Delete %d", chosen.Len())).Disabled(chosen.Len() == 0).Clicked() {
				files = slices.DeleteFunc(files, func(f *collectionsFile) bool { return chosen.Has(f.id) })
				chosen.Clear()
				row, opened = -1, ""
			}
		})
	})
}

func init() {
	// Table: files sorted by a column, chosen, opened and renamed in place.
	files := collectionsFiles()
	var (
		state  ui.ListState
		chosen ui.Selection[int]
		sort   = ui.SortOrder{Column: "Name"}
		row    = 2 // Budget 2026.xlsx, by name
		opened string
	)
	chosen.Add(1)
	cols := []ui.TableColumn{
		{Title: "Name", Sortable: true, MinWidth: 140},
		{Title: "Kind", Width: 140, Sortable: true},
		{Title: "Size", Width: 90, Align: ui.End, Sortable: true},
	}
	demo("table", func(c *ui.Context) {
		sorted := slices.Clone(files)
		slices.SortStableFunc(sorted, func(a, b *collectionsFile) int {
			var r int
			switch sort.Column {
			case "Kind":
				r = cmp.Compare(a.kind, b.kind)
			case "Size":
				r = cmp.Compare(a.size, b.size)
			}
			if r == 0 {
				r = cmp.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
			}
			if sort.Descending {
				r = -r
			}
			return r
		})
		state.Key = func(i int) any { return sorted[i].id }
		state.Label = func(i int) string { return sorted[i].name }
		state.Selected = &row
		state.Selection = &chosen
		state.Sort = &sort
		table := ui.Table(c, &state, cols, len(sorted), func(r, col int) {
			f := sorted[r]
			switch col {
			case 0:
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, collectionsFileIcon(f.name)).Size(16, 16)
					ui.EditableText(c, &f.name)
				})
			case 1:
				ui.Text(c, f.kind).SingleLine()
			case 2:
				ui.Text(c, collectionsSize(f.size))
			}
		}).AlignSelf(ui.Stretch).Height(236).Border(1, c.Theme().Border).Radius(8).Clip()
		if table.Submitted() && row >= 0 {
			opened = sorted[row].name
		}
		text := fmt.Sprintf("%d chosen. Click a header to sort; double-click a row to open it", chosen.Len())
		if opened != "" {
			text = "Opened " + opened
		}
		status(c, text)
	})
}

func init() {
	// Tree: a project's files, a folder opening with a click on its arrow.
	open := map[string]bool{"mygo": true, "mygo/ui": true}
	chosen := "mygo/ui/button.go"
	demo("tree", func(c *ui.Context) {
		var item func(p string, kids ...string)
		item = func(p string, kids ...string) {
			var o *bool
			var children func()
			if kids != nil {
				v := open[p]
				o = &v
				defer func() { open[p] = v }()
				children = func() {
					for _, k := range kids {
						switch k {
						case "ui":
							item(p+"/ui", "button.go", "list.go", "table.go")
						case "docs":
							item(p+"/docs", "README.md", "architecture.md")
						default:
							item(p + "/" + k)
						}
					}
				}
			}
			if ui.TreeItem(c, path.Base(p), o, children).Selected(chosen == p).Clicked() {
				chosen = p
			}
		}
		ui.Scroll(c).AlignSelf(ui.Stretch).Height(206).Padding(6).Border(1, c.Theme().Border).Radius(8).Children(func() {
			ui.Tree(c, func() {
				item("mygo", "ui", "docs", "app.go", "window.go", "go.mod")
			})
		})
		status(c, "Chose "+chosen)
	})
}

func init() {
	// Outline: the same tree, built only as rows show, alone or in a
	// table's columns.
	children := map[string][]string{
		"mygo":                  {"mygo/ui", "mygo/docs", "mygo/examples", "mygo/app.go", "mygo/window.go", "mygo/go.mod"},
		"mygo/ui":               {"mygo/ui/button.go", "mygo/ui/list.go", "mygo/ui/outline.go", "mygo/ui/table.go"},
		"mygo/docs":             {"mygo/docs/README.md", "mygo/docs/architecture.md"},
		"mygo/examples":         {"mygo/examples/todo", "mygo/examples/gallery"},
		"mygo/examples/todo":    {"mygo/examples/todo/main.go"},
		"mygo/examples/gallery": {"mygo/examples/gallery/main.go"},
	}
	kids := func(p string) []string { return children[p] }
	kind := func(p string) string {
		switch {
		case children[p] != nil:
			return "Folder"
		case path.Ext(p) == ".go":
			return "Go source"
		case path.Ext(p) == ".md":
			return "Markdown"
		}
		return "Document"
	}
	size := func(p string) string {
		if children[p] != nil {
			return "--"
		}
		n := 0
		for _, r := range p {
			n = n*31 + int(r)
		}
		return collectionsSize(n%60_000 + 800)
	}
	var (
		view      int
		plain     ui.OutlineState[string]
		columns   ui.OutlineState[string]
		plainRow  = 1
		columnRow = 1
	)
	plain.Open.Add("mygo")
	plain.Open.Add("mygo/ui")
	columns.Open.Add("mygo")
	columns.Open.Add("mygo/ui")
	cols := []ui.TableColumn{{Title: "Name"}, {Title: "Kind", Width: 120}, {Title: "Size", Width: 90, Align: ui.End}}
	demo("outline", func(c *ui.Context) {
		t := c.Theme()
		ui.Segmented(c, &view, "Outline", "In columns")
		roots := []string{"mygo"}
		s := &plain
		if view == 0 {
			plain.List.Selected = &plainRow
			ui.Outline(c, &plain, roots, kids, func(p string) {
				ic := iconFile
				if children[p] != nil {
					ic = iconFolder
				}
				ui.Icon(c, ic).Size(16, 16)
				ui.Text(c, path.Base(p)).SingleLine()
			}).AlignSelf(ui.Stretch).Height(206).Padding(4).Border(1, t.Border).Radius(8)
		} else {
			s = &columns
			columns.List.Selected = &columnRow
			ui.OutlineTable(c, &columns, cols, roots, kids, func(p string, col int) {
				switch col {
				case 0:
					ic := iconFile
					if children[p] != nil {
						ic = iconFolder
					}
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, ic).Size(16, 16)
						ui.Text(c, path.Base(p)).SingleLine()
					})
				case 1:
					ui.Text(c, kind(p)).SingleLine()
				case 2:
					ui.Text(c, size(p))
				}
			}).AlignSelf(ui.Stretch).Height(206).Border(1, t.Border).Radius(8).Clip()
		}
		text := "Right and Left open and close the item chosen"
		if r := *s.List.Selected; r >= 0 && r < s.Rows() {
			text = "Chose " + s.Item(r) + ". " + text
		}
		status(c, text)
	})
}

func init() {
	// Grid view: photos in as many columns as fit, chosen and dragged.
	type photo struct {
		name     string
		from, to ui.Color
	}
	names := []string{
		"Beach", "Canyon", "Desert", "Forest", "Glacier", "Harbor", "Island", "Lagoon",
		"Meadow", "Mountains", "Northern Lights", "Orchard", "Prairie", "Rainforest", "Reef", "Savanna",
		"Sunrise", "Sunset", "Tundra", "Valley", "Vineyard", "Volcano", "Waterfall", "Wetlands",
	}
	palette := [][2]string{
		{"#fde68a", "#38bdf8"}, {"#fb923c", "#9a3412"}, {"#fcd34d", "#d97706"}, {"#4ade80", "#166534"},
		{"#e0f2fe", "#60a5fa"}, {"#93c5fd", "#1e3a8a"}, {"#5eead4", "#0f766e"}, {"#67e8f9", "#0e7490"},
		{"#bef264", "#4d7c0f"}, {"#c4b5fd", "#475569"}, {"#34d399", "#312e81"}, {"#fca5a5", "#65a30d"},
		{"#fde047", "#a3a3a3"}, {"#22c55e", "#064e3b"}, {"#f472b6", "#0891b2"}, {"#fdba74", "#a16207"},
		{"#fecaca", "#f59e0b"}, {"#f97316", "#7c3aed"}, {"#e2e8f0", "#64748b"}, {"#86efac", "#15803d"},
		{"#d8b4fe", "#7f1d1d"}, {"#ef4444", "#1c1917"}, {"#a5f3fc", "#0369a1"}, {"#a3e635", "#3f6212"},
	}
	photos := make([]photo, len(names))
	for i, n := range names {
		photos[i] = photo{n + ".jpg", ui.Hex(palette[i][0]), ui.Hex(palette[i][1])}
	}
	var (
		state  ui.GridState
		chosen ui.Selection[string]
		item   = 3
		opened string
	)
	chosen.Add(photos[item].name)
	demo("grid-view", func(c *ui.Context) {
		state.Key = func(i int) any { return photos[i].name }
		state.Label = func(i int) string { return photos[i].name }
		state.Selected = &item
		state.Selection = &chosen
		state.Reorder = func(items []int, to int) { photos = collectionsMove(photos, items, to) }
		grid := ui.GridView(c, &state, len(photos), 110, 104, func(i int) {
			p := photos[i]
			ui.Box(c).Grow(1).Margin(6, 6, 2).Radius(6).Gradient(p.from, p.to, 135)
			ui.Text(c, p.name).SingleLine().FontSize(12).AlignSelf(ui.Center).Padding(0, 6, 4)
		}).AlignSelf(ui.Stretch).Height(236).Gap(4).Padding(4).Border(1, c.Theme().Border).Radius(8)
		if grid.Submitted() && item >= 0 {
			opened = photos[item].name
		}
		text := fmt.Sprintf("%d of %d chosen. The arrows move in both directions; drag photos to reorder them", chosen.Len(), len(photos))
		if opened != "" {
			text = "Opened " + opened
		}
		status(c, text)
	})
}
