// Package kit holds small native UI primitives shared by every slice.
package kit

import (
	"math"
	"strconv"
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Press gives custom buttons tactile feedback: a 0.96 scale while pressed,
// moved in paint-only frames.
func Press(button ui.Element) ui.Element {
	pose := ui.Pose{}
	if button.Pressed() && !button.IsDisabled() {
		pose.Scale = 0.96
	}
	return button.AnimatePose(pose, ui.Animation{Duration: 90 * time.Millisecond, Ease: ui.EaseOut})
}

// Variant is a button's emphasis, as coss names them.
type Variant int

const (
	// Primary is the one main action of a view: the yellow accent.
	Primary Variant = iota
	// Outline is a secondary action on a raised white face.
	Outline
	// Ghost is a quiet action with no face until hovered.
	Ghost
	// Live is a primary action while the microphone is open: the same
	// anatomy in the recording color, so the control reads as changing
	// state rather than being replaced.
	Live
)

// Button is a text button with coss anatomy over Sunbeam tokens: a 1px
// edge, a 1px top light and a contact shadow on raised variants, which
// press flat. A disabled button keeps its layout at 64% opacity.
func Button(c *ui.Context, tokens theme.Tokens, variant Variant, label string, enabled bool) ui.Element {
	button := ui.ButtonBase(c).Height(32).Padding(0, 12).Gap(6).Radius(theme.RadiusControl).AlignItems(ui.Center).
		Disabled(!enabled)
	Style(button, tokens, variant, enabled)
	button.Children(func() {
		ui.Text(c, label).FontSize(theme.Body).FontWeight(500).SingleLine()
	})
	return button
}

// IconButton is a square ghost button showing only an icon; label names it
// for assistive technology and its tooltip.
func IconButton(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, label string) ui.Element {
	button := ui.ButtonBase(c).Size(28, 28).Radius(theme.RadiusMD + 2).Justify(ui.Center).AlignItems(ui.Center).
		Label(label).Tooltip(label)
	Style(button, tokens, Ghost, true)
	button.TextColor(tokens.Muted)
	button.Children(func() { ui.Icon(c, icon).FontSize(13) })
	return button
}

// Style gives a button base the anatomy of a variant in its current
// state, for buttons with content of their own.
func Style(button ui.Element, tokens theme.Tokens, variant Variant, enabled bool) {
	hovered := enabled && button.Hovered()
	pressed := enabled && button.Pressed()
	var fill, edge ui.Color
	switch variant {
	case Primary:
		fill, edge = tokens.Accent, tokens.Accent.Mix(tokens.Shade, 0.06)
		if hovered {
			fill = tokens.Accent.Mix(tokens.Card, 0.1)
		}
		button.TextColor(tokens.AccentText)
		if enabled && !pressed {
			button.InsetShadow(0, 1, 0, 0, tokens.Highlight).Shadow(0, 1, 2, 0, tokens.Accent.Mix(tokens.Shade, 0.4).Alpha(0.24))
		}
	case Live:
		fill, edge = tokens.Live, tokens.Live.Mix(tokens.Shade, 0.06)
		if hovered {
			fill = tokens.Live.Mix(tokens.Card, 0.1)
		}
		button.TextColor(tokens.OnFill)
		if enabled && !pressed {
			button.InsetShadow(0, 1, 0, 0, tokens.Highlight).Shadow(0, 1, 2, 0, tokens.Live.Mix(tokens.Shade, 0.4).Alpha(0.24))
		}
	case Outline:
		fill, edge = tokens.Card, tokens.CardBorder
		if hovered {
			fill = tokens.AccentSoft.Alpha(0.5).Over(tokens.Card)
		}
		button.TextColor(tokens.Text)
		if enabled && !pressed {
			button.InsetShadow(0, 1, 0, 0, tokens.Highlight).Shadow(0, 1, 2, 0, tokens.Shade.Alpha(0.05))
		}
	case Ghost:
		fill, edge = ui.Transparent, ui.Transparent
		if hovered {
			fill = tokens.AccentSoft
		}
		button.TextColor(tokens.Text)
	}
	if pressed && variant != Ghost {
		button.InsetShadow(0, 1, 0, 0, tokens.Shade.Alpha(0.08))
	}
	button.Background(button.AnimateColor("face", fill, theme.Quick))
	if edge.A > 0 {
		button.Border(1, edge)
	}
	if !enabled {
		button.Opacity(0.64)
	}
	Press(button)
}

// Elevation is how high a surface floats: a tight contact shadow plus an
// ambient one whose blur tracks the height, in warm ink.
type Elevation int

const (
	Raised Elevation = iota + 1
	Lifted
	Floating
	Overlay
)

// Raise casts the shadows of an elevation.
func Raise(e ui.Element, tokens theme.Tokens, level Elevation) ui.Element {
	contact, ambient := float32(0.04), float32(0.08)
	if tokens.Dark {
		contact, ambient = 0.2, 0.36
	}
	switch level {
	case Raised:
		return e.Shadow(0, 1, 0, 0, tokens.Shade.Alpha(contact)).Shadow(0, 1, 2, 0, tokens.Shade.Alpha(ambient*0.75))
	case Lifted:
		return e.Shadow(0, 1, 0, 0, tokens.Shade.Alpha(contact)).Shadow(0, 4, 10, -2, tokens.Shade.Alpha(ambient))
	case Floating:
		return e.Shadow(0, 2, 0, 0, tokens.Shade.Alpha(contact)).Shadow(0, 10, 24, -4, tokens.Shade.Alpha(ambient*1.25))
	}
	return e.Shadow(0, 4, 0, 0, tokens.Shade.Alpha(contact)).Shadow(0, 20, 48, -8, tokens.Shade.Alpha(ambient*1.75))
}

// Card is a grouped surface: white on the paper canvas, one hairline edge,
// a contact shadow in the light and a top light in the dark.
func Card(c *ui.Context, tokens theme.Tokens) ui.Element {
	card := ui.Column(c).Radius(theme.RadiusCard).Background(tokens.Card).Border(1, tokens.CardBorder).Clip()
	if tokens.Dark {
		return card.InsetShadow(0, 1, 0, 0, tokens.Highlight)
	}
	return Raise(card, tokens, Raised)
}

// Group is a titled set of cards: the title sits close to what it names,
// groups stand apart in their parent's GapGroup.
func Group(c *ui.Context, tokens theme.Tokens, title string, body func()) {
	ui.Column(c).Gap(theme.GapRelated).Children(func() {
		if title != "" {
			SectionTitle(c, tokens, title)
		}
		body()
	})
}

// Row lays out a labeled setting: title and optional detail on the left,
// the control on the right.
func Row(c *ui.Context, tokens theme.Tokens, title, detail string, control func()) ui.Element {
	return ui.Row(c).Gap(16).Padding(12, 16).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.Body).FontWeight(500)
			if detail != "" {
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted)
			}
		})
		if control != nil {
			control()
		}
	})
}

