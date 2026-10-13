// Package theme holds Arion's semantic design tokens and motion presets.
//
// Colors follow Sunbeam, the design system Arion shares with Lyra: one
// yellow accent (#FFCC00) that always carries dark ink text, and warm
// low-chroma "ink" neutrals that read as paper in the light appearance and
// as ink in the dark one. The physical scales live only in this file;
// views use the semantic Tokens. See docs/frontend/design-system.md.
package theme

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Type scale in DIPs. Steps are at least 1.15× apart so hierarchy comes
// from size, not from shades of gray.
const (
	Display = 28
	Title   = 20
	Heading = 15
	Body    = 13
	Caption = 11.5
)

// Radius scale, as Lyra's: restrained, with nested corners computed from
// it (inner = outer − padding) rather than repeated.
const (
	RadiusXS      = 2
	RadiusSM      = 4
	RadiusMD      = 6
	RadiusLG      = 10
	RadiusXL      = 14
	RadiusPill    = 999
	RadiusControl = RadiusLG
	RadiusCard    = RadiusXL
)

// Spacing by relationship: tight inside a group, generous between groups.
const (
	GapTight   = 4
	GapRelated = 8
	GapGroup   = 28
	Gutter     = 24
)

// Sunbeam scales. Yellow is anchored at 500; ink rotates hue 40 → 22 from
// paper to ink. Kept unexported: views reach them through Tokens.
var (
	yellow100 = ui.Hex("#fff3c2")
	yellow400 = ui.Hex("#ffd21f")
	yellow500 = ui.Hex("#ffcc00")
	yellow600 = ui.Hex("#e6b800")
	yellow700 = ui.Hex("#a87e00")

	ink50  = ui.Hex("#faf8f5")
	ink100 = ui.Hex("#efece6")
	ink200 = ui.Hex("#dfdad3")
	ink300 = ui.Hex("#c0b9af")
	ink400 = ui.Hex("#988f86")
	ink500 = ui.Hex("#726b65")
	ink700 = ui.Hex("#38322e")
	ink800 = ui.Hex("#221e1b")
	ink900 = ui.Hex("#14110f")

	white = ui.Hex("#ffffff")

	success500 = ui.Hex("#1f9e5e")
	success400 = ui.Hex("#31c47a")
	danger500  = ui.Hex("#e03f1f")
	danger400  = ui.Hex("#eb6347")
	warning500 = ui.Hex("#e6780a")
	warning400 = ui.Hex("#f29233")

	// The substrate is one warm near-black in both appearances, as Lyra's
	// ontology surface: the dictation pill floats over any app on it.
	substrate     = ui.Hex("#161410")
	substrateLine = ui.Hex("#2d2820")
)

// Tokens are semantic colors resolved for the current appearance.
type Tokens struct {
	Dark bool

	Canvas     ui.Color // window background (paper)
	Card       ui.Color // grouped surfaces on the canvas
	CardBorder ui.Color // edge of cards and controls
	Hairline   ui.Color // dividers inside a card
	Text       ui.Color
	Muted      ui.Color
	Faint      ui.Color

	Accent     ui.Color // the yellow fill of primary actions and marks
	AccentText ui.Color // ink on the accent
	AccentSoft ui.Color // hover and selection wash
	AccentInk  ui.Color // the accent as text or a line on a surface
	OnFill     ui.Color // marks on saturated semantic fills (success, live)

	Live     ui.Color // an open microphone
	LiveSoft ui.Color
	Success  ui.Color
	Warning  ui.Color
	Danger   ui.Color

	// Shade is the color of shadows: warm ink, never neutral black.
	Shade ui.Color
	// Highlight is the 1px top light on raised controls.
	Highlight ui.Color
	// Scrim darkens photos under text.
	Scrim ui.Color

	PillSurface ui.Color
	PillText    ui.Color
	PillMuted   ui.Color
	PillBorder  ui.Color
}

