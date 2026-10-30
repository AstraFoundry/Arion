// Package clipboard owns local resource history interactions and settings.
package clipboard

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

const Width = 270
const MaxHeight = 198
const visibleRows = 4

// Height keeps short histories small while bounding long lists.
func Height(items []runtime.Clip) int {
	n := min(visibleRows, len(items))
	height := 48 + max(0, n-1)*2
	for _, item := range items[:n] {
		height += rowHeight(item)
	}
	return max(100, height)
}

func rowHeight(runtime.Clip) int { return 36 }

type Panel struct {
	Query         string
	searching     bool
	selected      int
	list          ui.ListState
	selectedID    string
	items         []runtime.Clip
	hoverID       string
	hoverSince    time.Time
	previewID     string
	previewAnchor ui.Rect
}

func clipTitle(item runtime.Clip, messages copy.Catalog) string {
	switch item.Payload.Kind {
	case runtime.ClipFiles:
		var names []string
		for _, path := range item.Payload.Files {
			names = append(names, filepath.Base(path))
		}
		return strings.Join(names, ", ")
	case runtime.ClipImage:
		return messages.Text(copy.ClipboardImages)
	default:
		return strings.Join(strings.Fields(item.Payload.Text), " ")
	}
}

func (p *Panel) Render(c *ui.Context, store runtime.Store) {
	tokens := theme.Apply(c)
	messages, state := store.Copy(), store.State.Clipboard
	items := runtime.FindClips(state.Items, p.Query, "")
	p.selected = max(0, min(p.selected, len(items)-1))
	if !slices.EqualFunc(p.items, items, func(a, b runtime.Clip) bool { return a.ID == b.ID }) {
		retained := false
		for i, item := range items {
			if item.ID == p.selectedID {
				p.selected = i
				retained = true
				break
			}
		}
		if !retained {
			p.selected, p.list = 0, ui.ListState{}
		}
		p.items = items
	}
	p.previewID, p.previewAnchor = "", ui.Rect{}
	ui.Column(c).Fill().Background(tokens.Card).Border(1, tokens.CardBorder).Radius(theme.RadiusLG).Children(func() {
		ui.Row(c).Height(40).Padding(4, 8, 4, 12).Gap(6).AlignItems(ui.Center).Children(func() {
			if p.searching {
				search := ui.TextInput(c.Key("search"), &p.Query).Label(messages.Text(copy.ClipboardSearch)).Placeholder(messages.Text(copy.ClipboardSearch)).
					AutoFocus().Grow(1).Shrink(1).Height(30).Background(ui.Transparent).Border(0, ui.Transparent)
				if search.Submitted() && len(items) > 0 {
					activate(store, items[p.selected].ID)
				}
				if kit.IconButton(c.Key("search-close"), tokens, icons.X, messages.Text(copy.ClipboardCloseSearch)).Clicked() {
					p.searching = false
					p.Query = ""
				}
			} else {
				ui.Text(c, messages.Text(copy.ClipboardTitle)).FontSize(theme.Body).FontWeight(500).TextColor(tokens.Muted).Grow(1)
				if kit.IconButton(c.Key("search-open"), tokens, icons.Search, messages.Text(copy.ClipboardSearch)).Clicked() {
					p.searching = true
				}
			}
		})
		if state.Error != "" {
			ui.Column(c).Padding(0, 10, 6, 10).Children(func() { Status(c, store, tokens) })
		}
		ui.Column(c).Grow(1).MinHeight(0).Padding(0, 6, 6, 6).Children(func() {
			if len(items) == 0 {
				key := copy.ClipboardEmpty
				if !store.State.Preferences.Clipboard.Enabled {
					key = copy.ClipboardOff
				}
				if p.Query != "" {
					key = copy.ClipboardNoMatches
				}
				kit.Empty(c, tokens, messages.Text(key))
			} else {
				p.history(c, store, tokens, items)
			}
		})
	})
	p.preview(c, store)
	choose := func(index int) {
		if len(items) == 0 {
			return
		}
		p.selected = max(0, min(index, len(items)-1))
		p.selectedID = items[p.selected].ID
		p.list.ScrollIntoView(p.selected)
	}
	if c.Shortcut(0, ui.KeyDown) {
		choose(p.selected + 1)
	}
	if c.Shortcut(0, ui.KeyUp) {
		choose(p.selected - 1)
	}
	if c.Shortcut(0, ui.KeyEnter) && len(items) > 0 {
		activate(store, items[p.selected].ID)
	}
	if c.Shortcut(0, ui.KeyEscape) {
		if p.searching {
			p.searching = false
			p.Query = ""
		} else {
			store.Actions.CloseClipboard()
		}
	}
	if len(items) > 0 {
		p.selectedID = items[p.selected].ID
	}
	// Restoring the list theme also resets the root background.
	c.Root().Background(ui.Transparent)
}

