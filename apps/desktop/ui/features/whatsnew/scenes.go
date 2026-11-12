package whatsnew

import (
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/screencast"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
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

// scene is one slide: its copy and the recording that plays above it.
// Cycle is one loop of the recording; final is the moment that stands for
// it under reduced motion, with its outcome showing.
type scene struct {
	title, body, label copy.Key
	cycle, final       time.Duration
	draw               func(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, t time.Duration, still bool)
}

// scenes maps every slide id of entities/whatsnew's catalog to its scene.
// A slide added to releases.json needs its entry here.
var scenes = map[string]scene{
	"selection-actions": {
		title: copy.WhatsNewSelectionTitle, body: copy.WhatsNewSelectionBody, label: copy.WhatsNewSelectionLabel,
		cycle: askCycle, final: askFinal, draw: drawAsk,
	},
	"selection-vocabulary": {
		title: copy.WhatsNewVocabularyTitle, body: copy.WhatsNewVocabularyBody, label: copy.WhatsNewVocabularyLabel,
		cycle: termCycle, final: termFinal, draw: drawTerm,
	},
}

// play draws the scene's recording, which began at since, on the stage.
// The recording is decoration: with reduced motion it holds its final
// frame, unzoomed.
func (s scene) play(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, since time.Time) {
	if c.Preferences().ReduceMotion {
		s.draw(c, tokens, messages, s.final, true)
		return
	}
	c.AnimationFrame()
	s.draw(c, tokens, messages, c.Now().Sub(since)%s.cycle, false)
}

var center = pt(Width/2, StageHeight/2)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// The stage both selection scenes share: another app's document with one
// phrase in its text, and Arion's bar under the selection. Everything the
// pointer lands on is placed, not laid out.
const (
	docLeft, docTop     = 50, 22
	docWidth, docHeight = 460, 150
	// lineTop is the phrase's line, from the stage's top; lineHeight is its
	// height.
	lineTop    = docTop + 76
	lineHeight = 18

	barLeft   = 80
	barTop    = lineTop + lineHeight + 8
	barHeight = 37
	barPad    = 5
	// The bar's buttons have fixed widths up to Add to vocabulary, so the
	// pointer finds each in either language.
	claudeButton = 94
	chatButton   = 110
	dividerSpan  = 7
	copyButton   = 66
	toolHeight   = barHeight - 2*barPad

	menuTop   = barTop + barHeight + 4
	menuWidth = 224
	menuRow   = 26
	menuPad   = 4
	menuGap   = 1
)

var (
	rest       = pt(280, 200)
	barMiddle  = float32(barTop) + barHeight/2
	claudeAt   = pt(barLeft+barPad+claudeButton/2, barMiddle)
	explainAt  = pt(barLeft+70, menuTop+menuPad+menuRow+menuGap+menuRow/2)
	vocabStart = float32(barLeft + barPad + claudeButton + chatButton + dividerSpan + copyButton)
	vocabAt    = pt(vocabStart+22, barMiddle)
)

// desk is where a scene's phrase sits in the document and how wide it is
// taken to be: the pointer drags across that width.
type desk struct {
	phrase copy.Key
	left   float32
	width  float32
}

func (d desk) start() point { return pt(d.left-1, lineTop+lineHeight/2+1) }
func (d desk) end() point   { return pt(d.left+d.width+2, lineTop+lineHeight/2+1) }

// document draws the other app's window. selected is how much of the
// phrase the drag has covered.
func document(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, d desk, selected float32) {
	window := ui.Column(c).Absolute().Left(docLeft).Top(docTop).Size(docWidth, docHeight).Radius(theme.RadiusLG+2).
		Background(tokens.Card).Border(1, tokens.CardBorder).Clip()
	kit.Raise(window, tokens, kit.Floating).Children(func() {
		ui.Row(c).Height(26).Padding(0, 10).Gap(6).AlignItems(ui.Center).Children(func() {
			for i := range 3 {
				ui.Box(c.Key(i)).Size(7, 7).Radius(3.5).Background(tokens.CardBorder)
			}
		})
		ui.Box(c).Height(1).Background(tokens.Hairline)
		rule := func(left, top, width float32) {
			ui.Box(c).Absolute().Left(left).Top(top).Size(width, 6).Radius(3).Background(tokens.Hairline)
		}
		line := float32(lineTop - docTop)
		middle := line + lineHeight/2 - 3
		left := d.left - docLeft
		rule(20, line-32, 330)
		rule(20, line-16, 270)
		rule(20, middle, left-28)
		rule(left+d.width+10, middle, docWidth-20-(left+d.width+10)-40)
		rule(20, line+lineHeight+10, 300)
		rule(20, line+lineHeight+26, 190)
		phrase := messages.Text(d.phrase)
		ui.Text(c, phrase).Absolute().Left(left).Top(line).Height(lineHeight).FontSize(theme.Body).SingleLine()
		if selected > 0 {
			// The selection is the phrase again on the highlight, clipped to
			// how far the drag has come.
			cover := ui.Row(c).Absolute().Left(left - 2).Top(line - 1).Height(lineHeight + 2).Clip().Radius(3)
			if selected < 1 {
				cover.Width((d.width + 4) * selected)
			}
			cover.Children(func() {
				ui.Text(c, phrase).Padding(1, 2).Height(lineHeight + 2).FontSize(theme.Body).SingleLine().Shrink(0).
					Background(tokens.AccentSoft).TextColor(tokens.Text)
			})
		}
	})
}

// bar is the selection bar as the real one draws it: the assistants, copy
// and vocabulary, then the X. Hot is the button under the pointer.
func bar(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, hot string) {
	surface := ui.Row(c.Key("bar")).Absolute().Left(barLeft).Top(barTop).Height(barHeight).Padding(barPad-1, barPad-1).AlignItems(ui.Center).
		Radius(theme.RadiusLG+2).Background(tokens.PillSurface).Border(1, tokens.PillBorder).TextColor(tokens.PillText).
		TransformOrigin(0.2, 0).Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: -4, Scale: 0.94}, Exit: &ui.Motion{Scale: 0.97}})
	kit.Raise(surface, tokens, kit.Lifted).Children(func() {
		button := func(id string, width float32, content func()) {
			b := ui.Row(c.Key(id)).Height(toolHeight).Padding(0, 8).Gap(5).Radius(theme.RadiusMD).AlignItems(ui.Center).Justify(ui.Center).Shrink(0)
			if width > 0 {
				b.Width(width)
			}
			if hot == id {
				b.Background(tokens.PillText.Alpha(0.14))
			}
			b.Children(content)
		}
		label := func(key copy.Key) {
			ui.Text(c, messages.Text(key)).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
		}
		assistant := func(id string, width float32, icon *ui.SVG, name copy.Key) {
			button(id, width, func() {
				ui.Icon(c, icon).FontSize(14).TextColor(tokens.PillText)
				label(name)
				ui.Icon(c, icons.ChevronDown).FontSize(12).TextColor(tokens.PillMuted)
			})
		}
		divider := func() {
			ui.Box(c).Width(1).Height(16).Margin(0, (dividerSpan-1)/2).Shrink(0).Background(tokens.PillBorder)
		}
		assistant("claude", claudeButton, icons.Anthropic, copy.SelectionClaude)
		assistant("chatgpt", chatButton, icons.OpenAI, copy.SelectionChatGPT)
		divider()
		button("copy", copyButton, func() {
			ui.Icon(c, icons.Copy).FontSize(14).TextColor(tokens.PillMuted)
			label(copy.Copy)
		})
		button("vocabulary", 0, func() {
			ui.Icon(c, icons.BookOpen).FontSize(14).TextColor(tokens.PillMuted)
			label(copy.SelectionVocabulary)
		})
		divider()
		button("close", toolHeight, func() { ui.Icon(c, icons.X).FontSize(13).TextColor(tokens.PillMuted) })
	})
}

