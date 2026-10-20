package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// galleryCharts is the state of the Charts page: the sales of the bar
// chart and the month clicked, the point chosen on the live line, how the
// areas stack, and the points of the scatter plot.
type galleryCharts struct {
	sales    []float64
	month    int
	selected int
	stacking int
	scatter  [3][]scatterPoint
}

// scatterPoint is a point of the scatter plot: a car's weight and fuel use.
type scatterPoint struct{ weight, use float64 }

var (
	chartMonths = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	// chartColors color the series, in light and dark windows alike.
	chartColors = []ui.Color{ui.Hex("#3b82f6"), ui.Hex("#10b981"), ui.Hex("#f59e0b"), ui.Hex("#ec4899")}
	// The visits of the stacked areas, by week, from three sources.
	chartSources = []string{"Search", "Social", "Direct"}
	chartVisits  = [][]float64{
		{30, 34, 38, 36, 42, 47, 45, 52, 58, 55, 61, 66},
		{12, 15, 13, 19, 24, 22, 28, 31, 27, 35, 38, 36},
		{20, 19, 22, 21, 23, 22, 25, 24, 27, 26, 28, 30},
	}
	chartPlatforms = []string{"macOS", "Windows", "Linux", "Browsers"}
	chartShares    = []float64{46, 31, 14, 9}
	// bars grow and shrink to new values.
	chartMove = ui.ElementTransition{Duration: 350 * time.Millisecond}
)

// newCharts returns the Charts page's data, the same at every start.
func newCharts() galleryCharts {
	r := rand.New(rand.NewPCG(1, 2))
	g := galleryCharts{month: -1, selected: -1}
	g.sales = make([]float64, len(chartMonths))
	for i := range g.sales {
		g.sales[i] = 40 + 50*math.Sin(float64(i)/3.5) + 20*r.Float64()
	}
	for k := range g.scatter {
		for range 24 {
			w := 1100 + 450*float64(k) + 170*r.NormFloat64()
			g.scatter[k] = append(g.scatter[k], scatterPoint{w, 3 + w/420 + 0.9*r.NormFloat64()})
		}
	}
	return g
}

// axisLabel is the look of the labels of the charts' axes.
func axisLabel(c *ui.Context) func(chart.Side, chart.Tick) {
	return func(_ chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(c.Theme().TextMuted)
	}
}

// gridLines paints a line across the plot at each tick of y, and along it
// at each of x.
func gridLines(area ui.Element, x, y []chart.Tick, color ui.Color) {
	area.Draw(func(p *ui.Painter, r ui.Rect) {
		for _, tick := range y {
			p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, color)
		}
		for _, tick := range x {
			p.Line(chart.X(r, tick.At), r.Y, chart.X(r, tick.At), r.Y+r.H, 1, color)
		}
	})
}

// legend shows the series' names by their colors.
func legend(c *ui.Context, names []string) {
	ui.Row(c).Gap(14).Wrap().Children(func() {
		for i, name := range names {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(10, 10).Radius(3).Background(chartColors[i])
				ui.Text(c, name).FontSize(12).TextColor(c.Theme().TextMuted)
			})
		}
	})
}

func (g *gallery) charts(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, "Package ui/chart has scales, axes and marks without a look: the marks are elements styled here, "+
		"and lines, areas and slices paths these cards paint.").TextColor(t.TextMuted)
	ui.Row(c).Gap(16).Wrap().AlignItems(ui.Start).Children(func() {
		card(c, "Sales", func() { g.salesChart(c) }).Basis(320).Grow(1).MinWidth(0)
		card(c, "Live data", func() { g.liveChart(c) }).Basis(320).Grow(1).MinWidth(0)
		card(c, "Visits by source", func() { g.stackedChart(c) }).Basis(320).Grow(1).MinWidth(0)
		card(c, "Weight and fuel use", func() { g.scatterChart(c) }).Basis(320).Grow(1).MinWidth(0)
		card(c, "Users by platform", func() { g.donutChart(c) }).Basis(320).Grow(1).MinWidth(0)
	})
}

