// Package theme holds Arion's semantic design tokens and motion presets.
//
// Colors follow Sunbeam, the design system Arion shares with Lyra, and come
// from its MyGo package (github.com/AstraFoundry/design/packages/mygo):
// one yellow accent that always carries dark ink text, and warm low-chroma
// "ink" neutrals that read as paper in the light appearance and as ink in
// the dark one. This file resolves Arion's semantic Tokens from that
// palette. The only physical colors it defines are those Sunbeam has no
// token for: the warning hue and the shortcut recording blue. Type, row
// geometry and motion stay Arion's. See docs/frontend/design-system.md.
package theme

import (
	"time"

	"github.com/AstraFoundry/design/packages/mygo/sunbeam"
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
	RadiusXS      = sunbeam.RadiusXS
	RadiusSM      = sunbeam.RadiusSM
	RadiusMD      = sunbeam.RadiusMD
	RadiusLG      = sunbeam.RadiusLG
	RadiusXL      = sunbeam.RadiusXL
	RadiusPill    = sunbeam.RadiusFull
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

// RowHeight is a settings row's height whatever control it holds — a
// switch, a button, a select or a segmented control — so the rows of a
// card line up. Only content that needs more room makes a row taller.
const RowHeight = 52

// Colors Sunbeam has no token for. Warning is Sunbeam's orange step,
// which the web package does not publish yet; the recording blue marks a
// shortcut field while it listens for keys.
var (
	warningLight       = ui.Hex("#e6780a")
	warningDark        = ui.Hex("#f29233")
	recordingInkLight  = ui.Hex("#0866ff")
	recordingSoftLight = ui.Hex("#f1f5ff")
	recordingInkDark   = ui.Hex("#8bb5ff")
	recordingSoftDark  = ui.Hex("#142441")
	white              = ui.RGB(255, 255, 255)
	black              = ui.RGB(0, 0, 0)
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
	Subtle     ui.Color // coss bg-muted: quiet fills such as keycaps
	Faint      ui.Color

	Accent        ui.Color // the yellow fill of primary actions and marks
	AccentText    ui.Color // ink on the accent
	AccentSoft    ui.Color // hover and selection wash
	AccentInk     ui.Color // the accent as text or a line on a surface
	RecordingInk  ui.Color // focused shortcut capture
	RecordingSoft ui.Color
	OnFill        ui.Color // marks on saturated semantic fills (success, live)

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

// Resolve returns the tokens for an appearance, from Sunbeam's palette.
func Resolve(dark bool) Tokens {
	p := sunbeam.Light()
	if dark {
		p = sunbeam.Dark()
	}
	t := Tokens{
		Dark:       dark,
		Canvas:     p.Background,
		Card:       p.Card,
		CardBorder: p.Border,
		// Dividers inside a card sit halfway between its edge and its face.
		Hairline:      p.Border.Mix(p.Card, 0.5),
		Text:          p.Foreground,
		Muted:         p.MutedForeground,
		Subtle:        p.Muted,
		Faint:         p.Faint,
		Accent:        p.Primary,
		AccentText:    p.PrimaryForeground,
		AccentSoft:    p.Accent,
		AccentInk:     p.AccentForeground,
		RecordingInk:  recordingInkLight,
		RecordingSoft: recordingSoftLight,
		OnFill:        white,
		Live:          p.Danger,
		LiveSoft:      p.Danger.Alpha(0.14),
		Success:       p.Success,
		Warning:       warningLight,
		Danger:        p.Danger,
		Shade:         sunbeam.Ink900,
		Highlight:     white.Alpha(0.16),
		Scrim:         p.Scrim,
		// The pill stands on the substrate in both appearances.
		PillSurface: sunbeam.SubstrateSurface.Alpha(0.94),
		PillText:    sunbeam.SubstrateHi,
		PillMuted:   sunbeam.SubstrateHi.Alpha(0.62),
		PillBorder:  sunbeam.SubstrateLine,
	}
	if dark {
		t.RecordingInk, t.RecordingSoft = recordingInkDark, recordingSoftDark
		t.LiveSoft = p.Danger.Alpha(0.16)
		t.Warning = warningDark
		t.Shade = black
		t.Highlight = white.Alpha(0.06)
		t.PillSurface = sunbeam.SubstrateSurface.Alpha(0.96)
		t.PillBorder = sunbeam.SubstrateEdge
	}
	return t
}

// Apply installs the Sunbeam theme for the window's appearance, so MyGo's
// own controls (switches, radios, fields, the sidebar) wear it too, and
// returns the tokens for Arion's views. It takes Sunbeam's colors and keeps
// the window theme's other settings, such as its font.
func Apply(c *ui.Context) Tokens {
	base := c.Theme()
	tokens := Resolve(base.Dark)
	s := sunbeam.Theme(base.Dark)
	t := *base
	t.Background, t.Text, t.TextMuted, t.Border = s.Background, s.Text, s.TextMuted, s.Border
	t.Surface, t.SurfaceHover, t.SurfacePressed = s.Surface, s.SurfaceHover, s.SurfacePressed
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = s.Accent, s.AccentHover, s.AccentPressed, s.AccentText
	t.Danger, t.Warning, t.Success = s.Danger, tokens.Warning, s.Success
	t.Selection, t.Focus = s.Selection, s.Focus
	t.Inverse, t.InverseText = tokens.Text, tokens.Canvas
	t.Scrollbar = tokens.Text.Alpha(0.28)
	if base.Dark {
		t.Selection = tokens.Accent.Alpha(0.3)
		t.Scrollbar = tokens.Text.Alpha(0.3)
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
// under 3:1. Yellow stays reserved for primary actions and the mark. The
// surface is cleared too: MyGo's Sidebar fills itself with it, which would
// set a white panel inside the window material or the column holding it.
// Restoring the theme resets the root's background, so a view that clears
// it for the window material does so after building.
func Ink(c *ui.Context, tokens Tokens, body func()) {
	outer := c.Theme()
	t := *outer
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = tokens.Text, tokens.Text, tokens.Text, tokens.Canvas
	t.Surface = ui.Transparent
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
	// Disclosure reveals measured content without overshoot.
	Disclosure = ui.Animation{Duration: 200 * time.Millisecond, Ease: ui.CubicBezier(0.23, 1, 0.32, 1)}
	// Exit leaves faster than things arrive.
	Exit = ui.Animation{Duration: 160 * time.Millisecond, Ease: ui.EaseIn}
	// Page slides onboarding steps.
	Page = ui.Animation{Spring: ui.SpringOptions{Duration: 520 * time.Millisecond, Bounce: 0.04}}
)
