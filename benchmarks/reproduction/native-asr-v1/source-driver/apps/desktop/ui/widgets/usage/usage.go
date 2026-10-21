// Package usage is the overview of the user's dictation: totals, a heatmap
// of words per day, where polishing went and the vocabulary, drawn from the
// per-day counts only (005).
package usage

import (
	"cmp"
	"slices"
	"time"

	"arion.local/desktop/ui/features/scene"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Range is the period a breakdown covers.
type Range int

const (
	Today Range = iota
	Week
	All
)

var ranges = []Range{Today, Week, All}

func (r Range) label() copy.Key {
	return [...]copy.Key{copy.UsageRangeToday, copy.UsageRangeWeek, copy.UsageRangeAll}[r]
}

// start is the first moment of the range ending now; zero for all time.
func (r Range) start(now time.Time) time.Time {
	switch r {
	case Today:
		return now
	case Week:
		return now.AddDate(0, 0, -(weekDays - 1))
	}
	return time.Time{}
}

// State keeps the overview's own choices.
type State struct {
	Scenes Range
	// played is set once the entrance has started.
	played bool
	rolls  kit.Rolls
}

// Replay plays the entrance again the next time the overview is drawn, as
// when it is entered.
func (s *State) Replay() { s.played = false }

// The entrance: cards rise into place one after another, figures roll up
// as their cards land, scene bars grow and the heatmap lights up last.
var (
	cardRise    = ui.SpringOptions{Duration: 520 * time.Millisecond}
	cardStagger = 45 * time.Millisecond
	// figureLead lets a card start rising before its figure rolls.
	figureLead = 90 * time.Millisecond
	barStagger = 50 * time.Millisecond
)

// arrive moves a card into place: on the entering frame it starts 8 DIPs
// low and transparent, then rises after the cards before it. With Reduce
// Motion MyGo only fades it, and every card fades at once.
func arrive(c *ui.Context, card ui.Element, entering bool, order int) ui.Element {
	pose := ui.Pose{}
	if entering {
		pose = ui.Pose{Y: 8, Fade: 1}
	}
	delay := time.Duration(order) * cardStagger
	if c.Preferences().ReduceMotion {
		delay = 0
	}
	return card.AnimatePose(pose, ui.Animation{Spring: cardRise, Delay: delay})
}

// figure draws a stat's value on wheels that roll with its card.
func figure(c *ui.Context, tokens theme.Tokens, state *State, key, value string, entering bool, order int) func() {
	return func() {
		kit.Figure(c, &state.rolls, key, value, kit.FigureStyle{Size: theme.Display, Weight: 600, Color: tokens.Text}, entering,
			time.Duration(order)*cardStagger+figureLead)
	}
}

const (
	weekDays    = 7
	recentTerms = 5
	gap         = 12
)

// Overview lays the cards out in pairs over the window's content column.
// editVocabulary opens the vocabulary editor.
func Overview(c *ui.Context, store runtime.Store, tokens theme.Tokens, state *State, editVocabulary func()) {
	messages := store.Copy()
	now := time.Now()
	days := store.State.Usage
	total := days.Since(time.Time{})
	entering := !state.played
	state.played = true
	speech, speechUnit := messages.Duration(total.SpeechMs)
	stats := []struct {
		key         string
		icon        *ui.SVG
		label, unit copy.Key
		value       string
	}{
		{"speech", icons.Clock, copy.UsageSpeechTime, "", speech},
		{"words", icons.Type, copy.UsageWords, copy.UnitWords, messages.Count(total.Words)},
		{"polished", icons.Sparkles, copy.UsagePolished, copy.UnitWords, messages.Count(total.PolishedWords())},
		{"speed", icons.Gauge, copy.UsageSpeed, copy.UnitWordsPerMinute, messages.Count(total.WordsPerMinute())},
	}
	stat := func(i int) func() ui.Element {
		return func() ui.Element {
			s := stats[i]
			unit := speechUnit
			if s.unit != "" {
				unit = messages.Text(s.unit)
			}
			return arrive(c, kit.Stat(c, tokens, s.icon, messages.Text(s.label), unit, figure(c, tokens, state, s.key, s.value, entering, i)), entering, i)
		}
	}
	ui.Column(c).Gap(gap).Children(func() {
		pair(c, stat(0), stat(1))
		pair(c, stat(2), stat(3))
		arrive(c, activity(c, tokens, messages, days, now, entering), entering, len(stats))
		pair(c, func() ui.Element {
			return arrive(c, scenes(c, tokens, messages, days.Since(state.Scenes.start(now)), &state.Scenes, entering, len(stats)+1), entering, len(stats)+1)
		}, func() ui.Element {
			return arrive(c, vocabulary(c, tokens, messages, store.State.Vocabulary, editVocabulary), entering, len(stats)+2)
		})
	})
}

// pair sets two cards side by side at equal widths and heights.
func pair(c *ui.Context, left, right func() ui.Element) {
	ui.Row(c).Gap(gap).AlignItems(ui.Stretch).Children(func() {
		left().Grow(1).Basis(0).MinWidth(0)
		right().Grow(1).Basis(0).MinWidth(0)
	})
}

type share struct {
	key   string
	icon  *ui.SVG
	label string
	words int
}

// scenes breaks the period's words down by the scene polishing laid them
// out for, the most used first, with the words left as recognized last.
func scenes(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, total runtime.UsageDay, period *Range, entering bool, order int) ui.Element {
	card := kit.Card(c, tokens)
	card.Children(func() {
		kit.CardHeader(c, tokens, messages.Text(copy.UsageScenes), func() { rangeSelect(c, tokens, messages, period) })
		if total.Words == 0 {
			kit.Empty(c, tokens, messages.Text(copy.UsageScenesEmpty))
			return
		}
		var shares []share
		for s, words := range total.Scenes {
			shares = append(shares, share{string(s), scene.Icon(s), messages.Text(scene.Label(s)), words})
		}
		slices.SortFunc(shares, func(a, b share) int {
			if a.words != b.words {
				return b.words - a.words
			}
			return cmp.Compare(a.key, b.key)
		})
		if plain := total.Words - total.PolishedWords(); plain > 0 {
			shares = append(shares, share{"plain", icons.Type, messages.Text(copy.UsageUnpolished), plain})
		}
		ui.Column(c).Gap(14).Padding(4, 16, 16, 16).Children(func() {
			// Bars grow once their card has landed, one after another.
			for i, s := range shares {
				delay := time.Duration(order)*cardStagger + figureLead + time.Duration(i)*barStagger
				shareRow(c.Key(s.key), tokens, messages, s, float32(s.words)/float32(total.Words), entering, delay)
			}
		})
	})
	return card
}

// shareRow names a scene over a bar of its part of the period's words.
func shareRow(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, s share, part float32, entering bool, delay time.Duration) {
	ui.Column(c).Gap(6).Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}).
		Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, s.icon).FontSize(13).TextColor(tokens.Muted)
				ui.Text(c, s.label).FontSize(theme.Body).SingleLine().Grow(1).Shrink(1)
				ui.Text(c, messages.Count(s.words)).FontSize(theme.Caption).FontWeight(500).FontFeatures("tnum").TextColor(tokens.Muted).SingleLine().
					ContentTransition(ui.ContentTransition{Numeric: true})
			})
			bar := ui.Box(c).Height(6)
			if entering {
				bar.Jump("part", 0)
			}
			width := bar.AnimateTo("part", part, ui.Animation{Spring: theme.Settle.Spring, Delay: delay})
			bar.Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(r, tokens.Hairline, r.H/2)
				if width > 0 {
					p.Fill(ui.Rect{X: r.X, Y: r.Y, W: max(r.W*width, r.H), H: r.H}, tokens.Accent, r.H/2)
				}
			})
		})
}