// prompts are the rows of an assistant's menu.
var prompts = []copy.Key{copy.SelectionTranslate, copy.SelectionExplain, copy.SelectionSummarize, copy.SelectionRewrite}

// menu is the prompt menu under an assistant, with the row under the
// pointer highlighted (-1 for none).
func menu(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, hot int) {
	surface := ui.Column(c.Key("menu")).Absolute().Left(barLeft).Top(menuTop).Width(menuWidth).Padding(menuPad).Gap(menuGap).
		Radius(theme.RadiusLG).Background(tokens.PillSurface).Border(1, tokens.PillBorder).TransformOrigin(0, 0).
		Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: -4, Scale: 0.96}, Exit: &ui.Motion{Scale: 0.97}})
	kit.Raise(surface, tokens, kit.Lifted).Children(func() {
		for i, key := range prompts {
			row := ui.Row(c.Key(i)).Height(menuRow).Padding(0, 8).Radius(theme.RadiusMD).AlignItems(ui.Center)
			ink := tokens.PillText
			if i == hot {
				row.Background(tokens.Accent)
				ink = tokens.AccentText
			}
			row.Children(func() {
				ui.Text(c, messages.Text(key)).FontSize(theme.Caption).TextColor(ink).SingleLine()
			})
		}
	})
}

