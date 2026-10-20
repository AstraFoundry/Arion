package chart

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The expected values below are d3's (d3-array 3.2.4, d3-scale 4.0.2,
// d3-shape 3.2.0) for the same input; times its scaleTime's, in UTC.

func TestTicks(t *testing.T) {
	for _, tc := range []struct {
		lo, hi float64
		count  int
		want   []float64
	}{
		{0, 1, 10, []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}},
		{0, 100, 5, []float64{0, 20, 40, 60, 80, 100}},
		{0, 87, 5, []float64{0, 20, 40, 60, 80}},
		{-3.2, 7.9, 6, []float64{-2, 0, 2, 4, 6}},
		{0.0012, 0.0047, 5, []float64{0.0015, 0.002, 0.0025, 0.003, 0.0035, 0.004, 0.0045}},
		{1e6, 3.4e6, 4, []float64{1e6, 1.5e6, 2e6, 2.5e6, 3e6}},
		{10, 0, 5, []float64{10, 8, 6, 4, 2, 0}},
		{0.1, 0.9, 3, []float64{0.2, 0.4, 0.6, 0.8}},
		{1, 1, 5, []float64{1}},
		{0, 1, 1, []float64{0, 1}},
	} {
		got := ticks(tc.lo, tc.hi, float64(tc.count))
		if !slices.Equal(got, tc.want) {
			t.Errorf("ticks(%v, %v, %d) = %v, want %v", tc.lo, tc.hi, tc.count, got, tc.want)
		}
	}
}

func TestNice(t *testing.T) {
	for _, tc := range []struct {
		lo, hi float64
		count  int
		want   [2]float64
	}{
		{0, 87, 5, [2]float64{0, 100}},
		{0, 87, 10, [2]float64{0, 90}},
		{0, 87, 2, [2]float64{0, 100}},
		{-3.2, 7.9, 6, [2]float64{-4, 8}},
		{0.0012, 0.0047, 5, [2]float64{0.001, 0.005}},
		{13, 1042, 10, [2]float64{0, 1100}},
		{90, 3, 5, [2]float64{100, 0}},
	} {
		s := NewLinear(tc.lo, tc.hi).Nice(tc.count)
		if got := [2]float64{s.Lo, s.Hi}; got != tc.want {
			t.Errorf("NewLinear(%v, %v).Nice(%d) = %v, want %v", tc.lo, tc.hi, tc.count, got, tc.want)
		}
	}
}

func TestTickLabels(t *testing.T) {
	labels := func(ts []Tick) string {
		var out []string
		for _, t := range ts {
			out = append(out, t.Label)
		}
		return strings.Join(out, " ")
	}
	for _, tc := range []struct {
		s     Linear
		count int
		want  string
	}{
		{NewLinear(0, 1), 5, "0.0 0.2 0.4 0.6 0.8 1.0"},
		{NewLinear(0, 100), 5, "0 20 40 60 80 100"},
		{NewLinear(0.0012, 0.0047), 5, "0.0015 0.0020 0.0025 0.0030 0.0035 0.0040 0.0045"},
		{NewLinear(-1, 1), 4, "-1.0 -0.5 0.0 0.5 1.0"},
		{Linear{Lo: 0, Hi: 1, Format: func(v float64) string { return "x" }}, 1, "x x"},
	} {
		if got := labels(tc.s.Ticks(tc.count)); got != tc.want {
			t.Errorf("%v.Ticks(%d) = %q, want %q", tc.s, tc.count, got, tc.want)
		}
	}
	if ts := NewLinear(0, 200).Ticks(2); ts[1].At != 0.5 || ts[1].Value != 100 {
		t.Errorf("ticks of 0 to 200: %+v", ts)
	}
}

func TestLinear(t *testing.T) {
	s := NewLinear(-10, 30)
	if s.At(0) != 0.25 || s.Invert(0.75) != 20 {
		t.Errorf("At(0) = %v, Invert(0.75) = %v", s.At(0), s.Invert(0.75))
	}
	if sp := s.Span(0, 10); sp != (Span{0.25, 0.5}) {
		t.Errorf("Span(0, 10) = %v", sp)
	}
	s.Clamp = true
	if s.At(50) != 1 || s.Invert(-1) != -10 {
		t.Errorf("clamped: At(50) = %v, Invert(-1) = %v", s.At(50), s.Invert(-1))
	}
	if NewLinear(3, 3).At(3) != 0.5 {
		t.Error("an empty domain maps to the middle")
	}
	if lo, hi := Extent(4, math.NaN(), -2, 9); lo != -2 || hi != 9 {
		t.Errorf("Extent = %v, %v", lo, hi)
	}
}

