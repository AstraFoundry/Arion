// Package capsule draws the floating dictation pill: a live meter while
// listening, the streaming draft, then the outcome, with spring motion
// between states.
package capsule

import (
	"hash/fnv"
	"time"
	"unicode/utf8"

	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// previewRunes bounds the draft shown in the pill; the tail is the part the
// user is speaking now.
const previewRunes = 64

var pillMotion = ui.ElementTransition{
	Animation: &theme.Settle,
	Enter:     &ui.Motion{Scale: 0.86, Y: 14},
	Exit:      &ui.Motion{Scale: 0.92, Y: 10},
}

var contentMotion = ui.ElementTransition{
	Animation: &theme.Pop,
	Enter:     &ui.Motion{Scale: 0.94},
	Exit:      &ui.Motion{Scale: 0.97},
}

// Render builds the pill bottom-centered in a transparent overlay window.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	state := store.State
	d := state.Dictation
	now := c.Now()
	phase := dictation.PhaseOf(d, now)
	if remaining := dictation.Remaining(d, now); remaining > 0 {
		c.After(remaining + 10*time.Millisecond)
	}
	ui.Column(c).Fill().Justify(ui.End).AlignItems(ui.Center).Padding(0, 0, 22, 0).Children(func() {
		if phase == dictation.Hidden {
			return
		}
		pill := kit.Pill(c.Key("pill"), tokens).MaxWidth(600).Transition(pillMotion)
		pill.Children(func() {
			row := ui.Row(c.Key(phase)).Gap(10).AlignItems(ui.Center).Transition(contentMotion)
			row.Children(func() {
				switch phase {
				case dictation.Preparing, dictation.Listening:
					listening(c, state, tokens, phase == dictation.Preparing)
				case dictation.Transcribing, dictation.Inserting:
					transcribing(c, d, tokens, store.Copy())
				case dictation.Done:
					outcome(c, d, tokens, store.Copy())
				case dictation.Failed:
					failure(c, state, tokens)
				case dictation.Cancelled:
					ui.Icon(c, icons.X).FontSize(15).TextColor(tokens.PillMuted)
					ui.Text(c, messages.Text(copy.PillCancelled)).FontSize(theme.Body).TextColor(tokens.PillMuted)
				}
			})
		})
	})
}

func listening(c *ui.Context, state *runtime.State, tokens theme.Tokens, preparing bool) {
	messages := copy.For(copy.Locale(state.Preferences.UILanguage))
	d := state.Dictation
	dot := ui.Box(c).Size(8, 8).Radius(4).Background(tokens.Live)
	if !preparing {
		pulse := dot.Loop("pulse", 1400*time.Millisecond, ui.Bounce(ui.EaseInOut))
		dot.Opacity(0.45 + 0.55*pulse)
	}
	kit.LevelBars(c, d.Levels, 11, 3, 20, 3, tokens.PillText)
	if text := dictation.Text(d); text != "" {
		ui.Text(c.Key("draft"), tail(text, previewRunes)).FontSize(theme.Body).MaxLines(2).MaxWidth(380).
			ContentTransition(ui.ContentTransition{})
	} else if !preparing {
		ui.Text(c, messages.Text(copy.PillListening)).FontSize(theme.Body).TextColor(tokens.PillMuted)
	}
	ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(0, 0, 0, 4).Children(func() {
		kit.Keycaps(c, tokens, state.Shortcut.Accelerator, true)
		ui.Text(c, messages.Text(copy.PillShortcutHint)).FontSize(theme.Caption).TextColor(tokens.PillMuted).SingleLine()
	})
}

func transcribing(c *ui.Context, d runtime.Dictation, tokens theme.Tokens, messages copy.Catalog) {
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		for i := 0; i < 3; i++ {
			dot := ui.Box(c.Key(i)).Size(5, 5).Radius(2.5).Background(tokens.PillText)
			wave := dot.Loop("wave", 900*time.Millisecond, ui.Linear)
			phase := wave - float32(i)*0.18
			if phase < 0 {
				phase += 1
			}
			lift := float32(0)
			if phase < 0.5 {
				lift = ui.EaseInOut(phase*2) * 4
			} else {
				lift = ui.EaseInOut((1-phase)*2) * 4
			}
			dot.Translate(0, -lift).Opacity(0.5 + 0.5*lift/4)
		}
	})
	text := dictation.Text(d)
	if text == "" {
		text = messages.Text(copy.PillTranscribing)
	}
	ui.Text(c.Key("draft"), tail(text, previewRunes)).FontSize(theme.Body).MaxLines(2).MaxWidth(420).TextColor(tokens.PillMuted)
}

func outcome(c *ui.Context, d runtime.Dictation, tokens theme.Tokens, messages copy.Catalog) {
	mark := ui.Box(c).Size(18, 18).Radius(9).Background(tokens.Success)
	drawn := mark.Keyframes("draw", []ui.Keyframe{{Value: 0}, {At: 320 * time.Millisecond, Value: 1, Ease: ui.EaseOut}},
		ui.KeyframeOptions{Run: run(d.SessionID), Delay: 60 * time.Millisecond})
	mark.Draw(func(p *ui.Painter, r ui.Rect) {
		check := (&ui.Path{}).MoveTo(r.X+5, r.Y+9.5).LineTo(r.X+8, r.Y+12.5).LineTo(r.X+13.5, r.Y+6)
		p.StrokePath(check.Trim(0, drawn), 2, tokens.OnFill)
	})
	ui.Text(c, messages.Text(dictation.OutcomeLabel(d, d.Pasted))).FontSize(theme.Body).FontWeight(500)
}

func failure(c *ui.Context, state *runtime.State, tokens theme.Tokens) {
	messages := copy.For(copy.Locale(state.Preferences.UILanguage))
	notice := dictation.NoticeOf(state.Dictation, state.Permissions)
	icon := ui.Icon(c, icons.TriangleAlert).FontSize(16).TextColor(tokens.Warning)
	shake := icon.Keyframes("shake", []ui.Keyframe{{Value: 0}, {At: 60 * time.Millisecond, Value: -3}, {At: 140 * time.Millisecond, Value: 3},
		{At: 220 * time.Millisecond, Value: -1.5}, {At: 300 * time.Millisecond, Value: 0, Ease: ui.EaseOut}}, ui.KeyframeOptions{Run: run(state.Dictation.SessionID)})
	icon.Translate(shake, 0)
	ui.Column(c).Gap(1).MaxWidth(420).Children(func() {
		ui.Text(c, messages.Text(notice.Title)).FontSize(theme.Body).FontWeight(500).SingleLine()
		if notice.Hint != "" {
			ui.Text(c, messages.Text(notice.Hint)).FontSize(theme.Caption).TextColor(tokens.PillMuted).SingleLine()
		}
	})
}

// tail keeps the last n runes so the pill shows what is being said now.
func tail(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	runes := []rune(text)
	return "…" + string(runes[len(runes)-n:])
}

// run turns a session ID into a keyframe run so outcome motion replays for
// each new session only.
func run(sessionID string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(sessionID))
	return h.Sum64()
}
