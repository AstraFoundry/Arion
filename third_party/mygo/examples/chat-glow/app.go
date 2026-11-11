package main

import (
	"math"
	"math/rand/v2"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

const (
	// thinkTime is how long the assistant thinks before it writes, in
	// seconds, and wordsPerSecond how fast it writes.
	thinkTime, wordsPerSecond = 1.6, 28
	// The glow comes up over fadeIn seconds and goes over fadeOut; the
	// orb's liveliness follows the work over liven.
	fadeIn, fadeOut, liven = 0.35, 0.9, 0.5
	// The composer's ring: Pad DIPs of glow around the field, whose
	// corners are fieldRadius.
	ringPadding, fieldRadius = 12, 16
	// maxWidth is the conversation's width at most.
	maxWidth = 660
	// jumpWords is how many words coming in one frame make the
	// conversation jump to its end rather than glide there: a reply
	// writes a word or so a frame.
	jumpWords = 12
	// replySize is the size of the replies' text and replyLeading its line
	// height, which "Thinking…" takes too, so that the row keeps its
	// height as the first words replace it.
	replySize, replyLeading = 15, 1.5
)

// suggestions are what the empty conversation offers to ask.
var suggestions = []string{"Summarize my week", "Draft a reply to the team", "Plan three days in Kyoto"}

// message is one of the conversation's: the user's, or the assistant's,
// of which shown words show so far.
type message struct {
	id    int
	mine  bool
	text  string
	words []string
	shown int
}

// app is the window's state.
type app struct {
	messages []*message
	input    string
	list     ui.ListState
	// seen is how many words of the replies the last frame showed.
	seen    int
	next    int
	replies int
	// The assistant's work, by the conversation's clock (now, in
	// seconds): when the last message was sent (asked), when the
	// assistant began to write (writing, 0 before), and whether it works.
	now, asked, writing float64
	busy                bool
	last                time.Time
	// glow is how much the window's edges glow, from 0 to 1, energy how
	// lively the orb is, and orbPhase how far its colors have moved.
	glow, energy, orbPhase float64
	// still stops what moves on its own, while the desktop asks for less
	// motion.
	still bool
	// stops are the edge glow's colors around the window, moving about,
	// and gpu tells that the last frame drew on a GPU, where it glows.
	stops stops
	gpu   bool
	sh    shaders
}

func newApp() *app {
	return &app{sh: newShaders(), stops: newStops(rand.Float64)}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.still = c.Preferences().ReduceMotion
	a.tick(c.Now())
	if a.moving() {
		c.AnimationFrame()
	}
	if c.Shortcut(0, ui.KeyEscape) && a.busy {
		a.stop()
	}

	root := ui.Column(c).Fill().Background(t.Background)
	// Whether frames draw on a GPU, for the next frames' glow.
	root.Draw(func(p *ui.Painter, r ui.Rect) { a.gpu = p.Accelerated() })
	root.Children(func() {
		if len(a.messages) == 0 {
			a.welcome(c)
		} else {
			a.conversation(c)
		}
		a.composer(c)
		// The title bar, under the window controls, drags the window.
		bar := c.TitleBar()
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(max(bar.Height, 32)).DragWindow()
		// The edges' glow, over everything, which the pointer goes through.
		a.edge(c)
	})
}

// welcome builds the empty conversation: the orb, a greeting, and
// suggestions to ask.
func (a *app) welcome(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).AlignItems(ui.Center).Justify(ui.Center).Gap(10).Padding(40, 24, 0, 24).Children(func() {
		ui.Box(c).Size(150, 150).Draw(a.drawOrb).Label("Assistant")
		ui.Text(c, "How can I help?").FontSize(28).FontWeight(600).TextColor(t.Text).Margin(4, 0, 0, 0)
		ui.Text(c, "Ask anything, or start with one of these.").FontSize(14).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Wrap().Justify(ui.Center).Margin(18, 0, 0, 0).Children(func() {
			for _, s := range suggestions {
				chip := ui.Box(c.Key(s)).Padding(9, 14).Radius(999).Border(1, t.Border).Label(s).Cursor(ui.CursorPointer)
				if chip.Hovered() {
					chip.Background(t.SurfaceHover)
				}
				chip.Children(func() { ui.Text(c, s).FontSize(13).TextColor(t.Text) })
				if chip.Clicked() {
					a.send(s)
				}
			}
		})
	})
}

