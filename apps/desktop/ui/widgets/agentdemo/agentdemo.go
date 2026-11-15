// Package agentdemo illustrates reporting a problem through a coding
// agent, as a screen recording would: the pointer copies the prompt,
// pastes it into the agent and sends it, and the agent opens a GitHub
// issue, while the camera zooms toward wherever the pointer works. It is
// an illustration drawn from fixed copy and the prompt it is given.
package agentdemo

import (
	goruntime "runtime"
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/screencast"
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

type (
	point = screencast.Point
	lens  = screencast.Lens
)

var (
	pt   = screencast.Pt
	view = screencast.View
)

// key holds value from ms milliseconds into the loop.
func key[T any](ms float32, value T) screencast.Key[T] { return screencast.At(ms, value) }

// Where things are on the stage: the copy button of the prompt card, the
// agent's input and send button, and the issue the agent opens.
var (
	center     = pt(Width/2, Height/2)
	rest       = pt(300, 440)
	copyButton = pt(257, 292)
	input      = pt(150, 350)
	send       = pt(286, 356)
	issue      = pt(170, 274)
)

// The pointer's path and the camera's moves, in milliseconds. Two keys
// with one value hold it still between them.
var (
	pointerKeys = []screencast.Key[point]{
		key(0, rest), key(500, rest), key(1300, copyButton), key(2500, copyButton), key(3200, input),
		key(3900, input), key(4500, send), key(4800, send), key(5500, rest),
	}
	lensKeys = []screencast.Key[lens]{
		key(0, view(1, center)), key(400, view(1, center)), key(1300, view(1.55, pt(250, 288))),
		key(1800, view(1.55, pt(250, 288))), key(2400, view(1, center)), key(2600, view(1, center)),
		key(3300, view(1.22, pt(170, 345))), key(4000, view(1.22, pt(170, 345))), key(4500, view(1.22, pt(200, 345))),
		key(4800, view(1.22, pt(200, 345))), key(5400, view(1.12, pt(170, 240))), key(6200, view(1.12, pt(170, 240))),
		key(6900, view(1.2, issue)), key(8400, view(1.2, issue)), key(9200, view(1, center)),
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
)

var clicks = []time.Duration{copyClick, inputClick, sendClick}

// frame is the whole scene at one moment.
type frame struct {
	Agent, Copied, Focused, Pasting, Pasted, Sent, Issue bool
	// Reply is how much of the agent's answer has streamed in.
	Reply  float32
	Cursor screencast.Cursor
	Lens   lens
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
		Cursor:  screencast.CursorAt(pointerKeys, clicks, t),
		Lens:    screencast.Sample(lensKeys, t, screencast.MixLens),
	}
	switch {
	case t >= replyEnd:
		f.Reply = 1
	case t > replyStart:
		f.Reply = float32(t-replyStart) / float32(replyEnd-replyStart)
	}
	return f
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
		f.Lens = view(1, center)
		render(c, tokens, messages, prompt, f)
		return
	}
	c.AnimationFrame()
	render(c, tokens, messages, prompt, frameAt(time.Duration(c.Now().UnixNano())))
}

// render draws the scene of f.
func render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, prompt string, f frame) {
	ui.Column(c).Fill().Clip().Background(tokens.Scrim).Role(ui.RoleImage).Label(messages.Text(copy.FeedbackDemoLabel)).Children(func() {
		screencast.Stage(c, Width, Height, f.Lens, func() {
			ui.Image(c, assets.Backdrop).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Cover)
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(96).PassThrough().
				LinearGradient(ui.LinearGradient{From: tokens.Scrim.Alpha(0.4), To: tokens.Scrim.Alpha(0), Angle: 180})
			if f.Agent {
				agent(c, tokens, messages, prompt, f)
			} else {
				promptCard(c, tokens, messages, prompt, f)
			}
			if f.Pasting {
				kit.Kbd(c.Key("paste"), tokens, pasteLegend(), false).Absolute().Left(f.Cursor.At.X + 14).Top(f.Cursor.At.Y + 18).
					Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.8}, Exit: &ui.Motion{Scale: 0.9}})
			}
			screencast.Pointer(c, tokens, f.Cursor)
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
		if reply := screencast.Head(messages.Text(copy.FeedbackDemoReply), f.Reply); reply != "" {
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
