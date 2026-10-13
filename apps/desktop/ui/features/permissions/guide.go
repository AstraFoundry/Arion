package permissions

import (
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// guidePadding insets the card's content; the app tile's corners follow.
const guidePadding = 10

// The guide sits under System Settings and slides down from its edge.
var guideMotion = ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: -8, Scale: 0.98}}

// nudge lifts the arrow toward the list a few times as the guide appears.
var nudge = []ui.Keyframe{
	{Value: 0},
	{At: 180 * time.Millisecond, Value: -4, Ease: ui.EaseOut},
	{At: 420 * time.Millisecond, Value: 0, Ease: ui.EaseIn},
}

// Guide walks the user through adding Arion to the System Settings list
// open above it: Arion's app is the drag source, and the list takes it.
// It fills its window; nothing shows while no guide is open.
func Guide(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	guide := store.State.Guide
	if guide.Pane == "" {
		return
	}
	card := ui.Column(c.Key(guide.Shown)).Fill().Padding(guidePadding).Gap(8).Radius(theme.RadiusCard).Background(tokens.Card).
		Transition(guideMotion)
	card.Children(func() {
		ui.Row(c).Height(36).Gap(8).Padding(0, 0, 0, 6).AlignItems(ui.Center).Children(func() {
			arrow := ui.Icon(c, icons.ArrowUp).FontSize(14).TextColor(tokens.AccentInk)
			lift := arrow.Keyframes("nudge", nudge, ui.KeyframeOptions{Run: uint64(guide.Shown), Repeat: 2,
				Delay: 240 * time.Millisecond, RepeatDelay: 360 * time.Millisecond})
			arrow.Translate(0, lift)
			ui.Text(c, messages.Text(copy.GuideDrag)).FontSize(theme.Body).Grow(1).Shrink(1)
			if kit.IconButton(c.Key("close"), tokens, icons.X, messages.Text(copy.GuideClose)).Clicked() {
				store.Actions.CloseGuide()
			}
		})
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