// titleBar is a mock window's bar: its three controls and its name.
func titleBar(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, name string) {
	ui.Row(c).Height(30).Padding(0, 10).Gap(6).AlignItems(ui.Center).Children(func() {
		for i := range 3 {
			ui.Box(c.Key(i)).Size(7, 7).Radius(3.5).Background(tokens.CardBorder)
		}
		ui.Spacer(c)
		if icon != nil {
			ui.Icon(c, icon).FontSize(12).TextColor(tokens.Muted)
		}
		ui.Text(c, name).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted).SingleLine()
		ui.Spacer(c)
		// Balances the controls, so the name sits in the middle.
		ui.Box(c).Width(33)
	})
	ui.Box(c).Height(1).Background(tokens.Hairline)
}

// Scene one: select a phrase, rest on Claude, choose Explain, and Claude
// opens with the prompt filled in.
const (
	askCycle = 9200 * time.Millisecond
	askFinal = 7000 * time.Millisecond

	askPress    = 1300 * time.Millisecond
	askRelease  = 2100 * time.Millisecond
	askBar      = 2250 * time.Millisecond
	askMenu     = 3200 * time.Millisecond
	askChoose   = 4150 * time.Millisecond
	askAnswered = 4400 * time.Millisecond
)

var (
	askDesk = desk{phrase: copy.WhatsNewDemoPhrase, left: 100, width: 194}

	// The assistant's window and, in it, the input holding the prompt.
	chatLeft, chatTop  float32 = 120, 38
	chatWide, chatTall float32 = 320, 224
	chatFocus                  = pt(chatLeft+chatWide/2, chatTop+chatTall/2+10)

	askAside = pt(chatLeft+chatWide/2+40, chatTop+chatTall-18)

	askPointer = []screencast.Key[point]{
		key(0, rest), key(150, rest), key(1300, askDesk.start()), key(2100, askDesk.end()), key(2250, askDesk.end()),
		key(3050, claudeAt), key(3350, claudeAt), key(3950, explainAt), key(4500, explainAt), key(5400, askAside), key(8300, askAside), key(9000, rest),
	}
	askLens = []screencast.Key[lens]{
		key(0, view(1, center)), key(700, view(1, center)), key(1400, view(1.4, pt(200, 112))), key(2300, view(1.4, pt(200, 112))),
		key(3000, view(1.32, pt(200, 170))), key(4200, view(1.32, pt(200, 170))), key(4900, view(1, center)), key(5300, view(1, center)),
		key(6100, view(1.14, chatFocus)), key(8300, view(1.14, chatFocus)), key(9000, view(1, center)),
	}
	askClicks = []time.Duration{askPress, askChoose}
)

// askFrame is scene one at one moment.
type askFrame struct {
	// Selected is how much of the phrase the drag has covered.
	Selected float32
	Bar      bool
	Menu     bool
	// Hot is the bar button the pointer is on, Row the menu row (-1 for
	// none).
	Hot      string
	Row      int
	Answered bool
	Cursor   screencast.Cursor
	Lens     lens
}

func askAt(t time.Duration) askFrame {
	t %= askCycle
	f := askFrame{
		Selected: span(t, askPress, askRelease),
		Bar:      t >= askBar && t < askChoose,
		Menu:     t >= askMenu && t < askChoose,
		Answered: t >= askAnswered,
		Row:      -1,
		Cursor:   screencast.CursorAt(askPointer, askClicks, t),
		Lens:     screencast.Sample(askLens, t, screencast.MixLens),
	}
	if f.Bar && over(f.Cursor.At, claudeAt, claudeButton/2, toolHeight/2) || f.Menu {
		f.Hot = "claude"
	}
	if f.Menu {
		f.Row = rowUnder(f.Cursor.At)
	}
	return f
}

