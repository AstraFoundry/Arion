# Charts

<!-- preview chart 300 -->

Package `ui/chart` builds charts without a look of their own, as headless
libraries such as visx do on the web, with the parts of Swift Charts:
scales that map data onto a plot and pick round ticks, a plot with axes
around it, marks that are elements you style as any other, paths of lines,
areas and pie slices for `Draw` functions, stacking, and a selection that
the pointer and the keyboard move. Every color, size and font is yours.

```go
import "github.com/egoist/mygo/ui/chart"

t := c.Theme()
x := chart.NewBand(app.months...).Padding(0.3, 0.15)
_, hi := chart.Extent(app.sales...)
y := chart.NewLinear(0, hi).Nice(4)
ticks := y.Ticks(4)
chart.Plot(c, chart.Axes{Left: ticks, Bottom: x.Ticks()}, func(area ui.Element) {
	area.Draw(func(p *ui.Painter, r ui.Rect) {
		for _, tick := range ticks { // grid lines
			p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
		}
	})
	for i, v := range app.sales {
		chart.Rect(c, x.Band(i), y.Span(0, v)).Radius(4).Background(t.Accent).
			Tooltip(fmt.Sprintf("%s: %.0f", app.months[i], v))
	}
}).Label("Sales by month").Height(200).Gap(6)
```

Positions on a plot are fractions of it, from 0 to 1: along x from its left
edge, and along y from its bottom up, the way values grow. Scales turn data
into fractions; marks place themselves in percentages of the plot, so they
are built without waiting for the layout and follow the plot as it
resizes; `Draw` functions get the plot's box, and `chart.X(r, f)` and
`chart.Y(r, f)` turn fractions into points in it.

## Scales

- `chart.NewLinear(lo, hi)` maps numbers. `Nice(n)` rounds the domain out
  to the round values of about n ticks, and `Ticks(n)` returns them: 1, 2
  or 5 times a power of ten apart, as d3 and Swift Charts pick them, with
  as many decimals as the step needs, or labeled by `Format`. `At(v)` is
  where a value is, `Span(a, b)` the stretch between two, as a bar's from 0
  to its value, and `Invert(f)` the value at a fraction, as under the
  pointer. `Clamp` keeps values outside the domain at its ends;
  `chart.Extent(values...)` returns the smallest and the largest.
- `chart.NewBand(labels...)` gives each label a band, as the bars of a bar
  chart: `Band(i)` is the i-th's stretch, `At(i)` its middle, `Width()`
  their width, and `Index(f)` the band at a fraction. `Padding(inner,
  outer)` leaves room between the bands and around them, in fractions of a
  band's step. Its `Ticks()` are the labels, in the middle of their bands.
- `chart.NewTime(from, to)` maps times. Its ticks fall on round times in
  the times' location, chosen for the span as d3's are: seconds, quarters
  of an hour, midnights, Sundays, the firsts of months or years; and each
  is labeled with what changed since the last round time of the coarser
  unit: "2026", "Mar", "Mar 5", "14:00", ":30".

Ticks are `chart.Tick` values, `At` a fraction, `Value` and `Label`: make
your own for axes a scale does not mark.

## The plot and its axes

`chart.Plot` lays the plot's area out in a grid with an axis on each side
that `chart.Axes` gives ticks to: `Left` and `Right` along the height,
`Top` and `Bottom` along the width. The labels stay in the layout, so the
axes take the room they need, and the area the rest: give the plot a
height, or `Fill` its parent, and a `Gap` (or `GapX` and `GapY`) between
the labels and the area. `Axes.Label` builds each label, centered on its
tick; it is plain text by default. Labels sit in boxes of no width or
height that center them, so keep their text on one line with `NoWrap`:

```go
label := func(side chart.Side, tick chart.Tick) {
	ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(t.TextMuted)
}
chart.Plot(c, chart.Axes{Left: y.Ticks(4), Bottom: x.Ticks(), Label: label}, build)
```

The function of `Plot` gets the area, an element to style, with a
background, a border or a `Draw` function painting grid lines and lines,
and to add marks to.

## Marks

Marks are elements in the area, placed in percentages of it: style them,
name them for assistive technology, give them tooltips, and handle their
clicks, as any element.

- `chart.Rect(c, x, y)` covers a stretch along x and up y: a bar, a band of
  a range, a cell of a heat map.
- `chart.Point(c, x, y, w, h)` is w by h DIPs, centered on a point: a dot
  of a scatter plot, the point a selection is on.
- `chart.RuleX(c, x, width)` is a vertical line across the plot, and
  `chart.RuleY(c, y, height)` a horizontal one: a threshold, an average.
- `chart.Anchor(c, x, y, h, v, build)` places what `build` adds at a point,
  of any size: `ui.Center` centers it there, `ui.Start` puts its left (h)
  or top (v) edge there, and `ui.End` its right or bottom edge, so that
  `ui.Center, ui.End` shows a label above a bar.

