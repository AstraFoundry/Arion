package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// The chart demo's data: visits of a site by month, from two sources, and
// a temperature through a day.
var (
	chartKind   = 0
	chartMonths = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	chartSearch = []float64{42, 58, 51, 74, 88, 69}
	chartDirect = []float64{18, 22, 30, 26, 35, 41}
	chartShares = []float64{48, 27, 15, 10}
	chartNames  = []string{"macOS", "Windows", "Linux", "Browsers"}
	chartDay    = time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	chartTemps  = []float64{6, 5, 4.5, 4, 5, 7.5, 10, 12.5, 14, 15, 14.5, 12, 9.5}
	chartPoint  = -1
	chartSlice  = -1
)

func init() {
	demo("chart", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(12).Children(func() {
			ui.Segmented(c, &chartKind, "Bars", "Line", "Stacked", "Donut")
			if chartKind < 3 && ui.Button(c, "Shuffle").Clicked() {
				for i := range chartSearch {
					chartSearch[i] = float64(20 + rand.IntN(80))
					chartDirect[i] = float64(10 + rand.IntN(40))
				}
				for i := range chartTemps {
					chartTemps[i] = 4 + 11*math.Sin(float64(i)/12*math.Pi) + rand.Float64()*3
				}
			}
		})
		ui.Column(c).Width(440).Height(200).Children(func() {
			switch chartKind {
			case 0:
				chartBars(c, t)
			case 1:
				chartLine(c, t)
			case 2:
				chartStacked(c, t)
			default:
				chartDonut(c, t)
			}
		})
	})
}

// chartGrid paints a line across the plot at each tick.
func chartGrid(area ui.Element, ticks []chart.Tick, color ui.Color) {
	area.Draw(func(p *ui.Painter, r ui.Rect) {
		for _, tick := range ticks {
			y := chart.Y(r, tick.At)
			p.Line(r.X, y, r.X+r.W, y, 1, color)
		}
	})
}

// chartLabel is the style of the demo's axis labels.
func chartLabel(c *ui.Context) func(chart.Side, chart.Tick) {
	return func(side chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(c.Theme().TextMuted)
	}
}

// bars move to their new heights as the data changes.
var chartBarMove = ui.ElementTransition{Duration: 300 * time.Millisecond}

func chartBars(c *ui.Context, t *ui.Theme) {
	x := chart.NewBand(chartMonths...).Padding(0.35, 0.2)
	_, hi := chart.Extent(chartSearch...)
	y := chart.NewLinear(0, hi).Nice(4)
	ticks := y.Ticks(4)
	chart.Plot(c, chart.Axes{Left: ticks, Bottom: x.Ticks(), Label: chartLabel(c)}, func(area ui.Element) {
		chartGrid(area, ticks, t.Border)
		for i, v := range chartSearch {
			chart.Rect(c, x.Band(i), y.Span(0, v)).Radius(4).Background(t.Accent).
				Transition(chartBarMove).Tooltip(fmt.Sprintf("%s: %.0f visits", chartMonths[i], v))
		}
	}).Label("Visits by month").Fill().GapX(8).GapY(6)
}