// conversation builds the messages, following the end as replies grow.
func (a *app) conversation(c *ui.Context) {
	a.list.Key = func(i int) any { return a.messages[i].id }
	a.list.FollowEnd = true
	// Glide up as the reply grows, but jump to the end when much came at
	// once, as when the window shows again after it was hidden.
	a.list.FollowAnimation = &ui.Smooth
	shown := 0
	for _, m := range a.messages {
		shown += m.shown
	}
	if shown-a.seen > jumpWords {
		a.list.FollowAnimation = nil
	}
	a.seen = shown
	ui.List(c, &a.list, len(a.messages), func(i int) {
		// Each message in the middle, the list as wide as the window, so
		// that it scrolls wherever the pointer is.
		ui.Row(c).Justify(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MaxWidth(maxWidth).Children(func() { a.message(c, a.messages[i]) })
		})
	}).Grow(1).Padding(52, 24, 16, 24).Gap(22).Justify(ui.End)
}

// message builds a message: the user's in a bubble on the right, the
// assistant's beside its orb, thinking until it writes.
func (a *app) message(c *ui.Context, m *message) {
	t := c.Theme()
	if m.mine {
		ui.Row(c).Children(func() {
			ui.Spacer(c)
			ui.Text(c, m.text).Selectable().Padding(9, 14).Radius(18).Background(t.Surface).MaxWidth(460)
		})
		return
	}
	last := m == a.messages[len(a.messages)-1]
	ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
		avatar := ui.Box(c).Size(30, 30).Margin(-3, 0, 0, 0)
		if last {
			avatar.Draw(a.drawOrb)
		} else {
			avatar.Draw(func(p *ui.Painter, r ui.Rect) { a.drawStillOrb(p, r, m.id) })
		}
		if m.shown == 0 && last && a.busy {
			ui.Box(c).Size(120, replySize*replyLeading).Label("Thinking").Draw(func(p *ui.Painter, r ui.Rect) { a.drawThinking(p, r, t) })
			return
		}
		ui.Text(c, strings.Join(m.words[:m.shown], "")).Selectable().FontSize(replySize).LineHeight(replyLeading).TextColor(t.Text).Grow(1).MinWidth(0)
	})
}

// composer builds the field to write in, in its glowing ring, and the
// button that sends, or stops the assistant.
func (a *app) composer(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Justify(ui.Center).Padding(0, 24-ringPadding, 22-ringPadding, 24-ringPadding).Children(func() {
		box := ui.Row(c).Grow(1).MaxWidth(maxWidth + 2*ringPadding).Padding(ringPadding).Draw(a.drawRing)
		box.Children(func() {
			ui.Row(c).Grow(1).AlignItems(ui.Center).Gap(8).Padding(6, 6, 6, 16).Radius(fieldRadius).Background(t.Surface).Children(func() {
				in := ui.TextInput(c, &a.input).Placeholder("Ask anything").Grow(1).AutoFocus().
					Background(ui.Color{}).Border(0, ui.Color{}).Padding(8, 0).Label("Message")
				ready := strings.TrimSpace(a.input) != ""
				label := "Send"
				if a.busy {
					label = "Stop"
				}
				send := ui.Box(c).Size(34, 34).Radius(17).Label(label).Cursor(ui.CursorPointer)
				switch {
				case a.busy || ready:
					send.Background(t.Text)
				default:
					send.Background(t.Border)
				}
				send.Draw(func(p *ui.Painter, r ui.Rect) { drawSendIcon(p, r, a.busy, t.Background) })
				if a.busy {
					if send.Clicked() {
						a.stop()
					}
				} else if (send.Clicked() || in.Submitted()) && ready {
					a.send(a.input)
				}
			})
		})
	})
}

