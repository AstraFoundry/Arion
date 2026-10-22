package kit

import (
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Tooltip is coss Tooltip over Sunbeam tokens: a short line on hover or
// focus of anchor, on a card face with a hairline, radius 6, Lifted
// elevation and Caption text, at most 260 DIP wide. It holds no actions;
// anything interactive belongs in the view.
func Tooltip(c *ui.Context, tokens theme.Tokens, anchor ui.Element, text string) {
	anchor.Description(text)
	ui.TooltipBase(c, anchor, func(tip ui.Element) {
		tip.Padding(5, 8).Radius(theme.RadiusMD).Background(tokens.Card).Border(1, tokens.CardBorder).MaxWidth(260)
		Raise(tip, tokens, Lifted)
		tip.Children(func() { ui.Text(c, text).FontSize(theme.Caption).TextColor(tokens.Text) })
	})
}

// Hint is an info glyph whose Tooltip explains the setting beside it, so a
// row keeps one line. It takes the keyboard focus, which shows the tip too.
func Hint(c *ui.Context, tokens theme.Tokens, text string) ui.Element {
	hint := ui.Box(c).Size(16, 16).Justify(ui.Center).AlignItems(ui.Center).Focusable()
	color := tokens.Faint
	if hint.Hovered() || hint.Focused() {
		color = tokens.Muted
	}
	hint.Children(func() { ui.Icon(c, icons.Info).FontSize(13).TextColor(color) })
	Tooltip(c, tokens, hint, text)
	return hint
}

// TipRow is Row with its explanation in a Hint after the title instead of
// a detail line.
func TipRow(c *ui.Context, tokens theme.Tokens, title, tip string, control func()) ui.Element {
	return ui.Row(c).Gap(16).Padding(12, 16).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).Shrink(1).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(theme.Body).FontWeight(500).Shrink(1)
			if tip != "" {
				Hint(c, tokens, tip)
			}
		})
		if control != nil {
			control()
		}
	})
}

// AppButton is an app's logo on a small tile that opens it: it rises a
// little on hover and names the app in a Tooltip. It reports a click.
func AppButton(c *ui.Context, tokens theme.Tokens, logo ui.ImageSource, name string) bool {
	button := ui.ButtonBase(c).Size(28, 28).Radius(7).Justify(ui.Center).AlignItems(ui.Center).
		Background(tokens.Card).Border(1, tokens.CardBorder).Label(name)
	pose := ui.Pose{}
	if button.Hovered() {
		pose.Y = -1.5
	}
	if button.Pressed() {
		pose.Scale = 0.96
	}
	button.AnimatePose(pose, theme.Quick)
	button.Children(func() { ui.Image(c, logo).Size(18, 18) })
	Tooltip(c, tokens, button, name)
	return button.Clicked()
}

// LinkRow is a row that opens a page: an optional leading mark, the title,
// its current value and a chevron. It reports a click.
func LinkRow(c *ui.Context, tokens theme.Tokens, title, value string, lead func()) bool {
	var trailing func()
	if value != "" {
		trailing = func() { ui.Text(c, value).FontSize(theme.Body).TextColor(tokens.Muted).SingleLine().Shrink(1) }
	}
	return LinkRowWith(c, tokens, title, lead, trailing)
}

// LinkRowWith is LinkRow with its own content before the chevron in place
// of a value, as a stack of avatars.
func LinkRowWith(c *ui.Context, tokens theme.Tokens, title string, lead, trailing func()) bool {
	row := ui.ButtonBase(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Label(title)
	if row.Hovered() {
		row.Background(tokens.AccentSoft.Alpha(0.5).Over(tokens.Card))
	}
	row.Children(func() {
		if lead != nil {
			lead()
		}
		ui.Text(c, title).FontSize(theme.Body).FontWeight(500).Grow(1).Shrink(1).SingleLine()
		if trailing != nil {
			trailing()
		}
		ui.Icon(c, icons.ChevronRight).FontSize(14).TextColor(tokens.Faint)
	})
	return row.Clicked()
}