func TestBand(t *testing.T) {
	b := NewBand("a", "b", "c", "d").Padding(0.2, 0.1)
	want := []float32{0.025, 0.275, 0.525, 0.775}
	for i, w := range want {
		if got := b.Band(i).Lo; math.Abs(float64(got-w)) > 1e-6 {
			t.Errorf("band %d starts at %v, want %v", i, got, w)
		}
	}
	if w := b.Width(); math.Abs(float64(w-0.2)) > 1e-6 {
		t.Errorf("Width = %v, want 0.2", w)
	}
	for _, tc := range []struct {
		f  float32
		i  int
		ok bool
	}{{0.01, 0, true}, {0.1, 0, true}, {0.249, 0, true}, {0.251, 1, true}, {0.99, 3, true}, {-0.01, 0, false}, {1.01, 0, false}} {
		if i, ok := b.Index(tc.f); i != tc.i || ok != tc.ok {
			t.Errorf("Index(%v) = %d, %v, want %d, %v", tc.f, i, ok, tc.i, tc.ok)
		}
	}
	if ts := b.Ticks(); len(ts) != 4 || ts[2].Label != "c" || ts[2].At != b.At(2) {
		t.Errorf("Ticks = %+v", ts)
	}
}

func TestTimeTicks(t *testing.T) {
	parse := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, tc := range []struct {
		from, to string
		count    int
		want     []string
		labels   string
	}{
		{"2026-01-01T00:00:00Z", "2026-12-31T00:00:00Z", 6,
			[]string{"2026-01-01T00:00:00Z", "2026-04-01T00:00:00Z", "2026-07-01T00:00:00Z", "2026-10-01T00:00:00Z"},
			"2026 Apr Jul Oct"},
		{"2026-03-01T00:00:00Z", "2026-03-08T00:00:00Z", 7,
			[]string{"2026-03-01T00:00:00Z", "2026-03-02T00:00:00Z", "2026-03-03T00:00:00Z", "2026-03-04T00:00:00Z", "2026-03-05T00:00:00Z", "2026-03-06T00:00:00Z", "2026-03-07T00:00:00Z", "2026-03-08T00:00:00Z"},
			"Mar Mar 2 Mar 3 Mar 4 Mar 5 Mar 6 Mar 7 Mar 8"},
		{"2026-03-05T09:13:00Z", "2026-03-05T17:40:00Z", 5,
			[]string{"2026-03-05T10:00:00Z", "2026-03-05T11:00:00Z", "2026-03-05T12:00:00Z", "2026-03-05T13:00:00Z", "2026-03-05T14:00:00Z", "2026-03-05T15:00:00Z", "2026-03-05T16:00:00Z", "2026-03-05T17:00:00Z"},
			"10:00 11:00 12:00 13:00 14:00 15:00 16:00 17:00"},
		{"2026-03-05T09:13:07Z", "2026-03-05T09:14:51Z", 6,
			[]string{"2026-03-05T09:13:15Z", "2026-03-05T09:13:30Z", "2026-03-05T09:13:45Z", "2026-03-05T09:14:00Z", "2026-03-05T09:14:15Z", "2026-03-05T09:14:30Z", "2026-03-05T09:14:45Z"},
			":15 :30 :45 09:14 :15 :30 :45"},
		{"2010-06-01T00:00:00Z", "2026-06-01T00:00:00Z", 5,
			[]string{"2015-01-01T00:00:00Z", "2020-01-01T00:00:00Z", "2025-01-01T00:00:00Z"},
			"2015 2020 2025"},
		{"2026-01-29T00:00:00Z", "2026-02-12T00:00:00Z", 8,
			[]string{"2026-01-29T00:00:00Z", "2026-01-31T00:00:00Z", "2026-02-01T00:00:00Z", "2026-02-03T00:00:00Z", "2026-02-05T00:00:00Z", "2026-02-07T00:00:00Z", "2026-02-09T00:00:00Z", "2026-02-11T00:00:00Z"},
			"Jan 29 Jan 31 Feb Feb 3 Feb 5 Feb 7 Feb 9 Feb 11"},
		{"2026-03-01T00:00:00Z", "2026-03-29T00:00:00Z", 4, // Sundays
			[]string{"2026-03-01T00:00:00Z", "2026-03-08T00:00:00Z", "2026-03-15T00:00:00Z", "2026-03-22T00:00:00Z", "2026-03-29T00:00:00Z"},
			"Mar Mar 8 Mar 15 Mar 22 Mar 29"},
		{"2026-01-01T00:00:00.120Z", "2026-01-01T00:00:00.870Z", 5,
			[]string{"2026-01-01T00:00:00.2Z", "2026-01-01T00:00:00.4Z", "2026-01-01T00:00:00.6Z", "2026-01-01T00:00:00.8Z"},
			".200 .400 .600 .800"},
	} {
		s := NewTime(parse(tc.from), parse(tc.to))
		var got, labels []string
		for _, tick := range s.Ticks(tc.count) {
			got = append(got, time.UnixMilli(int64(tick.Value)).UTC().Format(time.RFC3339Nano))
			labels = append(labels, tick.Label)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("ticks of %s to %s: %v, want %v", tc.from, tc.to, got, tc.want)
		}
		if l := strings.Join(labels, " "); l != tc.labels {
			t.Errorf("labels of %s to %s: %q, want %q", tc.from, tc.to, l, tc.labels)
		}
	}
	// Ticks fall on round times of the times' location.
	tokyo := time.FixedZone("JST", 9*3600)
	s := NewTime(time.Date(2026, 3, 1, 0, 0, 0, 0, tokyo), time.Date(2026, 3, 4, 0, 0, 0, 0, tokyo))
	if ts := s.Ticks(3); len(ts) != 4 || ts[1].Label != "Mar 2" {
		t.Errorf("ticks in Tokyo: %+v", ts)
	}
}

