// Package capsule draws the floating dictation pill: a live meter while
// listening, the streaming draft, then the outcome, with spring motion
// between states.
package capsule

import (
	"arion.local/desktop/ui/features/holdkey"
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

// previewRunes bounds the draft handed to the text line; the line shows its
// tail, the part the user is speaking now.
const previewRunes = 64

// Live is the pill's fixed width while the microphone is open and the
// draft is finishing: streaming hypotheses grow and shrink as they are
// revised, so a pill sized by its text would breathe with every partial.
// As Typeless's recording bar, the pill changes size only between states.
const Live = 380

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
	learned := phase == dictation.Hidden && dictation.LearnedShown(state.Learned, d, now)
	if learned {
		c.After(dictation.LearnedVisible - now.Sub(state.Learned.At) + 10*time.Millisecond)
	}
	ui.Column(c).Fill().Justify(ui.End).AlignItems(ui.Center).Padding(0, 0, 22, 0).Children(func() {
		if learned {
			kit.Pill(c.Key("learned"), tokens).MaxWidth(600).Transition(pillMotion).Children(func() {
				learnedNotice(c, store, tokens)
			})
			return
		}
		if phase == dictation.Hidden {
			return
		}
		pill := kit.Pill(c.Key("pill"), tokens).MaxWidth(600).Transition(pillMotion)
		switch phase {
		case dictation.Preparing, dictation.Listening, dictation.Transcribing, dictation.Inserting:
			pill.Width(pill.AnimateTo("width", Live, theme.Settle))
		}
		pill.Children(func() {
			row := ui.Row(c.Key(phase)).Grow(1).Gap(10).AlignItems(ui.Center).Transition(contentMotion)
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
	// The dot and the meter say "listening"; the word is their name for
	// assistive technology rather than text competing for the line.
	ui.Row(c).Gap(10).AlignItems(ui.Center).Shrink(0).Label(messages.Text(copy.PillListening)).Children(func() {
		dot := ui.Box(c).Size(8, 8).Radius(4).Background(tokens.Live)
		if !preparing {
			pulse := dot.Loop("pulse", 1400*time.Millisecond, ui.Bounce(ui.EaseInOut))
			dot.Opacity(0.45 + 0.55*pulse)
		}
		kit.LevelBars(c, d.Levels, 11, 3, 20, 3, tokens.PillText)
	})
	// One slot holds the draft once speech arrives, and until then how to
	// finish: a key and a few words, as Doubao's 「松手结束」.
	if text := dictation.Text(d); text != "" {
		draft(c, tokens, text, tokens.PillText)
		return
	}
	ui.Row(c.Key("hint")).Grow(1).Gap(6).AlignItems(ui.Center).Justify(ui.End).
		Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
		if state.Shortcut.Holding {
			kit.Keycap(c, tokens, holdkey.Name(messages, state.Preferences.HoldKey), true)
			ui.Text(c, messages.Text(copy.PillReleaseHint)).FontSize(theme.Caption).TextColor(tokens.PillMuted).SingleLine()
			return
		}
		kit.Keycaps(c, tokens, state.Shortcut.Accelerator, true)
		ui.Text(c, messages.Text(copy.PillShortcutHint)).FontSize(theme.Caption).TextColor(tokens.PillMuted).SingleLine()
	})
}

// draft shows the end of the streaming text on one line: as much of the
// tail as fits beside the meter, the start cut with an ellipsis, so
// the line keeps its place however long the sentence grows. Widths come
// from the previous frame's layout; before the first, the line clips.
func draft(c *ui.Context, tokens theme.Tokens, text string, color ui.Color) {
	line := ui.Row(c.Key("draft")).Grow(1).Shrink(1).Width(0).ClipX().AlignItems(ui.Center)
	room := line.Bounds().W
	if room > 0 {
		text = fit(c, tail(text, previewRunes), room)
	}
	line.Children(func() {
		ui.Text(c, text).FontSize(theme.Body).TextColor(color).SingleLine().Shrink(0)
	})
}

// fit returns the longest tail of text that fits in width at the body
// size, after an ellipsis when cut.
func fit(c *ui.Context, text string, width float32) string {
	measure := func(s string) float32 {
		w, _ := c.MeasureText(0, ui.Span{Text: s, Size: theme.Body})
		return w
	}
	if measure(text) <= width {
		return text
	}
	runes := []rune(text)
	// Smallest cut whose tail fits: tails shrink as the cut grows.
	lo, hi := 1, len(runes)
	for lo < hi {
		mid := (lo + hi) / 2
		if measure("…"+string(runes[mid:])) <= width {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return "…" + string(runes[lo:])
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
	draft(c, tokens, text, tokens.PillMuted)
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

// learnedNotice names the entry just learned from a correction, with an
// undo that forgets it.
func learnedNotice(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	l := store.State.Learned
	ui.Icon(c, icons.Sparkles).FontSize(15).TextColor(tokens.PillMuted)
	ui.Text(c, messages.Text(copy.PillLearned)).FontSize(theme.Body).TextColor(tokens.PillMuted).SingleLine().Shrink(0)
	ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(1).ClipX().Children(func() {
		ui.Text(c, l.From).FontSize(theme.Body).SingleLine()
		ui.Icon(c, icons.ArrowRight).FontSize(12).TextColor(tokens.PillMuted)
		ui.Text(c, l.To).FontSize(theme.Body).FontWeight(500).SingleLine()
	})
	undo := ui.ButtonBase(c.Key("undo")).Height(28).Padding(0, 10).Radius(theme.RadiusControl).AlignItems(ui.Center).
		Background(tokens.PillText.Alpha(0.12)).Shrink(0)
	if undo.Hovered() {
		undo.Background(tokens.PillText.Alpha(0.2))
	}
	kit.Press(undo).Children(func() {
		ui.Text(c, messages.Text(copy.Undo)).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
	})
	if undo.Clicked() {
		store.Actions.UndoLearned(l.ID)
	}
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