// salesChart is a bar chart whose bars grow to new values, show tooltips
// and choose a month when clicked, with the value above the one chosen.
func (g *gallery) salesChart(c *ui.Context) {
	t := c.Theme()
	ch := &g.chart
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		text := "Click a bar to choose its month."
		if ch.month >= 0 {
			text = fmt.Sprintf("%s: $%.1fk", chartMonths[ch.month], ch.sales[ch.month])
		}
		ui.Text(c, text).TextColor(t.TextMuted).Grow(1)
		if ui.Button(c, "Shuffle").Clicked() {
			for i := range ch.sales {
				ch.sales[i] = 10 + 110*rand.Float64()
			}
		}
	})
	x := chart.NewBand(chartMonths...).Padding(0.3, 0.1)
	_, hi := chart.Extent(ch.sales...)
	y := chart.NewLinear(0, hi).Nice(4)
	y.Format = func(v float64) string {
		if v == 0 {
			return "$0"
		}
		return fmt.Sprintf("$%.0fk", v)
	}
	ticks := y.Ticks(4)
	// Every other month is labeled, so that the labels keep apart.
	var months []chart.Tick
	for i, tick := range x.Ticks() {
		if i%2 == 0 {
			months = append(months, tick)
		}
	}
	chart.Plot(c, chart.Axes{Left: ticks, Bottom: months, Label: axisLabel(c)}, func(area ui.Element) {
		gridLines(area, nil, ticks, t.Border)
		for i, v := range ch.sales {
			bar := chart.Rect(c.Key(i), x.Band(i), y.Span(0, v)).Radius(3).Background(t.Accent).
				Transition(chartMove).Tooltip(fmt.Sprintf("%s: $%.1fk", chartMonths[i], v)).
				Label(chartMonths[i]).Cursor(ui.CursorPointer)
			if ch.month >= 0 && i != ch.month {
				bar.Background(t.Accent.Alpha(0.35))
			}
			if bar.Clicked() {
				if ch.month == i {
					ch.month = -1
				} else {
					ch.month = i
				}
			}
		}
		if i := ch.month; i >= 0 {
			chart.Anchor(c, x.At(i), y.At(ch.sales[i]), ui.Center, ui.End, func() {
				ui.Text(c, fmt.Sprintf("%.0f", ch.sales[i])).NoWrap().FontSize(11).Bold().Margin(0, 0, 4, 0)
			})
		}
	}).Label("Sales by month").Height(200).GapX(8).GapY(6)
}

// liveChart is a line of the last minute of samples, which a goroutine
// adds to every second, with the point under the pointer, or the arrows,
// chosen.
func (g *gallery) liveChart(c *ui.Context) {
	t := c.Theme()
	ch := &g.chart
	n := len(g.samples)
	end := g.now.Truncate(time.Second)
	x := chart.NewTime(end.Add(-59*time.Second), end)
	y := chart.NewLinear(0, 1)
	y.Format = func(v float64) string { return fmt.Sprintf("%.0f%%", 100*v) }
	xt, yt := x.Ticks(4), y.Ticks(4)
	xs := make([]float32, n)
	ys := make([]float32, n)
	at := func(i int) time.Time { return end.Add(time.Duration(i-n+1) * time.Second) }
	for i, v := range g.samples {
		xs[i], ys[i] = x.At(at(i)), y.At(v)
	}
	describe := func(i int) string { return fmt.Sprintf("%s: %.0f%%", at(i).Format("15:04:05"), 100*g.samples[i]) }
	text := "Point at the line, or focus it and use the arrows."
	if ch.selected >= 0 && ch.selected < n {
		text = describe(ch.selected)
	}
	ui.Text(c, text).TextColor(t.TextMuted)
	chart.Plot(c, chart.Axes{Left: yt, Bottom: xt, Label: axisLabel(c)}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range yt {
				p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
			}
			p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), t.Accent.Alpha(0.12))
			p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, t.Accent)
		})
		chart.Select(c, area, &ch.selected, xs, describe)
		if i := ch.selected; i >= 0 && i < n {
			chart.RuleX(c, xs[i], 1).Background(t.TextMuted)
			chart.Point(c, xs[i], ys[i], 10, 10).Radius(5).Background(t.Accent).Border(2, t.Background)
		}
	}).Label("Live samples").Description("The last minute of samples, one a second").Height(200).GapX(8).GapY(6)
}