// AttentionDot marks a sidebar item with something waiting for the user:
// a small yellow dot right of its label. Its edge keeps it visible on light
// surfaces, where yellow alone is too faint.
func AttentionDot(c *ui.Context, tokens theme.Tokens) ui.Element {
	return ui.Box(c).Size(7, 7).Radius(3.5).Shrink(0).Background(tokens.Accent).
		Border(1, tokens.Accent.Mix(tokens.Shade, 0.2)).Margin(0, 4, 0, 0)
}

// Divider separates rows inside a card, inset like AppKit's.
func Divider(c *ui.Context, tokens theme.Tokens) {
	ui.Box(c).Height(1).Margin(0, 0, 0, 16).Background(tokens.Hairline)
}

// SectionTitle labels a group of cards.
func SectionTitle(c *ui.Context, tokens theme.Tokens, text string) ui.Element {
	return ui.Text(c, text).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted).Margin(0, 4)
}

// Status states an outcome in words; the icon alone carries the color, so
// a row of states never turns into a traffic light.
func Status(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, color ui.Color, text string) ui.Element {
	return ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(color)
		ui.Text(c, text).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.Text).SingleLine().ContentTransition(ui.ContentTransition{})
	})
}

// Pill is the dictation HUD's surface: the warm substrate in both
// appearances, a hairline, a 1px top light and a contact plus ambient
// shadow, so it reads over any app. The live pill and its illustrations
// share it.
func Pill(c *ui.Context, tokens theme.Tokens) ui.Element {
	return ui.Row(c).Gap(10).Padding(0, 18, 0, 16).MinHeight(44).AlignItems(ui.Center).
		Radius(theme.RadiusPill).Background(tokens.PillSurface).Border(1, tokens.PillBorder).
		InsetShadow(0, 1, 0, 0, tokens.PillText.Alpha(0.06)).
		Shadow(0, 2, 4, 0, tokens.Shade.Alpha(0.22)).Shadow(0, 10, 24, -6, tokens.Shade.Alpha(0.36)).
		TextColor(tokens.PillText)
}

