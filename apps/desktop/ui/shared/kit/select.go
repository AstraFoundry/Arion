package kit

import (
	"arion.local/desktop/ui/shared/theme"
	"github.com/AstraFoundry/design/packages/mygo/sunbeam"
	"github.com/egoist/mygo/ui"
)

// PopupHeight caps a Select's popup: eight options show at once.
const PopupHeight = 256

// Select is ui.Select whose options may carry a preview before their
// labels, such as a Swatch: lead, when given, builds it for an option, in
// the popup and on the trigger. It keeps ui.Select's themed look, so it
// sits among those in a settings card unchanged, and its popup scrolls past
// PopupHeight, so a long list fits a small window. label names each option;
// the trigger's Changed reports a new choice into *selected.
func Select[T comparable](c *ui.Context, selected *T, options []T, label func(T) string, lead func(T)) ui.Element {
	t := c.Theme()
	sel := ui.SelectBase(c, selected)
	trigger := ControlFrame(c, sel.Trigger)
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
		if lead != nil {
			lead(current)
		}
		ui.Text(c, label(current)).SingleLine().Grow(1)
		chevron(c, t)
	})
	sel.Popup(func(panel ui.Element) {
		// The popup floats over the window: Sunbeam's raised panel surface.
		Popover(c, panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)), theme.RadiusControl+2)
		sunbeam.Layer(sunbeam.Raised, func() {
			ui.Scroll(c).MaxHeight(PopupHeight).Children(func() {
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
						if lead != nil {
							lead(option)
						}
						ui.Text(c, label(option)).SingleLine()
					})
				}
			})
		})
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
