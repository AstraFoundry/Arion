// Package agentdemo illustrates reporting a problem through a coding
// agent, as a screen recording would: the pointer copies the prompt,
// pastes it into the agent and sends it, and the agent opens a GitHub
// issue, while the camera zooms toward wherever the pointer works. It is
// an illustration drawn from fixed copy and the prompt it is given.
package agentdemo

import (
	goruntime "runtime"
	"time"
	"unicode/utf8"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/shared/typography"
	"github.com/egoist/mygo/ui"
)

// Width and Height are the stage the scene is laid out on; the page gives
// the illustration exactly this box.
const (
	Width  = 340
	Height = 500
)

// Cycle is one loop of the recording; Final is the frame that stands for
// it under reduced motion, with the issue opened.
const (
	Cycle = 9600 * time.Millisecond
	Final = 8000 * time.Millisecond
)

// point is a position on the stage.
type point struct{ X, Y float32 }

// lens is the camera: how far it zooms in, toward which point.
type lens struct {
	Zoom  float32
	Focus point
}

// key holds a value from At milliseconds into the loop, easing from the
// key before it.
type key[T any] struct {
	At    float32
	Value T
}

// Where things are on the stage: the copy button of the prompt card, the
// agent's input and send button, and the issue the agent opens.
var (
	center     = point{Width / 2, Height / 2}
	rest       = point{300, 440}
	copyButton = point{257, 292}
	input      = point{150, 350}
	send       = point{286, 356}
	issue      = point{170, 274}
)

// The pointer's path and the camera's moves, in milliseconds. Two keys
// with one value hold it still between them.
var (
	pointerKeys = []key[point]{
		{0, rest}, {500, rest}, {1300, copyButton}, {2500, copyButton}, {3200, input},
		{3900, input}, {4500, send}, {4800, send}, {5500, rest},
	}
	lensKeys = []key[lens]{
		{0, lens{1, center}}, {400, lens{1, center}}, {1300, lens{1.55, point{250, 288}}},
		{1800, lens{1.55, point{250, 288}}}, {2400, lens{1, center}}, {2600, lens{1, center}},
		{3300, lens{1.22, point{170, 345}}}, {4000, lens{1.22, point{170, 345}}}, {4500, lens{1.22, point{200, 345}}},
		{4800, lens{1.22, point{200, 345}}}, {5400, lens{1.12, point{170, 240}}}, {6200, lens{1.12, point{170, 240}}},
		{6900, lens{1.2, issue}}, {8400, lens{1.2, issue}}, {9200, lens{1, center}},
	}
)

// Moments of the story.
const (
	copyClick  = 1400 * time.Millisecond
	agentOpens = 2300 * time.Millisecond
	inputClick = 3300 * time.Millisecond
	pasteKeys  = 3400 * time.Millisecond
	pasted     = 3650 * time.Millisecond
	pasteDone  = 3950 * time.Millisecond
	sendClick  = 4600 * time.Millisecond
	replyStart = 5000 * time.Millisecond
	replyEnd   = 6000 * time.Millisecond
	issueOpens = 6300 * time.Millisecond
	press      = 160 * time.Millisecond
	ripple     = 450 * time.Millisecond
)

var clicks = []time.Duration{copyClick, inputClick, sendClick}

// frame is the whole scene at one moment.
type frame struct {
	Agent, Copied, Focused, Pasting, Pasted, Sent, Issue bool
	// Reply is how much of the agent's answer has streamed in.
	Reply float32
	// Press is how far the pointer is pressed, Ripple how far the ring of
	// the last click has spread (negative without one).
	Pointer       point
	Press, Ripple float32
	RippleAt      point
	Lens          lens
}

// frameAt computes the scene at t into the loop.
func frameAt(t time.Duration) frame {
	t %= Cycle
	f := frame{
		Agent:   t >= agentOpens,
		Copied:  t >= copyClick && t < agentOpens,
		Focused: t >= inputClick && t < sendClick,
		Pasting: t >= pasteKeys && t < pasteDone,
		Pasted:  t >= pasted && t < sendClick,
		Sent:    t >= sendClick,
		Issue:   t >= issueOpens,
		Pointer: sample(pointerKeys, t, mixPoint),
		Lens:    sample(lensKeys, t, mixLens),
		Ripple:  -1,
	}
	switch {
	case t >= replyEnd:
		f.Reply = 1
	case t > replyStart:
		f.Reply = float32(t-replyStart) / float32(replyEnd-replyStart)
	}
	for _, click := range clicks {
		since := t - click
		if since < 0 {
			continue
		}
		if since < press {
			f.Press = 1 - float32(since)/float32(press)
		}
		if since < ripple {
			f.Ripple = float32(since) / float32(ripple)
			f.RippleAt = sample(pointerKeys, click, mixPoint)
		}
	}
	return f
}

