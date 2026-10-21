package usage

import (
	"math"
	"sort"
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The heatmap is a week per column, Monday on top, ending with the current
// week; it shows as many weeks as fit, up to a year.
const (
	maxWeeks    = 53
	minCell     = 11
	maxPitch    = 18
	cellGap     = 3
	dayLabels   = 30
	monthLabels = 18
	// fallbackWidth is the grid's width in the default window, used for
	// the first frame before a layout exists.
	fallbackWidth = 470
	// levels are the shades above an empty day.
	levels = 4
)

// The entrance lights the cells column by column; each takes a quarter of
// the run to come on. The run starts below 0, so the wave waits for its
// card to land.
var (
	entrance  = ui.Animation{Duration: 1200 * time.Millisecond, Ease: func(t float32) float32 { return t }}
	cellRun   = float32(0.25)
	waveStart = float32(-0.18)
)

// grid places the cells within the heatmap's box.
type grid struct {
	x, pitch, cell float32
	weeks          int
}

func gridFor(width float32) grid {
	room := width - dayLabels + cellGap
	weeks := min(maxWeeks, max(1, int(room/(minCell+cellGap))))
	pitch := min(room/float32(weeks), maxPitch)
	return grid{x: dayLabels + room - pitch*float32(weeks), pitch: pitch, cell: pitch - cellGap, weeks: weeks}
}

func (g grid) height() float32 { return monthLabels + weekDays*g.pitch - cellGap }

func (g grid) rect(r ui.Rect, week, day int) ui.Rect {
	return ui.Rect{X: r.X + g.x + float32(week)*g.pitch, Y: r.Y + monthLabels + float32(day)*g.pitch, W: g.cell, H: g.cell}
}

// at returns the cell under a point of the box.
func (g grid) at(x, y float32) (week, day int, ok bool) {
	if x < g.x || y < monthLabels {
		return 0, 0, false
	}
	week, day = int((x-g.x)/g.pitch), int((y-monthLabels)/g.pitch)
	return week, day, week < g.weeks && day < weekDays
}

// heatDays are the days of the last maxWeeks weeks, Monday first, ending
// today; the current week ends early.
type heatDays struct {
	days []runtime.UsageDay
	end  time.Time
}

func lastWeeks(log runtime.UsageLog, now time.Time) heatDays {
	today := (int(now.Weekday()) + 6) % weekDays
	return heatDays{days: log.Days(now, (maxWeeks-1)*weekDays+today+1), end: now}
}

// day returns the day in a visible column, if it has come yet.
func (h heatDays) day(g grid, week, day int) (runtime.UsageDay, time.Time, bool) {
	index := (maxWeeks-g.weeks+week)*weekDays + day
	if index >= len(h.days) {
		return runtime.UsageDay{}, time.Time{}, false
	}
	return h.days[index], h.end.AddDate(0, 0, index-len(h.days)+1), true
}

// level shades a day by its rank among the visible days with dictation,
// as GitHub does, so one unusually busy day leaves the others readable:
// the busiest quarter is level 4. active is sorted.
func level(words int, active []int) int {
	if words <= 0 || len(active) == 0 {
		return 0
	}
	rank := sort.SearchInts(active, words+1)
	return min(levels, max(1, int(math.Ceil(float64(levels*rank)/float64(len(active))))))
}

// shades are the colors of levels 0 to 4: an empty well, then the accent
// growing to full strength and deepening to its ink.
func shades(tokens theme.Tokens) [levels + 1]ui.Color {
	return [levels + 1]ui.Color{tokens.Hairline, tokens.Accent.Alpha(0.3).Over(tokens.Card), tokens.Accent.Alpha(0.62).Over(tokens.Card),
		tokens.Accent, tokens.AccentInk}
}

// activity is the heatmap card: words per day, with a legend of shades.
func activity(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, log runtime.UsageLog, now time.Time, entering bool) ui.Element {
	colors := shades(tokens)
	card := kit.Card(c, tokens)
	card.Children(func() {
		kit.CardHeader(c, tokens, messages.Text(copy.UsageActivity), func() { legend(c, tokens, messages, colors) })
		heatmap(c, tokens, messages, lastWeeks(log, now), colors, entering)
	})
	return card
}

// legend shows the shades from fewest to most words.
func legend(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, colors [levels + 1]ui.Color) {
	ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
		ui.Text(c, messages.Text(copy.UsageLess)).FontSize(theme.Caption).TextColor(tokens.Muted).Margin(0, 3, 0, 0)
		for _, color := range colors {
			ui.Box(c).Size(10, 10).Radius(theme.RadiusXS).Background(color)
		}
		ui.Text(c, messages.Text(copy.UsageMore)).FontSize(theme.Caption).TextColor(tokens.Muted).Margin(0, 0, 0, 3)
	})
}