// Mark draws the app icon at size DIPs. The icon's own grid leaves a
// margin, so the visible squircle is about 80% of size.
func Mark(c *ui.Context, size float32) ui.Element {
	return ui.Image(c, assets.Mark).Size(size, size).Fit(ui.Contain)
}

// Kbd is coss Kbd: one key on a muted fill, 20 DIP high and at least as
// wide, radius 4, 12 DIP medium muted text. OnDark draws it on the pill's
// substrate.
func Kbd(c *ui.Context, tokens theme.Tokens, key string, onDark bool) ui.Element {
	fill, ink := tokens.Subtle, tokens.Muted
	if onDark {
		fill, ink = tokens.PillBorder, tokens.PillMuted
	}
	return ui.Row(c).Height(20).MinWidth(20).Padding(0, 4).Gap(4).Justify(ui.Center).AlignItems(ui.Center).
		Radius(theme.RadiusSM).Background(fill).Children(func() {
		ui.Text(c, key).FontSize(12).FontWeight(500).TextColor(ink).SingleLine()
	})
}

// KbdGroup is coss KbdGroup: the keys of one combination, 4 DIP apart.
func KbdGroup(c *ui.Context) ui.Element { return ui.Row(c).Gap(4).AlignItems(ui.Center) }

// LevelBars draws a live input meter: recent levels as rounded bars that
// ease toward their targets, louder in the middle like a voice.
func LevelBars(c *ui.Context, levels []float32, count int, width, height, gap float32, color ui.Color) ui.Element {
	box := ui.Box(c).Size(float32(count)*(width+gap)-gap, height)
	heights := make([]float32, count)
	for i := 0; i < count; i++ {
		shape := 1 - 0.55*float32(absInt(i-count/2))/float32(count/2+1)
		target := width + (height-width)*levelAt(levels, i, count)*shape
		heights[i] = box.AnimateTo(barKey(i), target, ui.Animation{Spring: ui.SpringOptions{Stiffness: 520, Damping: 32}})
	}
	return box.Draw(func(p *ui.Painter, r ui.Rect) {
		for i, h := range heights {
			x := r.X + float32(i)*(width+gap)
			p.Fill(ui.Rect{X: x, Y: r.Y + (r.H-h)/2, W: width, H: h}, color, width/2)
		}
	})
}

// levelAt is the level bar i of count shows: the newest level at the
// center, older ones spread outward.
func levelAt(levels []float32, i, count int) float32 {
	n := len(levels)
	if n == 0 {
		return 0
	}
	index := n - 1 - absInt(i-count/2)*max(n/count, 1)
	if index < 0 {
		return 0
	}
	return levels[index]
}

var barKeys [64]string