// span is how far t is from from to to, between 0 and 1.
func span(t, from, to time.Duration) float32 {
	return min(max(float32(t-from)/float32(to-from), 0), 1)
}

func over(p, middle point, halfWidth, halfHeight float32) bool {
	return p.X >= middle.X-halfWidth && p.X <= middle.X+halfWidth && p.Y >= middle.Y-halfHeight && p.Y <= middle.Y+halfHeight
}

// rowUnder is the menu row at p, or -1.
func rowUnder(p point) int {
	if p.X < barLeft || p.X > barLeft+menuWidth {
		return -1
	}
	y := p.Y - (menuTop + menuPad)
	if y < 0 {
		return -1
	}
	if row := int(y / (menuRow + menuGap)); row < len(prompts) {
		return row
	}
	return -1
}

func drawAsk(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, t time.Duration, still bool) {
	f := askAt(t)
	if still {
		f.Lens = view(1, center)
	}
	screencast.Stage(c, Width, StageHeight, f.Lens, func() {
		document(c, tokens, messages, askDesk, f.Selected)
		if f.Bar {
			bar(c, tokens, messages, f.Hot)
		}
		if f.Menu {
			menu(c, tokens, messages, f.Row)
		}
		if f.Answered {
			assistant(c, tokens, messages)
		}
		screencast.Pointer(c, tokens, f.Cursor)
	})
}

// assistant is Claude's window, opened with the prompt in its input for
// the person to send.
func assistant(c *ui.Context, tokens theme.Tokens, messages copy.Catalog) {
	window := ui.Column(c.Key("assistant")).Absolute().Left(chatLeft).Top(chatTop).Size(chatWide, chatTall).Radius(theme.RadiusLG+2).
		Background(tokens.Card).Border(1, tokens.CardBorder).Clip().
		Transition(ui.ElementTransition{Animation: &theme.Page, Enter: &ui.Motion{Y: 10, Scale: 0.96}})
	kit.Raise(window, tokens, kit.Floating).Children(func() {
		titleBar(c, tokens, icons.Anthropic, messages.Text(copy.SelectionClaude))
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Anthropic).FontSize(26).TextColor(tokens.Faint)
		})
		field := ui.Column(c).Margin(0, 12, 12, 12).Height(76).Padding(10, 12).Radius(theme.RadiusXL).
			Background(tokens.Canvas).Border(1, tokens.AccentInk)
		field.Children(func() {
			ui.Text(c, messages.Text(copy.WhatsNewDemoPrompt)).FontSize(theme.Caption).LineHeight(1.4).MaxLines(2).Ellipsis("…").Width(240)
			ui.Row(c).Absolute().Right(10).Bottom(10).Size(24, 24).Radius(12).Justify(ui.Center).AlignItems(ui.Center).
				Background(tokens.Text).TextColor(tokens.Card).Children(func() { ui.Icon(c, icons.ArrowUp).FontSize(12) })
		})
	})
}

// Scene two: select a term, choose Add to vocabulary, and the term is in
// Arion's vocabulary.
const (
	termCycle = 8400 * time.Millisecond
	termFinal = 6400 * time.Millisecond

	termPress   = 1300 * time.Millisecond
	termRelease = 1800 * time.Millisecond
	termBar     = 1950 * time.Millisecond
	termChoose  = 3100 * time.Millisecond
	termOpens   = 3400 * time.Millisecond
	termAdded   = 4000 * time.Millisecond
)