// Resolve returns the tokens for an appearance.
func Resolve(dark bool) Tokens {
	t := Tokens{
		Dark:        false,
		Canvas:      ink50,
		Card:        white,
		CardBorder:  ink200,
		Hairline:    ink100,
		Text:        ink900,
		Muted:       ink500,
		Faint:       ink400,
		Accent:      yellow500,
		AccentText:  ink900,
		AccentSoft:  yellow100,
		AccentInk:   yellow700,
		OnFill:      white,
		Live:        danger500,
		LiveSoft:    danger500.Alpha(0.14),
		Success:     success500,
		Warning:     warning500,
		Danger:      danger500,
		Shade:       ink900,
		Highlight:   white.Alpha(0.16),
		Scrim:       ink900,
		PillSurface: substrate.Alpha(0.94),
		PillText:    ink50,
		PillMuted:   ink50.Alpha(0.62),
		PillBorder:  substrateLine,
	}
	if dark {
		t.Dark = true
		t.Canvas = ui.Hex("#090706")
		t.Card = ink800
		t.CardBorder = ui.Hex("#322d29")
		t.Hairline = ui.Hex("#2a2622")
		t.Text = ink50
		t.Muted = ink300
		t.Faint = ink400
		t.AccentSoft = yellow500.Alpha(0.14)
		t.AccentInk = yellow400
		t.Live = danger400
		t.LiveSoft = danger400.Alpha(0.16)
		t.Success = success400
		t.Warning = warning400
		t.Danger = danger400
		t.Shade = ui.Hex("#000000")
		t.Highlight = white.Alpha(0.06)
		t.PillSurface = substrate.Alpha(0.96)
		t.PillBorder = ui.Hex("#3a342a")
	}
	return t
}

// Apply installs the Sunbeam theme for the window's appearance, so MyGo's
// own controls (switches, radios, fields, the sidebar) wear it too, and
// returns the tokens for Arion's views.
func Apply(c *ui.Context) Tokens {
	base := c.Theme()
	tokens := Resolve(base.Dark)
	t := *base
	t.Background, t.Text, t.TextMuted, t.Border = tokens.Canvas, tokens.Text, tokens.Muted, tokens.CardBorder
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = yellow500, yellow400, yellow600, ink900
	t.Danger, t.Warning, t.Success = tokens.Danger, tokens.Warning, tokens.Success
	if base.Dark {
		t.Surface, t.SurfaceHover, t.SurfacePressed = ink800, ink700, ui.Hex("#423b36")
		t.Selection, t.Focus = yellow500.Alpha(0.3), yellow500.Alpha(0.6)
		t.Scrollbar = white.Alpha(0.3)
		t.Inverse, t.InverseText = ink50, ink900
	} else {
		t.Surface, t.SurfaceHover, t.SurfacePressed = white, ink100, ink200
		t.Selection, t.Focus = yellow500.Alpha(0.4), yellow600.Alpha(0.8)
		t.Scrollbar = ink900.Alpha(0.28)
		t.Inverse, t.InverseText = ink900, ink50
	}
	// The desktop's Increase Contrast keeps its effect under our palette.
	if c.Preferences().HighContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.4)
		t.Focus.A = 255
		tokens.CardBorder, tokens.Muted = t.Border, t.TextMuted
		tokens.Hairline = tokens.Hairline.Mix(t.Text, 0.3)
	}
	t.Radius = RadiusControl
	c.SetTheme(&t)
	return tokens
}

// Ink builds body with ink as the accent: MyGo's navigation controls tint
// every icon with the accent, and yellow icons on a light surface fall
// under 3:1. Yellow stays reserved for primary actions and the mark.
// Restoring the theme resets the root's background, so a view that clears
// it for the window material does so after building.
func Ink(c *ui.Context, tokens Tokens, body func()) {
	outer := c.Theme()
	t := *outer
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = tokens.Text, tokens.Text, tokens.Text, tokens.Canvas
	c.SetTheme(&t)
	body()
	c.SetTheme(outer)
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
