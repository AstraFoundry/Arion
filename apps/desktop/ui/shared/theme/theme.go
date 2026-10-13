// Package theme holds Arion's semantic design tokens and motion presets.
// Colors derive from the desktop theme so light, dark, accent and contrast
// preferences follow the system; views never use physical colors directly.
package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Type scale in DIPs.
const (
	Display = 26
	Title   = 20
	Heading = 15
	Body    = 13
	Caption = 11.5
)

// Spacing and shape.
const (
	RadiusCard    = 12
	RadiusControl = 8
	RadiusPill    = 999
	Gutter        = 24
)

// Tokens are semantic colors resolved for the current appearance.
type Tokens struct {
	Canvas      ui.Color
	Card        ui.Color
	CardBorder  ui.Color
	Text        ui.Color
	Muted       ui.Color
	Faint       ui.Color
	Accent      ui.Color
	AccentText  ui.Color
	AccentSoft  ui.Color
	Live        ui.Color
	LiveSoft    ui.Color
	Success     ui.Color
	Warning     ui.Color
	Danger      ui.Color
	Hairline    ui.Color
	PillSurface ui.Color
	PillText    ui.Color
	PillMuted   ui.Color
	PillBorder  ui.Color
}

// Resolve derives tokens from the active desktop theme.
func Resolve(t *ui.Theme) Tokens {
	tokens := Tokens{
		Canvas:     t.Background,
		Card:       t.Surface,
		CardBorder: t.Border,
		Text:       t.Text,
		Muted:      t.TextMuted,
		Faint:      t.TextMuted.Alpha(0.6),
		Accent:     t.Accent,
		AccentText: t.AccentText,
		AccentSoft: t.Accent.Alpha(0.14),
		// Live marks an open microphone; a warm red reads as "recording"
		// on every accent color.
		Live:     ui.Oklch(0.66, 0.2, 27),
		LiveSoft: ui.Oklch(0.66, 0.2, 27).Alpha(0.16),
		Success:  t.Success,
		Warning:  t.Warning,
		Danger:   t.Danger,
		Hairline: t.Border.Alpha(0.7),
		// The dictation pill is dark in both appearances so it reads over
		// any app, as system HUDs do.
		PillSurface: ui.Oklch(0.2, 0.006, 270).Alpha(0.92),
		PillText:    ui.Oklch(0.97, 0, 0),
		PillMuted:   ui.Oklch(0.97, 0, 0).Alpha(0.62),
		PillBorder:  ui.Oklch(1, 0, 0).Alpha(0.12),
	}
	if t.Dark {
		tokens.Card = t.Surface
		tokens.PillSurface = ui.Oklch(0.24, 0.006, 270).Alpha(0.94)
	}
	return tokens
}

// Apply sets Arion's control metrics on top of the desktop theme.
func Apply(c *ui.Context) Tokens {
	t := *c.Theme()
	t.Radius = RadiusControl
	c.SetTheme(&t)
	return Resolve(c.Theme())
}

// Motion presets. Springs keep velocity when interrupted, which matters for
// a pill that grows while text streams in.
var (
	// Settle moves layout without overshoot.
	Settle = ui.Animation{Spring: ui.SpringOptions{Duration: 420 * time.Millisecond, Bounce: 0}}
	// Pop enters small surfaces with a slight overshoot.
	Pop = ui.Animation{Spring: ui.SpringOptions{Duration: 380 * time.Millisecond, Bounce: 0.18}}
	// Quick is for hover and color feedback.
	Quick = ui.Animation{Duration: 120 * time.Millisecond, Ease: ui.EaseOut}
	// Exit leaves faster than things arrive.
	Exit = ui.Animation{Duration: 160 * time.Millisecond, Ease: ui.EaseIn}
	// Page slides onboarding steps.
	Page = ui.Animation{Spring: ui.SpringOptions{Duration: 520 * time.Millisecond, Bounce: 0.04}}
)