// rangeSelect chooses the period of a breakdown.
func rangeSelect(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, period *Range) {
	chosen := *period
	sel := ui.SelectBase(c.Key("range"), period)
	kit.SelectTrigger(c, tokens, sel, messages.Text(copy.UsageRange), messages.Text(period.label()))
	kit.SelectPopup(tokens, sel, func() {
		for _, r := range ranges {
			kit.SelectItem(c, tokens, sel, r, messages.Text(r.label()), r == chosen)
		}
	})
}

// vocabulary shows the latest approved replacements and opens the editor.
func vocabulary(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, entries []runtime.Replacement, edit func()) ui.Element {
	card := kit.Card(c, tokens)
	card.Children(func() {
		kit.CardHeader(c, tokens, messages.Text(copy.NavVocabulary), func() {
			if kit.Button(c.Key("edit"), tokens, kit.Ghost, messages.Text(copy.UsageVocabEdit), true).Height(28).Clicked() {
				edit()
			}
		})
		if len(entries) == 0 {
			kit.Empty(c, tokens, messages.Text(copy.VocabEmpty))
			return
		}
		ui.Column(c).Gap(10).Padding(4, 16, 16, 16).Children(func() {
			for i := len(entries) - 1; i >= max(len(entries)-recentTerms, 0); i-- {
				entry := entries[i]
				ui.Row(c.Key(entry.ID)).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, entry.From).FontSize(theme.Body).TextColor(tokens.Muted).SingleLine().Shrink(1)
					ui.Icon(c, icons.ArrowRight).FontSize(12).TextColor(tokens.Faint)
					ui.Text(c, entry.To).FontSize(theme.Body).FontWeight(500).SingleLine().Shrink(1)
				})
			}
		})
	})
	return card
}