func init() {
	for i := range barKeys {
		barKeys[i] = "bar" + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
}

func barKey(i int) string { return barKeys[i%len(barKeys)] }

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// SelectPopup gives a select's popup coss anatomy over Sunbeam tokens: a
// card face with a hairline, a 1px top light and lifted elevation, 4 DIPs
// below the trigger and at least as wide. items builds the options with
// SelectItem and SelectSeparator.
func SelectPopup[T comparable](tokens theme.Tokens, sel ui.SelectParts[T], items func()) ui.Element {
	return sel.Popup(func(panel ui.Element) {
		panel.Padding(4).Margin(4, 0, 0, 0).Radius(theme.RadiusControl).Background(tokens.Card).
			Border(1, tokens.CardBorder).InsetShadow(0, 1, 0, 0, tokens.Highlight).TextColor(tokens.Text)
		Raise(panel, tokens, Lifted)
		items()
	})
}

// SelectItem is an option of a select, as coss draws it: a 28 DIP row
// with a check column before the label, washed while highlighted.
func SelectItem[T comparable](c *ui.Context, tokens theme.Tokens, sel ui.SelectParts[T], value T, label string, selected bool) ui.Element {
	item := sel.Item(value).MinHeight(28).Padding(4, 16, 4, 8).Gap(8).AlignItems(ui.Center).Radius(theme.RadiusMD).
		Label(label)
	if item.Highlighted() {
		item.Background(tokens.AccentSoft)
	}
	item.Children(func() {
		check := ui.Box(c).Size(16, 16).Shrink(0).Justify(ui.Center).AlignItems(ui.Center)
		if selected {
			check.Children(func() { ui.Icon(c, icons.Check).FontSize(14).TextColor(tokens.Text) })
		}
		ui.Text(c, label).FontSize(theme.Body).SingleLine().Shrink(1)
	})
	return item
}

// Swatch previews a color in a select: a small rounded square with a
// hairline, so a fill close to the surface keeps its edge.
func Swatch(c *ui.Context, tokens theme.Tokens, color ui.Color) ui.Element {
	return ui.Box(c).Size(12, 12).Shrink(0).Radius(theme.RadiusXS+1).Background(color).Border(1, tokens.CardBorder)
}

// SelectSeparator divides groups of options in a select's popup.
func SelectSeparator(c *ui.Context, tokens theme.Tokens) ui.Element {
	return ui.Box(c).Height(1).Margin(4, 8).Background(tokens.Hairline)
}

// CardHeader is a card's title row, as coss's CardHeader: the title on the
// left and an optional action (a select, a button, a figure) on the right.
func CardHeader(c *ui.Context, tokens theme.Tokens, title string, action func()) ui.Element {
	return ui.Row(c).Gap(8).Padding(12, 12, 4, 16).MinHeight(44).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(theme.Heading).FontWeight(600).TextColor(tokens.Text).SingleLine().Grow(1).Shrink(1)
		if action != nil {
			action()
		}
	})
}

// Stat is a card stating one measurement: an icon and label over the
// figure, which value draws large (with Figure), and its unit, set small.
func Stat(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, label, unit string, value func()) ui.Element {
	card := Card(c, tokens).Padding(14, 16, 16, 16).Gap(10)
	card.Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(tokens.Muted).Children(func() {
			ui.Icon(c, icon).FontSize(14)
			ui.Text(c, label).FontSize(theme.Caption).FontWeight(500).SingleLine().Shrink(1)
		})
		// MyGo has no baseline alignment outside fields: bottoms align and
		// the unit rises by the difference in descent.
		ui.Row(c).Gap(4).AlignItems(ui.End).Children(func() {
			value()
			if unit != "" {
				ui.Text(c, unit).FontSize(theme.Body).FontWeight(500).TextColor(tokens.Muted).SingleLine().Shrink(0).Margin(0, 0, 3, 0)
			}
		})
	})
	return card
}

// Rolls keeps a view's odometer wheels between frames, so a digit passing
// 9 keeps rolling up to 0 instead of back down.
type Rolls struct{ turns map[string]float32 }

// FigureStyle is the type a Figure is set in.
type FigureStyle struct {
	Size   float32
	Weight int
	Color  ui.Color
}

// figureRoll is how a wheel turns; each digit after the first waits
// digitStagger longer, so the figure rolls in from the left.
var (
	figureRoll   = ui.SpringOptions{Duration: 900 * time.Millisecond}
	digitStagger = 35 * time.Millisecond
)

