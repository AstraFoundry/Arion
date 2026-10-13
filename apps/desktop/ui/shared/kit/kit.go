// Package kit holds small native UI primitives shared by every slice.
package kit

import (
	"runtime"
	"strings"
	"time"

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

// Card is a grouped surface, as a section of System Settings.
func Card(c *ui.Context, tokens theme.Tokens) ui.Element {
	return ui.Column(c).Radius(theme.RadiusCard).Background(tokens.Card).Border(1, tokens.Hairline).Clip()
}

// Row lays out a labeled setting: title and optional detail on the left,
// the control on the right.
func Row(c *ui.Context, tokens theme.Tokens, title, detail string, control func()) ui.Element {
	return ui.Row(c).Gap(16).Padding(12, 16).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.Body)
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

// Keycap draws one key of a shortcut.
func Keycap(c *ui.Context, tokens theme.Tokens, key string, onDark bool) ui.Element {
	face, edge, ink := tokens.Card, tokens.CardBorder, tokens.Text
	if onDark {
		face, edge, ink = tokens.PillBorder, tokens.PillBorder, tokens.PillText
	}
	return ui.Row(c).MinWidth(24).Height(24).Padding(0, 7).Justify(ui.Center).AlignItems(ui.Center).
		Radius(6).Background(face).Border(1, edge).Shadow(0, 1, 0, 0, edge).Children(func() {
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