// drawSendIcon draws the button's icon: an arrow up, or a square while
// the assistant works.
func drawSendIcon(p *ui.Painter, r ui.Rect, busy bool, c ui.Color) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	if busy {
		p.Fill(ui.Rect{X: cx - 5, Y: cy - 5, W: 10, H: 10}, c, 2)
		return
	}
	var arrow ui.Path
	arrow.MoveTo(cx, cy+7).LineTo(cx, cy-7).MoveTo(cx-6, cy-1).LineTo(cx, cy-7).LineTo(cx+6, cy-1)
	p.StrokePath(&arrow, 2.2, c)
}

// glowLayers are the edge glow's layers, as the SwiftUI recreation of
// Apple Intelligence's glow draws them (jacobamobin/
// AppleIntelligenceGlowEffect): its outline stroked inside the window's
// edge this wide and blurred this much, by SwiftUI's radius, whose
// standard deviation is 0.95 of it, measured.
var glowLayers = []struct{ width, blur float32 }{{6, 0}, {9, 4}, {11, 12}, {15, 15}}

// edge builds the window's edges glowing while the assistant works: the
// window's outline stroked with a conic gradient of the glow's colors,
// their stops moving about, a crisp line and lines blurred over it, which
// the pointer goes through.
func (a *app) edge(c *ui.Context) {
	// Without a GPU, four layers of the window blurred every frame cost too
	// much: none.
	if a.glow <= 0.001 || !a.gpu {
		return
	}
	corner := float32(10)
	if runtime.GOOS == "darwin" {
		corner = 16
	}
	// From the right, as SwiftUI's AngularGradient goes, blended in Oklab
	// as SwiftUI blends it.
	g := ui.ConicGradient{Stops: a.stops.gradient(a.now), Angle: 90, Oklab: true}
	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Opacity(float32(ease(a.glow))).Children(func() {
		for _, l := range glowLayers {
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Radius(corner).
				Border(l.width, ui.Color{}).BorderConicGradient(g).Blur(0.95 * l.blur)
		}
	})
}

// drawRing draws the composer's ring, faint at rest and bright while the
// assistant works.
func (a *app) drawRing(p *ui.Painter, r ui.Rect) {
	if !p.Accelerated() {
		return
	}
	ring.Draw(p, r, shader.Options{Values: a.sh.ring.
		Float(ringTime, float32(a.now)).Float(ringStrength, float32(0.25+0.75*ease(a.glow))).
		Float(ringPad, ringPadding).Float(ringRadius, fieldRadius)})
}

// drawThinking draws "Thinking…" with a shine sweeping across it.
func (a *app) drawThinking(p *ui.Painter, r ui.Rect, t *ui.Theme) {
	const text = "Thinking…"
	_, h := p.MeasureText(0, ui.Span{Text: text, Size: replySize})
	y := r.Y + (r.H-h)/2
	if !p.Accelerated() {
		p.Text(r.X, y, text, replySize, t.TextMuted)
		return
	}
	base, shine := t.TextMuted, t.Text
	shimmer.Text(p, r.X, y, text, replySize, shader.Options{Values: a.sh.shimmer.
		Float(shimmerTime, float32(a.now)).Color(shimmerBase, base).Color(shimmerShine, shine)})
}

// drawStillOrb draws an earlier reply's orb, at rest, its colors where
// the reply's id leaves them.
func (a *app) drawStillOrb(p *ui.Painter, r ui.Rect, id int) {
	if !p.Accelerated() {
		a.drawOrb(p, r)
		return
	}
	orb.Draw(p, r, shader.Options{Values: a.sh.still.Float(orbPhase, float32(id)*1.7).Float(orbEnergy, 0)})
}