// sample eases between the keys around t.
func sample[T any](keys []key[T], at time.Duration, mix func(a, b T, p float32) T) T {
	t := float32(at) / float32(time.Millisecond)
	if t <= keys[0].At {
		return keys[0].Value
	}
	for i := 1; i < len(keys); i++ {
		if t < keys[i].At {
			from, to := keys[i-1], keys[i]
			return mix(from.Value, to.Value, ui.EaseInOut((t-from.At)/(to.At-from.At)))
		}
	}
	return keys[len(keys)-1].Value
}

func lerp(a, b, p float32) float32 { return a + (b-a)*p }

func mixPoint(a, b point, p float32) point { return point{lerp(a.X, b.X, p), lerp(a.Y, b.Y, p)} }

func mixLens(a, b lens, p float32) lens {
	return lens{lerp(a.Zoom, b.Zoom, p), mixPoint(a.Focus, b.Focus, p)}
}

// shift is the translation that brings the lens's focus to the middle of
// the stage scaled around its center, kept so the stage's edges never come
// into view.
func shift(l lens) (float32, float32) {
	axis := func(focus, size float32) float32 {
		half := size / 2
		low, high := half/l.Zoom, size-half/l.Zoom
		focus = min(max(focus, low), high)
		return (half - focus) * l.Zoom
	}
	return axis(l.Focus.X, Width), axis(l.Focus.Y, Height)
}

// pasteLegend is the paste shortcut on the keycap.
func pasteLegend() string {
	if goruntime.GOOS == "darwin" {
		return "⌘V"
	}
	return "Ctrl+V"
}

// Render fills its box with the recording. prompt is the text being
// copied and pasted.
func Render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, prompt string) {
	// The recording is decoration: with reduced motion it holds the frame
	// with the issue opened, unzoomed.
	if c.Preferences().ReduceMotion {
		f := frameAt(Final)
		f.Lens = lens{1, center}
		render(c, tokens, messages, prompt, f)
		return
	}
	c.AnimationFrame()
	render(c, tokens, messages, prompt, frameAt(time.Duration(c.Now().UnixNano())))
}

// render draws the scene of f.
func render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, prompt string, f frame) {
	x, y := shift(f.Lens)
	ui.Column(c).Fill().Clip().Background(tokens.Scrim).Role(ui.RoleImage).Label(messages.Text(copy.FeedbackDemoLabel)).Children(func() {
		ui.Column(c).Absolute().Top(0).Left(0).Size(Width, Height).TransformOrigin(0.5, 0.5).Scale(f.Lens.Zoom, f.Lens.Zoom).Translate(x, y).Children(func() {
			ui.Image(c, assets.Backdrop).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Cover)
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(96).PassThrough().
				LinearGradient(ui.LinearGradient{From: tokens.Scrim.Alpha(0.4), To: tokens.Scrim.Alpha(0), Angle: 180})
			if f.Agent {
				agent(c, tokens, messages, prompt, f)
			} else {
				promptCard(c, tokens, messages, prompt, f)
			}
			if f.Pasting {
				kit.Kbd(c.Key("paste"), tokens, pasteLegend(), false).Absolute().Left(f.Pointer.X + 14).Top(f.Pointer.Y + 18).
					Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.8}, Exit: &ui.Motion{Scale: 0.9}})
			}
			pointer(c, tokens, f)
		})
	})
}

// promptCard is the feedback window's prompt with its copy button.
func promptCard(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, prompt string, f frame) {
	card := ui.Column(c.Key("prompt-card")).Absolute().Left(30).Top(180).Size(280, 140).Padding(14).Gap(8).
		Radius(theme.RadiusXL).Background(tokens.Card).Border(1, tokens.CardBorder).
		Transition(ui.ElementTransition{Animation: &theme.Page, Enter: &ui.Motion{Y: 10}, Exit: &ui.Motion{Scale: 0.96}})
	kit.Raise(card, tokens, kit.Floating).Children(func() {
		ui.Text(c, messages.Text(copy.FeedbackPromptLabel)).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted)
		ui.Text(c, prompt).Font(typography.Mono).FontSize(10.5).LineHeight(1.4).MaxLines(3).Ellipsis("…")
		// Placed rather than laid out, so the pointer lands on it.
		label, icon := messages.Text(copy.Copy), icons.Copy
		if f.Copied {
			label, icon = messages.Text(copy.Copied), icons.Check
		}
		ui.Row(c).Absolute().Left(188).Top(98).Size(78, 28).Gap(5).Radius(theme.RadiusControl).Justify(ui.Center).AlignItems(ui.Center).
			Background(tokens.Accent).TextColor(tokens.AccentText).Children(func() {
			ui.Row(c.Key(label)).Gap(5).AlignItems(ui.Center).Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.9}}).Children(func() {
				ui.Icon(c, icon).FontSize(12)
				ui.Text(c, label).FontSize(theme.Caption).FontWeight(600).SingleLine()
			})
		})
	})
}

