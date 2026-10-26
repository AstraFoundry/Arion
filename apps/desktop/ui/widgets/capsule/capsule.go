// Package capsule draws the floating dictation pill: a live meter while
// listening, the streaming draft, then the outcome, with spring motion
// between states.
package capsule

import (
	"arion.local/desktop/ui/entities/holdkey"
	"hash/fnv"
	"math"
	"time"

	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// previewRunes bounds the draft laid out on the line; anything before it is
// far beyond the line's left edge. The line shows the tail, the part the
// user is speaking now.
const previewRunes = 600

// glide moves the line left as it grows: the typewriter adds a character
// every few frames, and the spring turns those steps into one motion.
var glide = ui.Animation{Spring: ui.SpringOptions{Duration: 180 * time.Millisecond, Bounce: 0}}

// maxLag is how far the gliding line may trail its target, about one body
// character: the newest, still-fading text never slips past the right edge.
const maxLag = 16

// edgeFade is the width over which text leaving the line's left edge fades
// into the pill.
const edgeFade = 28

// lineHeight holds the draft line open: its text is positioned absolutely.
const lineHeight = 22

// The text slot starts as wide as the finishing hint and grows with the
// draft in slotStep steps, keeping slotHeadroom free past its end, up to
// slotMax; a long draft then glides left. Streaming hypotheses grow and
// shrink as they are revised, so the slot never narrows within a session:
// a pill sized by its text would breathe with every partial, while a fixed
// width leaves a short reply in a long, mostly empty bar.
const (
	slotStep     = 24
	slotHeadroom = 16
	slotMax      = 248
)

// State keeps the pill's state across frames.
type State struct {
	// Typing paces the streaming draft.
	Typing dictation.Typewriter
	// session and slot are the widest the text slot has been in session.
	session string
	slot    float32
}

// widen grows the text slot of session to w, up to slotMax, and returns its
// width.
func (s *State) widen(session string, w float32) float32 {
	if session != s.session {
		s.session, s.slot = session, 0
	}
	s.slot = max(s.slot, min(slotMax, w))
	return s.slot
}

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
// s carries the draft's pacing and the slot's width across frames.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens, s *State) {
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
		kit.Pill(c.Key("pill"), tokens).MaxWidth(600).Transition(pillMotion).Children(func() {
			// The live phases share one row, so the draft stays in place when
			// speech ends instead of leaving and entering again.
			key := any(phase)
			switch phase {
			case dictation.Preparing, dictation.Listening, dictation.Transcribing, dictation.Polishing, dictation.Reviewing, dictation.Inserting:
				key = "live"
			}
			row := ui.Row(c.Key(key)).Grow(1).Gap(10).AlignItems(ui.Center).Transition(contentMotion)
			row.Children(func() {
				if key == "live" {
					liveMark(c, state, tokens, phase)
				}
				switch phase {
				case dictation.Preparing, dictation.Listening:
					listening(c, state, tokens, s)
				case dictation.Transcribing, dictation.Polishing, dictation.Reviewing, dictation.Inserting:
					transcribing(c, d, tokens, store.Copy(), s)
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

// liveMark is the live phases' one glyph: the app mark's bars as the meter
// while listening, folded into three waiting dots while the microphone
// opens and while recognition finishes, the dots turning on a circle while
// the rewrite organizes the text, and the full mark again once the text is
// decided. It stays the same element across those phases, so each change
// is one continuous motion.
func liveMark(c *ui.Context, state *runtime.State, tokens theme.Tokens, phase dictation.Phase) {
	mode := kit.MarkWaiting
	switch phase {
	case dictation.Listening:
		mode = kit.MarkListening
	case dictation.Polishing:
		mode = kit.MarkOrbiting
	case dictation.Reviewing, dictation.Inserting:
		mode = kit.MarkResting
	}
	glyph := kit.VoiceMark(c.Key("mark"), state.Dictation.Levels, 20, mode, tokens.PillText, tokens.Live).Shrink(0)
	if mode == kit.MarkListening {
		// The live dot and the meter say "listening"; the word is their name
		// for assistive technology rather than text competing for the line.
		glyph.Label(copy.For(copy.Locale(state.Preferences.UILanguage)).Text(copy.PillListening))
	}
}

func listening(c *ui.Context, state *runtime.State, tokens theme.Tokens, s *State) {
	messages := copy.For(copy.Locale(state.Preferences.UILanguage))
	d := state.Dictation
	// One slot holds the draft once speech arrives, and until then how to
	// finish: a key and a few words, as Doubao's 「松手结束」.
	if text := dictation.Text(d); text != "" {
		draft(c, tokens, s, d.SessionID, s.Typing.Update(d.SessionID, text, d.Final != "", c.Now()), tokens.PillText)
		return
	}
	hint := ui.Row(c.Key("hint")).Gap(6).AlignItems(ui.Center).
		Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}})
	// The draft takes over the hint's room, so the pill does not narrow
	// when speech arrives.
	s.widen(d.SessionID, hint.LayoutBounds().W)
	hint.Children(func() {
		key := state.Preferences.HoldKey
		if d.Translating && state.Preferences.TranslateKey != "" && state.Preferences.TranslateKey != runtime.HoldOff {
			key = state.Preferences.TranslateKey
		}
		kit.Kbd(c, tokens, holdkey.Legend(key), true)
		if state.Hold.Holding {
			ui.Text(c, messages.Text(copy.PillReleaseHint)).FontSize(theme.Caption).TextColor(tokens.PillMuted).SingleLine()
			return
		}
		ui.Text(c, messages.Text(copy.PillFinishHint)).FontSize(theme.Caption).TextColor(tokens.PillMuted).SingleLine()
	})
}

// draft shows the end of the streaming text on one line, in the text slot
// of session grown to fit it. Once the text is wider than the slot at its
// widest, it glides left so its end stays in view, and the characters
// leaving the left edge fade out. The typewriter's newest characters fade
// in at the end.
func draft(c *ui.Context, tokens theme.Tokens, s *State, session string, frame dictation.Frame, color ui.Color) {
	if frame.Animating {
		c.AnimationFrame()
	}
	runes := []rune(frame.Text)
	runes = runes[max(0, len(runes)-previewRunes):]
	// Where each character starts, from cached single-character widths. The
	// characters drawn at those exact positions are drawn one span each.
	starts := make([]float32, len(runes)+1)
	for i, r := range runes {
		starts[i+1] = starts[i] + advance(c, r)
	}
	// Shaped as runs, the line can come out wider than its characters
	// measured singly; keep the wider so the end stays inside.
	shaped, _ := c.MeasureText(0, ui.Span{Text: string(runes), Size: theme.Body})
	full := max(starts[len(runes)], shaped) + 1
	room := s.widen(session, slotStep*float32(math.Ceil(float64((full+slotHeadroom)/slotStep))))
	hidden := max(0, full-room)
	line := ui.Row(c.Key("draft")).Shrink(1).ClipX().AlignItems(ui.Center)
	line.Width(line.AnimateTo("width", room, glide))
	// The text dims when speech ends; ease it rather than switch.
	color = line.AnimateColor("color", color, glide)
	shift := max(line.AnimateTo("shift", hidden, glide), hidden-maxLag)
	// Characters wholly past the left edge are not drawn; those crossing the
	// fade zone are drawn singly, each fading with its distance from the edge.
	first := 0
	for first < len(runes) && starts[first+1] <= shift {
		first++
	}
	zone := first
	for zone < len(runes) && shift > 0 && starts[zone] < shift+edgeFade {
		zone++
	}
	tail := len(runes) - min(len(frame.Opacity), len(runes)-zone)
	var spans []ui.Span
	for i := first; i < zone; i++ {
		x := max(0, starts[i]-shift)
		// A span color with zero alpha reads as unset and draws in the full
		// text color, so the faintest character keeps a trace of alpha.
		edge := max(0.01, 1-(1-x/edgeFade)*min(1, shift/edgeFade))
		spans = append(spans, ui.Span{Text: string(runes[i]), Color: color.Alpha(edge * fadeIn(frame, len(runes), i))})
	}
	if zone < tail {
		spans = append(spans, ui.Span{Text: string(runes[zone:tail])})
	}
	for i := tail; i < len(runes); i++ {
		spans = append(spans, ui.Span{Text: string(runes[i]), Color: color.Alpha(fadeIn(frame, len(runes), i))})
	}
	// Absolute text no longer follows the row's centering; center its line
	// box in the line as the row did.
	_, textHeight := c.MeasureText(0, ui.Span{Text: "中", Size: theme.Body})
	line.Height(lineHeight).Children(func() {
		// Out of the row's layout and unwrapped, the text keeps its full
		// width instead of truncating to the line; the row clips it.
		ui.RichText(c, spans...).FontSize(theme.Body).TextColor(color).NoWrap().
			Absolute().Left(starts[first] - shift).Top((lineHeight - textHeight) / 2).Width(starts[len(runes)] - starts[first] + edgeFade)
	})
}

// fadeIn is the typewriter's opacity for rune i of n.
func fadeIn(frame dictation.Frame, n, i int) float32 {
	if k := i - (n - len(frame.Opacity)); k >= 0 {
		return frame.Opacity[k]
	}
	return 1
}

// advances caches each character's width at the body size; the draft line
// lays out hundreds of characters every frame while it animates.
var advances = map[rune]float32{}

func advance(c *ui.Context, r rune) float32 {
	if w, ok := advances[r]; ok {
		return w
	}
	w, _ := c.MeasureText(0, ui.Span{Text: string(r), Size: theme.Body})
	advances[r] = w
	return w
}

func transcribing(c *ui.Context, d runtime.Dictation, tokens theme.Tokens, messages copy.Catalog, s *State) {
	// Recognition has ended: the rest of the draft drains at full speed.
	frame := dictation.Frame{Text: messages.Text(copy.PillTranscribing)}
	if text := dictation.Text(d); text != "" {
		frame = s.Typing.Update(d.SessionID, text, true, c.Now())
	}
	draft(c, tokens, s, d.SessionID, frame, tokens.PillMuted)
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

// run turns a session ID into a keyframe run so outcome motion replays for
// each new session only.
func run(sessionID string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(sessionID))
	return h.Sum64()
}