func (p *Panel) history(c *ui.Context, store runtime.Store, tokens theme.Tokens, items []runtime.Clip) {
	messages, state := store.Copy(), store.State.Clipboard
	p.list.Selected = &p.selected
	p.list.Key = func(i int) any { return items[i].ID }
	p.list.Label = func(i int) string { return clipTitle(items[i], messages) }
	outer := c.Theme()
	listTheme := *outer
	listTheme.Accent, listTheme.AccentText = tokens.AccentSoft, tokens.Text
	listTheme.Radius = theme.RadiusMD
	c.SetTheme(&listTheme)
	list := ui.List(c.Key("history"), &p.list, len(items)).Grow(1).MinHeight(0).Gap(2)
	list.Rows(func(row ui.ListRow) {
		c, item := row.Context, items[row.Index]
		name := clipTitle(item, messages)
		line := ui.Row(c.Key(item.ID)).Height(float32(rowHeight(item))).Padding(0, 8).Gap(6).AlignItems(ui.Center)
		hovered := line.Hovered()
		pointerX, pointerY, pointerOver := line.PointerPosition()
		pointerX, pointerY = pointerX+line.Bounds().X, pointerY+line.Bounds().Y
		pinClicked := false
		available := !state.Using && filesAvailable(item, state)
		line.Children(func() {
			body := ui.ButtonBase(c.Key("paste-item")).Grow(1).Shrink(1).Height(float32(rowHeight(item))).Justify(ui.Start).AlignItems(ui.Center).
				Label(name).Disabled(!available)
			body.Children(func() {
				if thumb := state.Meta[item.ID].Thumbnail; item.Payload.Kind == runtime.ClipImage && thumb != nil {
					image := ui.Image(c, thumb).Size(56, 28).Fit(ui.Contain).Radius(theme.RadiusSM)
					if pointerOver && image.Bounds().Contains(pointerX, pointerY) {
						p.previewID, p.previewAnchor = item.ID, image.Bounds()
					}
				} else if item.Payload.Kind == runtime.ClipFiles {
					fileContent(c, item, tokens, messages)
				} else {
					text := ui.Text(c, name).SingleLine().FontSize(theme.Body).TextColor(tokens.Text).Grow(1).Shrink(1)
					if pointerOver && text.Bounds().Contains(pointerX, pointerY) {
						width, _ := c.MeasureText(0, ui.Span{Text: name, Size: theme.Body})
						if width > text.Bounds().W || name != item.Payload.Text {
							p.previewID, p.previewAnchor = item.ID, text.Bounds()
						}
					}
				}
			})
			if item.Payload.Kind == runtime.ClipFiles && pointerOver && body.Bounds().Contains(pointerX, pointerY) {
				p.previewID, p.previewAnchor = item.ID, body.Bounds()
			}
			if body.Clicked() {
				activate(store, item.ID)
			}
			ui.Row(c).Width(26).Height(28).Children(func() {
				if hovered {
					key := copy.ClipboardPin
					if item.Pinned {
						key = copy.ClipboardUnpin
					}
					button := kit.IconButton(c.Key("pin"), tokens, icons.Pin, messages.Text(key)).Size(26, 28).Disabled(state.Busy || state.Using)
					if item.Pinned {
						button.TextColor(tokens.AccentInk).Background(tokens.AccentSoft)
					}
					if button.Clicked() {
						pinClicked = true
						store.Actions.PinClipboard(item.ID, !item.Pinned)
					}
				}
			})
			action := copy.ClipboardPaste
			if state.CopyOnly {
				action = copy.Copy
			}
			if kit.IconButton(c.Key("activate"), tokens, icons.CornerDownLeft, messages.Text(action)).Size(26, 28).Disabled(!available).Clicked() {
				activate(store, item.ID)
			}
		})
		if line.Clicked() && !pinClicked && available {
			activate(store, item.ID)
		}
	})
	c.SetTheme(outer)
	if list.Changed() {
		p.selectedID = items[p.selected].ID
	}
	if list.Submitted() {
		activate(store, items[p.selected].ID)
	}
}

func filesAvailable(item runtime.Clip, state runtime.Clipboard) bool {
	if item.Payload.Kind != runtime.ClipFiles {
		return true
	}
	files := state.Meta[item.ID].Files
	if len(files) != len(item.Payload.Files) {
		return false
	}
	for _, f := range files {
		if !f.Available || f.Directory {
			return false
		}
	}
	return true
}

// preview uses a dwell delay so moving through the list does not flash cards.
func (p *Panel) preview(c *ui.Context, store runtime.Store) {
	if p.previewID != p.hoverID {
		p.hoverID, p.hoverSince = p.previewID, c.Now()
	}
	id := p.previewID
	if id != "" {
		remaining := 450*time.Millisecond - c.Now().Sub(p.hoverSince)
		if remaining > 0 {
			c.After(remaining)
			id = ""
		}
	}
	height := 0
	if id != "" {
		for _, item := range p.items {
			if item.ID == id {
				height = previewHeight(c, item, store.State.Clipboard.Meta[id].Thumbnail)
				break
			}
		}
	}
	store.Actions.HoverClipboard(id, p.previewAnchor, height)
}

func activate(store runtime.Store, id string) {
	if store.State.Clipboard.CopyOnly {
		store.Actions.CopyClipboard(id)
	} else {
		store.Actions.PasteClipboard(id)
	}
}