// send asks the assistant text, which it answers after thinking.
func (a *app) send(text string) {
	text = strings.TrimSpace(text)
	a.next++
	a.messages = append(a.messages, &message{id: a.next, mine: true, text: text})
	a.next++
	reply := replies[a.replies%len(replies)]
	a.replies++
	a.messages = append(a.messages, &message{id: a.next, text: reply, words: splitWords(reply)})
	a.input = ""
	a.asked, a.writing, a.busy = a.now, 0, true
}

// stop has the assistant stop where it is.
func (a *app) stop() {
	a.busy = false
	if n := len(a.messages); n > 0 {
		if m := a.messages[n-1]; !m.mine && m.shown == 0 {
			m.words, m.shown = []string{"Stopped."}, 1
		}
	}
}

// tick moves the conversation's clock on to now. The assistant works in
// real time, as a model streams whether the window shows or not: once
// the window shows again after frames paused while it was hidden, the
// clock catches up and the reply shows as far as it came.
func (a *app) tick(now time.Time) {
	dt := 0.0
	if !a.last.IsZero() {
		dt = max(now.Sub(a.last).Seconds(), 0)
	}
	a.last = now
	a.step(dt)
}

// step moves the conversation on by dt seconds: the assistant thinks,
// then writes word by word; the glow and the orb follow its work.
func (a *app) step(dt float64) {
	a.now += dt
	if a.busy {
		m := a.messages[len(a.messages)-1]
		if a.writing == 0 && a.now-a.asked >= thinkTime {
			// It began to write as it stopped thinking, however late the
			// clock tells.
			a.writing = a.asked + thinkTime
		}
		if a.writing > 0 {
			m.shown = min(len(m.words), int((a.now-a.writing)*wordsPerSecond)+1)
			if m.shown == len(m.words) {
				a.busy = false
			}
		}
	}
	if a.busy {
		a.glow = min(1, a.glow+dt/fadeIn)
	} else {
		a.glow = max(0, a.glow-dt/fadeOut)
	}
	goal := 0.0
	if a.busy {
		goal = 1
	}
	a.energy += (goal - a.energy) * (1 - math.Exp(-dt/liven))
	if a.glow > 0 && !a.still {
		a.stops.step(a.now)
	}
	if !a.still {
		a.orbPhase += dt * (0.5 + 1.8*a.energy)
	}
}

// moving reports whether something moves on its own: the assistant's
// work, the glow fading, the orb calming down, or the welcome's orb, which
// drifts unless the desktop asks for less motion.
func (a *app) moving() bool {
	return a.busy || a.glow > 0 || a.energy > 0.01 || (len(a.messages) == 0 && !a.still)
}

// ease is CSS's ease-in-out, for the glow's fades.
func ease(x float64) float64 { return x * x * (3 - 2*x) }

// splitWords splits text into words with the spaces after them, which
// joined give it back.
func splitWords(text string) []string {
	var words []string
	start := 0
	for i := 1; i < len(text); i++ {
		if text[i-1] == ' ' || text[i-1] == '\n' {
			if text[i] != ' ' && text[i] != '\n' {
				words = append(words, text[start:i])
				start = i
			}
		}
	}
	return append(words, text[start:])
}

var replies = []string{
	"Here's your week at a glance.\n\nYou shipped the new onboarding flow on Tuesday, and sign-ups are up 18% since. Thursday's design review settled the navigation question: tabs stay, the sidebar goes. Two things still need you: the Q3 budget draft is due Monday, and Priya is waiting on your feedback for the pricing page.",
	"Here's a draft:\n\nHi all, thanks for pushing through this sprint. The release is on track for Friday; the remaining risk is the migration, which Sam is testing tonight. If anything slips, we'll cut the export feature and ship it the week after. Let's keep tomorrow's stand-up to ten minutes.",
	"Day 1: Higashiyama. Start at Kiyomizu-dera early, walk down Sannenzaka to Gion, and end with dinner in Pontocho.\n\nDay 2: Arashiyama. The bamboo grove at sunrise, Tenryu-ji's garden, then the river by boat.\n\nDay 3: Fushimi Inari before the crowds, sake tasting in Fushimi, and a last evening at Nishiki Market.",
}
