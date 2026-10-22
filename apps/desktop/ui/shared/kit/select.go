package kit

import "github.com/egoist/mygo/ui"

// Select is ui.Select whose options carry a preview before their labels,
// such as a Swatch: lead builds it for an option, in the popup and on the
// trigger. It keeps ui.Select's themed look, so it sits among those in a
// settings card unchanged. label names each option; the trigger's Changed
// reports a new choice into *selected.
func Select[T comparable](c *ui.Context, selected *T, options []T, label func(T) string, lead func(T)) ui.Element {
	t := c.Theme()
	sel := ui.SelectBase(c, selected)
	trigger := sel.Trigger.Padding(t.Space(1.5), t.Space(3.5)).Gap(t.Space(1.5)).Radius(t.Radius).
		AlignItems(ui.Center).MinWidth(t.Space(35)).Border(1, t.Border).TextColor(t.Text)
	switch {
	case trigger.Pressed():
		trigger.Background(t.SurfacePressed)
	case trigger.Hovered():
		trigger.Background(t.SurfaceHover)
	default:
		trigger.Background(t.Surface)
	}
	current := *selected
	trigger.Children(func() {
		lead(current)
		ui.Text(c, label(current)).SingleLine().Grow(1)
		chevron(c, t)
	})
	sel.Popup(func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Radius+2).Background(t.Background).Border(1, t.Border).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18))
		for _, option := range options {
			item := sel.Item(option).Padding(t.Space(1.5), t.Space(2.5)).Gap(t.Space(1.5)).AlignItems(ui.Center).
				Radius(t.Radius).Label(label(option))
			switch {
			case item.Highlighted():
				item.Background(t.Accent).TextColor(t.AccentText)
			case option == current:
				item.Background(t.Surface)
			}
			item.Children(func() {
				lead(option)
				ui.Text(c, label(option)).SingleLine()
			})
		}
	})
	return trigger
}

// chevron is ui.Select's down arrow.
func chevron(c *ui.Context, t *ui.Theme) {
	ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		var path ui.Path
		path.MoveTo(r.X+r.W*0.1, r.Y+r.H*0.3).LineTo(r.X+r.W*0.5, r.Y+r.H*0.7).LineTo(r.X+r.W*0.9, r.Y+r.H*0.3)
		p.StrokePath(&path, 1.5, t.TextMuted)
	})
}