// nextTurn is where a wheel at target goes to show digit: forward, never
// back, so 9 → 0 rolls up through the turn.
func nextTurn(target float32, digit int) float32 {
	return target + float32((digit-int(target)%10+10)%10)
}

// Figure sets text in tabular digits on odometer wheels. Each digit rolls
// up to its value and keeps rolling up when the value grows; enter starts
// every wheel from 0, as when the view appears, after delay. Separators
// stay put. The text itself is laid out (invisibly) for size and for
// assistive technology; the wheels are painted over it, clipped to it.
func Figure(c *ui.Context, rolls *Rolls, key, text string, style FigureStyle, enter bool, delay time.Duration) ui.Element {
	if rolls.turns == nil {
		rolls.turns = map[string]float32{}
	}
	runes := []rune(text)
	digits := 0
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	holder := ui.Box(c.Key(key)).Clip()
	wheels := make([]float32, len(runes))
	place := 0
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < '0' || runes[i] > '9' {
			continue
		}
		// Wheels are named by place from the right, so 999 → 1,000 keeps
		// the ones on the ones.
		wheel := key + "/" + strconv.Itoa(place)
		digit := float32(runes[i] - '0')
		target, seen := rolls.turns[wheel]
		if enter || !seen {
			holder.Jump(wheel, 0)
			target = digit
		} else {
			target = nextTurn(target, int(digit))
		}
		rolls.turns[wheel] = target
		wait := delay + time.Duration(digits-1-place)*digitStagger
		wheels[i] = holder.AnimateTo(wheel, target, ui.Animation{Spring: figureRoll, Delay: wait})
		place++
	}
	holder.Children(func() {
		ui.Text(c, text).FontSize(style.Size).FontWeight(style.Weight).FontFeatures("tnum").SingleLine().Shrink(0).
			TextColor(ui.Transparent).Draw(func(p *ui.Painter, r ui.Rect) {
			x := r.X
			glyph := func(text string, y, alpha float32) {
				p.RichText(x, y, 0, ui.Span{Text: text, Size: style.Size, Weight: style.Weight, Features: "tnum", Color: style.Color.Alpha(alpha)})
			}
			for i, r0 := range runes {
				width, _ := p.MeasureText(0, ui.Span{Text: string(r0), Size: style.Size, Weight: style.Weight, Features: "tnum"})
				if r0 < '0' || r0 > '9' {
					glyph(string(r0), r.Y, 1)
				} else {
					turn := float32(math.Floor(float64(wheels[i])))
					along := wheels[i] - turn
					// The leaving digit rises out as the next rises in.
					glyph(strconv.Itoa(int(turn)%10), r.Y-along*r.H, 1-along)
					if along > 0.001 {
						glyph(strconv.Itoa((int(turn)+1)%10), r.Y+(1-along)*r.H, along)
					}
				}
				x += width
			}
		})
	})
	return holder
}

// SelectTrigger draws a select's trigger as coss does: an outline control
// showing the chosen value and an up-down chevron. label names the select
// for assistive technology.
func SelectTrigger[T comparable](c *ui.Context, tokens theme.Tokens, sel ui.SelectParts[T], label, value string) ui.Element {
	trigger := sel.Trigger.Label(label).Height(28).Padding(0, 8, 0, 10).Gap(6).MinWidth(96).
		Radius(theme.RadiusControl).AlignItems(ui.Center)
	Style(trigger, tokens, Outline, true)
	trigger.Children(func() {
		ui.Text(c, value).FontSize(theme.Body).FontWeight(500).SingleLine().Grow(1).ContentTransition(ui.ContentTransition{})
		ui.Icon(c, icons.ChevronsUpDown).FontSize(13).TextColor(tokens.Muted)
	})
	return trigger
}

// Empty fills a card's body when there is nothing to show yet, as coss's
// Empty: one quiet line, centered.
func Empty(c *ui.Context, tokens theme.Tokens, text string) ui.Element {
	return ui.Column(c).Grow(1).Padding(24, 16).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(theme.Body).TextColor(tokens.Muted).TextAlign(ui.Center)
	})
}