func TestStack(t *testing.T) {
	seg := func(v [][][2]float64) [][]Segment {
		out := make([][]Segment, len(v))
		for s := range v {
			for _, p := range v[s] {
				out[s] = append(out[s], Segment{p[0], p[1]})
			}
		}
		return out
	}
	check := func(name string, got [][]Segment, want [][][2]float64) {
		t.Helper()
		w := seg(want)
		for s := range w {
			for i := range w[s] {
				if math.Abs(got[s][i].Lo-w[s][i].Lo) > 1e-9 || math.Abs(got[s][i].Hi-w[s][i].Hi) > 1e-9 {
					t.Errorf("%s: [%d][%d] = %v, want %v", name, s, i, got[s][i], w[s][i])
				}
			}
		}
	}
	// d3's stackOffsetDiverging, stackOffsetExpand and stackOffsetSilhouette.
	check("standard", Stack([][]float64{{3, -1, 2}, {4, 2, -5}, {1, 1, 1}}, StackStandard),
		[][][2]float64{{{0, 3}, {-1, 0}, {0, 2}}, {{3, 7}, {0, 2}, {-5, 0}}, {{7, 8}, {2, 3}, {2, 3}}})
	pos := [][]float64{{3, 1, 2}, {4, 2, 5}, {1, 1, 1}}
	check("normalized", Stack(pos, StackNormalized),
		[][][2]float64{{{0, 0.375}, {0, 0.25}, {0, 0.25}}, {{0.375, 0.875}, {0.25, 0.75}, {0.25, 0.875}}, {{0.875, 1}, {0.75, 1}, {0.875, 1}}})
	check("center", Stack(pos, StackCenter),
		[][][2]float64{{{-4, -1}, {-2, -1}, {-4, -2}}, {{-1, 3}, {-1, 1}, {-2, 3}}, {{3, 4}, {1, 2}, {3, 4}}})
	// Shorter series and NaN take no room.
	got := Stack([][]float64{{1, math.NaN()}, {2}}, StackStandard)
	if got[0][1] != (Segment{0, 0}) || got[1][1] != (Segment{0, 0}) || got[1][0] != (Segment{1, 3}) {
		t.Errorf("ragged stack: %v", got)
	}
}

func TestPie(t *testing.T) {
	want := [][2]float64{{0, 0.6383185307179586}, {0.6383185307179586, 1.8949555921538759}, {1.8949555921538759, 3.7699111843077517}, {3.7699111843077517, 3.7899111843077518}, {3.7899111843077518, 6.283185307179586}}
	got := Pie([]float64{1, 2, 3, 0, 4}, 0.02)
	for i, w := range want {
		if math.Abs(got[i].Start-w[0]) > 1e-9 || math.Abs(got[i].End-w[1]) > 1e-9 {
			t.Errorf("slice %d = %v, want %v", i, got[i], w)
		}
	}
	// Twelve o'clock, three, six and nine.
	slices := Pie([]float64{1, 1, 1, 1}, 0)
	for i, p := range [][2]float32{{1, -10}, {10, 1}, {-1, 10}, {-10, -1}} {
		if j, ok := SliceAt(slices, p[0], p[1], 5, 20); !ok || j != i {
			t.Errorf("SliceAt(%v) = %d, %v, want %d", p, j, ok, i)
		}
	}
	if _, ok := SliceAt(slices, 1, 1, 5, 20); ok {
		t.Error("the hole of a donut has no slice")
	}
	if x, y := Polar(0, 0, 10, math.Pi/2); math.Abs(float64(x-10)) > 1e-5 || math.Abs(float64(y)) > 1e-5 {
		t.Errorf("Polar at a quarter turn = %v, %v", x, y)
	}
}

