// Package showcase illustrates Arion at work: a mock editor window and the
// dictation pill floating over the soft photograph. It is an illustration,
// drawn from fixed copy rather than live state, except for the microphone
// levels a caller may pass in.
package showcase

import (
	"math"
	"time"
	"unicode/utf8"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Scene is what the illustration shows.
type Scene int

const (
	// Dictating loops through a dictation: the draft streams in the pill,
	// then lands in the editor. Reduced motion shows Inserted instead.
	Dictating Scene = iota
	// Listening holds the pill open over an empty editor.
	Listening
	// Inserted shows the sentence in the editor and the pill's outcome.
	Inserted
)

// The dictating loop: speech streams for speak, the text stays inserted
// until cycle ends.
const (
	speak = 3600 * time.Millisecond
	cycle = 6400 * time.Millisecond
)

// Window is the mock editor's width; the pill hangs over its lower edge.
const Window = 248

// Render fills its box with the photograph and the scene. Levels drive the
// pill's meter when given (a live microphone); otherwise it breathes.
// Radius rounds the photograph's corners (all, or top-left, top-right,
// bottom-right, bottom-left) where it meets a rounded container: each layer
// carries the corners itself, because the GPU renderers leave bitmaps and
// gradients square under a parent's rounded clip.
func Render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, scene Scene, accelerator string, levels []float32, radius ...float32) {
	now := c.Now()
	// The dictation loop is decoration: with reduced motion it holds its
	// final frame.
	if scene == Dictating && c.Preferences().ReduceMotion {
		scene = Inserted
	}
	if scene != Inserted {
		c.AnimationFrame()
	}
	spoken, inserted := progress(scene, now)
	if len(levels) == 0 {
		levels = breathing(now)
	}
	corners := [4]float32{}
	switch len(radius) {
	case 1:
		corners = [4]float32{radius[0], radius[0], radius[0], radius[0]}
	case 4:
		corners = [4]float32{radius[0], radius[1], radius[2], radius[3]}
	}
	ui.Column(c).Fill().Clip().Radius(corners[:]...).Background(tokens.Scrim).Role(ui.RoleImage).Label(messages.Text(copy.ShowcaseLabel)).Children(func() {
		ui.Image(c, assets.Backdrop).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Cover).Radius(corners[:]...)
		// Scrims keep the title bar controls and the pill's shadow legible
		// over the brightest petals. The top one meets the top corners, the
		// bottom one the bottom corners.
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(96).PassThrough().Radius(corners[0], corners[1], 0, 0).
			LinearGradient(ui.LinearGradient{From: tokens.Scrim.Alpha(0.4), To: tokens.Scrim.Alpha(0), Angle: 180})
		ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(240).PassThrough().Radius(0, 0, corners[2], corners[3]).
			LinearGradient(ui.LinearGradient{From: tokens.Scrim.Alpha(0), To: tokens.Scrim.Alpha(0.45), Angle: 180})
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			editor(c, tokens, messages, inserted)
			pill(c, tokens, messages, scene, accelerator, levels, spoken, inserted)
		})
	})
}

// progress returns how much of the sentence has been spoken and whether it
// has landed in the editor.
func progress(scene Scene, now time.Time) (float32, bool) {
	switch scene {
	case Listening:
		return 0, false
	case Inserted:
		return 1, true
	}
	at := time.Duration(now.UnixNano()) % cycle
	if at >= speak {
		return 1, true
	}
	return float32(at) / float32(speak), false
}

// editor draws a document window of another app: a quiet title bar, two
// lines of existing text as rules, and the dictated sentence at the caret.
func editor(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, inserted bool) {
	window := ui.Column(c).Width(Window).Radius(theme.RadiusLG+2).Background(tokens.Card).Border(1, tokens.CardBorder).Clip()
	kit.Raise(window, tokens, kit.Floating).Children(func() {
		ui.Row(c).Height(26).Padding(0, 10).Gap(6).AlignItems(ui.Center).Children(func() {
			for i := 0; i < 3; i++ {
				ui.Box(c.Key(i)).Size(7, 7).Radius(3.5).Background(tokens.CardBorder)
			}
		})
		ui.Box(c).Height(1).Background(tokens.Hairline)
		ui.Column(c).Padding(14, 16, 34, 16).Gap(8).Children(func() {
			ui.Box(c).Height(6).Width(168).Radius(3).Background(tokens.Hairline)
			ui.Box(c).Height(6).Width(124).Radius(3).Background(tokens.Hairline)
			ui.Row(c).MinHeight(36).AlignItems(ui.Start).Children(func() {
				if inserted {
					ui.Text(c.Key("sentence"), messages.Text(copy.ShowcaseSentence)).FontSize(theme.Body).MaxLines(2).Shrink(1).
						Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Y: 4}})
					return
				}
				// The caret waits where the text will land.
				caret := ui.Box(c).Size(1.5, 16).Margin(1, 0, 0, 0).Background(tokens.Text)
				caret.Opacity(caret.Loop("blink", 1100*time.Millisecond, ui.Bounce(ui.EaseInOut)))
			})
		})
	})
}

// pill mirrors the live pill: an open microphone with its meter and the
// streaming draft, or the outcome once the text landed.
func pill(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, scene Scene, accelerator string, levels []float32, spoken float32, inserted bool) {
	phase := "listening"
	if inserted {
		phase = "inserted"
	}
	kit.Pill(c.Key("pill"), tokens).Margin(-22, 0, 0, 0).MaxWidth(Window + 24).Children(func() {
		ui.Row(c.Key(phase)).Gap(10).AlignItems(ui.Center).
			Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.97}}).
			Children(func() {
				if inserted {
					mark := ui.Row(c).Size(18, 18).Radius(9).Background(tokens.Success).Justify(ui.Center).AlignItems(ui.Center)
					mark.TextColor(tokens.OnFill).Children(func() { ui.Icon(c, icons.Check).FontSize(11) })
					ui.Text(c, messages.Text(copy.PillInserted)).FontSize(theme.Body).FontWeight(500)
					return
				}
				dot := ui.Box(c).Size(8, 8).Radius(4).Background(tokens.Live)
				dot.Opacity(0.45 + 0.55*dot.Loop("pulse", 1400*time.Millisecond, ui.Bounce(ui.EaseInOut)))
				kit.LevelBars(c, levels, 9, 3, 18, 3, tokens.PillText)
				// The streaming draft, or the shortcut that finishes: the
				// illustration is too narrow for the live pill's full row.
				if draft := head(messages.Text(copy.ShowcaseSentence), spoken); draft != "" && scene == Dictating {
					ui.Text(c, tail(draft, 8)).FontSize(theme.Body).SingleLine().MaxWidth(150)
				} else {
					kit.Keycaps(c, tokens, accelerator, true)
				}
			})
	})
}

// head returns the first fraction of text's runes.
func head(text string, fraction float32) string {
	n := int(float32(utf8.RuneCountInString(text)) * fraction)
	return string([]rune(text)[:n])
}

// tail keeps the last n runes, the part being spoken now.
func tail(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return "…" + string(runes[len(runes)-n:])
}

// breathing is an idle, voice-like level pattern.
func breathing(now time.Time) []float32 {
	t := float64(now.UnixMilli()) / 1000
	levels := make([]float32, 9)
	for i := range levels {
		v := 0.35 + 0.3*math.Sin(t*2.1+float64(i)*0.9) + 0.2*math.Sin(t*3.7+float64(i)*1.7)
		levels[i] = float32(math.Max(0.08, math.Min(1, v)))
	}
	return levels
}