// agent is the coding agent's window: its conversation above the input.
func agent(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, prompt string, f frame) {
	window := ui.Column(c.Key("agent")).Absolute().Left(20).Top(110).Size(300, 280).Radius(theme.RadiusLG+2).
		Background(tokens.Card).Border(1, tokens.CardBorder).Clip().
		Transition(ui.ElementTransition{Animation: &theme.Page, Enter: &ui.Motion{Scale: 0.96}})
	kit.Raise(window, tokens, kit.Floating).Children(func() {
		ui.Row(c).Height(30).Padding(0, 10).Gap(6).AlignItems(ui.Center).Children(func() {
			for i := 0; i < 3; i++ {
				ui.Box(c.Key(i)).Size(7, 7).Radius(3.5).Background(tokens.CardBorder)
			}
			ui.Text(c, config.Agent).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted).Margin(0, 0, 0, 82)
		})
		ui.Box(c).Height(1).Background(tokens.Hairline)
		if f.Sent {
			ui.Text(c.Key("asked"), prompt).Absolute().Right(12).Top(44).Width(196).Padding(8, 10).Radius(theme.RadiusLG).
				Background(tokens.Subtle).FontSize(10.5).LineHeight(1.35).MaxLines(3).Ellipsis("…").
				Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Y: 8}})
		}
		if reply := head(messages.Text(copy.FeedbackDemoReply), f.Reply); reply != "" {
			ui.Text(c, reply).Absolute().Left(12).Top(104).Width(260).FontSize(theme.Caption).MaxLines(2)
		}
		if f.Issue {
			opened(c, tokens, messages)
		}
		field := ui.Column(c).Absolute().Left(12).Top(208).Size(276, 60).Padding(8, 10).Radius(theme.RadiusXL).
			Background(tokens.Canvas).Border(1, tokens.CardBorder)
		if f.Focused {
			field.Border(1, tokens.AccentInk)
		}
		field.Children(func() {
			if f.Pasted {
				ui.Text(c.Key("pasted"), prompt).FontSize(10.5).LineHeight(1.35).MaxLines(2).Ellipsis("…").Width(224).
					Transition(ui.ElementTransition{Animation: &theme.Quick, Enter: &ui.Motion{}})
			} else {
				ui.Text(c, messages.FeedbackDemoAsk(config.Agent)).FontSize(theme.Caption).TextColor(tokens.Faint).SingleLine()
			}
			ink := tokens.Subtle
			if f.Pasted {
				ink = tokens.Text
			}
			ui.Row(c).Absolute().Right(10).Bottom(10).Size(24, 24).Radius(12).Justify(ui.Center).AlignItems(ui.Center).
				Background(ink).TextColor(tokens.Card).Children(func() { ui.Icon(c, icons.ArrowUp).FontSize(12) })
		})
	})
}

// opened is the GitHub issue the agent opened.
func opened(c *ui.Context, tokens theme.Tokens, messages copy.Catalog) {
	card := ui.Row(c.Key("issue")).Absolute().Left(12).Top(132).Size(276, 64).Padding(10, 12).Gap(10).AlignItems(ui.Center).
		Radius(theme.RadiusLG).Background(tokens.Card).Border(1, tokens.CardBorder).
		Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: 10, Scale: 0.96}})
	kit.Raise(card, tokens, kit.Lifted).Children(func() {
		ui.Icon(c, icons.GitHub).FontSize(20).TextColor(tokens.Text)
		ui.Column(c).Gap(5).Shrink(1).Children(func() {
			ui.Row(c).Height(18).Padding(0, 7).Gap(4).Radius(9).AlignItems(ui.Center).AlignSelf(ui.Start).
				Background(tokens.Success).TextColor(tokens.OnFill).Children(func() {
				ui.Icon(c, icons.CircleCheck).FontSize(10)
				ui.Text(c, messages.Text(copy.FeedbackDemoOpened)).FontSize(10).FontWeight(600).SingleLine()
			})
			ui.Text(c, messages.Text(copy.FeedbackDemoIssue)).FontSize(theme.Caption).FontWeight(600).SingleLine().Ellipsis("…")
		})
	})
}

// pointer draws the arrow and the ring a click leaves.
func pointer(c *ui.Context, tokens theme.Tokens, f frame) {
	if f.Ripple >= 0 {
		size := 14 + 22*f.Ripple
		ui.Box(c).Absolute().Left(f.RippleAt.X-size/2).Top(f.RippleAt.Y-size/2).Size(size, size).Radius(size/2).
			Border(2, tokens.AccentInk).Opacity(1 - f.Ripple).PassThrough()
	}
	scale := 1 - 0.15*f.Press
	// The arrow's tip sits at (4, 2) of its 20 DIP box.
	ui.Image(c, icons.Pointer).Absolute().Left(f.Pointer.X-4).Top(f.Pointer.Y-2).Size(20, 20).
		TransformOrigin(0.2, 0.1).Scale(scale, scale).PassThrough()
}

// head returns the first fraction of text's runes.
func head(text string, fraction float32) string {
	n := int(float32(utf8.RuneCountInString(text)) * fraction)
	return string([]rune(text)[:n])
}