func TestNearest(t *testing.T) {
	xs := []float32{0, 0.25, 0.5, 1}
	for _, tc := range []struct {
		f float32
		i int
	}{{-1, 0}, {0.1, 0}, {0.125, 0}, {0.2, 1}, {0.4, 2}, {0.8, 3}, {2, 3}} {
		if i := Nearest(xs, tc.f); i != tc.i {
			t.Errorf("Nearest(%v) = %d, want %d", tc.f, i, tc.i)
		}
	}
	if Nearest(nil, 0.5) != -1 {
		t.Error("no points, no nearest")
	}
}

func TestCurves(t *testing.T) {
	pts := [][2]float32{{0, 0}, {10, 10}, {20, 10}, {30, 0}}
	// Monotone never goes past the values of the points it joins: at the
	// flat stretch, the control points stay at its height.
	segs := curve(pts, CurveMonotone)
	if len(segs) != 3 || !segs[1].cubic || segs[1].c1[1] != 10 || segs[1].c2[1] != 10 {
		t.Errorf("monotone: %+v", segs)
	}
	if segs := curve(pts, CurveStepAfter); len(segs) != 6 || segs[0].to != [2]float32{10, 0} {
		t.Errorf("step after: %+v", segs)
	}
	if segs := curve(pts, CurveStepBefore); segs[0].to != [2]float32{0, 10} {
		t.Errorf("step before: %+v", segs)
	}
	if segs := curve(pts, CurveStep); len(segs) != 9 || segs[0].to != [2]float32{5, 0} || segs[1].to != [2]float32{5, 10} {
		t.Errorf("step: %+v", segs)
	}
	if segs := curve(pts[:2], CurveMonotone); len(segs) != 1 || segs[0].cubic {
		t.Errorf("two points join straight: %+v", segs)
	}
}

// pixel reads the color at (x, y) of a tester's frame, at scale 1.
func pixel(tt *ui.Tester, x, y int) [4]uint8 {
	c := tt.Image().RGBAAt(x, y)
	return [4]uint8{c.R, c.G, c.B, c.A}
}

func TestPaths(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	xs := []float32{0, 0.5, 1}
	view := func(c *ui.Context) {
		ui.Box(c).Size(100, 100).Draw(func(p *ui.Painter, r ui.Rect) {
			// An area under a line from (0, 0.5) up to (1, 1), broken
			// nowhere, and a line with a gap.
			p.FillPath(Area(r, xs, nil, []float32{0.5, 0.75, 1}, CurveLinear), red)
		})
		ui.Box(c).Size(100, 100).Draw(func(p *ui.Painter, r ui.Rect) {
			p.FillPath(Sector(r.X+50, r.Y+50, 20, 40, Slice{Start: 0, End: math.Pi}), red)
		})
	}
	tt := ui.NewTester(view, 100, 200)
	for _, tc := range []struct {
		x, y int
		in   bool
	}{{50, 90, true}, {10, 40, false}, {90, 20, true}, {50, 30, true}, // the area
		{80, 150, true}, {50, 150, false}, {20, 150, false}, {50, 105, false}, {50, 195, false}} { // the right half of a donut
		if got := pixel(tt, tc.x, tc.y) == [4]uint8{255, 0, 0, 255}; got != tc.in {
			t.Errorf("(%d, %d) filled %v, want %v", tc.x, tc.y, got, tc.in)
		}
	}
	// A NaN breaks a line into two.
	var path *ui.Path
	ui.Render(func(c *ui.Context) {
		ui.Box(c).Size(10, 10).Draw(func(p *ui.Painter, r ui.Rect) {
			path = Line(r, []float32{0, 0.25, 0.5, 0.75}, []float32{0, 1, float32(math.NaN()), 1}, CurveLinear)
		})
	}, 10, 10, 1)
	if path == nil {
		t.Fatal("not drawn")
	}
}