// stackedChart piles three series of areas up, as values or as shares.
func (g *gallery) stackedChart(c *ui.Context) {
	t := c.Theme()
	ch := &g.chart
	ui.Segmented(c, &ch.stacking, "Values", "Shares").Label("Stacking").AlignSelf(ui.Start)
	how := chart.StackStandard
	if ch.stacking == 1 {
		how = chart.StackNormalized
	}
	stacked := chart.Stack(chartVisits, how)
	hi := 0.0
	for _, seg := range stacked[len(stacked)-1] {
		hi = max(hi, seg.Hi)
	}
	y := chart.NewLinear(0, hi).Nice(4)
	if how == chart.StackNormalized {
		y = chart.NewLinear(0, 1)
		y.Format = func(v float64) string { return fmt.Sprintf("%.0f%%", 100*v) }
	}
	weeks := len(chartVisits[0])
	x := chart.NewLinear(1, float64(weeks))
	x.Format = func(v float64) string { return fmt.Sprintf("W%.0f", v) }
	xt, yt := x.Ticks(6), y.Ticks(4)
	xs := make([]float32, weeks)
	for i := range xs {
		xs[i] = x.At(float64(i + 1))
	}
	chart.Plot(c, chart.Axes{Left: yt, Bottom: xt, Label: axisLabel(c)}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range yt {
				p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
			}
			for s, series := range stacked {
				lo, hi := make([]float32, weeks), make([]float32, weeks)
				for i, seg := range series {
					lo[i], hi[i] = y.At(seg.Lo), y.At(seg.Hi)
				}
				p.FillPath(chart.Area(r, xs, lo, hi, chart.CurveMonotone), chartColors[s].Alpha(0.75))
				p.StrokePath(chart.Line(r, xs, hi, chart.CurveMonotone), 1.5, chartColors[s])
			}
		})
	}).Label("Visits by source").Height(180).GapX(8).GapY(6)
	legend(c, chartSources)
}

// scatterChart shows cars as points, by weight and fuel use, in three
// classes, each point with a tooltip.
func (g *gallery) scatterChart(c *ui.Context) {
	t := c.Theme()
	ch := &g.chart
	legend(c, []string{"Small", "Midsize", "Large"})
	var weights, uses []float64
	for _, class := range ch.scatter {
		for _, p := range class {
			weights, uses = append(weights, p.weight), append(uses, p.use)
		}
	}
	x := chart.NewLinear(chart.Extent(weights...)).Nice(4)
	y := chart.NewLinear(chart.Extent(uses...)).Nice(4)
	x.Format = func(v float64) string { return fmt.Sprintf("%.1f t", v/1000) }
	y.Format = func(v float64) string { return fmt.Sprintf("%.0f L", v) }
	xt, yt := x.Ticks(4), y.Ticks(4)
	chart.Plot(c, chart.Axes{Left: yt, Bottom: xt, Label: axisLabel(c)}, func(area ui.Element) {
		gridLines(area, xt, yt, t.Border)
		for k, class := range ch.scatter {
			for _, p := range class {
				chart.Point(c, x.At(p.weight), y.At(p.use), 9, 9).Radius(4.5).
					Background(chartColors[k].Alpha(0.8)).Border(1, chartColors[k]).
					Tooltip(fmt.Sprintf("%.0f kg, %.1f L/100 km", p.weight, p.use))
			}
		}
	}).Label("Weight and fuel use of cars").Height(200).GapX(8).GapY(6)
}

// donutChart divides a ring among platforms, growing the slice under the
// pointer, whose share shows in the middle.
func (g *gallery) donutChart(c *ui.Context) {
	t := c.Theme()
	slices := chart.Pie(chartShares, 0.025)
	const size, inner, outer = 180, 56, 86
	ui.Row(c).Gap(28).AlignItems(ui.Center).Children(func() {
		ring := ui.Box(c).Size(size, size).Label("Users by platform")
		hover := -1
		if x, y, over := ring.PointerPosition(); over {
			if i, ok := chart.SliceAt(slices, x-size/2, y-size/2, inner, outer); ok {
				hover = i
			}
		}
		ring.Draw(func(p *ui.Painter, r ui.Rect) {
			for i, s := range slices {
				out := float32(outer - 4)
				if i == hover {
					out = outer
				}
				p.FillPath(chart.Sector(r.X+size/2, r.Y+size/2, inner, out, s), chartColors[i])
			}
		})
		ring.Children(func() {
			ui.Column(c).Fill().Center().Children(func() {
				if hover < 0 {
					ui.Text(c, "4.2M").FontSize(24).Bold()
					ui.Text(c, "users").FontSize(12).TextColor(t.TextMuted)
					return
				}
				ui.Text(c, fmt.Sprintf("%.0f%%", chartShares[hover])).FontSize(24).Bold()
				ui.Text(c, chartPlatforms[hover]).FontSize(12).TextColor(t.TextMuted)
			})
		})
		ui.Column(c).Gap(8).Children(func() {
			for i, name := range chartPlatforms {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(10, 10).Radius(3).Background(chartColors[i])
					ui.Text(c, name).FontSize(13).Width(70)
					ui.Text(c, fmt.Sprintf("%.0f%%", chartShares[i])).FontSize(13).TextColor(t.TextMuted)
				})
			}
		})
	})
}