var (
	termDesk = desk{phrase: copy.WhatsNewDemoTerm, left: 250, width: 56}

	// Arion's vocabulary window and the row the term lands in.
	vocabLeft, vocabTop     float32 = 130, 40
	vocabWidth, vocabHeight float32 = 300, 214
	termRow                         = pt(vocabLeft+vocabWidth/2, vocabTop+31+14+20)

	termAside = pt(termRow.X+110, termRow.Y+70)

	termPointer = []screencast.Key[point]{
		key(0, rest), key(500, rest), key(1300, termDesk.start()), key(1800, termDesk.end()), key(2100, termDesk.end()),
		key(2800, vocabAt), key(3300, vocabAt), key(4200, termAside), key(7500, termAside), key(8200, rest),
	}
	termLens = []screencast.Key[lens]{
		key(0, view(1, center)), key(700, view(1, center)), key(1400, view(1.4, pt(270, 118))), key(2200, view(1.4, pt(270, 118))),
		key(2800, view(1.3, pt(300, 140))), key(3200, view(1.3, pt(300, 140))), key(3800, view(1, center)), key(4100, view(1, center)),
		key(4900, view(1.3, termRow)), key(7500, view(1.3, termRow)), key(8200, view(1, center)),
	}
	termClicks = []time.Duration{termPress, termChoose}
)

// termFrame is scene two at one moment.
type termFrame struct {
	Selected          float32
	Bar, Hot          bool
	Vocabulary, Added bool
	Cursor            screencast.Cursor
	Lens              lens
}

func termAt(t time.Duration) termFrame {
	t %= termCycle
	f := termFrame{
		Selected:   span(t, termPress, termRelease),
		Bar:        t >= termBar && t < termChoose,
		Vocabulary: t >= termOpens,
		Added:      t >= termAdded,
		Cursor:     screencast.CursorAt(termPointer, termClicks, t),
		Lens:       screencast.Sample(termLens, t, screencast.MixLens),
	}
	f.Hot = f.Bar && f.Cursor.At.X >= vocabStart && over(f.Cursor.At, pt(f.Cursor.At.X, barMiddle), 1, toolHeight/2)
	return f
}

func drawTerm(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, t time.Duration, still bool) {
	f := termAt(t)
	if still {
		f.Lens = view(1, center)
	}
	screencast.Stage(c, Width, StageHeight, f.Lens, func() {
		document(c, tokens, messages, termDesk, f.Selected)
		if f.Bar {
			hot := ""
			if f.Hot {
				hot = "vocabulary"
			}
			bar(c, tokens, messages, hot)
		}
		if f.Vocabulary {
			vocabulary(c, tokens, messages, f.Added)
		}
		screencast.Pointer(c, tokens, f.Cursor)
	})
}

// vocabulary is Arion's vocabulary list, where the term arrives as its
// newest entry.
func vocabulary(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, added bool) {
	window := ui.Column(c.Key("vocabulary")).Absolute().Left(vocabLeft).Top(vocabTop).Size(vocabWidth, vocabHeight).Radius(theme.RadiusLG+2).
		Background(tokens.Canvas).Border(1, tokens.CardBorder).Clip().
		Transition(ui.ElementTransition{Animation: &theme.Page, Enter: &ui.Motion{Y: 10, Scale: 0.96}})
	kit.Raise(window, tokens, kit.Floating).Children(func() {
		titleBar(c, tokens, nil, messages.Text(copy.NavVocabulary))
		ui.Column(c).Margin(14).Radius(theme.RadiusLG).Background(tokens.Card).Border(1, tokens.CardBorder).Clip().Children(func() {
			row := func(content func()) {
				ui.Row(c).Height(40).Padding(0, 14).Gap(10).AlignItems(ui.Center).Children(content)
			}
			row(func() {
				if !added {
					return
				}
				ui.Row(c.Key("term")).Grow(1).Gap(10).AlignItems(ui.Center).
					Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: -6, Scale: 0.96}}).Children(func() {
					ui.Text(c, messages.Text(copy.WhatsNewDemoTerm)).FontSize(theme.Body).FontWeight(500).SingleLine()
					ui.Spacer(c)
					mark := ui.Row(c).Size(18, 18).Radius(9).Background(tokens.Success).Justify(ui.Center).AlignItems(ui.Center)
					mark.TextColor(tokens.OnFill).Children(func() { ui.Icon(c, icons.Check).FontSize(11) })
				})
			})
			for i, width := range []float32{96, 132, 74} {
				ui.Box(c.Key(i)).Height(1).Background(tokens.Hairline)
				row(func() { ui.Box(c).Size(width, 6).Radius(3).Background(tokens.Hairline) })
			}
		})
	})
}
