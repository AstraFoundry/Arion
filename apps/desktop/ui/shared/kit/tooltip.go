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

// LinkRow is a row that opens a page: an optional leading mark, the title,
// its current value and a chevron. It reports a click.
func LinkRow(c *ui.Context, tokens theme.Tokens, title, value string, lead func()) bool {
	row := ui.ButtonBase(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Label(title)
	if row.Hovered() {
		row.Background(tokens.AccentSoft.Alpha(0.5).Over(tokens.Card))
	}
	row.Children(func() {
		if lead != nil {
			lead()
		}
		ui.Text(c, title).FontSize(theme.Body).FontWeight(500).Grow(1).Shrink(1).SingleLine()
		if value != "" {
			ui.Text(c, value).FontSize(theme.Body).TextColor(tokens.Muted).SingleLine().Shrink(1)
		}
		ui.Icon(c, icons.ChevronRight).FontSize(14).TextColor(tokens.Faint)
	})
	return row.Clicked()
}
