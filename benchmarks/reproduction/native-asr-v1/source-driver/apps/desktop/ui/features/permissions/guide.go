package permissions

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// guidePadding insets the card's content; the app tile's corners follow.
const guidePadding = 10

// Guide offers a drag source only for lists that accept one.
func Guide(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	guide := store.State.Guide
	if guide.Pane == "" {
		return
	}
	var instruction copy.Key
	switch guide.Pane {
	case runtime.GuideMicrophone:
		instruction = copy.GuideMicrophone
	case runtime.GuideScreenRecording:
		instruction = copy.GuideScreenRecording
	}
	if instruction != "" {
		kit.SettingsGuide(c, tokens, guide.Shown, messages.Text(instruction), messages.Text(copy.GuideClose), store.Actions.CloseGuide, nil)
		return
	}
	kit.SettingsGuide(c, tokens, guide.Shown, messages.Text(copy.GuideDrag), messages.Text(copy.GuideClose), store.Actions.CloseGuide, func() {
		appTile(c, tokens, messages, guide)
	})
}

// appTile is Arion's app as a file: dragging it into the list adds Arion.
func appTile(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, guide runtime.Guide) {
	name := messages.Text(copy.Title)
	tile := ui.Row(c.Key("app")).Height(48).Padding(0, 12, 0, 8).Gap(10).AlignItems(ui.Center).
		Radius(theme.RadiusCard-guidePadding).Border(1, tokens.CardBorder).Cursor(ui.CursorGrab).Label(name).DragData(guide.App)
	face := tokens.Canvas
	if tile.Hovered() {
		face = tokens.AccentSoft.Alpha(0.5).Over(tokens.Canvas)
	}
	tile.Background(tile.AnimateColor("face", face, theme.Quick))
	if tile.Dragging() {
		tile.Opacity(0.48)
	}
	tile.Children(func() {
		kit.Mark(c, 32)
		ui.Text(c, name).FontSize(theme.Body).FontWeight(500).Grow(1).SingleLine()
		ui.Icon(c, icons.GripVertical).FontSize(14).TextColor(tokens.Faint)
	})
}