func chartLine(c *ui.Context, t *ui.Theme) {
	x := chart.NewTime(chartDay, chartDay.Add(24*time.Hour))
	lo, hi := chart.Extent(chartTemps...)
	y := chart.NewLinear(lo, hi).Nice(4)
	ticks := y.Ticks(4)
	xs := make([]float32, len(chartTemps))
	ys := make([]float32, len(chartTemps))
	for i, v := range chartTemps {
		xs[i] = x.At(chartDay.Add(time.Duration(i) * 2 * time.Hour))
		ys[i] = y.At(v)
	}
	chart.Plot(c, chart.Axes{Left: ticks, Bottom: x.Ticks(6), Label: chartLabel(c)}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range ticks {
				y := chart.Y(r, tick.At)
				p.Line(r.X, y, r.X+r.W, y, 1, t.Border)
			}
			p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), t.Accent.Alpha(0.15))
			p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, t.Accent)
		})
		describe := func(i int) string {
			return fmt.Sprintf("%s: %.1f °C", chartDay.Add(time.Duration(i)*2*time.Hour).Format("15:04"), chartTemps[i])
		}
		chart.Select(c, area, &chartPoint, xs, describe)
		if i := chartPoint; i >= 0 {
			chart.RuleX(c, xs[i], 1).Background(t.TextMuted)
			chart.Point(c, xs[i], ys[i], 10, 10).Radius(5).Background(t.Accent).Border(2, t.Background)
			h := ui.Start
			if xs[i] > 0.5 {
				h = ui.End
			}
			chart.Anchor(c, xs[i], 1, h, ui.Start, func() {
				ui.Text(c, describe(i)).NoWrap().FontSize(12).Padding(3, 8).Margin(0, 6).
					Radius(6).Background(t.Surface).Border(1, t.Border)
			})
		}
	}).Label("Temperature on March 5").Fill().GapX(8).GapY(6)
}

func chartStacked(c *ui.Context, t *ui.Theme) {
	x := chart.NewBand(chartMonths...).Padding(0.35, 0.2)
	stacked := chart.Stack([][]float64{chartSearch, chartDirect}, chart.StackStandard)
	hi := 0.0
	for _, seg := range stacked[1] {
		hi = max(hi, seg.Hi)
	}
	y := chart.NewLinear(0, hi).Nice(4)
	ticks := y.Ticks(4)
	colors := []ui.Color{t.Accent, t.Accent.Alpha(0.45)}
	names := []string{"Search", "Direct"}
	ui.Row(c).Gap(14).Children(func() {
		for s, name := range names {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(10, 10).Radius(3).Background(colors[s])
				ui.Text(c, name).FontSize(12).TextColor(t.TextMuted)
			})
		}
	}).Margin(0, 0, 8, 0)
	chart.Plot(c, chart.Axes{Left: ticks, Bottom: x.Ticks(), Label: chartLabel(c)}, func(area ui.Element) {
		chartGrid(area, ticks, t.Border)
		for s, series := range stacked {
			for i, seg := range series {
				chart.Rect(c, x.Band(i), y.Span(seg.Lo, seg.Hi)).Background(colors[s]).
					Transition(chartBarMove).Tooltip(fmt.Sprintf("%s, %s: %.0f", chartMonths[i], names[s], seg.Hi-seg.Lo))
			}
		}
	}).Label("Visits by month and source").Grow(1).GapX(8).GapY(6)
}

func chartDonut(c *ui.Context, t *ui.Theme) {
	slices := chart.Pie(chartShares, 0.03)
	colors := make([]ui.Color, len(slices))
	for i := range colors {
		colors[i] = t.Accent.Mix(t.Background, float32(i)*0.22)
	}
	const inner, outer = 52, 88
	ui.Row(c).Fill().Gap(24).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ring := ui.Box(c).Size(2*outer, 2*outer).Label("Users by platform")
		chartSlice = -1
		if x, y, over := ring.PointerPosition(); over {
			if i, ok := chart.SliceAt(slices, x-outer, y-outer, inner, outer); ok {
				chartSlice = i
			}
		}
		ring.Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			for i, s := range slices {
				out := float32(outer - 4)
				if i == chartSlice {
					out = outer
				}
				p.FillPath(chart.Sector(cx, cy, inner, out, s), colors[i])
			}
		})
		ring.Children(func() {
			ui.Column(c).Fill().Center().Children(func() {
				i := max(chartSlice, 0)
				ui.Text(c, fmt.Sprintf("%.0f%%", chartShares[i])).FontSize(22).Bold()
				ui.Text(c, chartNames[i]).FontSize(12).TextColor(t.TextMuted)
			})
		})
		ui.Column(c).Gap(6).Children(func() {
			for i, name := range chartNames {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(10, 10).Radius(3).Background(colors[i])
					ui.Text(c, fmt.Sprintf("%s  %.0f%%", name, chartShares[i])).FontSize(12)
				})
			}
		})
	})
}