```go
for i, v := range app.sales {
	bar := chart.Rect(c, x.Band(i), y.Span(0, v)).Background(t.Accent)
	if bar.Clicked() {
		app.month = i
	}
	chart.Anchor(c, x.At(i), y.At(v), ui.Center, ui.End, func() {
		ui.Text(c, fmt.Sprint(v)).NoWrap().FontSize(11).Margin(0, 0, 4, 0)
	})
}
```

A mark with a [`Transition`](transitions.md) moves to its new place as the
data changes, as bars growing to new values:

```go
chart.Rect(c, x.Band(i), y.Span(0, v)).Background(t.Accent).
	Transition(ui.ElementTransition{Duration: 300 * time.Millisecond})
```

## Lines and areas

`chart.Line(r, xs, ys, curve)` returns the path of a line through points,
and `chart.Area(r, xs, lo, hi, curve)` the path of the area between two
lines, for a `Draw` function of the area to stroke and fill. `xs` and `ys`
are fractions, as scales return; nil `lo` is the bottom of the plot, and a
NaN breaks the line or the area there, for missing values. The curve joins
the points: `chart.CurveLinear` with straight lines, `chart.CurveMonotone`
smoothly, without going past the values of the points it joins, and
`chart.CurveStepAfter`, `CurveStepBefore` and `CurveStep` in steps.

```go
area.Draw(func(p *ui.Painter, r ui.Rect) {
	p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), t.Accent.Alpha(0.15))
	p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, t.Accent)
})
```

## Stacks

`chart.Stack(series, how)` piles series up, as stacked bars and areas show
them: `series[s][i]` is the i-th value of the s-th series, and the result's
`[s][i]` the `Segment` it covers in the i-th stack. `chart.StackStandard`
piles values up from 0, and negative values down from it;
`chart.StackNormalized` in shares of the stack's sum, from 0 to 1; and
`chart.StackCenter` around 0, as a streamgraph.

```go
stacked := chart.Stack([][]float64{app.search, app.direct}, chart.StackStandard)
for s, series := range stacked {
	for i, seg := range series {
		chart.Rect(c, x.Band(i), y.Span(seg.Lo, seg.Hi)).Background(colors[s])
	}
}
```

## Pies and donuts

`chart.Pie(values, pad)` divides a circle among values, in their order
from twelve o'clock, with `pad` radians between them, and
`chart.Sector(cx, cy, inner, outer, slice)` returns the path of a slice: a
wedge with `inner` 0, a piece of a donut otherwise. `chart.SliceAt` finds
the slice at a point, as under the pointer, and `chart.Polar` the point at
an angle, where a slice's label goes:

```go
slices := chart.Pie(app.shares, 0.03)
ring := ui.Box(c).Size(160, 160).Label("Users by platform")
hover := -1
if x, y, over := ring.PointerPosition(); over {
	if i, ok := chart.SliceAt(slices, x-80, y-80, 50, 80); ok {
		hover = i
	}
}
ring.Draw(func(p *ui.Painter, r ui.Rect) {
	for i, s := range slices {
		outer := float32(76)
		if i == hover {
			outer = 80
		}
		p.FillPath(chart.Sector(r.X+80, r.Y+80, 50, outer, s), colors[i])
	}
})
```

## Selection

`chart.Select` makes the area choose one of the points at `xs`, in
ascending order, as Swift Charts' `chartXSelection`: the one nearest the
pointer while it is over the area, and none (-1) once it leaves. The area
takes the keyboard focus, as one stop of Tab, and the left and right
arrows, Home and End move the choice while it has it, and Escape clears
it. It reports a new choice. Show the choice with marks:

```go
describe := func(i int) string { return fmt.Sprintf("%s: %.1f °C", hours[i], temps[i]) }
chart.Select(c, area, &app.selected, xs, describe)
if i := app.selected; i >= 0 {
	chart.RuleX(c, xs[i], 1).Background(t.TextMuted)
	chart.Point(c, xs[i], ys[i], 10, 10).Radius(5).Background(t.Accent)
	chart.Anchor(c, xs[i], 1, ui.Center, ui.Start, func() {
		ui.Text(c, describe(i)).NoWrap().Padding(3, 8).Radius(6).Background(t.Surface)
	})
}
```

For the points of a band scale, `xs` are their middles; `chart.Nearest`
finds the point nearest a fraction for selections of your own, and a
band's `Index` the band.

## Accessibility

Name the plot with `Label` and say what it shows with `Description`.
Marks are elements: name each one assistive technology should read, as a
bar's month and value. `Select` tells it about the chosen point: what
`describe` returns is the area's value, which screen readers read out as
the arrows come to a point.

What the package leaves out: logarithmic scales, zooming and panning,
legends (rows of your own, as above) and labels that avoid each other.