func TestPlotLayout(t *testing.T) {
	x := NewBand("Mon", "Tue", "Wed", "Thu").Padding(0.2, 0)
	y := NewLinear(0, 100)
	view := func(c *ui.Context) {
		Plot(c, Axes{Left: y.Ticks(2), Bottom: x.Ticks()}, func(area ui.Element) {
			area.Label("area")
			Rect(c, x.Band(1), y.Span(0, 50)).Label("bar")
			Point(c, 0.5, 0.5, 10, 10).Label("dot")
			RuleY(c, 1, 2).Label("top rule")
			Anchor(c, 1, 0, ui.End, ui.End, func() { ui.Text(c, "corner").NoWrap() })
		}).Size(400, 200).Margin(20)
	}
	tt := ui.NewTester(view, 440, 240)
	area, _ := tt.Find("area")
	label100, _ := tt.Find("100")
	mon, _ := tt.Find("Mon")
	if area.X != 20+label100.W || label100.X != 20 {
		t.Errorf("the area starts at %v, after the widest label, %+v", area.X, label100)
	}
	if mid := label100.Y + label100.H/2; math.Abs(float64(mid-area.Y)) > 0.5 {
		t.Errorf("the label 100 is centered at %v, not on the top of the area at %v", mid, area.Y)
	}
	if area.Y+area.H != mon.Y || area.Y+area.H+mon.H != 220 {
		t.Errorf("the area %+v and the labels below it %+v", area, mon)
	}
	b := x.Band(0)
	if mid := mon.X + mon.W/2; math.Abs(float64(mid-(area.X+area.W*b.Mid()))) > 0.5 {
		t.Errorf("Mon is centered at %v, not on its band at %v", mid, area.X+area.W*b.Mid())
	}
	bar, _ := tt.Find("bar")
	b = x.Band(1)
	want := ui.Rect{X: area.X + area.W*b.Lo, Y: area.Y + area.H/2, W: area.W * (b.Hi - b.Lo), H: area.H / 2}
	if math.Abs(float64(bar.X-want.X)) > 0.5 || math.Abs(float64(bar.Y-want.Y)) > 0.5 || math.Abs(float64(bar.W-want.W)) > 0.5 || math.Abs(float64(bar.H-want.H)) > 0.5 {
		t.Errorf("bar at %+v, want %+v", bar, want)
	}
	dot, _ := tt.Find("dot")
	if dot.X+5 != area.X+area.W/2 || dot.Y+5 != area.Y+area.H/2 {
		t.Errorf("dot at %+v, not centered in %+v", dot, area)
	}
	rule, _ := tt.Find("top rule")
	if rule.Y != area.Y-1 || rule.W != area.W || rule.H != 2 {
		t.Errorf("rule at %+v", rule)
	}
	corner, _ := tt.Find("corner")
	if math.Abs(float64(corner.X+corner.W-(area.X+area.W))) > 0.5 || math.Abs(float64(corner.Y+corner.H-(area.Y+area.H))) > 0.5 {
		t.Errorf("corner at %+v, not ending at the bottom right of %+v", corner, area)
	}
}

func TestSelect(t *testing.T) {
	xs := []float32{0.1, 0.5, 0.9}
	selected := -1
	changes := 0
	view := func(c *ui.Context) {
		Plot(c, Axes{}, func(area ui.Element) {
			area.Label("plot")
			if Select(c, area, &selected, xs, func(i int) string { return []string{"one", "two", "three"}[i] }) {
				changes++
			}
		}).Size(200, 100)
	}
	tt := ui.NewTester(view, 200, 100)
	tt.Move(95, 50)
	if selected != 1 {
		t.Fatalf("pointer at the middle chose %d", selected)
	}
	tt.Move(190, 50)
	if selected != 2 {
		t.Fatalf("pointer at the right chose %d", selected)
	}
	tt.Move(300, 50) // out of the window, and of the plot
	if selected != -1 {
		t.Fatalf("pointer gone, still %d", selected)
	}
	if changes != 3 {
		t.Errorf("%d changes, want 3", changes)
	}
	// The keyboard, once the plot has the focus.
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("plot") {
		t.Fatal("Tab does not focus the plot")
	}
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyRight)
	if selected != 1 {
		t.Errorf("two rights chose %d", selected)
	}
	tt.Key(0, ui.KeyEnd)
	tt.Key(0, ui.KeyRight) // stays at the last
	if selected != 2 {
		t.Errorf("End chose %d", selected)
	}
	tt.Key(0, ui.KeyHome)
	if a := tt.Announcements(); !slices.Equal(a, []string{"one", "two", "three", "one"}) {
		t.Errorf("announced %q", a)
	}
	tt.Key(0, ui.KeyEscape)
	if selected != -1 {
		t.Errorf("Escape left %d", selected)
	}
	tt.Key(0, ui.KeyLeft) // from none, to the last
	if selected != 2 {
		t.Errorf("left from none chose %d", selected)
	}
}