// heatmap draws the cells with month and weekday labels. Cells light up
// in a wave from the oldest week when the overview is entered; the hovered
// day names its words.
func heatmap(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, h heatDays, colors [levels + 1]ui.Color, entering bool) {
	box := ui.Box(c.Key("heatmap")).Margin(4, 16, 16, 16)
	width := box.LayoutBounds().W
	if width <= 0 {
		width = fallbackWidth
	}
	g := gridFor(width)
	box.Height(g.height())
	if entering {
		box.Jump("reveal", waveStart)
	}
	progress := box.AnimateTo("reveal", 1, entrance)

	var active []int
	for week := 0; week < g.weeks; week++ {
		for day := 0; day < weekDays; day++ {
			if d, _, ok := h.day(g, week, day); ok && d.Words > 0 {
				active = append(active, d.Words)
			}
		}
	}
	sort.Ints(active)
	hoverWeek, hoverDay := -1, -1
	if x, y, over := box.PointerPosition(); over {
		if week, day, ok := g.at(x, y); ok {
			if d, date, ok := h.day(g, week, day); ok {
				hoverWeek, hoverDay = week, day
				box.Tooltip(messages.DayWords(date, d.Words))
			}
		}
	}

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		g := gridFor(r.W)
		label := func(text string, x, y float32) {
			p.RichText(x, y, 0, ui.Span{Text: text, Size: theme.Caption, Color: tokens.Muted})
		}
		_, line := p.MeasureText(0, ui.Span{Text: "0", Size: theme.Caption})
		for day := 0; day < weekDays; day += 2 {
			cell := g.rect(r, 0, day)
			label(messages.Weekday(time.Weekday((day+1)%weekDays)), r.X, cell.Y+(g.cell-line)/2)
		}
		labelEnd := float32(-1)
		previous := time.Month(0)
		for week := 0; week < g.weeks; week++ {
			_, date, ok := h.day(g, week, 0)
			if !ok {
				continue
			}
			if date.Month() != previous {
				previous = date.Month()
				// A month label needs room before the next one starts.
				text := messages.Month(date.Month())
				w, _ := p.MeasureText(0, ui.Span{Text: text, Size: theme.Caption})
				x := g.rect(r, week, 0).X
				if x > labelEnd+6 && x+w <= r.X+r.W {
					label(text, x, r.Y)
					labelEnd = x + w
				}
			}
		}
		radius := min(float32(theme.RadiusSM)/2+1, g.cell/4)
		for week := 0; week < g.weeks; week++ {
			for day := 0; day < weekDays; day++ {
				d, _, ok := h.day(g, week, day)
				if !ok {
					continue
				}
				cell := g.rect(r, week, day)
				p.Fill(cell, colors[0], radius)
				if shade := level(d.Words, active); shade > 0 {
					// Each column starts a little after the one before it.
					start := (1 - cellRun) * (float32(week) + float32(day)*0.35) / (float32(g.weeks) + weekDays*0.35)
					t := min(max((progress-start)/cellRun, 0), 1)
					if t > 0 {
						grow := ui.EaseOut(t)
						size := g.cell * (0.4 + 0.6*grow)
						inset := (g.cell - size) / 2
						p.Fill(ui.Rect{X: cell.X + inset, Y: cell.Y + inset, W: size, H: size}, colors[shade].Alpha(t), radius)
					}
				}
				if week == hoverWeek && day == hoverDay {
					p.Stroke(cell, tokens.Text.Alpha(0.55), radius, 1)
				}
			}
		}
	})
}
