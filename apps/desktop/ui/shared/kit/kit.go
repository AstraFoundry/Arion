// Package kit holds small native UI primitives shared by every slice.
package kit

import (
	"runtime"
	"strings"
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

// Keycap draws one key of a shortcut.
func Keycap(c *ui.Context, tokens theme.Tokens, key string, onDark bool) ui.Element {
	face, edge, ink := tokens.Card, tokens.CardBorder, tokens.Text
	if onDark {
		face, edge, ink = tokens.PillBorder, tokens.PillBorder, tokens.PillText
	}
	return ui.Row(c).MinWidth(24).Height(24).Padding(0, 7).Justify(ui.Center).AlignItems(ui.Center).
		Radius(theme.RadiusMD).Background(face).Border(1, edge).Shadow(0, 1, 0, 0, edge).Children(func() {
		ui.Text(c, key).FontSize(12).FontWeight(500).TextColor(ink).SingleLine()
	})
}

// Keycaps draws an accelerator such as "Alt+Space" as platform keys.
func Keycaps(c *ui.Context, tokens theme.Tokens, accelerator string, onDark bool) ui.Element {
	return ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		for _, key := range Keys(accelerator) {
			Keycap(c, tokens, key, onDark)
		}
	})
}

var macKeys = map[string]string{
	"cmd": "⌘", "command": "⌘", "cmdorctrl": "⌘", "commandorcontrol": "⌘", "super": "⌘", "meta": "⌘",
	"alt": "⌥", "option": "⌥", "ctrl": "⌃", "control": "⌃", "shift": "⇧",
	"space": "Space", "enter": "↩", "return": "↩", "escape": "esc", "tab": "⇥", "backspace": "⌫",
	"up": "↑", "down": "↓", "left": "←", "right": "→",
}

var windowsKeys = map[string]string{
	"cmdorctrl": "Ctrl", "commandorcontrol": "Ctrl", "ctrl": "Ctrl", "control": "Ctrl", "alt": "Alt", "option": "Alt",
	"shift": "Shift", "super": "Win", "meta": "Win", "space": "Space", "escape": "Esc", "enter": "Enter",
}

// Keys splits an accelerator into display keys for this platform.
func Keys(accelerator string) []string {
	names := windowsKeys
	if runtime.GOOS == "darwin" {
		names = macKeys
	}
	var keys []string
	for _, part := range strings.Split(accelerator, "+") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if display, ok := names[strings.ToLower(part)]; ok {
			keys = append(keys, display)
		} else {
			keys = append(keys, strings.ToUpper(part[:1])+part[1:])
		}
	}
	return keys
}

// LevelBars draws a live input meter: recent levels as rounded bars that
// ease toward their targets, louder in the middle like a voice.
func LevelBars(c *ui.Context, levels []float32, count int, width, height, gap float32, color ui.Color) ui.Element {
	box := ui.Box(c).Size(float32(count)*(width+gap)-gap, height)
	heights := make([]float32, count)
	for i := 0; i < count; i++ {
		var level float32
		if n := len(levels); n > 0 {
			// Newest level at the center, older ones spread outward.
			distance := i - count/2
			if distance < 0 {
				distance = -distance
			}
			index := n - 1 - distance*max(n/count, 1)
			if index >= 0 {
				level = levels[index]
			}
		}
		shape := 1 - 0.55*float32(absInt(i-count/2))/float32(count/2+1)
		target := width + (height-width)*level*shape
		heights[i] = box.AnimateTo(barKey(i), target, ui.Animation{Spring: ui.SpringOptions{Stiffness: 520, Damping: 32}})
	}
	return box.Draw(func(p *ui.Painter, r ui.Rect) {
		for i, h := range heights {
			x := r.X + float32(i)*(width+gap)
			p.Fill(ui.Rect{X: x, Y: r.Y + (r.H-h)/2, W: width, H: h}, color, width/2)
		}
	})
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

// SelectSeparator divides groups of options in a select's popup.
func SelectSeparator(c *ui.Context, tokens theme.Tokens) ui.Element {
	return ui.Box(c).Height(1).Margin(4, 8).Background(tokens.Hairline)
}
